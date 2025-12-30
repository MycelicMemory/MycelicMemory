# Development Environment Setup

## Prerequisites Installation

### 1. Install Go 1.21+

**macOS (Homebrew):**
```bash
brew install go
```

**Verify installation:**
```bash
go version  # Should show go1.21 or higher
```

**Set Go environment variables:**
```bash
# Add to ~/.zshrc or ~/.bashrc
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

### 2. Install SQLite 3.50.0+

**Check current version:**
```bash
sqlite3 --version
```

**Install/upgrade if needed (macOS):**
```bash
brew install sqlite3
```

### 3. Install Node.js 16+ (for npm wrapper)

**Check current version:**
```bash
node --version
```

**Install if needed:**
```bash
brew install node
```

### 4. Optional: Install Ollama (for AI features)

```bash
brew install ollama
ollama serve  # Start Ollama service
ollama pull nomic-embed-text  # Download embedding model (768-dim)
ollama pull qwen2.5:3b  # Download chat model
```

### 5. Optional: Install Qdrant (for vector search)

**Using Docker:**
```bash
docker run -p 6333:6333 qdrant/qdrant
```

**Or via Homebrew:**
```bash
brew install qdrant
```

## Project Setup

### 1. Clone and Initialize

```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink
```

### 2. Install Go Dependencies

```bash
go mod download
go mod tidy
```

### 3. Build the Project

```bash
go build -o ultrathink cmd/ultrathink/main.go
```

### 4. Run Tests

```bash
go test ./...
```

### 5. Run the Application

```bash
./ultrathink --help
```

## Verification

Run the following to verify your environment is ready:

```bash
# Check Go
go version

# Check SQLite
sqlite3 --version

# Check Node.js (for npm wrapper)
node --version
npm --version

# Optional: Check Ollama
ollama list

# Optional: Check Qdrant
curl http://localhost:6333/health
```

## Next Steps

1. Complete Phase 1 setup (dependencies, structure)
2. Implement Phase 2 (Database layer)
3. Follow the GitHub issues in order

## Troubleshooting

**Go command not found:**
- Make sure Go is installed: `brew install go`
- Verify GOPATH is set: `echo $GOPATH`

**SQLite too old:**
- Upgrade: `brew upgrade sqlite3`
- Verify: `sqlite3 --version`

**Build errors:**
- Clean and rebuild: `go clean && go build`
- Update dependencies: `go mod tidy`

---

For detailed implementation plan, see [docs/BUILD_PLAN.md](docs/BUILD_PLAN.md)
