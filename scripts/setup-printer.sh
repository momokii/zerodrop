#!/usr/bin/env bash
#
# ZeroDrop Terminal — Thermal Printer Automatic Setup
#
# Detects your USB thermal printer, adds your user to the required
# groups, generates + installs udev rules, and verifies everything
# works. Designed to be safe to run multiple times.
#
# Usage:
#   ./scripts/setup-printer.sh          # run the full setup
#   ./scripts/setup-printer.sh --dry-run  # preview without making changes
#   ./scripts/setup-printer.sh --help     # show this message
#
# Options:
#   --dry-run   Show what would be done without actually doing it
#   --help      Show this help and exit
#

set -euo pipefail

# ──────────────────────────────────────────────
# Configuration
# ──────────────────────────────────────────────

UDEV_RULES_FILE="/tmp/99-zerodrop-printer.rules"
UDEV_RULES_DEST="/etc/udev/rules.d/99-zerodrop-printer.rules"
KNOWN_PRINTERS_VID=(
  "1504"  # POS-5890 / Generic ESC/POS
  "04b8"  # Epson TM-T88
  "0416"  # Rongta RP58
  "0456"  # XPrinter XP-58III
  "0493"  # Citizen CT-S310
  "0519"  # Star Micronics TSP650
  "0dd4"  # BCST Printers
  "20d1"  # Gainscha
  "0fe6"  # Zjiang
  "0418"  # Custom VG205
)

# ──────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

log_ok()   { printf "${GREEN}✓${NC} %s\n" "$1"; }
log_warn() { printf "${YELLOW}⚠${NC} %s\n" "$1"; }
log_err()  { printf "${RED}✗${NC} %s\n" "$1" >&2; }
log_info() { printf "${BLUE}→${NC} %s\n" "$1"; }
log_step() { printf "\n${BOLD}%s${NC}\n" "$1"; }
log_section() { printf "\n━━━ %s ━━━\n" "$1"; }

DRY_RUN=false
NEEDS_RELOGIN=false

# ──────────────────────────────────────────────
# Argument parsing
# ──────────────────────────────────────────────

for arg in "$@"; do
  case "$arg" in
    --help|-h)
      head -30 "$0" | tail -28 | sed 's/^#//'
      exit 0
      ;;
    --dry-run)
      DRY_RUN=true
      ;;
    *)
      log_err "Unknown option: $arg"
      echo "Usage: $0 [--dry-run] [--help]"
      exit 1
      ;;
  esac
done

if [ "$DRY_RUN" = true ]; then
  log_warn "DRY RUN MODE — no changes will be made"
  echo ""
fi

# ──────────────────────────────────────────────
# Step 0: OS check
# ──────────────────────────────────────────────

log_section "Step 0: System check"

if [ "$(uname)" != "Linux" ]; then
  log_err "This script only supports Linux. Detected: $(uname)"
  exit 1
fi
log_ok "Linux detected"

if [ "$(id -u)" -eq 0 ]; then
  log_warn "Running as root. You should run this as your normal user (sudo prompts will appear when needed)."
fi

# ──────────────────────────────────────────────
# Step 1: Detect printer hardware
# ──────────────────────────────────────────────

log_section "Step 1: Detecting printer hardware"

DETECTED_DEVICE=""
DETECTED_MODEL=""
DETECTED_VID=""
DETECTED_PID=""

# 1a. Check standard device paths
for path in /dev/usb/lp[0-2] /dev/usblp[0-2] /dev/lp[0-2]; do
  if [ -e "$path" ]; then
    DETECTED_DEVICE="$path"
    log_ok "Found printer device: $path"
    break
  fi
done

# 1b. Try to identify via sysfs or lsusb
if [ -n "$DETECTED_DEVICE" ]; then
  # Try sysfs identification
  DEVICE_NAME=$(basename "$DETECTED_DEVICE")
  for syspath in "/sys/class/usbmisc/$DEVICE_NAME/../../" "/sys/class/printer/$DEVICE_NAME/../../"; do
    if [ -f "$syspath/idVendor" ] && [ -f "$syspath/idProduct" ]; then
      DETECTED_VID=$(cat "$syspath/idVendor" 2>/dev/null | tr -d '[:space:]')
      DETECTED_PID=$(cat "$syspath/idProduct" 2>/dev/null | tr -d '[:space:]')
      break
    fi
  done
