#!/bin/bash
# E2E Test Orchestrator
# Runs all E2E tests in sequence and reports results

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY=${ULTRATHINK_BINARY:-"ultrathink"}
API_PORT=${API_PORT:-3099}
SERVER_PID=""
OVERALL_PASSED=0
OVERALL_FAILED=0

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_header() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} $1"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
}

log_result() {
    if [ "$2" = "0" ]; then
        echo -e "  ${GREEN}✓${NC} $1: ${GREEN}PASSED${NC}"
        OVERALL_PASSED=$((OVERALL_PASSED + 1))
    else
        echo -e "  ${RED}✗${NC} $1: ${RED}FAILED${NC}"
        OVERALL_FAILED=$((OVERALL_FAILED + 1))
    fi
}

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        echo ""
        echo "Stopping ultrathink server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
}

trap cleanup EXIT

log_header "Ultrathink E2E Test Suite"
echo ""
echo "Binary: $BINARY"
echo "API Port: $API_PORT"
echo "Test Directory: $SCRIPT_DIR"
echo ""

# Verify binary exists
if ! command -v "$BINARY" &> /dev/null; then
    echo -e "${RED}Error: Binary '$BINARY' not found in PATH${NC}"
    exit 1
fi

# Display version
echo "Version: $($BINARY --version 2>&1 | head -1)"
echo ""

# ===== Start Server =====
log_header "Starting Ultrathink Server"

# Check if server is already running
if curl -s "http://localhost:$API_PORT/api/v1/health" > /dev/null 2>&1; then
    echo "Server already running on port $API_PORT"
else
    echo "Starting server..."
    $BINARY start --port $API_PORT > /tmp/ultrathink-e2e.log 2>&1 &
    SERVER_PID=$!
    echo "Server started with PID: $SERVER_PID"

    # Wait for server to be ready
    MAX_WAIT=30
    for i in $(seq 1 $MAX_WAIT); do
        if curl -s "http://localhost:$API_PORT/api/v1/health" > /dev/null 2>&1; then
            echo "Server is ready!"
            break
        fi
        if [ $i -eq $MAX_WAIT ]; then
            echo -e "${RED}Server failed to start within ${MAX_WAIT}s${NC}"
            cat /tmp/ultrathink-e2e.log
            exit 1
        fi
        echo "  Waiting for server... ($i/$MAX_WAIT)"
        sleep 1
    done
fi

# ===== Run Tests =====
log_header "Running E2E Tests"

# Test 1: CLI Tests
echo ""
echo -e "${BLUE}[1/3] CLI Tests${NC}"
if bash "$SCRIPT_DIR/test-cli.sh"; then
    log_result "CLI Tests" 0
else
    log_result "CLI Tests" 1
fi

# Test 2: REST API Tests
echo ""
echo -e "${BLUE}[2/3] REST API Tests${NC}"
export API_BASE="http://localhost:$API_PORT/api/v1"
if bash "$SCRIPT_DIR/test-rest-api.sh"; then
    log_result "REST API Tests" 0
else
    log_result "REST API Tests" 1
fi

# Test 3: Memory Operations Tests (CLI mode)
echo ""
echo -e "${BLUE}[3/3] Memory Operations Tests (CLI)${NC}"
export USE_API="false"
if bash "$SCRIPT_DIR/test-memory-ops.sh"; then
    log_result "Memory Operations Tests" 0
else
    log_result "Memory Operations Tests" 1
fi

# Test 4: Memory Operations Tests (API mode)
echo ""
echo -e "${BLUE}[Bonus] Memory Operations Tests (API)${NC}"
export USE_API="true"
if bash "$SCRIPT_DIR/test-memory-ops.sh"; then
    log_result "Memory Operations Tests (API)" 0
else
    log_result "Memory Operations Tests (API)" 1
fi

# ===== Summary =====
log_header "E2E Test Summary"
echo ""
echo -e "  Test Suites Passed: ${GREEN}$OVERALL_PASSED${NC}"
echo -e "  Test Suites Failed: ${RED}$OVERALL_FAILED${NC}"
echo ""

TOTAL=$((OVERALL_PASSED + OVERALL_FAILED))
if [ $OVERALL_FAILED -eq 0 ]; then
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ALL E2E TESTS PASSED! ($OVERALL_PASSED/$TOTAL test suites)                        ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║  SOME E2E TESTS FAILED ($OVERALL_FAILED/$TOTAL test suites failed)                ║${NC}"
    echo -e "${RED}╚══════════════════════════════════════════════════════════════╝${NC}"
    exit 1
fi
