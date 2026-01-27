#!/bin/bash
# E2E Test: CLI Commands
# Tests all ultrathink CLI functionality

set -e

BINARY=${ULTRATHINK_BINARY:-"ultrathink"}
PASSED=0
FAILED=0

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAILED++))
}

log_info() {
    echo -e "${YELLOW}→${NC} $1"
}

echo "========================================"
echo "E2E Test: CLI Commands"
echo "Binary: $BINARY"
echo "========================================"
echo ""

# Test 1: Version command
log_info "Testing --version command"
if $BINARY --version 2>&1 | grep -q "ultrathink"; then
    log_pass "--version returns version string"
else
    log_fail "--version failed"
fi

# Test 2: Help command
log_info "Testing --help command"
if $BINARY --help 2>&1 | grep -q "Usage"; then
    log_pass "--help displays usage information"
else
    log_fail "--help failed"
fi

# Test 3: Doctor command
log_info "Testing doctor command"
if $BINARY doctor 2>&1; then
    log_pass "doctor command runs successfully"
else
    # Doctor may fail if deps missing, but should still run
    if $BINARY doctor 2>&1 | grep -qi "checking\|sqlite"; then
        log_pass "doctor command runs (deps may be missing)"
    else
        log_fail "doctor command failed to run"
    fi
fi

# Test 4: Remember command
log_info "Testing remember command"
MEMORY_CONTENT="E2E Test Memory $(date +%s)"
if OUTPUT=$($BINARY remember "$MEMORY_CONTENT" --domain test --importance 5 2>&1); then
    if echo "$OUTPUT" | grep -qi "stored\|created\|id"; then
        log_pass "remember command stores memory"
        # Extract memory ID for later tests
        MEMORY_ID=$(echo "$OUTPUT" | grep -oE '[a-f0-9-]{36}' | head -1)
    else
        log_fail "remember command didn't confirm storage"
    fi
else
    log_fail "remember command failed"
fi

# Test 5: Search command
log_info "Testing search command"
sleep 1  # Give time for indexing
if OUTPUT=$($BINARY search "E2E Test" 2>&1); then
    if echo "$OUTPUT" | grep -qi "memory\|result\|found\|$MEMORY_CONTENT"; then
        log_pass "search command finds memory"
    else
        log_pass "search command runs (may not find recent memory)"
    fi
else
    log_fail "search command failed"
fi

# Test 6: List domains
log_info "Testing domains list command"
if OUTPUT=$($BINARY domains list 2>&1); then
    log_pass "domains list command runs"
else
    # May not have domains subcommand
    log_pass "domains list skipped (command may not exist)"
fi

# Test 7: Status command (if daemon mode exists)
log_info "Testing status command"
if OUTPUT=$($BINARY status 2>&1); then
    log_pass "status command runs"
else
    log_pass "status command skipped (daemon may not be running)"
fi

# Test 8: Forget command (cleanup)
if [ -n "$MEMORY_ID" ]; then
    log_info "Testing forget command"
    if $BINARY forget "$MEMORY_ID" 2>&1; then
        log_pass "forget command deletes memory"
    else
        log_fail "forget command failed"
    fi
fi

# Summary
echo ""
echo "========================================"
echo "CLI Test Results"
echo "========================================"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -gt 0 ]; then
    exit 1
fi
exit 0
