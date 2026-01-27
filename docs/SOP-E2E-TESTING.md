# SOP: End-to-End Testing

## Overview

This document describes the comprehensive E2E testing strategy for ultrathink, covering all functionality from installation to usage across all supported platforms.

## Test Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    E2E Test Orchestrator                     │
│                   (run-all-e2e.sh)                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  CLI Tests  │  │  API Tests  │  │ Memory Operations   │  │
│  │             │  │             │  │                     │  │
│  │ • --version │  │ • /health   │  │ • Create memory     │  │
│  │ • --help    │  │ • /stats    │  │ • Search (keyword)  │  │
│  │ • doctor    │  │ • /memories │  │ • Search (semantic) │  │
│  │ • remember  │  │ • /search   │  │ • Update memory     │  │
│  │ • search    │  │ • /domains  │  │ • Delete memory     │  │
│  │ • forget    │  │ • /sessions │  │ • Edge cases        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Test Scripts

### Location
```
scripts/e2e-tests/
├── run-all-e2e.sh      # Orchestrator - runs all tests
├── test-cli.sh         # CLI command tests
├── test-rest-api.sh    # REST API endpoint tests
└── test-memory-ops.sh  # Memory operations tests
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ULTRATHINK_BINARY` | `ultrathink` | Path to binary |
| `API_BASE` | `http://localhost:3099/api/v1` | API base URL |
| `API_PORT` | `3099` | Server port |
| `USE_API` | `false` | Use API mode for memory tests |

## Running Tests Locally

### Prerequisites
- ultrathink binary built or installed
- `curl` and `jq` installed
- Bash shell (Git Bash on Windows)

### Quick Start
```bash
# Run all tests
./scripts/e2e-tests/run-all-e2e.sh

# Run individual test suites
./scripts/e2e-tests/test-cli.sh
./scripts/e2e-tests/test-rest-api.sh
./scripts/e2e-tests/test-memory-ops.sh
```

### Custom Binary Path
```bash
export ULTRATHINK_BINARY=/path/to/ultrathink
./scripts/e2e-tests/run-all-e2e.sh
```

### API-Only Testing
```bash
export USE_API=true
export API_BASE=http://localhost:3099/api/v1
./scripts/e2e-tests/test-memory-ops.sh
```

## CI/CD Integration

### GitHub Actions Workflows

1. **Installation Test** (`installation-test.yml`)
   - Builds binaries for all platforms
   - Tests basic execution and npm installation
   - Runs on every PR and merge

2. **E2E Test** (`e2e-test.yml`)
   - Comprehensive functionality tests
   - Runs on Linux, macOS, Windows
   - Includes Docker isolated test
   - Tests CLI, API, and memory operations

### Workflow Triggers
- Push to `development` or `main`
- Pull requests to these branches
- Manual trigger via workflow dispatch

## Test Categories

### 1. CLI Tests (`test-cli.sh`)

| Test | Command | Expected |
|------|---------|----------|
| Version | `--version` | Returns version string |
| Help | `--help` | Shows usage info |
| Doctor | `doctor` | Runs without crash |
| Remember | `remember "content"` | Returns memory ID |
| Search | `search "query"` | Returns results |
| Forget | `forget <id>` | Deletes memory |

### 2. REST API Tests (`test-rest-api.sh`)

| Endpoint | Method | Test |
|----------|--------|------|
| `/health` | GET | Returns health status |
| `/stats` | GET | Returns statistics |
| `/memories` | GET | Lists memories |
| `/memories` | POST | Creates memory |
| `/memories/:id` | GET | Retrieves memory |
| `/memories/:id` | PUT | Updates memory |
| `/memories/:id` | DELETE | Deletes memory |
| `/memories/search` | POST | Searches memories |
| `/domains` | GET | Lists domains |

### 3. Memory Operations Tests (`test-memory-ops.sh`)

| Category | Tests |
|----------|-------|
| Basic Ops | Create, retrieve, update, delete |
| Search | Keyword, semantic, domain-filtered |
| Metadata | Tags, importance, domain |
| Edge Cases | Empty query, special chars, long content |

## Isolated Environment Testing

### Docker (Recommended)
```bash
docker run --rm -it ubuntu:22.04 bash
apt-get update && apt-get install -y curl jq ca-certificates

# Download binary
curl -L -o ultrathink https://github.com/MycelicMemory/ultrathink/releases/latest/download/ultrathink-linux-x64
chmod +x ultrathink
mv ultrathink /usr/local/bin/

# Clone repo for test scripts
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink

# Run tests
./scripts/e2e-tests/run-all-e2e.sh
```

### Windows Sandbox
1. Open Windows Sandbox
2. Download binary and test scripts
3. Run in PowerShell/Git Bash

### Fresh VM
1. Create minimal OS install
2. Install only required deps (curl, jq)
3. Download binary
4. Run test scripts

## Adding New Tests

### Adding a CLI Test
```bash
# In test-cli.sh
log_info "Testing new-command"
if $BINARY new-command 2>&1 | grep -q "expected"; then
    log_pass "new-command works"
else
    log_fail "new-command failed"
fi
```

### Adding an API Test
```bash
# In test-rest-api.sh
log_info "Testing GET /new-endpoint"
RESPONSE=$(curl -s "$API_BASE/new-endpoint")
if echo "$RESPONSE" | grep -qi "expected"; then
    log_pass "GET /new-endpoint works"
else
    log_fail "GET /new-endpoint failed"
fi
```

### Adding Memory Operation Test
```bash
# In test-memory-ops.sh
log_info "Testing new memory operation"
# ... test implementation
if [ condition ]; then
    log_pass "New operation works"
else
    log_fail "New operation failed"
fi
```

## Troubleshooting

### Server Won't Start
```bash
# Check if port is in use
lsof -i :3099
netstat -tulpn | grep 3099

# Kill existing process
pkill ultrathink
```

### Tests Fail on Windows
- Use Git Bash, not PowerShell
- Ensure line endings are LF
- Check path separators

### Semantic Search Fails
- Requires Ollama running
- Test is non-blocking (logs warning)

## Metrics and Reporting

Each test script outputs:
- Pass/fail count
- Individual test results
- Exit code (0 = all pass, 1 = failures)

The orchestrator aggregates results and provides summary.

## Best Practices

1. **Run locally before pushing** - Catch issues early
2. **Check CI logs on failure** - Artifacts uploaded on failure
3. **Keep tests independent** - Each test cleans up after itself
4. **Use meaningful assertions** - Check specific output, not just exit codes
5. **Handle optional features** - Gracefully skip if deps missing (Ollama, etc.)
