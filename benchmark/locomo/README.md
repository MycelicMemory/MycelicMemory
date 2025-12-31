# LoCoMo Benchmarks

Memory-augmented benchmarks for evaluating Ultrathink's long-term conversational memory capabilities using the LoCoMo dataset.

---

## Overview

Two benchmark variants are available:

| Benchmark | Dataset | Answer Format | Metric | Questions |
|-----------|---------|---------------|--------|-----------|
| **locomo10** (Primary) | locomo10.json | Free-text | F1 Score | 1,986 |
| locomo_mc10 | locomo-mc10-full.jsonl | 10 choices | Accuracy | ~200 |

Both test 5 conversation memory abilities:

| Category | Description |
|----------|-------------|
| Single-Hop | Direct fact retrieval from a single memory |
| Multi-Hop / Temporal | Time-based reasoning across multiple events |
| Inferential | Combining multiple facts to reach conclusions |
| Open-Domain | Requiring general knowledge beyond conversation |
| Adversarial | Robustness testing with misleading context |

---

## Prerequisites

### Required

- **Ultrathink** installed and built ([see Quick Start](../../docs/QUICKSTART.md))
- **Python 3.8+** with pip
- **API Key** for an OpenAI-compatible LLM (default: DeepSeek)

### Disk Space

- ~50MB for LoCoMo free-response dataset
- ~20MB for LoCoMo multiple-choice dataset
- ~100MB for Python dependencies and NLTK data

---

## Quick Start

### Step 1: Start Ultrathink Server

```bash
# In one terminal
ultrathink start --port 3099

# Verify it's running
curl http://localhost:3099/api/v1/health
```

### Step 2: Configure API Key

**Option A: Environment variable**

```bash
export DEEPSEEK_API_KEY=your_api_key_here
```

**Option B: Config file**

```bash
cd benchmark/locomo
cp config.example.yaml config.yaml
# Edit config.yaml and add your API key under llm.api_key
```

### Step 3: Install Python Dependencies

```bash
cd benchmark/locomo
make setup
```

This installs:
- `openai`, `requests`, `tqdm` - API and progress
- `huggingface_hub` - Dataset downloads
- `numpy`, `pandas` - Data processing
- `nltk` - Text tokenization and stemming
- `fastapi`, `uvicorn` - Bridge server

It also downloads NLTK data (punkt tokenizer, wordnet).

### Step 4: Download Datasets

```bash
# Free-response dataset (1,986 questions)
make download-fr

# Multiple-choice dataset (~200 questions)
make download-mc
```

### Step 5: Run Benchmarks

```bash
# Quick test (20 questions) - recommended first run
make run-fr-quick

# Full benchmark (1,986 questions) - takes several hours
make run-fr

# Multiple-choice quick test (10 questions)
make run-mc-quick

# Full multiple-choice benchmark
make run-mc
```

---

## Understanding Results

### Free-Response (locomo10)

**Primary Metric: Mean F1 Score**

The F1 score measures the overlap between generated and gold answers:
- **Precision**: What fraction of generated tokens appear in gold answer
- **Recall**: What fraction of gold tokens appear in generated answer
- **F1**: Harmonic mean of precision and recall

Scores range from 0.0 to 1.0. Higher is better.

Example output:
```
Category Results:
  single-hop:     F1=0.72  (398 questions)
  multi-hop:      F1=0.58  (412 questions)
  inferential:    F1=0.45  (389 questions)
  open-domain:    F1=0.31  (401 questions)
  adversarial:    F1=0.52  (386 questions)

Overall Mean F1: 0.516
```

### Multiple-Choice (locomo_mc10)

**Primary Metric: Accuracy**

Accuracy = correct answers / total questions. Scores range from 0% to 100%.

### Additional Metrics

Both benchmarks track:
- **Latency**: LLM response time + memory retrieval time (ms)
- **Tokens**: Input/output tokens per question
- **Cost**: Estimated API cost using DeepSeek pricing

---

## Configuration

The benchmark uses `config.yaml` for settings. Copy from example:

```bash
cp config.example.yaml config.yaml
```

### Key Settings

```yaml
llm:
  model: deepseek-chat       # Model name
  api_key: ${DEEPSEEK_API_KEY}  # Or set directly
  base_url: https://api.deepseek.com/v1
  temperature: 0.0           # Deterministic outputs
  max_tokens: 100            # Max answer length

retrieval:
  top_k: 10                  # Memories to retrieve
  min_similarity: 0.0        # Minimum similarity threshold
  use_ai: true               # Use semantic search

baselines:
  fr_tokens: 20000           # Full conversation token estimate
  mc_tokens: 16690

pricing:
  input_price_per_mtok: 0.014   # $/M input tokens
  output_price_per_mtok: 0.056  # $/M output tokens
```

### Using Other LLM Providers

The benchmark supports any OpenAI-compatible API. Configure in `config.yaml`:

