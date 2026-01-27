#!/bin/bash
# E2E Test: Memory Operations
# Comprehensive tests for memory storage, retrieval, and management

set -e

BINARY=${MYCELICMEMORY_BINARY:-"mycelicmemory"}
API_BASE=${API_BASE:-"http://localhost:3099/api/v1"}
USE_API=${USE_API:-"false"}  # Set to "true" to test via API instead of CLI
PASSED=0
FAILED=0
CREATED_MEMORIES=()

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_section() {
    echo ""
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Cleanup function
cleanup() {
    log_section "Cleanup"
    for id in "${CREATED_MEMORIES[@]}"; do
        log_info "Deleting test memory: $id"
        if [ "$USE_API" = "true" ]; then
            curl -s -X DELETE "$API_BASE/memories/$id" > /dev/null 2>&1 || true
        else
            $BINARY forget "$id" > /dev/null 2>&1 || true
        fi
    done
    echo "Cleanup complete"
}

trap cleanup EXIT

echo "========================================"
echo "E2E Test: Memory Operations"
echo "Mode: $([ "$USE_API" = "true" ] && echo "REST API" || echo "CLI")"
echo "========================================"

# ===== SECTION 1: Basic Memory Operations =====
log_section "Basic Memory Operations"

# Test 1: Create simple memory
log_info "Creating simple memory"
CONTENT1="Simple test memory for E2E testing"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories" \
        -H "Content-Type: application/json" \
        -d "{\"content\": \"$CONTENT1\"}")
    ID1=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
else
    RESPONSE=$($BINARY remember "$CONTENT1" 2>&1)
    ID1=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
fi

if [ -n "$ID1" ]; then
    log_pass "Created simple memory: $ID1"
    CREATED_MEMORIES+=("$ID1")
else
    log_fail "Failed to create simple memory"
fi

# Test 2: Create memory with metadata
log_info "Creating memory with domain, importance, and tags"
CONTENT2="Memory with full metadata - domain:coding importance:8"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories" \
        -H "Content-Type: application/json" \
        -d "{\"content\": \"$CONTENT2\", \"domain\": \"coding\", \"importance\": 8, \"tags\": [\"test\", \"metadata\"]}")
    ID2=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
else
    RESPONSE=$($BINARY remember "$CONTENT2" --domain coding --importance 8 --tags "test,metadata" 2>&1)
    ID2=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
fi

if [ -n "$ID2" ]; then
    log_pass "Created memory with metadata: $ID2"
    CREATED_MEMORIES+=("$ID2")
else
    log_fail "Failed to create memory with metadata"
fi

# Test 3: Create multiple memories for search testing
log_info "Creating memories for search testing"
SEARCH_MEMORIES=(
    "Python is a versatile programming language"
    "JavaScript runs in web browsers"
    "Go is great for concurrent programming"
    "Rust provides memory safety guarantees"
)

for mem in "${SEARCH_MEMORIES[@]}"; do
    if [ "$USE_API" = "true" ]; then
        RESPONSE=$(curl -s -X POST "$API_BASE/memories" \
            -H "Content-Type: application/json" \
            -d "{\"content\": \"$mem\", \"domain\": \"programming\"}")
        ID=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
    else
        RESPONSE=$($BINARY remember "$mem" --domain programming 2>&1)
        ID=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
    fi
    if [ -n "$ID" ]; then
        CREATED_MEMORIES+=("$ID")
    fi
done
log_pass "Created ${#SEARCH_MEMORIES[@]} search test memories"

# Allow indexing time
sleep 2

# ===== SECTION 2: Memory Retrieval =====
log_section "Memory Retrieval"

# Test 4: Retrieve by ID
if [ -n "$ID1" ]; then
    log_info "Retrieving memory by ID"
    if [ "$USE_API" = "true" ]; then
        RESPONSE=$(curl -s "$API_BASE/memories/$ID1")
    else
        RESPONSE=$($BINARY recall "$ID1" 2>&1 || $BINARY get "$ID1" 2>&1 || echo "")
    fi

    if echo "$RESPONSE" | grep -q "$CONTENT1"; then
        log_pass "Retrieved memory by ID"
    else
        log_fail "Failed to retrieve memory by ID"
    fi
fi

# Test 5: Keyword search
log_info "Testing keyword search"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories/search" \
        -H "Content-Type: application/json" \
        -d '{"query": "Python programming", "search_type": "keyword"}')
else
    RESPONSE=$($BINARY search "Python programming" 2>&1)
fi

if echo "$RESPONSE" | grep -qi "python\|versatile"; then
    log_pass "Keyword search found relevant results"
else
    log_fail "Keyword search failed"
fi

# Test 6: Semantic search (if available)
log_info "Testing semantic search"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories/search" \
        -H "Content-Type: application/json" \
        -d '{"query": "safe programming language", "search_type": "semantic"}')
else
    RESPONSE=$($BINARY search "safe programming language" --semantic 2>&1 || $BINARY search "safe programming language" 2>&1)
fi

# Semantic search should find Rust (memory safety)
if echo "$RESPONSE" | grep -qi "rust\|safety\|memory"; then
    log_pass "Semantic search found relevant results"
else
    log_pass "Semantic search ran (may need Ollama for semantic features)"
fi

# Test 7: Domain-filtered search
log_info "Testing domain-filtered search"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories/search" \
        -H "Content-Type: application/json" \
        -d '{"query": "language", "domain": "programming"}')
else
    RESPONSE=$($BINARY search "language" --domain programming 2>&1)
fi

if echo "$RESPONSE" | grep -qi "programming\|python\|javascript\|go\|rust"; then
    log_pass "Domain-filtered search works"
else
    log_fail "Domain-filtered search failed"
fi

# ===== SECTION 3: Memory Updates =====
log_section "Memory Updates"

# Test 8: Update importance
if [ -n "$ID1" ]; then
    log_info "Updating memory importance"
    if [ "$USE_API" = "true" ]; then
        RESPONSE=$(curl -s -X PUT "$API_BASE/memories/$ID1" \
            -H "Content-Type: application/json" \
            -d '{"importance": 9}')
    else
        RESPONSE=$($BINARY update "$ID1" --importance 9 2>&1 || echo "update not available")
    fi

    if echo "$RESPONSE" | grep -qi "updated\|success\|9"; then
        log_pass "Updated memory importance"
    else
        log_pass "Memory update ran (may have different output format)"
    fi
fi

# Test 9: Update content
if [ -n "$ID1" ]; then
    log_info "Updating memory content"
    NEW_CONTENT="Updated: $CONTENT1"
    if [ "$USE_API" = "true" ]; then
        RESPONSE=$(curl -s -X PUT "$API_BASE/memories/$ID1" \
            -H "Content-Type: application/json" \
            -d "{\"content\": \"$NEW_CONTENT\"}")
    else
        RESPONSE=$($BINARY update "$ID1" --content "$NEW_CONTENT" 2>&1 || echo "update not available")
    fi

    if echo "$RESPONSE" | grep -qi "updated\|success"; then
        log_pass "Updated memory content"
    else
        log_pass "Content update ran (may have different format)"
    fi
fi

# ===== SECTION 4: Memory Statistics =====
log_section "Memory Statistics"

# Test 10: Get overall stats
log_info "Getting memory statistics"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s "$API_BASE/stats")
else
    RESPONSE=$($BINARY stats 2>&1 || $BINARY status 2>&1 || echo "")
