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
DEVICE_NODE_WAIT=3  # seconds to wait for device node after binding
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
RAW_USB_MODE=false

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
# Helper: scan all possible printer device nodes
# ──────────────────────────────────────────────

scan_device_nodes() {
  # Check standard usblp device paths
  for path in /dev/usb/lp[0-2] /dev/usblp[0-2]; do
    if [ -e "$path" ]; then
      echo "$path"
      return 0
    fi
  done

  # Check serial device paths (some USB-to-serial thermal printers)
  for path in /dev/ttyUSB[0-3]; do
    if [ -e "$path" ]; then
      # Only return if we have reason to believe it's a printer
      # (checked by caller with sysfs VID matching)
      echo "$path"
      return 0
    fi
  done

  return 1
}

# ──────────────────────────────────────────────
# Helper: try to bind usblp driver to a USB device
# ──────────────────────────────────────────────

try_bind_usblp() {
  local target_vid="$1"
  local target_pid="$2"
  local found=false

  for dev_dir in /sys/bus/usb/devices/*/; do
    [ -f "$dev_dir/idVendor" ] || continue
    local vid pid
    vid=$(cat "$dev_dir/idVendor" 2>/dev/null | tr -d '[:space:]')
    pid=$(cat "$dev_dir/idProduct" 2>/dev/null | tr -d '[:space:]')
    [ "$vid" = "$target_vid" ] && [ "$pid" = "$target_pid" ] || continue

    found=true
    local dev_name
    dev_name="${dev_dir%/}"
    dev_name="${dev_name##*/}"
    log_info "Found USB device at sysfs: $dev_name (VID:PID = $vid:$pid)"

    # Check if usblp is already bound to any interface of this device
    for bound in /sys/bus/usb/drivers/usblp/*/; do
      [ -d "$bound" ] || continue
      local bound_link
      bound_link=$(readlink -f "$bound" 2>/dev/null || true)
      if [ -n "$bound_link" ] && echo "$bound_link" | grep -q "$dev_name"; then
        log_ok "usblp driver is already bound to this device"
        return 0
      fi
    done

    # Try binding each interface (typically 1-4:1.0 format)
    for intf_dir in "$dev_dir"*:*.*/; do
      [ -d "$intf_dir" ] || continue
      local intf_name
      intf_name="${intf_dir%/}"
      intf_name="${intf_name##*/}"

      # Skip if already bound to usblp
      if [ -L "${intf_dir}driver" ]; then
        local current_driver
        current_driver=$(readlink "${intf_dir}driver" 2>/dev/null | xargs basename 2>/dev/null || true)
        if [ "$current_driver" = "usblp" ]; then
          log_ok "usblp already bound to interface $intf_name"
          return 0
        fi
      fi

      if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would bind usblp to $intf_name"
        continue
      fi

      # Method 1: Direct bind
      log_info "Method 1: Direct bind to $intf_name..."
      if echo "$intf_name" | sudo tee /sys/bus/usb/drivers/usblp/bind 2>/dev/null | grep -q "$intf_name"; then
        log_ok "Direct bind succeeded for $intf_name"
        return 0
      fi
      local bind_err
      bind_err=$(echo "$intf_name" | sudo tee /sys/bus/usb/drivers/usblp/bind 2>&1 >/dev/null || true)
      log_info "Direct bind result: ${bind_err:-no error message}"

      # Method 2: driver_override (forces kernel to use this driver)
      log_info "Method 2: driver_override for $intf_name..."
      if echo "usblp" | sudo tee "${intf_dir}driver_override" 2>/dev/null | grep -q "usblp"; then
        # Trigger probe by writing to bind
        if echo "$intf_name" | sudo tee /sys/bus/usb/drivers/usblp/bind 2>/dev/null | grep -q "$intf_name"; then
          log_ok "driver_override bind succeeded for $intf_name"
          return 0
        fi
        log_info "driver_override set but bind still failed"
        # Clear override
        echo "" | sudo tee "${intf_dir}driver_override" >/dev/null 2>&1 || true
      fi

      # Method 3: Unbind from current driver, then bind to usblp
      if [ -L "${intf_dir}driver" ]; then
        local current_driver
        current_driver=$(readlink "${intf_dir}driver" 2>/dev/null | xargs basename 2>/dev/null || true)
        if [ -n "$current_driver" ] && [ "$current_driver" != "usblp" ]; then
          log_info "Method 3: Unbind from '$current_driver', then rebind to usblp..."
          echo "$intf_name" | sudo tee "/sys/bus/usb/drivers/${current_driver}/unbind" >/dev/null 2>&1 || true
          sleep 0.5
          if echo "$intf_name" | sudo tee /sys/bus/usb/drivers/usblp/bind >/dev/null 2>&1; then
            log_ok "Rebind succeeded for $intf_name"
            return 0
          fi
          # Restore original driver
          echo "$intf_name" | sudo tee "/sys/bus/usb/drivers/${current_driver}/bind" >/dev/null 2>&1 || true
        fi
      fi
    done

    # Method 4: Add VID PID to usblp's new_id table (triggers re-probe)
    log_info "Method 4: Adding $target_vid $target_pid to usblp new_id table..."
    if [ "$DRY_RUN" = false ]; then
      echo "$target_vid $target_pid" | sudo tee /sys/bus/usb/drivers/usblp/new_id >/dev/null 2>&1
      sleep 1
      # Check if it bound after new_id
      for intf_dir in "$dev_dir"*:*.*/; do
        [ -d "$intf_dir" ] || continue
        if [ -L "${intf_dir}driver" ]; then
          local drv
          drv=$(readlink "${intf_dir}driver" 2>/dev/null | xargs basename 2>/dev/null || true)
          if [ "$drv" = "usblp" ]; then
            log_ok "new_id triggered successful bind"
            return 0
          fi
        fi
      done
      log_info "new_id did not trigger auto-bind"
    fi
  done

  if [ "$found" = true ]; then
    log_warn "Could not bind usblp driver to the detected USB device"
    return 1
  else
    log_warn "USB device not found in sysfs (may have disconnected)"
    return 1
  fi
}