**OpenAI:**
```yaml
llm:
  model: gpt-4o-mini
  api_key: ${OPENAI_API_KEY}
  base_url: https://api.openai.com/v1
```

**Anthropic (via OpenRouter):**
```yaml
llm:
  model: anthropic/claude-3-haiku
  api_key: ${OPENROUTER_API_KEY}
  base_url: https://openrouter.ai/api/v1
```

**Local Ollama:**
```yaml
llm:
  model: qwen2.5:7b
  api_key: ollama
  base_url: http://localhost:11434/v1
```

---

## Make Targets Reference

### Setup & Download

| Target | Description |
|--------|-------------|
| `make setup` | Install Python deps + NLTK data |
| `make download-fr` | Download free-response dataset |
| `make download-mc` | Download multiple-choice dataset |

### Free-Response Benchmarks

| Target | Description |
|--------|-------------|
| `make run-fr-quick` | Run 20 questions (quick test) |
| `make run-fr` | Run full benchmark (1,986 questions) |

### Multiple-Choice Benchmarks

| Target | Description |
|--------|-------------|
| `make run-mc-quick` | Run 10 questions (quick test) |
| `make run-mc` | Run full benchmark (~200 questions) |
| `make evaluate-mc` | Generate detailed evaluation metrics |

### Utilities

| Target | Description |
|--------|-------------|
| `make clean` | Remove results and logs |
| `make server` | Start bridge server (port 9876) |

---

## Advanced Usage

### Random Sampling

Run a random subset of questions:

```bash
python -m locomo10.main \
  --dataset data/locomo10.json \
  --output results/sample_results.json \
  --max-questions 100 \
  --random-sample \
  --seed 42
```

### Custom Ultrathink URL

```bash
python -m locomo10.main \
  --dataset data/locomo10.json \
  --output results/results.json \
  --ultrathink-url http://localhost:9999/api/v1
```

---

## Directory Structure

```
benchmark/locomo/
├── shared/                    # Common modules
│   ├── config.py             # Configuration & env loading
│   ├── ultrathink_client.py  # Ultrathink API client
│   ├── llm_call_tracker.py   # LLM API tracking
│   ├── logging_system.py     # Benchmark logging
│   └── prompts.py            # Shared prompts
│
├── locomo10/                  # Free-response benchmark
│   ├── main.py               # Benchmark runner
│   ├── f1_evaluator.py       # F1 scoring with Porter stemming
│   ├── metrics_tracker.py    # F1/latency/cost metrics
│   └── progress_display.py   # Live console output
│
├── locomo_mc10/               # Multiple-choice benchmark
│   ├── main.py               # Benchmark runner
│   ├── download_dataset.py   # HuggingFace dataset fetcher
│   ├── comprehensive_evals.py# Detailed evaluation metrics
│   ├── metrics_tracker.py    # Accuracy/latency/cost metrics
│   └── progress_display.py   # Live console output
│
├── data/                      # Downloaded datasets (gitignored)
├── results/                   # Benchmark results (gitignored)
├── logs/                      # Detailed logs (gitignored)
├── config.yaml               # Your configuration (gitignored)
├── config.example.yaml       # Example configuration
├── Makefile                  # Build commands
└── requirements.txt          # Python dependencies
```

---

## How Benchmarks Work

For each question in the dataset:

1. **Ingest**: Store the conversation history as memories in Ultrathink
2. **Retrieve**: Search for relevant memories using semantic search (top-k)
3. **Generate**: Call LLM with retrieved context to answer the question
4. **Evaluate**: Compare generated answer to gold answer (F1 or accuracy)
5. **Track**: Record latency, tokens, and cost
6. **Cleanup**: Delete memories to prepare for next question

This tests Ultrathink's ability to:
- Store conversational information correctly
- Retrieve relevant context for questions
- Support accurate answer generation

---

## Troubleshooting

### "Connection refused" to Ultrathink

```bash
# Start the server
ultrathink start --port 3099

# Verify
curl http://localhost:3099/api/v1/health
```

### "API key not found"

```bash
# Set environment variable
export DEEPSEEK_API_KEY=your_key

# Or check config.yaml has api_key set
```

### NLTK download errors

```bash
python3 -c "import nltk; nltk.download('punkt'); nltk.download('punkt_tab'); nltk.download('wordnet')"
```

### Slow performance

- Use `--max-questions 20` to run fewer questions
- Check Ollama is running for semantic search
- Ensure you're not running other intensive processes

---

## References

- [LoCoMo Dataset (GitHub)](https://github.com/snap-research/locomo)
- [LoCoMo-MC10 (HuggingFace)](https://huggingface.co/datasets/Percena/locomo-mc10)
- [Mem0 Research Paper](https://arxiv.org/abs/2504.19413)
- [Original LoCoMo Paper (ACL 2024)](https://aclanthology.org/2024.acl-long.747/)
