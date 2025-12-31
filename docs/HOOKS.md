# Ultrathink Auto-Memory Hooks

Automatically capture knowledge from Claude Code sessions using hooks.

---

## Overview

Ultrathink hooks integrate with Claude Code's hook system to:

1. **Capture memories automatically** when you make decisions or solve problems
2. **Load relevant context** when starting new sessions
3. **Track configuration changes** to important files

No manual "remember this" needed - knowledge flows into your memory system as you work.

---

## Quick Setup

### 1. Create Hooks Directory

```bash
mkdir -p ~/.claude/hooks
```

### 2. Install Hook Scripts

**Option A: Download from GitHub**

```bash
curl -o ~/.claude/hooks/ultrathink-memory-capture.py \
  https://raw.githubusercontent.com/MycelicMemory/ultrathink/main/hooks/ultrathink-memory-capture.py

curl -o ~/.claude/hooks/ultrathink-context-loader.py \
  https://raw.githubusercontent.com/MycelicMemory/ultrathink/main/hooks/ultrathink-context-loader.py

chmod +x ~/.claude/hooks/*.py
```

**Option B: Copy from local install**

```bash
cp /path/to/ultrathink/hooks/*.py ~/.claude/hooks/
chmod +x ~/.claude/hooks/*.py
```

### 3. Configure Hooks