# ──────────────────────────────────────────────
# Step 1: Detect printer hardware
# ──────────────────────────────────────────────

log_section "Step 1: Detecting printer hardware"

DETECTED_DEVICE=""
DETECTED_MODEL=""
DETECTED_VID=""
DETECTED_PID=""
USB_DETECTED_BUT_NO_NODE=false

# 1a. Check standard device paths first
DETECTED_DEVICE=$(scan_device_nodes || true)
if [ -n "$DETECTED_DEVICE" ]; then
  log_ok "Found printer device: $DETECTED_DEVICE"
  # Try to identify via sysfs
  DEVICE_NAME=$(basename "$DETECTED_DEVICE")
  for syspath in "/sys/class/usbmisc/$DEVICE_NAME/../../" "/sys/class/printer/$DEVICE_NAME/../../"; do
    if [ -f "${syspath}idVendor" ] && [ -f "${syspath}idProduct" ]; then
      DETECTED_VID=$(cat "${syspath}idVendor" 2>/dev/null | tr -d '[:space:]')
      DETECTED_PID=$(cat "${syspath}idProduct" 2>/dev/null | tr -d '[:space:]')
      log_ok "Identified: VID:PID = $DETECTED_VID:$DETECTED_PID"
      break
    fi
  done
fi

# 1b. If no device node, scan USB bus with lsusb
if [ -z "$DETECTED_DEVICE" ]; then
  if command -v lsusb &>/dev/null; then
    log_info "No device node found. Scanning USB bus..."
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
            USB_DETECTED_BUT_NO_NODE=true
            break 2
          fi
        done <<< "$MATCH"
      fi
    done

    if [ -z "$DETECTED_VID" ]; then
      log_warn "No supported thermal printer detected on USB bus."
      log_info "If your printer is connected, check: lsusb"
      log_info "The udev rules include a generic fallback for lp devices."
    fi
  else
    log_warn "lsusb not available — cannot scan USB bus."
    log_info "Plug in your printer and check: ls -la /dev/usb/lp0"
  fi
fi

