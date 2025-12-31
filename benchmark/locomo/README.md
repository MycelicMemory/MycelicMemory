# LoCoMo Benchmarks

Memory-augmented benchmarks for evaluating ultrathink's long-term conversational memory capabilities using the LoCoMo dataset.

## Overview

Two benchmark variants are available:

| Benchmark | Dataset | Answer Format | Metric | Questions |
|-----------|---------|---------------|--------|-----------|
| **locomo10** (Primary) | locomo10.json | Free-text | F1 Score | 1,986 |
| locomo_mc10 | locomo-mc10-full.jsonl | 10 choices | Accuracy | ~200 |

Both test 5 conversation memory abilities:

| Category | Description |
|----------|-------------|
| Single-Hop | Direct fact retrieval |
| Multi-Hop / Temporal | Time-based reasoning |
| Inferential | Combining multiple facts |
| Open-Domain | General knowledge |
| Adversarial | Robustness testing |

## Quick Start

```bash
# Prerequisites: ultrathink server running on port 3099
ultrathink serve --port 3099

# Set API key
export DEEPSEEK_API_KEY=your_key

# Run free-response benchmark (primary)
cd benchmark/locomo
make setup         # Install dependencies
make download-fr   # Download dataset
make run-fr-quick  # Run 20 questions
make run-fr        # Run full benchmark (1986 questions)

# Run multiple-choice benchmark
make download-mc   # Download dataset
make run-mc-quick  # Run 10 questions
make run-mc        # Run full benchmark
```

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
├── locomo10/                  # Free-response benchmark (primary)
│   ├── main.py               # Benchmark runner
│   ├── f1_evaluator.py       # F1 scoring with Porter stemming
│   ├── metrics_tracker.py    # F1/latency/cost metrics
│   └── progress_display.py   # Live console output
│
├── locomo_mc10/               # Multiple-choice benchmark
│   ├── main.py               # Benchmark runner
│   ├── metrics_tracker.py    # Accuracy/latency/cost metrics
│   ├── progress_display.py   # Live console output
│   ├── comprehensive_evals.py# Evaluation report
│   ├── download_dataset.py   # Dataset downloader
│   └── metrics/              # LLM judge, utils
│
├── data/                      # Downloaded datasets
├── results/                   # Benchmark results
├── logs/                      # Detailed logs
├── Makefile                   # Build commands
└── requirements.txt           # Python dependencies
```

## How It Works

For each question:
1. Ingest conversation history as memories into ultrathink
2. Retrieve relevant memories via semantic search (top-k)
3. Generate answer using retrieved context (not full context)
4. Evaluate: F1 score (locomo10) or exact match (locomo_mc10)
5. Track metrics: latency, tokens, cost
6. Clean up memories for next question

## Metrics

| Metric | locomo10 | locomo_mc10 |
|--------|----------|-------------|
| Primary | Mean F1 Score | Accuracy % |
| Latency | LLM + retrieval time | LLM + retrieval time |
| Tokens | Input/output per question | Input/output per question |
| Cost | DeepSeek pricing | DeepSeek pricing |

## GitHub Actions

The benchmark runs automatically on commits to main:
- Runs locomo10 (free-response) with 20 questions
- Reports Mean F1 score by category
- Artifacts saved for 30 days

## References

- [LoCoMo Dataset (GitHub)](https://github.com/snap-research/locomo)
- [LoCoMo-MC10 (HuggingFace)](https://huggingface.co/datasets/Percena/locomo-mc10)
- [Mem0 Research Paper](https://arxiv.org/abs/2504.19413)
- [Original LoCoMo Paper (ACL 2024)](https://aclanthology.org/2024.acl-long.747/)
