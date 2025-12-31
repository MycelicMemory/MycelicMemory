# LoCoMo Benchmark Implementation

This directory contains the implementation of the [LoCoMo benchmark](https://github.com/snap-research/locomo) for evaluating ultrathink's long-term conversational memory capabilities.

## Overview

LoCoMo (Long-term Conversational Memory) is an ACL 2024 benchmark that evaluates LLM agents on their ability to:
- Answer questions requiring long-term memory recall
- Summarize events across extended conversation histories
- Handle multiple conversation sessions spanning weeks/months

### Dataset Statistics
- **10 conversations** with human-verified annotations
- **~300 turns** per conversation average
- **~9K tokens** per conversation average
- **Up to 35 sessions** per conversation
- **5 question categories**: Single-hop, Multi-hop, Temporal, Commonsense, Adversarial

---

## Architecture

### How Ultrathink Maps to LoCoMo

```
┌─────────────────────────────────────────────────────────────────────┐
│                         LoCoMo Dataset                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │
│  │ Conversation│  │ Conversation│  │     ...     │  (10 total)      │
│  │     #1      │  │     #2      │  │             │                  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                  │
└─────────┼────────────────┼────────────────┼─────────────────────────┘
          │                │                │
          ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Ultrathink Ingestion                            │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Each dialogue turn → Memory                                 │    │
│  │  - content: dialogue text                                    │    │
│  │  - tags: [locomo, conv_N, session_M, speaker_X]             │    │
│  │  - domain: locomo-benchmark                                  │    │
│  │  - importance: based on content significance                 │    │
│  │  - timestamp: session datetime                               │    │
│  └─────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Persona data → High-importance memories                     │    │
│  │  - importance: 10                                            │    │
│  │  - tags: [locomo, persona, speaker_X]                       │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Ultrathink Storage                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │
│  │   SQLite    │  │   Qdrant    │  │   Ollama    │                  │
│  │  (memories) │  │  (vectors)  │  │    (AI)     │                  │
│  └─────────────┘  └─────────────┘  └─────────────┘                  │
└─────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Evaluation Tasks                                │
│  ┌───────────────────┐  ┌───────────────────┐                       │
│  │   QA Evaluation   │  │ Event Summarization│                       │
│  │                   │  │                   │                       │
│  │ For each question:│  │ For each conv:    │                       │
│  │ 1. Search memories│  │ 1. Retrieve all   │                       │
│  │ 2. AI analysis    │  │ 2. Summarize      │                       │
│  │ 3. Compare answer │  │ 3. Compare events │                       │
│  │ 4. Calculate F1   │  │ 4. Score          │                       │
│  └───────────────────┘  └───────────────────┘                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Data Ingestion

**File: `ingest.go`**

```go
// LoCoMo data structures
type LoCoMoDataset struct {
    Conversations []Conversation `json:"conversations"`
}

type Conversation struct {
    ID        string              `json:"id"`
    SpeakerA  string              `json:"speaker_a"`
    SpeakerB  string              `json:"speaker_b"`
    Personas  map[string][]string `json:"personas"`
    Sessions  map[string][]Turn   `json:"sessions"`
    Dates     map[string]string   `json:"dates"`
    QA        []QAAnnotation      `json:"qa"`
    Events    []EventAnnotation   `json:"events"`
}

type Turn struct {
    DiaID   string `json:"dia_id"`
    Speaker string `json:"speaker"`
    Content string `json:"content"`
}
```

**Ingestion Process:**
1. Download `locomo10.json` from GitHub
2. Parse JSON into Go structures
3. For each conversation:
   - Store personas as high-importance memories
   - For each session:
     - Store each turn as a memory with metadata
     - Preserve dialogue IDs for evidence matching
4. Create index for efficient QA evaluation

### Phase 2: QA Evaluation

**File: `qa_eval.go`**

**Question Categories:**
| Category | Description | Retrieval Strategy |
|----------|-------------|-------------------|
| Single-hop | Direct retrieval | Semantic search |
| Multi-hop | Combine multiple memories | Multi-query search |
| Temporal | Time-based reasoning | Date-filtered search |
| Commonsense | External knowledge | AI reasoning |
| Adversarial | Robustness testing | All strategies |

**Evaluation Loop:**
```go
func EvaluateQA(conv Conversation, strategy RetrievalStrategy) Results {
    results := Results{}

    for _, qa := range conv.QA {
        // 1. Retrieve relevant memories
        memories := strategy.Retrieve(qa.Question)

        // 2. Generate answer using AI
        answer := aiManager.AnswerQuestion(qa.Question, memories)

        // 3. Calculate F1 score
        f1 := calculateF1(answer, qa.Answer)

        results.Add(qa.Category, f1)
    }

    return results
}
```

### Phase 3: Retrieval Strategies

**File: `retrieval.go`**

1. **Direct Context**
   - Pass all memories as context
   - Limited by context window

2. **Dialog RAG**
   ```go
   func (d *DialogRAG) Retrieve(question string) []Memory {
       return searchEngine.Search(&SearchOptions{
           Query:  question,
           Limit:  10,
           UseAI:  true,
           Domain: "locomo-benchmark",
       })
   }
   ```

3. **Observation RAG**
   - Pre-generate observations per speaker
   - Store as separate memories
   - Retrieve observations instead of raw dialogue

4. **Summary RAG**
   - Generate session summaries
   - Use summaries for retrieval

### Phase 4: Results & Reporting

**File: `results.go`**

```go
type BenchmarkResults struct {
    Benchmark  string    `json:"benchmark"`
    Timestamp  time.Time `json:"timestamp"`
    Model      string    `json:"model"`
    Strategy   string    `json:"retrieval_strategy"`

    Overall    Metrics           `json:"overall"`
    Categories map[string]Metrics `json:"categories"`
    Questions  []QuestionResult  `json:"per_question"`
}

type Metrics struct {
    F1        float64 `json:"f1"`
    Precision float64 `json:"precision"`
    Recall    float64 `json:"recall"`
    Count     int     `json:"count"`
}
```

---

## File Structure

```
benchmark/locomo/
├── README.md           # This file
├── types.go            # LoCoMo data structures
├── ingest.go           # Data ingestion pipeline
├── qa_eval.go          # QA evaluation logic
├── event_eval.go       # Event summarization evaluation
├── retrieval.go        # Retrieval strategies
├── metrics.go          # F1 score calculation
├── results.go          # Results storage
├── report.go           # Report generation
└── compare.go          # Baseline comparison
```

---

## CLI Commands

### Ingestion
```bash
# Download and ingest LoCoMo dataset
ultrathink benchmark ingest locomo

# Ingest with custom data path
ultrathink benchmark ingest locomo --data-path ./locomo10.json

# Verify ingestion
ultrathink benchmark status locomo
```

### Evaluation
```bash
# Run full QA evaluation
ultrathink benchmark run locomo --task qa

# Run with specific retrieval strategy
ultrathink benchmark run locomo --task qa --retrieval dialog-rag --top-k 10

# Run specific category
ultrathink benchmark run locomo --task qa --category temporal

# Run event summarization
ultrathink benchmark run locomo --task events
```

### Results
```bash
# View results
ultrathink benchmark results locomo

# Export as JSON
ultrathink benchmark results locomo --format json > results.json

# Compare with baselines
ultrathink benchmark compare locomo --baseline gpt4
```

---

## Expected Results

### Published Baselines (LoCoMo Paper)

| Model | Overall F1 | Single-hop | Multi-hop | Temporal | Commonsense | Adversarial |
|-------|------------|------------|-----------|----------|-------------|-------------|
| Human | 87.9 | - | - | - | - | - |
| GPT-4 | 32.1 | - | - | - | - | - |
| GPT-3.5 | 24.2 | - | - | - | - | - |
| Llama-2-70B | 16.9 | - | - | - | - | - |

### Ultrathink Target

With proper retrieval and memory management, we aim to achieve:
- **Dialog RAG**: Competitive with GPT-4 (~30+ F1)
- **Observation RAG**: Improved temporal reasoning
- **Advantage**: Persistent storage enables iterative improvement

---

## Requirements

- Ollama running with embedding model (`nomic-embed-text`)
- Ollama running with chat model (`qwen2.5:3b` or better)
- Qdrant (optional, for vector search)
- ~500MB disk space for dataset and memories

---

## Related Issues

- #22 - Data Ingestion Pipeline
- #23 - QA Evaluation Runner
- #24 - Event Summarization Evaluation
- #25 - RAG Retrieval Strategies
- #26 - Results Dashboard & Comparison
- #27 - Tracking Issue

---

## References

- [LoCoMo Paper (ACL 2024)](https://aclanthology.org/2024.acl-long.747/)
- [LoCoMo GitHub](https://github.com/snap-research/locomo)
- [arXiv](https://arxiv.org/abs/2402.17753)
