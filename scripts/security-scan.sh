#!/usr/bin/env bash
# ZeroDrop Security Verification Scanner
# Runs code-scanning checks that cannot be expressed as Go unit tests.
# These checks verify structural security claims about the codebase.
set -euo pipefail

PASS=0
FAIL=0
TOTAL=0

pass() {
    TOTAL=$((TOTAL + 1))
    PASS=$((PASS + 1))
    printf "[PASS] %-30s %s\n" "$1" "$2"
}

fail() {
    TOTAL=$((TOTAL + 1))
    FAIL=$((FAIL + 1))
    printf "[FAIL] %-30s %s\n" "$1" "$2"
}

# --- Zero-Knowledge Claims ---

# S1: No Decrypt/Unseal functions in server code
if grep -rn "func.*[Dd]ecrypt\|func.*[Uu]nseal" pkg/ cmd/ 2>/dev/null | grep -v "_test.go" | grep -q .; then
    fail "Zero-Knowledge" "Decrypt/Unseal functions found in server code"
else
    pass "Zero-Knowledge" "No Decrypt/Unseal functions in server code"
fi

# S2: No database imports
if grep -rn '"database/sql\|gorm\|sqlite\|postgres\|mysql\|bolt\|badger\|leveldb' pkg/ cmd/ 2>/dev/null | grep -q .; then
    fail "Zero-Knowledge" "Database imports found in server code"
else
    pass "Zero-Knowledge" "No database imports (ephemeral processing only)"
fi

# S3: No math/rand in security-critical code
if grep -rn '"math/rand"' pkg/crypto/ pkg/api/middleware.go 2>/dev/null | grep -q .; then
    fail "Cryptography" "math/rand found in security-critical code"
else
    pass "Cryptography" "No math/rand in security code (crypto/rand only)"
fi

# S10: Only crypto/rand for all rand.Read / rand.Reader calls in security code
# Check each file that uses rand.Read or rand.Reader and verify it imports crypto/rand
s10_fail=false
for f in $(grep -rl 'rand\.Read\|rand\.Reader' pkg/crypto/ pkg/api/middleware.go 2>/dev/null | grep -v "_test.go"); do
    if ! grep -q '"crypto/rand"' "$f"; then
        fail "Cryptography" "$f uses rand.Read/Reader without crypto/rand import"
        s10_fail=true
    fi
done
if [ "$s10_fail" = false ]; then
    pass "Cryptography" "All rand.Read/Reader calls use crypto/rand"
fi

# --- Secrets Management ---

# S4: .env in .gitignore
if grep -q '^\.env$' .gitignore 2>/dev/null; then
    pass "Secrets" ".env files excluded from version control"
else
    fail "Secrets" ".env NOT in .gitignore — secret leak risk!"
fi

# S5: Key files in .gitignore
if grep -q 'private_key' .gitignore 2>/dev/null; then
    pass "Secrets" "Key files excluded from version control"
else
    fail "Secrets" "Key files NOT in .gitignore"
fi

# S6: No hardcoded passwords/tokens in source (excluding tests)
if grep -rn 'password\s*=\s*"[^"]\+"' pkg/ cmd/ 2>/dev/null | grep -v "_test.go" | grep -q .; then
    fail "Secrets" "Hardcoded password found in source code"
else
    pass "Secrets" "No hardcoded passwords/tokens in source code"
fi

# --- Container Security ---

# S7: Dockerfile uses non-root user
if grep -q 'USER zerodrop' Dockerfile 2>/dev/null; then
    pass "Container" "Dockerfile runs as non-root user (zerodrop)"
else
    fail "Container" "Dockerfile does NOT use non-root user"
fi

# S8: Production compose has no-new-privileges
if grep -q 'no-new-privileges' docker-compose.prod.yml 2>/dev/null; then
    pass "Container" "Production compose has no-new-privileges"
else
    fail "Container" "Production compose missing no-new-privileges"
fi

# S9: Production compose has resource limits
if grep -q 'cpus:' docker-compose.prod.yml 2>/dev/null && grep -q 'memory:' docker-compose.prod.yml 2>/dev/null; then
    pass "Container" "Production compose has resource limits (CPU + memory)"
else
    fail "Container" "Production compose missing resource limits"
fi

# --- Summary ---
echo ""
echo "==============================================="
echo "  SECURITY SCAN RESULTS"
echo "==============================================="
echo "  Checks: $TOTAL"
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo "==============================================="

if [ "$FAIL" -gt 0 ]; then
    echo "  ⚠ SOME CHECKS FAILED — review above"
    exit 1
else
    echo "  ✓ ALL CHECKS PASSED"
fi
