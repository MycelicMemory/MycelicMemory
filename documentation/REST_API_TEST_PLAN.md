# Ultrathink REST API Comprehensive Test Plan

This document contains comprehensive tests for all REST API endpoints to verify parity with local-memory v1.2.0.

## Test Environment
- **Local Memory**: http://localhost:3002
- **Ultrathink**: http://localhost:3099

## Test Execution Status

| Category | Endpoint | Status | Notes |
|----------|----------|--------|-------|
| Health | GET /health | [ ] | |
| Memory | POST /memories | [ ] | Create |
| Memory | GET /memories | [ ] | List all |
| Memory | GET /memories/:id | [ ] | Get single |
| Memory | PUT /memories/:id | [ ] | Update |
| Memory | DELETE /memories/:id | [ ] | Delete |
| Memory | GET /memories/stats | [ ] | Statistics |
| Search | GET /memories/search | [ ] | GET search |
| Search | POST /memories/search | [ ] | POST search |
| Search | POST /memories/search/intelligent | [ ] | AI search |
| Search | POST /search/tags | [ ] | Tag search |
| Search | POST /search/date-range | [ ] | Date range |
| Analysis | POST /analyze | [ ] | AI analysis |
| Relationships | POST /relationships | [ ] | Create |
| Relationships | POST /relationships/discover | [ ] | AI discover |
| Relationships | GET /memories/:id/related | [ ] | Find related |
| Relationships | GET /memories/:id/graph | [ ] | Map graph |
| Categories | POST /categories | [ ] | Create |
| Categories | GET /categories | [ ] | List |
| Categories | GET /categories/stats | [ ] | Stats |
| Categories | POST /memories/:id/categorize | [ ] | Categorize |
| Domains | POST /domains | [ ] | Create |
| Domains | GET /domains | [ ] | List |
| Domains | GET /domains/:domain/stats | [ ] | Stats |
| Sessions | GET /sessions | [ ] | List |
| Sessions | GET /sessions/stats | [ ] | Stats |
| Stats | GET /stats | [ ] | System stats |

---

## 1. HEALTH ENDPOINT

### Test 1.1: Basic Health Check
```bash
# Local Memory
curl -s http://localhost:3002/api/v1/health | jq .

# Ultrathink
curl -s http://localhost:3099/api/v1/health | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Server is healthy",
  "data": {
    "status": "healthy",
    "session": "<session-id>",
    "timestamp": "<ISO-8601>"
  }
}
```

---

## 2. MEMORY CRUD OPERATIONS

### Test 2.1: Create Memory (Basic)
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test memory content",
    "importance": 7,
    "tags": ["test", "api"]
  }' | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Memory stored successfully",
  "data": {
    "id": "<uuid>",
    "content": "Test memory content",
    "source": null,
    "slug": null,
    "importance": 7,
    "tags": ["test", "api"],
    "session_id": "<session>",
    "domain": null,
    "created_at": "<timestamp>",
    "updated_at": "<timestamp>"
  }
}
```

### Test 2.2: Create Memory (With Domain and Source)
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Memory with all fields",
    "importance": 9,
    "tags": ["complete", "test"],
    "domain": "programming",
    "source": "manual-test"
  }' | jq .
```

### Test 2.3: Create Memory (Minimal - content only)
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Minimal memory"
  }' | jq .
```

### Test 2.4: Create Memory (Empty content - should fail)
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories \
  -H "Content-Type: application/json" \
  -d '{}' | jq .
```

### Test 2.5: List All Memories
```bash
curl -s http://localhost:PORT/api/v1/memories | jq .
```

### Test 2.6: List Memories with Limit
```bash
curl -s "http://localhost:PORT/api/v1/memories?limit=2" | jq .
```

### Test 2.7: List Memories with Offset
```bash
curl -s "http://localhost:PORT/api/v1/memories?limit=2&offset=1" | jq .
```

### Test 2.8: Get Single Memory
```bash
curl -s http://localhost:PORT/api/v1/memories/<memory-id> | jq .
```