# 1c. If printer detected via lsusb but no device node, try to create one
if [ "$USB_DETECTED_BUT_NO_NODE" = true ] && [ -z "$DETECTED_DEVICE" ]; then
  log_info "Printer detected on USB bus but no device node exists."
  log_info "Will attempt to create one (requires sudo)."

  # Ensure usblp module is loaded
  if ! lsmod | grep -q "^usblp"; then
    log_info "Loading usblp kernel module..."
    if [ "$DRY_RUN" = false ]; then
      sudo modprobe usblp 2>/dev/null && log_ok "usblp module loaded" || log_warn "Could not load usblp module"
    else
      log_info "[DRY RUN] Would run: sudo modprobe usblp"
    fi
  else
    log_ok "usblp kernel module is already loaded"
  fi

  # --- Attempt A: usbreset (force USB re-enumeration) ---
  # Many USB-to-parallel bridge chips (e.g. Zjiang 0fe6:811e) enumerate as
  # USB Printer class 07 but the kernel doesn't auto-bind usblp on first plug.
  # Resetting the device forces a clean re-enumeration which usually fixes it.
  if command -v usbreset &>/dev/null; then
    USBRESET_TARGET=$(lsusb 2>/dev/null | grep -i "${DETECTED_VID}:${DETECTED_PID}" | head -1 || true)
    if [ -n "$USBRESET_TARGET" ]; then
      USBRESET_BUS=$(echo "$USBRESET_TARGET" | awk '{print $2}')
      USBRESET_DEV=$(echo "$USBRESET_TARGET" | awk '{print $4}' | tr -d ':')
      USBRESET_PATH="/dev/bus/usb/${USBRESET_BUS}/${USBRESET_DEV}"

      if [ -e "$USBRESET_PATH" ]; then
        if [ "$DRY_RUN" = false ]; then
          log_info "Resetting USB device (usbreset $USBRESET_PATH)..."
          if sudo usbreset "$USBRESET_PATH" 2>/dev/null; then
            log_ok "USB reset succeeded. Waiting for device to re-enumerate..."
            sleep 2
            DETECTED_DEVICE=$(scan_device_nodes || true)
            if [ -n "$DETECTED_DEVICE" ]; then
              log_ok "Device node appeared after usbreset: $DETECTED_DEVICE"
              USB_DETECTED_BUT_NO_NODE=false
            fi
          else
            log_warn "usbreset failed (device may not support it)"
          fi
        else
          log_info "[DRY RUN] Would run: sudo usbreset $USBRESET_PATH"
        fi
      fi
    fi
  fi

  # --- Attempt B: manual sysfs driver binding ---
  if [ "$USB_DETECTED_BUT_NO_NODE" = true ] && [ "$DRY_RUN" = false ]; then
    log_info "Attempting manual usblp driver binding via sysfs..."
    if try_bind_usblp "$DETECTED_VID" "$DETECTED_PID"; then
      log_ok "Driver binding succeeded. Waiting for device node..."
      local_wait=0
      while [ $local_wait -lt "$DEVICE_NODE_WAIT" ]; do
        sleep 1
        local_wait=$((local_wait + 1))
        DETECTED_DEVICE=$(scan_device_nodes || true)
        if [ -n "$DETECTED_DEVICE" ]; then
          log_ok "Device node appeared: $DETECTED_DEVICE"
          USB_DETECTED_BUT_NO_NODE=false
          break
        fi
      done

      if [ -z "$DETECTED_DEVICE" ]; then
        log_warn "Driver bound but device node did not appear within ${DEVICE_NODE_WAIT}s"
        log_info "The udev rules (Step 3) may help. Continue with setup and re-plug the printer."
      fi
    else
      log_warn "Driver binding did not succeed."
      log_info "This is expected for some USB-to-parallel bridge chips (e.g. Zjiang 0fe6:811e)."
      log_info "ZeroDrop will use direct USB bulk transfers instead of usblp."
      USB_DETECTED_BUT_NO_NODE=false  # Not a failure — raw USB mode will be used
      RAW_USB_MODE=true
    fi
  elif [ "$USB_DETECTED_BUT_NO_NODE" = true ] && [ "$DRY_RUN" = true ]; then
    log_info "[DRY RUN] Would attempt usbreset + usblp driver binding for $DETECTED_VID:$DETECTED_PID"
    RAW_USB_MODE=true
  fi
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
  if [ "$USB_DETECTED_BUT_NO_NODE" = true ]; then
    log_warn "Printer found on USB bus but no device node available."
    log_info "Possible fixes (in order):"
    log_info "  1. Re-plug the USB cable (resets the device and triggers udev)"
    log_info "  2. Power-cycle the printer (off then on)"
    log_info "  3. Run: sudo usbreset \"$(echo "$DETECTED_MODEL" | head -1)\""
    log_info "  4. Run: sudo modprobe -r usblp && sudo modprobe usblp"
    log_info "  5. Then re-run this script"
  else
    log_warn "No printer device node found to test write access."
    log_info "Plug in your USB thermal printer and run: ls -la /dev/usb/lp0"
    log_info "Then re-run this script."
  fi
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

