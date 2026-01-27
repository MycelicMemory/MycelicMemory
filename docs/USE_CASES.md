# MyclicMemory Use Cases

15 detailed examples showing how to use MyclicMemory for different workflows.

---

## Table of Contents

### For Developers
1. [Code Decision Journal](#1-code-decision-journal)
2. [Debugging Knowledge Base](#2-debugging-knowledge-base)
3. [API Reference Cache](#3-api-reference-cache)
4. [Configuration Vault](#4-configuration-vault)
5. [Learning Log](#5-learning-log)

### For Teams
6. [Project Context](#6-project-context)
7. [Onboarding Assistant](#7-onboarding-assistant)
8. [Best Practices Library](#8-best-practices-library)
9. [Incident Postmortems](#9-incident-postmortems)

### For Research
10. [Literature Notes](#10-literature-notes)
11. [Concept Relationships](#11-concept-relationships)
12. [Progress Tracking](#12-progress-tracking)
13. [Citation Manager](#13-citation-manager)

### For Personal Knowledge
14. [Second Brain](#14-second-brain)
15. [Daily Learnings (TIL)](#15-daily-learnings-til)

---

## For Developers

### 1. Code Decision Journal

**Problem:** You make architectural decisions but forget the rationale months later.

**Solution:** Store decisions with context and reasoning.

```
You: "Remember: We chose PostgreSQL over MongoDB for the user service because we need
     strong ACID transactions for payment processing. MongoDB's eventual consistency
     would risk duplicate charges. Importance: 9, tags: architecture, database, decision"
```

**Search later:**
```
You: "Why did we choose PostgreSQL for user service?"
Claude: [Searches memories and retrieves the decision with full context]
```

**MCP Tool Usage:**
```json
{
  "tool": "store_memory",
  "arguments": {
    "content": "Chose PostgreSQL over MongoDB for user service: need ACID transactions for payments",
    "importance": 9,
    "tags": ["architecture", "database", "decision", "user-service"],
    "domain": "databases"
  }
}
```

---

### 2. Debugging Knowledge Base

**Problem:** You solve the same bugs repeatedly, forgetting previous solutions.

**Solution:** Store debugging insights as you solve problems.

```
You: "Gotcha: React 18 strict mode runs useEffect twice in development.
     This isn't a bug - it's intentional to help find side effect issues.
     Fix: ensure effects are idempotent or use cleanup functions."
```

**Search later:**
```
You: "Why is my useEffect running twice?"
Claude: [Finds the React 18 strict mode memory]
```

**Auto-capture with hooks:** The memory-capture hook automatically detects "gotcha", "bug was", "fix:" patterns and stores them.

---

### 3. API Reference Cache

**Problem:** You keep looking up the same API patterns.

**Solution:** Store frequently-used API patterns with examples.

```
You: "Remember this Go pattern for graceful HTTP shutdown:

     srv := &http.Server{Addr: ':8080'}
     go srv.ListenAndServe()

     quit := make(chan os.Signal, 1)
     signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
     <-quit

     ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
     defer cancel()
     srv.Shutdown(ctx)

     Tags: go, http, patterns, graceful-shutdown"
```

**Search later:**
```
You: "How do I gracefully shutdown an HTTP server in Go?"
```

---

### 4. Configuration Vault

**Problem:** Environment-specific configs are scattered across docs, Slack, and your memory.

**Solution:** Store configuration knowledge in searchable memories.

```
You: "Production database config for payment-service:
     - Host: payments-db.prod.internal
     - Port: 5432
     - Pool size: 20 (increased from 10 after load testing)
     - SSL: required
     - Connection timeout: 10s
     Tags: configuration, production, payment-service, database"
```

**Note:** Don't store actual credentials - just configuration patterns and references.

---

### 5. Learning Log

**Problem:** You learn new concepts but forget them without practice.

**Solution:** Document learnings as you go, building a searchable knowledge base.

```
You: "TIL: Rust's ownership system has three rules:
     1. Each value has exactly one owner
     2. When owner goes out of scope, value is dropped
     3. References must not outlive the owner

     This prevents memory bugs at compile time without garbage collection.
     Importance: 8, tags: rust, ownership, memory-safety"
```

**Review later:**
```
You: "Summarize what I've learned about Rust this month"
Claude: [Uses analysis tool to summarize Rust-tagged memories]
```

---

## For Teams

### 6. Project Context

**Problem:** Context is lost between sessions when working on the same project.

**Solution:** Use project-tagged memories that load automatically.

**With hooks enabled**, memories are automatically tagged with `project:<name>`:

```
You: "For this project: we use Jest for testing, React Query for data fetching,
     and Tailwind for styling. Build with 'npm run build', test with 'npm test'."
```

The context-loader hook will automatically surface these memories when you start a new session in the project directory.

---

### 7. Onboarding Assistant

**Problem:** New team members ask the same questions repeatedly.

**Solution:** Build a searchable knowledge base of common questions.

```
You: "Store these onboarding notes:

     Q: How do I set up the dev environment?
     A: Run 'make setup' which installs deps, creates .env from template,
        and starts Docker services. Takes about 5 minutes.

     Q: Where are the API docs?
     A: Swagger UI at localhost:3000/docs when running locally.
        Production docs at docs.ourcompany.com/api

     Tags: onboarding, setup, faq"
```

New team members can ask Claude: "How do I set up the dev environment?"

---

### 8. Best Practices Library

**Problem:** Code review feedback repeats the same suggestions.

**Solution:** Store coding standards as searchable memories.

```
You: "Best practice for our Go services:

     Error handling:
     - Always wrap errors with context: fmt.Errorf('failed to X: %w', err)
     - Use sentinel errors for expected conditions
     - Log at the boundary, not everywhere

     Naming:
     - Receivers: single letter (s for Service)
     - Interfaces: -er suffix (Reader, Writer)
     - Packages: single word, no underscores

     Importance: 9, tags: best-practices, go, code-review"
```

---

### 9. Incident Postmortems

**Problem:** Similar incidents recur because learnings aren't easily searchable.

**Solution:** Store incident summaries with tags for easy retrieval.

```
You: "Incident 2024-03-15: Payment service outage

     Duration: 45 minutes
     Impact: 12% of payments failed
     Root cause: Database connection pool exhausted due to slow query

     What went wrong:
     - Missing index on orders.created_at column
     - No connection pool monitoring alerts

     Fixes applied:
     - Added index (query time 3s -> 10ms)
     - Added pool exhaustion alert
     - Increased pool size as buffer

     Tags: incident, payment-service, database, postmortem
     Importance: 9"
```

**Search later:**
```
You: "Have we had connection pool issues before?"
```

---

## For Research

### 10. Literature Notes

**Problem:** You read papers but forget key insights.

**Solution:** Store paper summaries with searchable tags.

```
You: "Paper: Attention Is All You Need (Vaswani et al., 2017)

     Key insight: Self-attention can replace recurrence entirely for sequence modeling.

     Architecture: Encoder-decoder with multi-head attention
     - Query, Key, Value projections
     - Scaled dot-product attention: softmax(QK^T/sqrt(d_k))V
     - Multi-head allows attending to different representation subspaces

     Impact: Foundation for BERT, GPT, and modern LLMs

     Tags: paper, transformers, attention, deep-learning, nlp
     Importance: 10"
```

---

### 11. Concept Relationships

**Problem:** You learn concepts in isolation without connecting them.

**Solution:** Use the relationships tool to map connections.

```
You: "How does the transformer attention mechanism relate to what I know about RNNs?"

Claude: [Searches for transformer and RNN memories, then creates a relationship]
```

**Explicit relationship:**
```
You: "Create a relationship between my transformer notes and LSTM notes -
     mark it as 'replaces' since transformers largely replaced LSTMs for NLP"
```

**MCP Tool:**
```json
{
  "tool": "relationships",
  "arguments": {
    "relationship_type": "create",
    "source_memory_id": "transformer-uuid",
    "target_memory_id": "lstm-uuid",
    "relationship_type_enum": "replaces",
    "strength": 0.9
  }
}
```

---

### 12. Progress Tracking

**Problem:** Hard to see learning progress over time.

**Solution:** Use temporal analysis to track knowledge growth.

```
You: "Show me my learning progression for machine learning over the past month"

Claude: [Uses analysis with temporal_patterns]
```

**MCP Tool:**
```json
{
  "tool": "analysis",
  "arguments": {
    "analysis_type": "temporal_patterns",
    "concept": "machine learning",
    "temporal_timeframe": "month"
  }
}
```

This shows:
- Number of memories added over time
- Key themes that emerged
- Knowledge gaps identified

---

### 13. Citation Manager

**Problem:** You forget where you learned things.

**Solution:** Store sources with your memories.

```
You: "From 'Clean Code' by Robert Martin, Chapter 2:

     Functions should do one thing, do it well, and do it only.

     Signs a function does too much:
     - More than one level of abstraction
     - Multiple reasons to change
     - Hard to name concisely

     Source: Clean Code, p.35-40
     Tags: book, clean-code, functions, best-practices"
```

**Search later:**
```
You: "What did Clean Code say about functions?"
```

---

## For Personal Knowledge

### 14. Second Brain

**Problem:** Knowledge scattered across notes, bookmarks, and memory.

**Solution:** Use MyclicMemory as your searchable external brain.

```
You: "Whenever I learn something interesting, store it. For example:

     The Dunning-Kruger effect isn't just about incompetent people being
     overconfident. The original study showed experts slightly underestimate
     their abilities too. The main finding is that metacognition (knowing
     what you don't know) improves with expertise.

     Tags: psychology, cognitive-bias, learning"
```

**Build the habit:**
- End sessions with "What should I remember from today?"
- Ask "What do I know about X?" before researching
- Use hooks to auto-capture insights

---

### 15. Daily Learnings (TIL)

**Problem:** You learn something new every day but rarely retain it.

**Solution:** Capture TILs with automatic tagging.

```
You: "TIL: The 'git reflog' command shows all reference updates, even ones
     not in the regular log. This means you can recover 'lost' commits
     after a bad rebase or reset. Commits aren't truly gone until garbage
     collected (usually 90 days)."
```

The memory-capture hook automatically detects "TIL" and stores with appropriate tags.

**Weekly review:**
```
You: "Summarize my TILs from this week"

Claude: [Uses analysis(summarize) with timeframe='week' filtered to TIL tag]
```

---

## Pro Tips

### 1. Use Importance Wisely

- **1-3:** Trivia, temporary notes
- **4-6:** Useful but not critical
- **7-8:** Important patterns and decisions
- **9-10:** Critical knowledge, core concepts

### 2. Tag Consistently

Create a tagging taxonomy:
- **Type:** decision, gotcha, pattern, til, best-practice
- **Domain:** frontend, backend, devops, database
- **Project:** project:myapp, project:api
- **Language:** go, python, typescript, rust

### 3. Review Regularly

```
You: "What high-importance memories have I added this month?"
You: "What patterns do you see across my debugging notes?"
You: "Summarize my learning progress in databases"
```

### 4. Connect Knowledge

Use relationships to build a knowledge graph:
```
You: "Find concepts related to my Docker notes"
You: "Map how my Kubernetes knowledge connects to Docker"
```

---

## Next Steps

- [Set up auto-capture hooks](HOOKS.md)
- [Quick Start Guide](QUICKSTART.md)
- [REST API Reference](../README.md#rest-api)