### Test 2.9: Get Non-existent Memory
```bash
curl -s http://localhost:PORT/api/v1/memories/00000000-0000-0000-0000-000000000000 | jq .
```

### Test 2.10: Update Memory (Content)
```bash
curl -s -X PUT http://localhost:PORT/api/v1/memories/<memory-id> \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Updated content"
  }' | jq .
```

### Test 2.11: Update Memory (Tags)
```bash
curl -s -X PUT http://localhost:PORT/api/v1/memories/<memory-id> \
  -H "Content-Type: application/json" \
  -d '{
    "tags": ["updated", "new-tag"]
  }' | jq .
```

### Test 2.12: Update Memory (Importance)
```bash
curl -s -X PUT http://localhost:PORT/api/v1/memories/<memory-id> \
  -H "Content-Type: application/json" \
  -d '{
    "importance": 10
  }' | jq .
```

### Test 2.13: Update Non-existent Memory
```bash
curl -s -X PUT http://localhost:PORT/api/v1/memories/00000000-0000-0000-0000-000000000000 \
  -H "Content-Type: application/json" \
  -d '{"content": "test"}' | jq .
```

### Test 2.14: Delete Memory
```bash
curl -s -X DELETE http://localhost:PORT/api/v1/memories/<memory-id> | jq .
```

### Test 2.15: Delete Non-existent Memory
```bash
curl -s -X DELETE http://localhost:PORT/api/v1/memories/00000000-0000-0000-0000-000000000000 | jq .
```

### Test 2.16: Memory Stats
```bash
curl -s http://localhost:PORT/api/v1/memories/stats | jq .
```

---

## 3. SEARCH OPERATIONS

### Test 3.1: GET Search (Basic)
```bash
curl -s "http://localhost:PORT/api/v1/memories/search?query=test" | jq .
```

### Test 3.2: GET Search with Limit
```bash
curl -s "http://localhost:PORT/api/v1/memories/search?query=test&limit=5" | jq .
```

### Test 3.3: POST Search (Basic)
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "test",
    "limit": 5
  }' | jq .
```

**Expected Response Structure:**
```json
{
  "data": [
    {
      "id": "<uuid>",
      "summary": "<content>",
      "relevance_score": 1.0,
      "tags": ["tag1"],
      "importance": 5,
      "created_at": ""
    }
  ],
  "pagination_metadata": {
    "has_next_page": false,
    "has_previous_page": false,
    "total_count": 1,
    "page_size": 5,
    "current_page": 1
  },
  "query_hash": "abc123",
  "search_info": {
    "query": "test",
    "search_type": "enhanced_text",
    "total_results": 1,
    "processing_time_ms": 0,
    "has_more_results": false
  }
}
```

### Test 3.4: POST Search with AI
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "test",
    "use_ai": true,
    "limit": 5
  }' | jq .
```

### Test 3.5: Intelligent Search
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories/search/intelligent \
  -H "Content-Type: application/json" \
  -d '{
    "query": "test memory",
    "limit": 5
  }' | jq .
```

### Test 3.6: Tag Search (Single Tag)
```bash
curl -s -X POST http://localhost:PORT/api/v1/search/tags \
  -H "Content-Type: application/json" \
  -d '{
    "tags": ["test"]
  }' | jq .
```

### Test 3.7: Tag Search (Multiple Tags - OR)
```bash
curl -s -X POST http://localhost:PORT/api/v1/search/tags \
  -H "Content-Type: application/json" \
  -d '{
    "tags": ["test", "api"],
    "tag_operator": "OR"
  }' | jq .
```

### Test 3.8: Tag Search (Multiple Tags - AND)
```bash
curl -s -X POST http://localhost:PORT/api/v1/search/tags \
  -H "Content-Type: application/json" \
  -d '{
    "tags": ["test", "api"],
    "tag_operator": "AND"
  }' | jq .
```

### Test 3.9: Date Range Search
```bash
curl -s -X POST http://localhost:PORT/api/v1/search/date-range \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2025-01-01",
    "end_date": "2025-12-31",
    "limit": 10
  }' | jq .
