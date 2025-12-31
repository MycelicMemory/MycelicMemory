# Comprehensive Metrics Tracking Guide

## Overview

The LoCoMo-MC10 benchmark now tracks comprehensive metrics beyond simple accuracy, following industry best practices for LLM evaluation. Metrics cover performance (latency), resource consumption (tokens), cost, and error analysis.

## Metrics Tracked

### 1. Accuracy Metrics

**Overall Accuracy**
- Total correct predictions / total questions
- Formula: `(correct / total) × 100`
- Example: 72% accuracy on 50 questions

**Per-Type Accuracy**
- Breakdown by question type: single-hop, multi-hop, temporal, open-domain, adversarial
- Identifies which question types are easier/harder
- Example: Single-hop 100%, Multi-hop 45.8%, Temporal 85.7%

**Error Analysis**
- Total errors and error rate
- Errors categorized by question type
- Identifies patterns in failures

### 2. Latency Metrics (seconds)

**Aggregate Latency Statistics**
- **Mean**: Average latency across all questions
- **Median**: 50th percentile (less affected by outliers)
- **P95/P99**: 95th and 99th percentiles (tail latency, SLA targets)
- **Min/Max**: Range of latencies
- **StdDev**: Standard deviation (latency stability)

**Example Output (50 questions)**
```
Mean: 1.839s
Median: 1.762s
P95: 2.852s
P99: 2.869s
Min: 1.270s
Max: 2.869s
StdDev: 0.397s
```

**Latency Breakdown**
- **Context Building Time**: Time to construct prompt from conversation history
- **LLM Response Time**: Time waiting for DeepSeek API response
- **Total Latency**: Context + LLM time

Example breakdown for 50 questions:
```
Context Building: 0.000s mean (0.02s total)
LLM Response: 1.839s mean (91.93s total)
Total: 1.839s mean (91.95s total)
```

### 3. Token Usage Metrics

**Token Counts**
- **Input Tokens**: Tokens in prompt (question + context + choices)
- **Output Tokens**: Tokens in LLM response (typically 1-2 for choice index)
- **Total Tokens**: Input + Output

**Per-Question Averages**
- Mean input tokens/question
- Mean output tokens/question
- Mean total tokens/question

**Example (50 questions)**
```
Total Input: 834,481 tokens
Total Output: 50 tokens
Total Tokens: 834,531 tokens
Mean per Question: 16,689.6 input, 1.0 output
```

### 4. Cost Estimation

Based on **DeepSeek API Pricing (v2024)**
- Input: $0.014 per 1M tokens
- Output: $0.056 per 1M tokens

**Costs Calculated**
- Input cost
- Output cost
- Total cost
- Cost per question (total / num_questions)

**Example (50 questions)**
```
Input Cost: $0.011683
Output Cost: $0.000003
Total Cost: $0.011686
Cost per Question: $0.000234
```

### 5. Per-Type Metrics

For each question type, we track:
- Count and accuracy (number correct / total)
- Latency statistics (mean, median, P95, min, max)
- Token usage (total and averages)

Example breakdown:
```
Single-Hop (Direct Recall)
  Total: 19, Correct: 19, Accuracy: 100.0%
  Mean Latency: 1.946s, P95: 2.869s
  Tokens: 16,686.1 per question

Multi-Hop (Multi-step Reasoning)
  Total: 24, Correct: 11, Accuracy: 45.8%
  Mean Latency: 1.806s, P95: 2.309s
  Tokens: 16,687.5 per question
```

## Running the Benchmark with Metrics

### 1. Run the Experiment

```bash
cd benchmark/locomo
python run_experiments.py --max_questions 50
```

**Output**: `results/ultrathink_results.json`

The script will automatically:
- Track latency, tokens, and cost for each question
- Print summary metrics at completion
- Save detailed results with all metrics

### 2. Generate Comprehensive Report

```bash
python comprehensive_evals.py
```

**Output**:
- `results/comprehensive_metrics.json` - Machine-readable metrics
- Console output - Formatted metrics report

### 3. Examine Raw Results

Check `results/ultrathink_results.json` for per-question data:

```json
{
  "question_id": {
    "question": "...",
    "correct_choice_index": 5,
    "predicted_choice_index": 5,
    "question_type": "multi_hop",
    "latency_total": 1.839,
    "latency_context_building": 0.001,
    "latency_llm_response": 1.838,
    "tokens": {
      "input_tokens": 16673,
      "output_tokens": 1,
      "total_tokens": 16674
    }
  }
}
```

## Interpretation Guidelines

### Latency
- **Typical range**: 1.3s - 2.9s per question
- **Acceptable P95**: < 3.0s
- **Acceptable Mean**: < 2.0s
- **StdDev indicates stability**: < 0.5s is good, > 0.5s suggests high variance

### Token Usage
- **Typical per question**: ~16,650-16,750 tokens
- **Mostly input**: Input >> Output (full context passed to LLM)
- **Expected growth**: Scales linearly with dataset size
- **Can be optimized**: Implement retrieval to reduce context size

### Cost
- **Current approach**: $0.00023/question (~$23 for 1,986 questions)
- **At scale**: Full dataset ~$46 for both input+output
- **With optimization**: Retrieval could reduce to $0.00010/question

### Accuracy Patterns
- **Single-hop**: Should be high (90-100%)
- **Multi-hop**: Should be lower (30-60%)
- **Temporal**: Should be moderate (60-80%)
- **Gap indicates**: Well-calibrated difficulty of questions

## Files

### New Files
- **metrics_tracker.py**: Core metrics collection and aggregation
  - `MetricsTracker` class
  - `TokenMetrics` dataclass
  - `QuestionResult` dataclass

- **comprehensive_evals.py**: Metrics report generation
  - Load and reconstruct metrics from results
  - Generate formatted reports
  - Save to JSON

### Modified Files
- **run_experiments.py**: Integrated metrics tracking
  - Updated `generate_answer()` to return token metrics
  - Track context building vs LLM latency separately
  - Initialize and populate MetricsTracker during run
  - Print summary metrics at completion

## Future Enhancements

Potential metrics to add:
1. **Retrieval Metrics** (when using memory system)
   - Recall@k for relevant context retrieval
   - Precision of retrieved information
   - Ranking metrics

2. **Confidence Metrics**
   - Model confidence scores if available
   - Correlation between confidence and correctness

3. **Efficiency Metrics**
   - Throughput (questions per second)
   - Goodput (correct answers per second)
   - Resource utilization

4. **Comparative Metrics**
   - Baseline comparison (different context sizes)
   - Model comparison (different LLMs)
   - Prompt engineering impact

## Best Practices

1. **Always track P95/P99 latency**, not just mean
   - Mean can be misleading with tail latencies
   - P95 is often used for SLA targets

2. **Monitor cost per question**, not just total cost
   - Helps identify expensive question types
   - Useful for budgeting

3. **Check per-type breakdown**
   - Identifies which optimizations would help most
   - Multi-hop is typically the bottleneck

4. **Analyze latency components**
   - Context building often negligible
   - LLM response is dominant
   - Suggests focus on retrieval optimization

5. **Track token growth**
   - Full context scales linearly with conversation length
   - Retrieval-based approach could provide sublinear scaling
