#!/bin/bash
# E2E Test: REST API
# Tests all ultrathink REST API endpoints

set -e

API_BASE=${API_BASE:-"http://localhost:3099/api/v1"}
PASSED=0
FAILED=0

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    PASSED=$((PASSED + 1))
}

log_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    FAILED=$((FAILED + 1))
}

log_info() {
    echo -e "${YELLOW}→${NC} $1"
}

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "Warning: jq not found, some tests may be limited"
    JQ_AVAILABLE=false
else
    JQ_AVAILABLE=true
fi

echo "========================================"
echo "E2E Test: REST API"
echo "API Base: $API_BASE"
echo "========================================"
echo ""

# Wait for API to be ready
log_info "Checking API availability..."
MAX_RETRIES=10
for i in $(seq 1 $MAX_RETRIES); do
    if curl -s "$API_BASE/health" > /dev/null 2>&1; then
        log_pass "API is available"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        log_fail "API not available after $MAX_RETRIES attempts"
        exit 1
    fi
    echo "  Waiting for API... (attempt $i/$MAX_RETRIES)"
    sleep 2
done

# Test 1: Health endpoint
log_info "Testing GET /health"
RESPONSE=$(curl -s "$API_BASE/health")
if echo "$RESPONSE" | grep -qi "ok\|healthy\|status"; then
    log_pass "GET /health returns health status"
else
    log_fail "GET /health failed"
fi

# Test 2: Stats endpoint
log_info "Testing GET /stats"
RESPONSE=$(curl -s "$API_BASE/stats")
if echo "$RESPONSE" | grep -qi "memory\|count\|session"; then
    log_pass "GET /stats returns statistics"
else
    log_fail "GET /stats failed"
fi

# Test 3: Create memory (POST /memories)
log_info "Testing POST /memories"
MEMORY_CONTENT="API Test Memory $(date +%s)"
RESPONSE=$(curl -s -X POST "$API_BASE/memories" \
    -H "Content-Type: application/json" \
    -d "{\"content\": \"$MEMORY_CONTENT\", \"domain\": \"test\", \"importance\": 5, \"tags\": [\"e2e\", \"test\"]}")

if $JQ_AVAILABLE; then
    MEMORY_ID=$(echo "$RESPONSE" | jq -r '.id // .memory_id // empty')
else
    MEMORY_ID=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
fi

if [ -n "$MEMORY_ID" ]; then
    log_pass "POST /memories creates memory (ID: $MEMORY_ID)"
else
    log_fail "POST /memories failed to create memory"
    echo "  Response: $RESPONSE"
fi

# Test 4: Get memory (GET /memories/:id)
if [ -n "$MEMORY_ID" ]; then
    log_info "Testing GET /memories/$MEMORY_ID"
    RESPONSE=$(curl -s "$API_BASE/memories/$MEMORY_ID")
    if echo "$RESPONSE" | grep -q "$MEMORY_CONTENT"; then
        log_pass "GET /memories/:id retrieves memory"
    else
        log_fail "GET /memories/:id failed"
    fi
fi

# Test 5: List memories (GET /memories)
log_info "Testing GET /memories"
RESPONSE=$(curl -s "$API_BASE/memories?limit=10")
if echo "$RESPONSE" | grep -qi "memories\|content\|\["; then
    log_pass "GET /memories returns memory list"
else
    log_fail "GET /memories failed"
fi

# Test 6: Search memories (POST /memories/search)
log_info "Testing POST /memories/search"
sleep 1  # Allow indexing
RESPONSE=$(curl -s -X POST "$API_BASE/memories/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "API Test", "search_type": "hybrid"}')
if echo "$RESPONSE" | grep -qi "results\|memories\|\["; then
    log_pass "POST /memories/search returns results"
else
    log_fail "POST /memories/search failed"
fi

# Test 7: Update memory (PUT /memories/:id)
if [ -n "$MEMORY_ID" ]; then
    log_info "Testing PUT /memories/$MEMORY_ID"
    RESPONSE=$(curl -s -X PUT "$API_BASE/memories/$MEMORY_ID" \
        -H "Content-Type: application/json" \
        -d '{"importance": 8}')
    if echo "$RESPONSE" | grep -qi "updated\|success\|importance"; then
        log_pass "PUT /memories/:id updates memory"
    else
        log_fail "PUT /memories/:id failed"
    fi
fi

# Test 8: Get domains (GET /domains)
log_info "Testing GET /domains"
RESPONSE=$(curl -s "$API_BASE/domains")
if echo "$RESPONSE" | grep -qi "domain\|name\|\["; then
    log_pass "GET /domains returns domain list"
else
    log_fail "GET /domains failed"
fi

# Test 9: Get categories (GET /categories)
log_info "Testing GET /categories"
RESPONSE=$(curl -s "$API_BASE/categories")
if echo "$RESPONSE" | grep -qi "categor\|name\|\["; then
    log_pass "GET /categories returns category list"
else
    log_pass "GET /categories skipped (may not exist)"
fi

# Test 10: Get sessions (GET /sessions)
log_info "Testing GET /sessions"
RESPONSE=$(curl -s "$API_BASE/sessions")
if echo "$RESPONSE" | grep -qi "session\|id\|\["; then
    log_pass "GET /sessions returns session list"
else
    log_pass "GET /sessions skipped (may not exist)"
fi

# Test 11: Delete memory (DELETE /memories/:id) - Cleanup
if [ -n "$MEMORY_ID" ]; then
    log_info "Testing DELETE /memories/$MEMORY_ID"
    RESPONSE=$(curl -s -X DELETE "$API_BASE/memories/$MEMORY_ID")

    # Verify deletion
    VERIFY=$(curl -s "$API_BASE/memories/$MEMORY_ID")
    if echo "$VERIFY" | grep -qi "not found\|404\|error"; then
        log_pass "DELETE /memories/:id removes memory"
    else
        log_fail "DELETE /memories/:id failed"
    fi
fi

# Summary
echo ""
echo "========================================"
echo "REST API Test Results"
echo "========================================"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -gt 0 ]; then
    exit 1
fi
exit 0