```

---

## 4. ANALYSIS OPERATIONS

### Test 4.1: Question Answering
```bash
curl -s -X POST http://localhost:PORT/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "analysis_type": "question",
    "question": "What have I learned about testing?"
  }' | jq .
```

### Test 4.2: Summarization
```bash
curl -s -X POST http://localhost:PORT/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "analysis_type": "summarize",
    "timeframe": "all"
  }' | jq .
```

### Test 4.3: Pattern Analysis
```bash
curl -s -X POST http://localhost:PORT/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "analysis_type": "analyze",
    "query": "test"
  }' | jq .
```

### Test 4.4: Temporal Patterns
```bash
curl -s -X POST http://localhost:PORT/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "analysis_type": "temporal_patterns",
    "concept": "testing",
    "temporal_timeframe": "month"
  }' | jq .
```

---

## 5. RELATIONSHIP OPERATIONS

### Test 5.1: Create Relationship
```bash
curl -s -X POST http://localhost:PORT/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "source_memory_id": "<memory-id-1>",
    "target_memory_id": "<memory-id-2>",
    "relationship_type_enum": "similar",
    "strength": 0.8,
    "context": "Both about testing"
  }' | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Relationship created successfully",
  "data": {
    "id": "<uuid>",
    "source_memory_id": "<uuid>",
    "target_memory_id": "<uuid>",
    "relationship_type": "similar",
    "strength": 0.8,
    "context": "Both about testing",
    "auto_generated": false,
    "created_at": "<timestamp>"
  }
}
```

### Test 5.2: Create Relationship (Invalid Type)
```bash
curl -s -X POST http://localhost:PORT/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "source_memory_id": "<memory-id>",
    "target_memory_id": "<memory-id>",
    "relationship_type_enum": "invalid_type",
    "strength": 0.5
  }' | jq .
```

### Test 5.3: Find Related Memories
```bash
curl -s http://localhost:PORT/api/v1/memories/<memory-id>/related | jq .
```

### Test 5.4: Find Related with Limit
```bash
curl -s "http://localhost:PORT/api/v1/memories/<memory-id>/related?limit=5" | jq .
```

### Test 5.5: Find Related with Type Filter
```bash
curl -s "http://localhost:PORT/api/v1/memories/<memory-id>/related?type=similar" | jq .
```

### Test 5.6: Map Memory Graph
```bash
curl -s http://localhost:PORT/api/v1/memories/<memory-id>/graph | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Graph mapped successfully",
  "data": {
    "nodes": [
      {
        "id": "<uuid>",
        "content": "...",
        "distance": 0,
        "importance": 5
      }
    ],
    "edges": [
      {
        "source": "<uuid>",
        "target": "<uuid>",
        "type": "similar",
        "strength": 0.8
      }
    ],
    "total_nodes": 2,
    "total_edges": 1,
    "max_depth": 2
  }
}
```

### Test 5.7: Map Graph with Depth
```bash
curl -s "http://localhost:PORT/api/v1/memories/<memory-id>/graph?depth=3" | jq .
```

### Test 5.8: Discover Relationships (AI)
```bash
curl -s -X POST http://localhost:PORT/api/v1/relationships/discover \
  -H "Content-Type: application/json" \
  -d '{
    "limit": 5
  }' | jq .
```

---

## 6. CATEGORY OPERATIONS

### Test 6.1: Create Category
```bash
curl -s -X POST http://localhost:PORT/api/v1/categories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Category",
    "description": "A category for testing"
  }' | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Category created successfully",
  "data": {
    "id": "<uuid>",
    "name": "Test Category",
    "description": "A category for testing",
    "parent_category_id": null,
    "confidence_threshold": 0.7,
    "auto_generated": false,
    "created_at": "<timestamp>"
  }
}
```

### Test 6.2: Create Category with Parent
```bash
curl -s -X POST http://localhost:PORT/api/v1/categories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sub Category",
    "description": "A child category",
    "parent_id": "<parent-category-id>"
  }' | jq .