elif command -v lsusb &>/dev/null; then
  # No device node but maybe the printer is connected and needs a driver
  log_info "Scanning USB devices with lsusb..."
  LSUSB_OUTPUT=$(lsusb 2>/dev/null || true)
  for VID in "${KNOWN_PRINTERS_VID[@]}"; do
    MATCH=$(echo "$LSUSB_OUTPUT" | grep -i "$VID:" || true)
    if [ -n "$MATCH" ]; then
      DETECTED_VID="$VID"
      while IFS= read -r line; do
        PID_HEX=$(echo "$line" | grep -oiP "$VID:([0-9a-f]+)" | head -1 | sed "s/$VID://i")
        if [ -n "$PID_HEX" ]; then
          DETECTED_PID="$PID_HEX"
          DETECTED_MODEL=$(echo "$line" | grep -oP '\)\s*\K.*' || echo "Unknown ESC/POS printer")
          log_ok "Found printer via USB: $DETECTED_MODEL (VID:PID = $VID:$PID_HEX)"
          log_warn "Device node not found at /dev/usb/lp*. The printer may need a kernel module."
          log_info "Try: sudo modprobe usblp"
          break 2
        fi
      done <<< "$MATCH"
    fi
  done
  if [ -z "$DETECTED_VID" ]; then
    log_warn "No supported thermal printer detected. Run 'lsusb' to check your USB devices."
    log_info "If your printer is listed but not detected, it may not be in the supported list."
    log_info "You can still try the setup — the udev rules include a generic fallback for lp devices."
  fi
else
  log_info "No printer device node found in /dev/usb/lp*"
  log_info "lsusb not available — skipping USB scan."
  log_info "Plug in your printer and check: ls -la /dev/usb/lp0"
fi

# ──────────────────────────────────────────────
# Step 2: Add user to required groups
# ──────────────────────────────────────────────

log_section "Step 2: Group permissions"

CURRENT_USER=$(whoami)
GROUPS_NEEDED=("lp" "dialout")
GROUPS_MISSING=()

for group in "${GROUPS_NEEDED[@]}"; do
  if groups "$CURRENT_USER" 2>/dev/null | grep -qw "$group"; then
    log_ok "User '$CURRENT_USER' is already in '$group' group"
  else
    log_warn "User '$CURRENT_USER' is NOT in '$group' group"
    GROUPS_MISSING+=("$group")
  fi
done

