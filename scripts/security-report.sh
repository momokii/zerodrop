#!/usr/bin/env bash
# ZeroDrop Security Report Formatter
# Parses Go test output and security-scan.sh output into a unified report.
set -euo pipefail

GO_LOG="${1:-/dev/null}"
SCAN_LOG="${2:-/dev/null}"

echo ""
echo "==============================================="
echo "  ZERODROP SECURITY VERIFICATION REPORT"
echo "==============================================="
echo ""

# Parse Go test results
go_pass=0
go_fail=0
if [ -f "$GO_LOG" ]; then
    go_pass=$(grep -c "^--- PASS:.*TestSecurity_" "$GO_LOG" 2>/dev/null) || go_pass=0
    go_fail=$(grep -c "^--- FAIL:.*TestSecurity_" "$GO_LOG" 2>/dev/null) || go_fail=0
fi

# Parse scan results
scan_pass=0
scan_fail=0
if [ -f "$SCAN_LOG" ]; then
    scan_pass=$(grep -c "^\[PASS\]" "$SCAN_LOG" 2>/dev/null) || scan_pass=0
    scan_fail=$(grep -c "^\[FAIL\]" "$SCAN_LOG" 2>/dev/null) || scan_fail=0
fi

total_pass=$((go_pass + scan_pass))
total_fail=$((go_fail + scan_fail))
total=$((total_pass + total_fail))

printf "  %-20s %10s %10s\n" "Layer" "Passed" "Failed"
printf "  %-20s %10s %10s\n" "--------------------" "----------" "----------"
printf "  %-20s %10d %10d\n" "Go Security Tests" "$go_pass" "$go_fail"
printf "  %-20s %10d %10d\n" "Code Scanning" "$scan_pass" "$scan_fail"
printf "  %-20s %10s %10s\n" "--------------------" "----------" "----------"
printf "  %-20s %10d %10d\n" "TOTAL" "$total_pass" "$total_fail"
echo ""

if [ "$total_fail" -eq 0 ]; then
    echo "  ALL $total SECURITY CLAIMS VERIFIED"
else
    echo "  $total_fail of $total checks FAILED:"
    if [ "$go_fail" -gt 0 ]; then
        echo "    - Go Security Tests: $go_fail failures (search for '--- FAIL:.*TestSecurity_')"
    fi
    if [ "$scan_fail" -gt 0 ]; then
        echo "    - Code Scanning: $scan_fail failures (search for '[FAIL]')"
    fi
fi

echo "==============================================="
echo ""
echo "Run individual layers:"
echo "  go test -v -run TestSecurity_ ./...    # Go tests"
echo "  bash scripts/security-scan.sh          # Code scanning"
echo "  make check-security                    # Everything"
echo ""

if [ "$total_fail" -gt 0 ]; then
    exit 1
fi