```

### Test 6.3: List Categories
```bash
curl -s http://localhost:PORT/api/v1/categories | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Found X categories",
  "data": [
    {
      "id": "<uuid>",
      "name": "Test Category",
      "description": "...",
      "parent_category_id": null,
      "confidence_threshold": 0.7,
      "auto_generated": false,
      "created_at": "<timestamp>"
    }
  ]
}
```

### Test 6.4: Category Stats
```bash
curl -s http://localhost:PORT/api/v1/categories/stats | jq .
```

### Test 6.5: Categorize Memory
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories/<memory-id>/categorize \
  -H "Content-Type: application/json" \
  -d '{
    "category_id": "<category-id>",
    "confidence": 0.9,
    "reasoning": "This memory fits the test category"
  }' | jq .
```

---

## 7. DOMAIN OPERATIONS

### Test 7.1: Create Domain
```bash
curl -s -X POST http://localhost:PORT/api/v1/domains \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-domain",
    "description": "A domain for testing"
  }' | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Domain created successfully",
  "data": {
    "id": "<uuid>",
    "name": "test-domain",
    "description": "A domain for testing",
    "created_at": "<timestamp>",
    "updated_at": "<timestamp>"
  }
}
```

### Test 7.2: List Domains
```bash
curl -s http://localhost:PORT/api/v1/domains | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Domains retrieved successfully",
  "data": {
    "domains": [
      {
        "id": "<uuid>",
        "name": "test-domain",
        "description": "...",
        "created_at": "<timestamp>",
        "updated_at": "<timestamp>"
      }
    ]
  }
}
```

### Test 7.3: Domain Stats
```bash
curl -s http://localhost:PORT/api/v1/domains/test-domain/stats | jq .
```

---

## 8. SESSION OPERATIONS

### Test 8.1: List Sessions
```bash
curl -s http://localhost:PORT/api/v1/sessions | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "Found X sessions",
  "data": ["session-id-1", "session-id-2"]
}
```

### Test 8.2: Session Stats
```bash
curl -s http://localhost:PORT/api/v1/sessions/stats | jq .
```

---

## 9. SYSTEM STATS

### Test 9.1: System Statistics
```bash
curl -s http://localhost:PORT/api/v1/stats | jq .
```

**Expected Response Structure:**
```json
{
  "success": true,
  "message": "System statistics retrieved successfully",
  "data": {
    "total_memories": 10,
    "average_importance": 5.5,
    "unique_tags": ["tag1", "tag2"],
    "most_common_tags": [],
    "date_range": {
      "earliest": "<timestamp>",
      "latest": "<timestamp>"
    },
    "session_id": "<current-session>"
  }
}
```

---

## 10. ERROR HANDLING

### Test 10.1: Invalid JSON
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories \
  -H "Content-Type: application/json" \
  -d 'not valid json' | jq .
```

### Test 10.2: Missing Required Field
```bash
curl -s -X POST http://localhost:PORT/api/v1/memories \
  -H "Content-Type: application/json" \
  -d '{"importance": 5}' | jq .
```

### Test 10.3: Invalid ID Format
```bash
curl -s http://localhost:PORT/api/v1/memories/not-a-uuid | jq .
```

---

## Full Workflow Tests

### Workflow 1: Complete Memory Lifecycle
1. Create memory
2. Verify in list
3. Get single memory
4. Update memory
5. Verify update
6. Delete memory
7. Verify deletion

### Workflow 2: Category + Memory Association
1. Create category
2. Verify category in list
3. Create memory
4. Categorize memory
5. Verify category stats

### Workflow 3: Relationship Graph Building
1. Create memory A
2. Create memory B
3. Create memory C
4. Create relationship A->B (similar)
5. Create relationship B->C (expands)
6. Get graph from A (depth 2)
7. Find related from A

### Workflow 4: Search Variations
1. Create memories with various tags
2. Test keyword search
3. Test tag search (OR)
4. Test tag search (AND)
5. Test date range search
6. Verify result formats match

---

## Test Execution Log

| Test ID | Timestamp | LM Result | UT Result | Match | Notes |
|---------|-----------|-----------|-----------|-------|-------|
| | | | | | |

