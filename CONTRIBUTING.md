# Contributing to Ultrathink

Thank you for your interest in contributing to Ultrathink! This document provides guidelines and information for contributors.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Maintain professional communication

## Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite 3.50.0 or higher
- Node.js 16+ (for npm wrapper development)
- Git
- GitHub account

### Setting Up Development Environment

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/ultrathink.git
   cd ultrathink
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Run Tests**
   ```bash
   go test ./...
   ```

4. **Build**
   ```bash
   go build -o ultrathink cmd/ultrathink/main.go
   ```

## Development Workflow

### 1. Create a Branch

Always create a new branch for your work:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

**Branch Naming Convention:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test additions/updates

### 2. Make Changes

- Write clean, readable code
- Follow Go best practices
- Add tests for new functionality
- Update documentation as needed
- Keep commits atomic and focused

### 3. Test Your Changes

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/database/...

# Run benchmarks
go test -bench=. ./...
```

### 4. Commit Your Changes

Follow conventional commit message format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```bash
git commit -m "feat(database): add FTS5 full-text search support"
git commit -m "fix(api): resolve memory leak in search endpoint"
git commit -m "docs(readme): update installation instructions"
```

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:
- Clear title describing the change
- Detailed description of what and why
- Reference any related issues
- Screenshots/examples if applicable

## Coding Standards

### Go Style Guide

Follow [Effective Go](https://golang.org/doc/effective_go.html) and these additional guidelines:

1. **Naming**
   - Use camelCase for unexported names
   - Use PascalCase for exported names
   - Use meaningful, descriptive names
   - Avoid abbreviations unless widely known

2. **Functions**
   - Keep functions small and focused
   - Single responsibility principle
   - Return errors, don't panic (except in truly exceptional cases)
   - Document exported functions

3. **Error Handling**
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to create memory: %w", err)
   }

   // Bad
   if err != nil {
       panic(err)
   }
   ```

4. **Comments**
   - Document all exported types, functions, constants
   - Use complete sentences
   - Explain "why", not "what"

5. **Testing**
   - Test file: `*_test.go`
   - Test function: `func TestFunctionName(t *testing.T)`
   - Use table-driven tests for multiple cases
   - Aim for 80%+ code coverage

### Example Test Structure

```go
func TestMemoryStore(t *testing.T) {
    tests := []struct {
        name    string
        input   Memory
        want    Memory
        wantErr bool
    }{
        {
            name: "valid memory",
            input: Memory{Content: "test", Importance: 5},
            want: Memory{Content: "test", Importance: 5},
            wantErr: false,
        },
        {
            name: "invalid importance",
            input: Memory{Content: "test", Importance: 11},
            want: Memory{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Store(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Store() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Project Structure

```
ultrathink/
├── cmd/
│   └── ultrathink/          # Main application entry point
├── internal/                # Private application code
│   ├── database/           # SQLite operations
│   ├── api/                # REST API handlers
│   ├── mcp/                # MCP server implementation
│   ├── cli/                # CLI commands
│   ├── memory/             # Core memory logic
│   ├── search/             # Search engines
│   ├── relationships/      # Graph algorithms
│   ├── ai/                 # AI integrations
│   └── vector/             # Vector store client
├── pkg/                    # Public libraries
│   └── config/             # Configuration
├── scripts/                # Build/deployment scripts
├── npm/                    # npm wrapper package
└── docs/                   # Documentation
```

## Pull Request Guidelines

### Before Submitting

- [ ] Code follows style guidelines
- [ ] All tests pass
- [ ] New tests added for new features
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] No merge conflicts

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Related Issues
Fixes #123

## How Has This Been Tested?
Description of testing performed

## Checklist
- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

## Issue Guidelines

### Reporting Bugs

Use the bug report template and include:
- Clear, descriptive title
- Steps to reproduce
- Expected vs actual behavior
- System information (OS, Go version, etc.)
- Error messages/logs
- Screenshots if applicable

### Requesting Features

Use the feature request template and include:
- Clear description of the feature
- Use cases and motivation
- Proposed implementation (if any)
- Alternatives considered

## Development Phases

Current development follows a 10-phase roadmap. See [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues) for details:

1. **Phase 1**: Project Setup & Foundation
2. **Phase 2**: Database Layer (SQLite + FTS5)
3. **Phase 3**: Core Memory Logic
4. **Phase 4**: AI Integration
5. **Phase 5**: REST API
6. **Phase 6**: CLI
7. **Phase 7**: MCP Server
8. **Phase 8**: Daemon Management
9. **Phase 9**: npm Distribution
10. **Phase 10**: Build & Deployment

## Questions?

- Check [GitHub Discussions](https://github.com/MycelicMemory/ultrathink/discussions)
- Review existing issues and PRs
- Ask in your PR/issue

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Ultrathink!