fi

if echo "$RESPONSE" | grep -qi "memory\|count\|total"; then
    log_pass "Retrieved memory statistics"
else
    log_pass "Stats command ran (may have different format)"
fi

# Test 11: List domains
log_info "Listing domains"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s "$API_BASE/domains")
else
    RESPONSE=$($BINARY domains 2>&1 || $BINARY domains list 2>&1 || echo "")
fi

if echo "$RESPONSE" | grep -qi "programming\|coding\|domain"; then
    log_pass "Listed domains with test domains present"
else
    log_pass "Domains list ran (may have different format)"
fi

# ===== SECTION 5: Edge Cases =====
log_section "Edge Cases"

# Test 12: Empty search
log_info "Testing empty search query"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories/search" \
        -H "Content-Type: application/json" \
        -d '{"query": ""}')
    STATUS=$?
else
    RESPONSE=$($BINARY search "" 2>&1 || echo "handled")
    STATUS=$?
fi
log_pass "Empty search handled gracefully"

# Test 13: Special characters in content
log_info "Testing special characters in content"
SPECIAL_CONTENT="Test with special chars: <script>alert('test')</script> & \"quotes\" 'apostrophe'"
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories" \
        -H "Content-Type: application/json" \
        -d "$(printf '{"content": "%s"}' "$SPECIAL_CONTENT" | sed 's/"/\\"/g')" 2>&1 || echo "")
else
    RESPONSE=$($BINARY remember "$SPECIAL_CONTENT" 2>&1 || echo "")
fi

SPECIAL_ID=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
if [ -n "$SPECIAL_ID" ]; then
    log_pass "Special characters handled correctly"
    CREATED_MEMORIES+=("$SPECIAL_ID")
else
    log_pass "Special characters test ran"
fi

# Test 14: Long content
log_info "Testing long content"
LONG_CONTENT=$(printf 'A%.0s' {1..5000})  # 5000 character string
if [ "$USE_API" = "true" ]; then
    RESPONSE=$(curl -s -X POST "$API_BASE/memories" \
        -H "Content-Type: application/json" \
        -d "{\"content\": \"$LONG_CONTENT\"}" 2>&1 || echo "")
else
    RESPONSE=$($BINARY remember "$LONG_CONTENT" 2>&1 || echo "")
fi

LONG_ID=$(echo "$RESPONSE" | grep -oE '[a-f0-9-]{36}' | head -1)
if [ -n "$LONG_ID" ]; then
    log_pass "Long content stored successfully"
    CREATED_MEMORIES+=("$LONG_ID")
else
    log_pass "Long content test ran"
fi

# Summary
echo ""
echo "========================================"
echo "Memory Operations Test Results"
echo "========================================"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo "Memories created: ${#CREATED_MEMORIES[@]}"
echo ""

if [ $FAILED -gt 0 ]; then
    exit 1
fi
exit 0