# For raw USB mode, detect and write the raw device path
# (e.g., /dev/bus/usb/001/090) so docker-compose can map it
if [ "$RAW_USB_MODE" = true ] && [ -n "$DETECTED_VID" ] && [ -n "$DETECTED_PID" ]; then
  RAW_DEVICE_PATH=""
  for dev_dir in /sys/bus/usb/devices/*/; do
    [ -f "$dev_dir/idVendor" ] || continue
    vid=$(cat "$dev_dir/idVendor" 2>/dev/null | tr -d '[:space:]')
    pid=$(cat "$dev_dir/idProduct" 2>/dev/null | tr -d '[:space:]')
    [ "$vid" = "$DETECTED_VID" ] && [ "$pid" = "$DETECTED_PID" ] || continue
    bus=$(cat "$dev_dir/busnum" 2>/dev/null | tr -d '[:space:]')
    dev=$(cat "$dev_dir/devnum" 2>/dev/null | tr -d '[:space:]')
    if [ -n "$bus" ] && [ -n "$dev" ]; then
      RAW_DEVICE_PATH="/dev/bus/usb/$(printf '%03d' "$bus")/$(printf '%03d' "$dev")"
      break
    fi
  done
  if [ -n "$RAW_DEVICE_PATH" ]; then
    update_env_var "USB_RAW_DEVICE" "$RAW_DEVICE_PATH"
  fi
else
  # Clear USB_RAW_DEVICE if not in raw USB mode
  if grep -qs '^USB_RAW_DEVICE=' "$ENV_FILE" 2>/dev/null; then
    update_env_var "USB_RAW_DEVICE" ""
  fi
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

echo "  USB detected:  $([ -n "$DETECTED_VID" ] && echo "$DETECTED_VID:$DETECTED_PID $DETECTED_MODEL" || echo "none")"
echo "  User groups:   $( (groups "$CURRENT_USER" 2>/dev/null | grep -qw "lp" && echo "✅ lp ok") || echo "⚠️  lp missing")"
echo "                 $( (groups "$CURRENT_USER" 2>/dev/null | grep -qw "dialout" && echo "✅ dialout ok") || echo "⚠️  dialout missing")"
echo "  udev rules:    $( [ -f "$UDEV_RULES_DEST" ] && echo "✅ installed" || echo "⚠️  not installed")"
echo "  .env config:   $ENV_STATUS"
echo "  Device node:   $([ -n "$DETECTED_DEVICE" ] && echo "$DETECTED_DEVICE" || echo "none found")"
echo "  Write access:  $([ -n "$DETECTED_DEVICE" ] && [ -w "$DETECTED_DEVICE" ] && echo "✅ yes" || ([ "$RAW_USB_MODE" = true ] && echo "N/A (raw USB)"))"
echo "  USB mode:      $([ "$RAW_USB_MODE" = true ] && echo "direct bulk (no usblp)" || echo "usblp driver")"
if [ "$RAW_USB_MODE" = true ] && [ -n "$RAW_DEVICE_PATH" ]; then
  echo "  Raw USB path:  $RAW_DEVICE_PATH"
fi
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
elif [ "$RAW_USB_MODE" = true ]; then
  echo "  ${BOLD}Printer ready via direct USB mode.${NC}"
  echo "  The usblp driver cannot bind to this chip (common with Zjiang/YICHIP printers)."
  echo "  ZeroDrop will bypass usblp and send data directly to the USB device."
  echo ""
  echo "  Run ZeroDrop:"
  echo "    ./bin/zerodrop                        # reads .env automatically"
  echo "    make docker-up                        # Docker (dev mode)"
  echo "    make docker-up-prod                   # Docker (production mode)"
  echo ""
  echo "  Verify:"
  echo "    curl -s http://localhost:8080/health | python3 -m json.tool"
  echo ""
elif [ "$USB_DETECTED_BUT_NO_NODE" = true ]; then
  echo "  ${BOLD}Printer detected but device node missing.${NC}"
  echo "  The udev rules are installed. Try these steps:"
  echo ""
  echo "  1. Re-plug the USB cable (this triggers udev rules)"
  echo "  2. If that doesn't work, power-cycle the printer"
  echo "  3. Reset the USB device:"
  echo "       sudo usbreset \"$DETECTED_MODEL\""
  echo "     Or by path:"
  echo "       sudo usbreset /dev/bus/usb/*/*"
  echo "  4. Reload the usblp module:"
  echo "       sudo modprobe -r usblp && sudo modprobe usblp"
  echo "  5. Re-run this script"
  echo ""
  echo "  After the device node appears:"
  echo "    ./bin/zerodrop                        # reads .env automatically"
  echo "    make docker-up                        # Docker (dev mode)"
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
  echo "  ${BOLD}Docker note:${NC} docker-compose files read PRINTER_TYPE and"
  echo "  PRINTER_DEVICE from .env automatically. For raw USB printers"
  echo "  (Zjiang, etc.), docker-compose.prod.yml mounts /dev/bus/usb and"
  echo "  /sys/bus/usb automatically — no manual device path needed."
  echo ""
fi
