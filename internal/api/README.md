# REST API Package

## Purpose

HTTP REST API server with all 27 verified endpoints.

## Components

### server.go
- Gin HTTP server setup
- CORS configuration
- Route registration
- Middleware chain

### handlers.go
- Request handlers for all 27 endpoints
- Request validation
- Response formatting

### response.go
- Standard response format (success, message, data)
- Error response helpers
- HTTP status code mapping

### middleware.go
- Authentication middleware
- Rate limiting
- Request logging
- Error recovery

## Verified Endpoints (27 total)

**1. Memory Operations (10)**
- POST /api/v1/memories
- GET /api/v1/memories
- GET /api/v1/memories/search
- POST /api/v1/memories/search
- POST /api/v1/memories/search/intelligent
- GET /api/v1/memories/:id
- PUT /api/v1/memories/:id
- DELETE /api/v1/memories/:id
- GET /api/v1/memories/stats
- GET /api/v1/memories/:id/related

**2. AI Operations (1)**
- POST /api/v1/analyze

**3. Relationships (3)**
- POST /api/v1/relationships
- POST /api/v1/relationships/discover
- GET /api/v1/memories/:id/graph

**4. Categories (4)**
- POST /api/v1/categories
- GET /api/v1/categories
- POST /api/v1/memories/:id/categorize
- GET /api/v1/categories/stats

**5. Temporal Analysis (4)**
- POST /api/v1/temporal/patterns
- POST /api/v1/temporal/progression
- POST /api/v1/temporal/gaps
- POST /api/v1/temporal/timeline

**6. Advanced Search (2)**
- POST /api/v1/search/tags
- POST /api/v1/search/date-range

**7. System & Management (5)**
- GET /api/v1/health
- GET /api/v1/sessions
- GET /api/v1/stats
- POST /api/v1/domains
- GET /api/v1/domains/:domain/stats

## Verified Response Format

```json
{
  "success": true|false,
  "message": "Human-readable message",
  "data": { ... }
}
```

## Usage Example

```go
server := api.NewServer(config, memoryService)
server.SetupRoutes()
server.Start(":3002")
```

## Related Issues

- #15: Implement standard REST response format
- #16: Implement all 27 REST API endpoints
