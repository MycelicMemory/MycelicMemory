# LoCoMo-MC10 Benchmark

This directory contains the LoCoMo-MC10 benchmark implementation for evaluating ultrathink's long-term conversational memory capabilities.

Based on the [mem0ai/mem0 evaluation framework](https://github.com/mem0ai/mem0/tree/main/evaluation).

## Overview

LoCoMo-MC10 is a 1,986-item multiple-choice benchmark derived from LoCoMo that tests LLM agents on 5 conversation memory abilities:

| Category | Description | Count |
|----------|-------------|-------|
| 1. Single-Hop | Direct fact retrieval | ~400 |
| 2. Multi-Hop | Combining multiple facts | ~400 |
| 3. Temporal | Time-based reasoning | ~400 |
| 4. Open-Domain | General knowledge | ~400 |
| 5. Adversarial | Robustness testing | ~400 |

**Dataset**: [HuggingFace - Percena/locomo-mc10](https://huggingface.co/datasets/Percena/locomo-mc10)

## Evaluation Metrics

Following mem0's evaluation approach:

1. **LLM Judge Accuracy** - DeepSeek evaluates if the generated answer is semantically correct compared to gold answer
2. **F1 Score** - Token-level overlap between prediction and reference
3. **BLEU-1 Score** - N-gram precision with smoothing

## Quick Start

```bash
# Install dependencies
make setup

# Run quick test (20 questions)
make quick

# Run full benchmark
make all
```

## Usage

### 1. Setup

```bash
pip install -r requirements.txt
```

### 2. Run Experiments

```bash
# Quick test with 20 questions
python run_experiments.py --max_questions 20 --download

# Full benchmark
python run_experiments.py --download

# With session summaries instead of full dialogues
python run_experiments.py --use_summaries --download
```

### 3. Evaluate Results

```bash
# Run LLM judge evaluation
python evals.py --input_file results/ultrathink_results.json

# Generate score summary
python generate_scores.py
```

## Configuration

Set your DeepSeek API key:

```bash
export DEEPSEEK_API_KEY=your_api_key_here
```

Or edit the default in `metrics/llm_judge.py`.

## File Structure

```
benchmark/locomo/
├── README.md                 # This file
├── Makefile                  # Convenience commands
├── requirements.txt          # Python dependencies
├── prompts.py               # Answer generation prompts
├── run_experiments.py       # Main experiment runner
├── evals.py                 # Evaluation with LLM judge
├── generate_scores.py       # Score aggregation
├── metrics/
│   ├── __init__.py
│   ├── llm_judge.py         # DeepSeek LLM judge
│   └── utils.py             # BLEU, F1 metrics
├── dataset/                 # Downloaded dataset
└── results/                 # Evaluation results
```

## LLM Judge Prompt

The LLM judge uses this prompt to evaluate answers:

```
Your task is to label an answer to a question as 'CORRECT' or 'WRONG'.
You will be given:
  (1) a question (posed by one user to another user),
  (2) a 'gold' (ground truth) answer,
  (3) a generated answer

The gold answer will usually be concise. The generated answer might be longer,
but you should be generous - as long as it touches on the same topic as the
gold answer, it should be CORRECT.

For time-related questions, if the generated answer refers to the same date
or time period as the gold answer, it should be CORRECT.
```

## Expected Results

### Baseline Comparisons (from mem0 research)

| Method | LLM Judge Accuracy |
|--------|-------------------|
| Human | ~95% |
| Mem0 | 66.9% |
| OpenAI Memory | 52.9% |
| Full Context | ~50% |
| RAG | ~40% |
| Random (10-choice) | 10% |

## References

- [LoCoMo-MC10 Dataset](https://huggingface.co/datasets/Percena/locomo-mc10)
- [mem0ai/mem0 Evaluation](https://github.com/mem0ai/mem0/tree/main/evaluation)
- [Mem0 Research Paper](https://arxiv.org/abs/2504.19413)
- [Original LoCoMo Paper (ACL 2024)](https://aclanthology.org/2024.acl-long.747/)
