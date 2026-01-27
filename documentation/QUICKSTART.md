# MyclicMemory Quickstart Guide

Get productive with MyclicMemory in 5 minutes.

## Table of Contents

- [Core Concepts](#core-concepts)
- [Basic Commands](#basic-commands)
- [Using with Claude](#using-with-claude)
- [Building Your Knowledge Base](#building-your-knowledge-base)
- [Searching Memories](#searching-memories)
- [AI Analysis](#ai-analysis)
- [Memory Relationships](#memory-relationships)
- [Best Practices](#best-practices)

---

## Core Concepts

**Memories** are pieces of knowledge you want to persist across conversations:
- Technical decisions and their rationale
- Debugging insights and solutions
- Project-specific conventions
- Learning and discoveries

**Each memory has:**
- `content` - The actual knowledge
- `importance` - Priority level (1-10)
- `tags` - Categories for organization
- `domain` - Knowledge domain (e.g., "programming", "devops")

---

## Basic Commands

### Store a Memory

```bash
# Simple memory
mycelicmemory remember "React useEffect runs after render, not before"

# With metadata
mycelicmemory remember "Never use force push on main branch" \
  --importance 9 \
  --tags git,safety,teamwork \
  --domain devops
```

### Search Memories

```bash
# Basic search
mycelicmemory search "react hooks"

# Search within a domain
mycelicmemory search "deployment" --domain devops

# Search by tags
mycelicmemory search --tags debugging
```

### Analyze Memories

```bash
# Ask a question
mycelicmemory analyze "What do I know about error handling?"

# Get a summary
mycelicmemory analyze --type summarize --timeframe week

# Discover patterns
mycelicmemory analyze --type patterns
```

### System Commands

```bash
# Health check
mycelicmemory doctor

# Version info
mycelicmemory --version

# Start REST API server
mycelicmemory start

# Stop server
mycelicmemory stop
```

---

## Using with Claude

Once MyclicMemory is configured as an MCP server, Claude can use it directly.

### Storing Memories

Ask Claude:
- "Remember that our API uses JWT tokens with 24h expiry"
- "Store this with importance 9: always run tests before deploying"
- "Save this debugging tip tagged with 'python': use pdb.set_trace() for debugging"

### Searching Memories

Ask Claude:
- "What do I know about authentication?"
- "Search my memories for React patterns"
- "Find all memories tagged with 'debugging'"
- "What high-importance memories do I have?"

### Getting Insights

Ask Claude:
- "Summarize what I learned this week"
- "Based on my memories, how should I structure this Go project?"
- "What patterns do you see in my debugging notes?"

---

## Building Your Knowledge Base

### What to Store

**Technical Decisions**
```
"Using PostgreSQL for the user service because we need ACID transactions
and complex queries. Redis considered but rejected for primary storage."
```
Tags: `decision`, `database`, `architecture`
Importance: 8-9

**Debugging Insights**
```
"The 'cannot read property of undefined' error in UserList was caused by
the API returning null instead of empty array. Always default to []."
```
Tags: `debugging`, `gotcha`, `javascript`
Importance: 7-8

**Project Conventions**
```
"This project uses kebab-case for file names and PascalCase for components.
All API routes are prefixed with /api/v1/."
```
Tags: `convention`, `project`
Importance: 9

**Learnings**
```
"TIL: Go interfaces are satisfied implicitly - no 'implements' keyword needed.
Just implement the methods and the type satisfies the interface."
```
Tags: `learning`, `go`
Importance: 6-7

### What NOT to Store

- Generic programming knowledge (Claude already knows this)
- Temporary information (one-time debugging)
- Sensitive data (passwords, API keys, personal info)
- Obvious facts that don't need remembering

### Importance Levels

| Level | Use For |
|-------|---------|
| 9-10 | Critical decisions, security rules, project-defining choices |
| 7-8 | Important patterns, significant learnings, common gotchas |
| 5-6 | Useful tips, preferences, minor conventions |
| 1-4 | Low-priority notes, temporary reminders |

---

## Searching Memories

### Search Types

**Semantic Search** (default)
Finds memories by meaning, not just keywords:
```bash
mycelicmemory search "how to handle errors"
# Finds memories about error handling, exception management, etc.
```

**Tag Search**
Find memories with specific tags:
```bash
mycelicmemory search --tags debugging,python
```

**Domain Search**
Search within a knowledge domain:
```bash
mycelicmemory search "testing" --domain backend
```

**Hybrid Search**
Combine approaches:
```bash
mycelicmemory search "authentication" --tags security --domain backend
```

### Search Tips

1. **Be specific**: "React useState cleanup" works better than "hooks"
2. **Use domains**: Narrow results to relevant areas
3. **Check tags**: Use consistent tags for better filtering
4. **Try synonyms**: If one search fails, rephrase

---

## AI Analysis

Requires Ollama with models installed. See [Installation Guide](INSTALLATION.md#ollama-ai-powered-features).

### Question Answering

Ask questions and get answers based on your stored knowledge:

```bash
mycelicmemory analyze "What authentication methods have I used?"
mycelicmemory analyze "How do I typically structure my Go projects?"
mycelicmemory analyze "What debugging strategies work best for React?"
```

### Summarization

Get summaries of your memories:

```bash
# Recent activity
mycelicmemory analyze --type summarize --timeframe today
mycelicmemory analyze --type summarize --timeframe week

# All memories in a domain
mycelicmemory analyze --type summarize --domain programming
```

### Pattern Discovery

Find patterns across your knowledge:

```bash
mycelicmemory analyze --type patterns
mycelicmemory analyze --type patterns --domain devops
```

---

## Memory Relationships

Connect related memories to build a knowledge graph.

### Create Relationships

```bash
# Link two related memories
mycelicmemory relate <memory-id-1> <memory-id-2> --type similar

# Relationship types:
# - similar: Related concepts
# - references: One mentions the other
# - contradicts: Conflicting information
# - expands: One elaborates on the other
# - causes: Cause and effect
# - enables: One enables the other
```

### Find Related Memories

```bash
# Find memories related to a specific one
mycelicmemory find_related <memory-id>

# View the knowledge graph
mycelicmemory graph <memory-id> --depth 2
```

### Automatic Discovery

Let AI find connections:
```bash
mycelicmemory discover-relationships
```

---

## Best Practices

### 1. Store Knowledge Immediately

When you learn something valuable, store it right away:
```bash
mycelicmemory remember "Just discovered that..." --importance 7 --tags learning
```

### 2. Use Consistent Tags

Create a personal taxonomy:
- `decision` - Architectural choices
- `debugging` - Bug fixes and solutions
- `gotcha` - Non-obvious issues
- `learning` - New knowledge
- `preference` - Personal preferences
- Language tags: `go`, `python`, `javascript`
- Domain tags: `frontend`, `backend`, `devops`

### 3. Review Regularly

Periodically review and update your knowledge:
```bash
# What did I learn recently?
mycelicmemory analyze --type summarize --timeframe week

# Find outdated information
mycelicmemory search "deprecated" --importance 5
```

### 4. Connect Related Knowledge

Build a knowledge graph:
```bash
# After storing related memories
mycelicmemory relate <id1> <id2> --type similar
```

### 5. Set Appropriate Importance

- **9-10**: Must never forget (security rules, critical decisions)
- **7-8**: Important patterns and learnings
- **5-6**: Useful but not critical
- **1-4**: Nice to have

---

## Example Workflow

### Starting a New Project

```bash
# Store project context
mycelicmemory remember "Project X uses Go 1.23, PostgreSQL 16, and Redis 7.
Deployed on AWS EKS with ArgoCD for GitOps." \
  --importance 9 \
  --tags project-x,architecture,stack \
  --domain backend

# Store coding conventions
mycelicmemory remember "Project X conventions: use snake_case for DB columns,
camelCase for Go variables, kebab-case for API endpoints." \
  --importance 8 \
  --tags project-x,convention
```

### During Development

```bash
# Store a decision
mycelicmemory remember "Chose to use pgx over database/sql for better
PostgreSQL features and performance. Considered GORM but wanted more control." \
  --importance 8 \
  --tags decision,database,go

# Store a debugging insight
mycelicmemory remember "The connection pool exhaustion was caused by not closing
rows after scanning. Always defer rows.Close() immediately after Query()." \
  --importance 9 \
  --tags debugging,gotcha,go,database
```

### Getting Help from Claude

With MyclicMemory connected, ask Claude:
- "Based on my memories, what database library should I use for this Go project?"
- "What gotchas should I watch out for with PostgreSQL in Go?"
- "Remind me of the conventions for Project X"

---

## Next Steps

- **Explore advanced features**: Relationships, domains, categories
- **Set up auto-capture**: See [Installation Guide](INSTALLATION.md#auto-memory-hooks)
- **Enable AI features**: Install Ollama for semantic search and analysis
- **Join the community**: https://github.com/MycelicMemory/mycelicmemory
