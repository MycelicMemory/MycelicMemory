# SOP: Installation Testing in Isolated Environments

## Purpose
This document provides reproducible steps for testing mycelicmemory installation in clean environments with no pre-existing dependencies.

## Automated Testing
Installation tests run automatically via GitHub Actions when:
- A PR is opened against `development` or `main`
- Code is merged into `development` or `main`
- Manually triggered via workflow dispatch

See: `.github/workflows/installation-test.yml`

---

## Manual Testing Procedures

### Option 1: Docker (Recommended for Linux)

**Prerequisites:** Docker installed

```bash
# Test in clean Ubuntu environment
docker run --rm -it ubuntu:22.04 bash

# Inside container:
apt-get update && apt-get install -y curl ca-certificates

# Download binary directly
curl -L -o mycelicmemory https://github.com/MycelicMemory/mycelicmemory/releases/latest/download/mycelicmemory-linux-x64
chmod +x mycelicmemory
./mycelicmemory --version
./mycelicmemory --help
./mycelicmemory doctor
```

**With Node.js (npm installation):**
```bash
docker run --rm -it node:20 bash

# Inside container:
npm install -g github:MycelicMemory/mycelicmemory
mycelicmemory --version
mycelicmemory doctor
```

### Option 2: Fresh VM (Windows/macOS/Linux)

**Prerequisites:** VirtualBox, VMware, or cloud VM

1. **Create fresh VM:**
   - Windows: Use Windows 11 ISO
   - macOS: Use macOS VM (requires Apple hardware)
   - Linux: Ubuntu 22.04 minimal install

2. **Test binary installation:**
   ```bash
   # Download appropriate binary from releases
   # https://github.com/MycelicMemory/mycelicmemory/releases

   # macOS/Linux:
   curl -L -o mycelicmemory https://github.com/MycelicMemory/mycelicmemory/releases/latest/download/mycelicmemory-<platform>
   chmod +x mycelicmemory
   ./mycelicmemory --version

   # Windows (PowerShell):
   Invoke-WebRequest -Uri "https://github.com/MycelicMemory/mycelicmemory/releases/latest/download/mycelicmemory-windows-x64.exe" -OutFile "mycelicmemory.exe"
   .\mycelicmemory.exe --version
   ```

3. **Test npm installation:**
   ```bash
   # Install Node.js first if not present
   npm install -g github:MycelicMemory/mycelicmemory
   mycelicmemory --version
   ```

### Option 3: GitHub Codespaces

1. Create new codespace from mycelicmemory repo
2. In terminal:
   ```bash
   # Build from source
   go build -tags fts5 -o mycelicmemory ./cmd/mycelicmemory
   ./mycelicmemory --version

   # Or test npm install
   npm install -g github:MycelicMemory/mycelicmemory
   mycelicmemory --version
   ```

### Option 4: Windows Sandbox

**Prerequisites:** Windows 10/11 Pro with Windows Sandbox enabled

1. Open Windows Sandbox (fresh Windows each time)
2. Download binary or install Node.js
3. Test installation

---

## Test Checklist

### Basic Functionality
- [ ] Binary executes without errors
- [ ] `--version` shows correct version
- [ ] `--help` displays usage information
- [ ] `doctor` command runs (may show missing optional deps)

### Database Initialization
- [ ] Config directory created (`~/.mycelicmemory/`)
- [ ] SQLite database initializes on first use
- [ ] FTS5 full-text search works

### MCP Server Mode
- [ ] `--mcp` flag starts JSON-RPC server
- [ ] Server accepts initialize request
- [ ] Server responds with capabilities

### npm Package
- [ ] `npm install -g github:MycelicMemory/mycelicmemory` succeeds
- [ ] `mycelicmemory` command available in PATH
- [ ] Binary downloads on first run (correct platform detected)

### Optional Dependencies (informational only)
- [ ] Ollama connection attempted (OK to fail if not installed)
- [ ] Qdrant connection attempted (OK to fail if not installed)

---

## Expected Results by Platform

### macOS (arm64/x64)
```
$ mycelicmemory --version
mycelicmemory version 1.2.x

$ mycelicmemory doctor
Checking system dependencies...
✓ SQLite with FTS5: Available
✓ Configuration directory: ~/.mycelicmemory
⚠ Ollama: Not found (optional - install from https://ollama.ai)
⚠ Qdrant: Not found (optional - install from https://qdrant.tech)
```

### Linux (x64/arm64)
```
$ mycelicmemory --version
mycelicmemory version 1.2.x

$ mycelicmemory doctor
Checking system dependencies...
✓ SQLite with FTS5: Available
✓ Configuration directory: ~/.mycelicmemory
⚠ Ollama: Not found (optional)
⚠ Qdrant: Not found (optional)
```

### Windows (x64)
```
PS> .\mycelicmemory.exe --version
mycelicmemory version 1.2.x

PS> .\mycelicmemory.exe doctor
Checking system dependencies...
✓ SQLite with FTS5: Available
✓ Configuration directory: C:\Users\<user>\.mycelicmemory
⚠ Ollama: Not found (optional)
⚠ Qdrant: Not found (optional)
```

---

## Troubleshooting

### Binary won't execute (macOS)
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine mycelicmemory
```

### npm install fails with MODULE_NOT_FOUND
- Ensure you're installing from the correct branch/commit
- The wrapper script downloads binary on first run, not during install

### "go: command not found" during build
- Go is only needed for building from source
- Pre-built binaries don't require Go

### Permission denied
```bash
chmod +x mycelicmemory  # Unix
# Windows: Run as Administrator if needed
```

---

## CI/CD Integration

The `installation-test.yml` workflow automatically:
1. Builds binaries for all 5 platform/arch combinations
2. Tests binary execution on macOS, Linux, Windows
3. Tests npm installation from GitHub
4. Tests MCP server protocol
5. Reports success/failure summary

Trigger manually: Actions → Installation Test → Run workflow
