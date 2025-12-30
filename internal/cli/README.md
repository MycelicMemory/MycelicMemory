# CLI Package

## Purpose

Command-line interface with 32+ commands using Cobra framework.

## Components

### root.go
- Root command setup
- Global flags
- Command registration

### commands/
- Individual command implementations
- Output formatting
- Interactive prompts

### formatter.go
- CLI output formatting
- Color support
- Table rendering

## Verified Commands (32+)

**1. Core Memory (6)**
- remember, search, get, list, update, forget

**2. Relationships (4)**
- relate, find_related, discover, map_graph

**3. Organization (7)**
- list_categories, create_category, categorize, category_stats
- list_domains, create_domain, domain_stats

**4. Sessions (2)**
- list_sessions, session_stats

**5. Analysis (1 with 4 modes)**
- analyze (summarize, question, analyze, temporal_patterns)

**6. Service (8)**
- start, stop, status, ps, kill, kill_all, doctor, validate

**7. Setup (4)**
- setup, install mcp, license activate, license status

## Verified Output Format

```
Memory List
===========

Found 1 memories

1. Go routines enable concurrent programming
   ID: 19e71855... | Importance: 9/10 | Tags: golang, concurrency

💡 Suggestions:
   💡 View details: ultrathink get <id>
```

## Related Issues

- #17: Implement 32+ CLI commands using Cobra