if [ ${#GROUPS_MISSING[@]} -gt 0 ]; then
  GROUP_LIST=$(IFS=,; echo "${GROUPS_MISSING[*]}")
  if [ "$DRY_RUN" = true ]; then
    log_info "[DRY RUN] Would run: sudo usermod -aG $GROUP_LIST $CURRENT_USER"
  else
    log_info "Running: sudo usermod -aG $GROUP_LIST $CURRENT_USER"
    if sudo usermod -aG "$GROUP_LIST" "$CURRENT_USER"; then
      log_ok "Added '$CURRENT_USER' to groups: $GROUP_LIST"
      NEEDS_RELOGIN=true
    else
      log_err "Failed to add user to groups. Run manually: sudo usermod -aG $GROUP_LIST $CURRENT_USER"
    fi
  fi
fi

# ──────────────────────────────────────────────
# Step 3: Generate and install udev rules
# ──────────────────────────────────────────────

log_section "Step 3: udev rules"

RULE_CONTENT='# ZeroDrop Terminal — Thermal printer udev rules
# Grants lp group write access to supported ESC/POS printers

# POS-5890 / Generic ESC/POS
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="1504", ATTRS{idProduct}=="0006", MODE="0660", GROUP="lp"
# Epson TM-T88 series
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="04b8", MODE="0660", GROUP="lp"
# Rongta RP58
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0416", ATTRS{idProduct}=="5011", MODE="0660", GROUP="lp"
# XPrinter XP-58III
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0456", ATTRS{idProduct}=="0808", MODE="0660", GROUP="lp"
# Citizen CT-S310
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0493", ATTRS{idProduct}=="b002", MODE="0660", GROUP="lp"
# Star Micronics TSP650
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0519", ATTRS{idProduct}=="0001", MODE="0660", GROUP="lp"
# Gainscha
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="20d1", ATTRS{idProduct}=="0001", MODE="0660", GROUP="lp"
# Zjiang
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0fe6", ATTRS{idProduct}=="811e", MODE="0660", GROUP="lp"
# BCST Printers
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0dd4", ATTRS{idProduct}=="01a5", MODE="0660", GROUP="lp"
# Custom VG205
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="0418", ATTRS{idProduct}=="0156", MODE="0660", GROUP="lp"
# Generic fallback for all lp printer devices
KERNEL=="lp[0-9]*", SUBSYSTEM=="printer", MODE="0660", GROUP="lp"
KERNEL=="usb/lp[0-9]*", SUBSYSTEM=="usb", MODE="0660", GROUP="lp"
'

if [ -f "$UDEV_RULES_DEST" ]; then
  EXISTING_HASH=$(md5sum "$UDEV_RULES_DEST" 2>/dev/null | cut -d' ' -f1 || sha1sum "$UDEV_RULES_DEST" 2>/dev/null | cut -d' ' -f1)
  NEW_HASH=$(echo "$RULE_CONTENT" | md5sum 2>/dev/null | cut -d' ' -f1 || echo "$RULE_CONTENT" | sha1sum 2>/dev/null | cut -d' ' -f1)
  if [ "$EXISTING_HASH" = "$NEW_HASH" ]; then
    log_ok "udev rules already installed and up to date at $UDEV_RULES_DEST"
    SKIP_UDEV=true
  else
    log_info "udev rules exist but differ — will update"
    SKIP_UDEV=false
  fi
else
  SKIP_UDEV=false
fi

if [ "$SKIP_UDEV" = false ]; then
  if [ "$DRY_RUN" = true ]; then
    log_info "[DRY RUN] Would write udev rules to: $UDEV_RULES_FILE"
    log_info "[DRY RUN] Would install to: $UDEV_RULES_DEST"
    log_info "[DRY RUN] Would run: sudo udevadm control --reload-rules && sudo udevadm trigger"
  else
    # Write rules to temp file
    echo "$RULE_CONTENT" > "$UDEV_RULES_FILE"
    log_ok "Generated udev rules at $UDEV_RULES_FILE ($(wc -l < "$UDEV_RULES_FILE") lines)"

    # Install rules
    log_info "Installing udev rules (requires sudo)..."
    if sudo cp "$UDEV_RULES_FILE" "$UDEV_RULES_DEST"; then
      log_ok "Installed udev rules to $UDEV_RULES_DEST"
      log_info "Reloading udev rules..."
      sudo udevadm control --reload-rules 2>/dev/null || log_warn "Could not reload udev rules"
      sudo udevadm trigger 2>/dev/null || log_warn "Could not trigger udev"
      log_ok "udev rules reloaded"
    else
      log_err "Failed to install udev rules. Try manually:"
      echo "  sudo cp $UDEV_RULES_FILE $UDEV_RULES_DEST"
      echo "  sudo udevadm control --reload-rules"
      echo "  sudo udevadm trigger"
    fi
  fi
fi

# ──────────────────────────────────────────────
# Step 4: Verify write access
# ──────────────────────────────────────────────

log_section "Step 4: Verifying write access"

if [ -z "$DETECTED_DEVICE" ]; then
  log_warn "No printer device node found to test write access."
  log_info "Plug in your USB thermal printer and run: ls -la /dev/usb/lp0"
  log_info "Then re-run this script."
else
  if [ -w "$DETECTED_DEVICE" ]; then
    log_ok "User '$CURRENT_USER' can write to $DETECTED_DEVICE"
  else
    log_warn "User '$CURRENT_USER' cannot write to $DETECTED_DEVICE"
    DEVICE_PERMS=$(ls -la "$DETECTED_DEVICE" | awk '{print $1, $3, $4}')
    log_info "Device permissions: $DEVICE_PERMS"

    # Try with sudo to check if the device is writable at all
    if sudo test -w "$DETECTED_DEVICE" 2>/dev/null; then
      log_info "The device IS writable with sudo — group membership hasn't taken effect yet."
      if [ "$NEEDS_RELOGIN" = true ]; then
        log_warn "You need to LOG OUT and log back in for group changes to apply."
      else
        log_warn "Your user may not be in the 'lp' group. Run: groups | grep lp"
      fi
    else
      log_err "Device exists but is not writable even with sudo. Check cable and printer power."
    fi
  fi
fi

# ──────────────────────────────────────────────
# Step 5: Configure .env file
# ──────────────────────────────────────────────

log_section "Step 5: Configuring .env file"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$PROJECT_DIR/.env"
ENV_EXAMPLE="$PROJECT_DIR/.env.example"

if [ -n "$DETECTED_DEVICE" ]; then
  DEV_VALUE="$DETECTED_DEVICE"
else
  DEV_VALUE=""  # empty = auto-detect
fi

ENV_CHANGED=false

if [ ! -f "$ENV_FILE" ]; then
  if [ -f "$ENV_EXAMPLE" ]; then
    log_info "Creating $ENV_FILE from $(basename "$ENV_EXAMPLE")"
    if [ "$DRY_RUN" = false ]; then
      cp "$ENV_EXAMPLE" "$ENV_FILE"
      log_ok "Created $ENV_FILE from template"
    else
      log_info "[DRY RUN] Would create $ENV_FILE from $(basename "$ENV_EXAMPLE")"
    fi
    ENV_CHANGED=true
  else
    log_info "Creating minimal $ENV_FILE"
    if [ "$DRY_RUN" = false ]; then
      touch "$ENV_FILE"
      log_ok "Created $ENV_FILE"
    else
      log_info "[DRY RUN] Would create $ENV_FILE"
    fi
    ENV_CHANGED=true
  fi
else
  log_ok "$ENV_FILE already exists"
fi

# Helper: update a single variable in .env
# Usage: update_env_var VAR_NAME VALUE
# Handles: commented lines, uncommented lines, missing lines
update_env_var() {
  local var_name="$1"
  local var_value="$2"
  local file="$ENV_FILE"

  # Escape value for sed replacement (escapes | / & \)
  local escaped_value
  escaped_value=$(printf '%s\n' "$var_value" | sed 's|[\/&]|\\&|g')

  if grep -qs "^[[:space:]]*#\?[[:space:]]*${var_name}=" "$file" 2>/dev/null; then
    # Line exists (commented or not) — replace it uncommented
    if [ "$DRY_RUN" = false ]; then
      sed -i "s|^[[:space:]]*#\?[[:space:]]*${var_name}=.*|${var_name}=${escaped_value}|" "$file"
    fi
  else
    if [ "$DRY_RUN" = false ]; then
      echo "${var_name}=${var_value}" >> "$file"
    fi
  fi

  if [ "$DRY_RUN" = false ]; then
    log_ok "Set ${var_name}=${var_value} in $(basename "$ENV_FILE")"
  else
    log_info "[DRY RUN] Would set ${var_name}=${var_value} in $(basename "$ENV_FILE")"
  fi
  ENV_CHANGED=true
}

update_env_var "PRINTER_TYPE" "usb"

if [ -n "$DEV_VALUE" ]; then
  update_env_var "PRINTER_DEVICE" "$DEV_VALUE"
else
  update_env_var "PRINTER_DEVICE" ""
fi

if [ "$ENV_CHANGED" = false ]; then
  log_ok "Printer variables already configured correctly in $(basename "$ENV_FILE")"
fi

# ──────────────────────────────────────────────
# Summary
# ──────────────────────────────────────────────

log_section "Summary"

if [ "$DRY_RUN" = true ]; then
  echo "  ❖ Dry run complete. No changes were made."
  echo "  ❖ Run without --dry-run to apply."
  echo ""
  exit 0
fi

ENV_STATUS=""
if [ -f "$ENV_FILE" ]; then
  ENV_PRINTER_TYPE=$(grep -s '^PRINTER_TYPE=' "$ENV_FILE" | head -1 | cut -d= -f2 || true)
  ENV_PRINTER_DEVICE=$(grep -s '^PRINTER_DEVICE=' "$ENV_FILE" | head -1 | cut -d= -f2- || true)
  if [ "$ENV_PRINTER_TYPE" = "usb" ]; then
    ENV_STATUS="✅ configured (PRINTER_TYPE=usb"
    if [ -n "$ENV_PRINTER_DEVICE" ]; then
      ENV_STATUS="$ENV_STATUS, PRINTER_DEVICE=$ENV_PRINTER_DEVICE)"
    else
      ENV_STATUS="$ENV_STATUS, auto-detect)"
    fi
  else
    ENV_STATUS="⚠️  PRINTER_TYPE not set to usb"
  fi
else
  ENV_STATUS="⚠️  not found"
fi

echo "  User groups:   $( (groups "$CURRENT_USER" 2>/dev/null | grep -qw "lp" && echo "✅ lp ok") || echo "⚠️  lp missing")"
echo "                 $( (groups "$CURRENT_USER" 2>/dev/null | grep -qw "dialout" && echo "✅ dialout ok") || echo "⚠️  dialout missing")"
echo "  udev rules:    $( [ -f "$UDEV_RULES_DEST" ] && echo "✅ installed" || echo "⚠️  not installed")"
echo "  .env config:   $ENV_STATUS"
echo "  Device node:   $([ -n "$DETECTED_DEVICE" ] && echo "$DETECTED_DEVICE" || echo "none found")"
echo "  Write access:  $([ -n "$DETECTED_DEVICE" ] && [ -w "$DETECTED_DEVICE" ] && echo "✅ yes" || echo "⚠️  no")"
echo ""

if [ "$NEEDS_RELOGIN" = true ]; then
  log_warn "Group changes require a new login session to take effect."
  echo ""
  echo "  ${BOLD}Next step:${NC}"
  echo "  1. Log out and log back in"
  echo "  2. Run the app (reads .env automatically):"
  echo "       ./bin/zerodrop"
  echo "     Or with Docker:"
  echo "       make docker-up           # dev mode (reads .env)"
  echo "       make docker-up-prod      # production mode (reads .env)"
  echo "  3. Verify: curl -s http://localhost:8080/health"
  echo ""
elif [ -n "$DETECTED_DEVICE" ] && [ -w "$DETECTED_DEVICE" ]; then
  echo "  ${BOLD}Setup complete!${NC} You can now run ZeroDrop with your printer:"
  echo ""
  echo "    ./bin/zerodrop                        # reads .env automatically"
  echo "    make docker-up                        # Docker (dev mode)"
  echo "    make docker-up-prod                   # Docker (production mode)"
  echo ""
  echo "  Verify:"
  echo "    curl -s http://localhost:8080/health | python3 -m json.tool"
  echo ""
else
  echo "  ${BOLD}Partial setup.${NC} Address the warnings above, then re-run this script."
  echo ""

  if [ -z "$DETECTED_DEVICE" ]; then
    echo "  ${BOLD}Printer not detected. Tips:${NC}"
    echo "  • Make sure the printer is plugged in via USB and powered on"
    echo "  • Check kernel messages: dmesg | tail -20 | grep -i usb"
    echo "  • Check USB devices: lsusb | grep -i \"1504\\|04b8\\|0416\\|0456\""
    echo "  • Load USB printer module: sudo modprobe usblp"
    echo ""
  fi
fi

if [ -f "$ENV_FILE" ] && grep -qs '^PRINTER_TYPE=usb' "$ENV_FILE"; then
  echo "  ${BOLD}Docker note:${NC} docker-compose files now read"
  echo "  PRINTER_TYPE and PRINTER_DEVICE from .env automatically."
  echo "  No manual editing needed."
  echo ""
fi
