# LoCoMo-MC10 Benchmark

Memory-augmented benchmark for evaluating ultrathink's long-term conversational memory capabilities using the LoCoMo-MC10 dataset.

## Overview

LoCoMo-MC10 is a 1,986-item multiple-choice benchmark derived from LoCoMo that tests LLM agents on 5 conversation memory abilities:

| Category | Description |
|----------|-------------|
| Single-Hop | Direct fact retrieval |
| Multi-Hop | Combining multiple facts |
| Temporal | Time-based reasoning |
| Open-Domain | General knowledge |
| Adversarial | Robustness testing |

**Dataset**: [HuggingFace - Percena/locomo-mc10](https://huggingface.co/datasets/Percena/locomo-mc10)

## Quick Start

```bash
# From repository root
make benchmark-setup    # Install dependencies
make benchmark-quick    # Run with 10 questions
make benchmark          # Run full benchmark
make benchmark-evaluate # Evaluate results
```

Or from this directory:

```bash
make setup      # Install dependencies
make download   # Download dataset (if needed)
make run-quick  # Run with 10 questions
make run        # Run full benchmark
make evaluate   # Evaluate results
```

## Prerequisites

1. **Ultrathink server running** on port 3099:
   ```bash
   ultrathink serve --port 3099
   ```

2. **DeepSeek API key** (optional - has default):
   ```bash
   export DEEPSEEK_API_KEY=your_key
   ```

## How It Works

For each question:
1. Ingest conversation history as memories into ultrathink
2. Retrieve relevant memories via semantic search
3. Generate answer using retrieved context (not full context)
4. Compare prediction to correct answer
5. Clean up memories for next question

## File Structure

```
benchmark/locomo/
├── Makefile                  # Build commands
├── README.md                 # This file
├── requirements.txt          # Python dependencies
├── memory_augmented.py       # Main benchmark runner
├── comprehensive_evals.py    # Evaluation metrics generator
├── ultrathink_client.py      # Ultrathink API client
├── metrics_tracker.py        # Metrics collection
├── logging_system.py         # Benchmark logging
├── llm_call_tracker.py       # LLM API tracking
├── download_dataset.py       # Dataset downloader
├── prompts.py                # Answer generation prompts
├── metrics/                  # Evaluation metrics
│   ├── __init__.py
│   ├── llm_judge.py
│   └── utils.py
├── data/                     # Downloaded dataset
└── results/                  # Benchmark results
```

## Metrics

The benchmark measures:
- **Accuracy**: Correct answers / Total questions
- **Latency**: LLM response time + context building time
- **Token Usage**: Input/output tokens per question
- **Cost Estimation**: Based on DeepSeek pricing

## Expected Results

| Method | Accuracy |
|--------|----------|
| Human | ~95% |
| Mem0 | 66.9% |
| OpenAI Memory | 52.9% |
| Full Context | ~50% |
| RAG | ~40% |
| Random | 10% |

## References

- [LoCoMo-MC10 Dataset](https://huggingface.co/datasets/Percena/locomo-mc10)
- [Mem0 Research Paper](https://arxiv.org/abs/2504.19413)
- [Original LoCoMo Paper (ACL 2024)](https://aclanthology.org/2024.acl-long.747/)