Edit `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/ultrathink-memory-capture.py",
            "timeout": 10
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/ultrathink-memory-capture.py",
            "timeout": 15
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/ultrathink-context-loader.py",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

### 4. Restart Claude Code

```bash
claude
```

---

## How It Works

### Memory Capture Hook

**Trigger:** `PostToolUse` (Edit/Write) and `Stop` events

**What it captures:**

| Pattern | Example | Auto-Tag |
|---------|---------|----------|
| Decisions | "because we need...", "decided to..." | `decision` |
| Bug fixes | "the bug was...", "fixed by..." | `debugging` |
| Gotchas | "gotcha:", "TIL:" | `gotcha`, `til` |
| Best practices | "should always/never...", "best practice:" | `best-practices` |
| Important notes | "important:", "remember:" | varies |

**Files tracked:**

| File Pattern | Why |
|--------------|-----|
| `.env*` | Environment configuration |
| `config.*`, `settings.*` | Application configuration |
| `docker-compose.yml`, `Dockerfile` | Container configuration |
| `package.json`, `go.mod`, `Cargo.toml` | Dependency manifests |
| `.github/workflows/*` | CI/CD configuration |
| `schema.*`, `migrations/*` | Database schema |
| `.claude/*` | Claude configuration |

### Context Loader Hook

**Trigger:** `SessionStart` event

**What it does:**

1. Detects project type from files (Go, Python, Node, Rust, etc.)
2. Searches for memories tagged with project name
3. Searches for memories tagged with detected languages
4. Injects top 5 high-importance memories as session context

**Project detection:**

| File | Detected As |
|------|-------------|
| `go.mod` | Go project |
| `package.json` | Node/TypeScript project |
| `requirements.txt`, `pyproject.toml` | Python project |
| `Cargo.toml` | Rust project |
| `docker-compose.yml` | DevOps context |
| `.github/workflows` | CI/CD context |

---

## Configuration Options

### Memory Capture Settings

Edit the top of `ultrathink-memory-capture.py`:

```python
# Rate limiting: minimum seconds between memory stores
RATE_LIMIT_SECONDS = 30

# Maximum memories to extract per session stop
MAX_SESSION_MEMORIES = 3

# Minimum content length for memories
MIN_CONTENT_LENGTH = 40
```

### Context Loader Settings

Edit the top of `ultrathink-context-loader.py`:

```python
# Minimum importance for memories to be loaded as context
MIN_IMPORTANCE = 6

# Maximum memories to load
MAX_MEMORIES = 5
```

---

## Features

### Deduplication

Memories are deduplicated using content hashing:
- Same content won't be stored twice within 24 hours
- Normalized comparison (case-insensitive, whitespace-normalized)

### Rate Limiting

Prevents memory spam:
- Minimum 30 seconds between stores (configurable)
- Maximum 3 memories per session stop

### Project Tagging

Memories are automatically tagged with project context:
- `project:<project-name>` tag added to all memories
- Makes memories searchable by project

### Importance Scoring

Automatic importance calculation based on:
- Decision patterns (+2)
- Bug fix indicators (+3)
- Important file changes (+1 to +3)
- Content length (+1 for >200 chars, +2 for >500)
- Explicit importance markers (+1)

### Domain Detection

Auto-detects knowledge domain:
- `devops`: Docker, Kubernetes, Terraform
- `databases`: SQL, PostgreSQL, Redis
- `frontend`: React, Vue, CSS
- `backend`: API, server, REST
- `security`: Auth, encryption, JWT
- `programming`: General code (fallback)

---

## Logging

Hooks log to `~/.claude/hooks/memory-capture.log`:

```bash
# View recent logs
tail -f ~/.claude/hooks/memory-capture.log
```

Log format:
```
2024-03-15T10:30:45.123456 [INFO] Hook event: PostToolUse
2024-03-15T10:30:45.234567 [INFO] Stored memory: [myproject] Modified config.json...
2024-03-15T10:30:45.345678 [DEBUG] Rate limited, skipping: ...
```

---

## Troubleshooting

### Hooks Not Running

1. Check hook scripts are executable:
   ```bash
   ls -la ~/.claude/hooks/
   chmod +x ~/.claude/hooks/*.py
   ```

2. Verify settings.json syntax:
   ```bash
   python3 -m json.tool ~/.claude/settings.json
   ```

3. Check logs for errors:
   ```bash
   cat ~/.claude/hooks/memory-capture.log
   ```

### Memories Not Being Stored

1. Check ultrathink is installed:
   ```bash
   which ultrathink
   ultrathink --version
   ```

2. Verify rate limiting isn't blocking:
   ```bash
   cat ~/.claude/hooks/.memory-state.json
   ```

3. Check the content matches capture patterns

### Context Not Loading

1. Verify SessionStart hook is configured
2. Check if memories exist with project tags:
   ```bash
   ultrathink search --tags "project:myproject"
   ```

3. Check minimum importance threshold (default: 6)

---

## Customization

### Adding Custom Patterns

Edit `DECISION_PATTERNS` in `ultrathink-memory-capture.py`:

```python
DECISION_PATTERNS = [
    (r"because\s+\w+", 2),
    (r"decided\s+to", 2),
    # Add your patterns:
    (r"lesson\s+learned", 3),
    (r"pro\s+tip", 2),
]
```

### Adding Custom File Tracking

Edit `IMPORTANT_FILES` in `ultrathink-memory-capture.py`:

```python
IMPORTANT_FILES = {
    r"\.env": 2,
    r"config\.\w+$": 2,
    # Add your patterns:
    r"\.terraform": 2,
    r"helm/values": 2,
}
```

### Adding Project Detection

Edit `PROJECT_MARKERS` in `ultrathink-context-loader.py`:

```python
PROJECT_MARKERS = {
    "go.mod": {"lang": "go", "tags": ["go"], "domain": "programming"},
    # Add your markers:
    "mix.exs": {"lang": "elixir", "tags": ["elixir"], "domain": "programming"},
    "build.gradle": {"lang": "java", "tags": ["java", "gradle"], "domain": "programming"},
}
```

---

## Hook Events Reference

| Event | When Fired | Use Case |
|-------|------------|----------|
| `SessionStart` | Session begins | Load relevant context |
| `PostToolUse` | After tool executes | Capture file changes |
| `Stop` | Claude stops responding | Analyze session for insights |
| `PreToolUse` | Before tool executes | (Not used by ultrathink) |

---

## Privacy Considerations

- Hooks only capture content matching specific patterns
- Credentials in `.env` files are NOT captured (pattern matching, not file content)
- All data stays local in your SQLite database
- No data is sent to external services

---

## Disabling Hooks

Remove the hook entries from `~/.claude/settings.json` or delete the hook files:

```bash
rm ~/.claude/hooks/ultrathink-*.py
```

---

## Next Steps

- [Quick Start Guide](QUICKSTART.md)
- [Use Cases](USE_CASES.md)
- [REST API Reference](../README.md#rest-api)
