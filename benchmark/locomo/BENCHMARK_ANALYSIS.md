# LoCoMo-MC10 Benchmark: Deep Analysis & Implementation Issues

## Executive Summary

**CRITICAL ISSUE IDENTIFIED:** Our implementation completely misunderstands the LoCoMo-MC10 benchmark task.

### What We Implemented (WRONG ❌)
- Free-form answer generation using DeepSeek
- Full context (all haystack_sessions) passed directly to LLM
- Evaluation using LLM judge, F1 score, BLEU score
- No memory system integration
- No choice selection from provided options

### What LoCoMo-MC10 Actually Is (CORRECT ✅)
- **Multiple-choice QA task** with 10 answer options per question
- Evaluation metric: **Simple accuracy** (choice index prediction)
- Tests: Memory retrieval + multi-option reasoning
- Supports: Memory ingestion, retrieval, and fusion approaches

---

## 1. Official LoCoMo-MC10 Specification

### Dataset Overview
- **Total Questions:** 1,986
- **Format:** Multiple-choice with 10 options
- **Ground Truth:** `correct_choice_index` (0-9)
- **Source:** Percena/locomo-mc10 on HuggingFace

### Question Types Distribution
| Type | Count | % | Focus |
|------|-------|---|-------|
| **Open-Domain** | 841 | 42.3% | General conversation knowledge |
| **Adversarial** | 446 | 22.5% | Misleading/unanswerable questions |
| **Multi-Hop** | 321 | 16.2% | Multi-turn reasoning |
| **Single-Hop** | 282 | 14.2% | Single-turn fact retrieval |
| **Temporal** | 96 | 4.8% | Timing/sequence understanding |

### Key Dataset Fields
```python
{
    "question_id": str,              # Unique ID
    "question": str,                 # Natural-language question
    "choices": list[str] (len=10),   # 10 shuffled answer options
    "answer": str,                   # Correct answer text
    "correct_choice_index": int,     # Index of correct option (0-9)
    "question_type": str,            # single_hop, multi_hop, temporal_reasoning, open_domain, adversarial
    "num_sessions": int,
    "haystack_session_ids": list[str],
    "haystack_session_summaries": list[str],
    "haystack_session_datetimes": list[datetime],
    "haystack_sessions": list[list[dict]]  # Full conversation turns
}
```

---

## 2. Correct Evaluation Methodology

### Evaluation Task Flow
```
Question + 10 Choices + Context
         ↓
   Memory System:
   1. Ingest context (haystack_sessions)
   2. Retrieve relevant information for question
   3. Use retrieved + choices → LLM selects best choice
         ↓
   LLM Selection: index ∈ {0,1,2,...,9}
         ↓
   Compare: predicted_index vs correct_choice_index
         ↓
   Metric: Simple Accuracy = (correct / total) × 100
```

### Correct Evaluation Metrics
- **Primary Metric:** Simple Accuracy by choice index matching
- **Secondary Metrics:**
  - Accuracy per question type
  - Balanced accuracy (for class-imbalanced test sets)
  - Recall@k for retrieval evaluation

### Evaluation Approaches (from official docs)
| Approach | Method | Use Case |
|----------|--------|----------|
| **Direct MC** | Choose from 10 options with full context | Baseline |
| **Memory+RAG** | Ingest context, retrieve, then choose | Main task |
| **Context-stress** | Drop full haystack_sessions, measure performance decay | Robustness |
| **Retrieval-only** | Measure Recall@k on conversation turns | RAG quality |

---

## 3. Official Implementation vs Our Implementation

### Official Approach (SNAP Research + Percena)
```python
# For each question:
context = haystack_sessions  # Full conversation history
question = question_text      # Natural language question
choices = [choice_0, ..., choice_9]  # 10 options

# Memory system processes:
# 1. Ingest(context)
# 2. retrieved = Retrieve(question)
# 3. prediction = Select(question, choices, retrieved)

# Evaluation:
accuracy = (prediction == correct_choice_index) ? 1 : 0
```

### Our Implementation (WRONG ❌)
```python
# For each question:
context = haystack_sessions  # Full context (WRONG - no filtering)
question = question_text
# choices = IGNORED! (WRONG - not used)

# Our current process:
# 1. Full context → DeepSeek LLM
# 2. Generate free-form answer (WRONG - should select choice)
# 3. Evaluate with LLM judge, F1, BLEU (WRONG - should be accuracy)

# Result:
# - 50% LLM judge accuracy (meaningless for MC task)
# - F1/BLEU scores (not applicable to MC)
# - No memory system being tested
```

---

## 4. Critical Issues in Our Implementation

### Issue 1: Ignoring Multiple Choice Options
**Problem:** The `choices` field in dataset is completely ignored.
```python
# Current code in run_experiments.py
question_text = q.get("question", "")
generated_answer = self.generate_answer(question_text, context)
# choices = q.get("choices", [])  # NOT USED!
```
**Fix Required:** Must select from provided 10 options

### Issue 2: Free-Form Answer Generation
**Problem:** Generating free-text instead of selecting from choices
```python
# Wrong: Generate any text
response = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": prompt}],
    max_tokens=100  # Generates variable-length response
)

# Should: Select from 10 options
```
**Fix Required:** LLM must output choice index (0-9) or match answer to one of the choices

### Issue 3: Wrong Evaluation Metrics
**Problem:** Using metrics designed for open-ended QA
```python
# Current evaluation (WRONG for MC):
metrics = calculate_metrics(pred_answer, gt_answer)        # F1 score
bleu_scores = calculate_bleu_scores(pred_answer, gt_answer) # BLEU score
llm_score = evaluate_llm_judge(...)                        # LLM judge

# Correct evaluation (for MC):
accuracy = (predicted_choice_index == correct_choice_index) ? 1 : 0
```
**Fix Required:** Use simple choice-index comparison

### Issue 4: No Memory System Integration
**Problem:** Passing full context directly instead of using memory system
```python
# Current: Full context → LLM (bypasses memory system entirely)
context = build_context_from_sessions(q)
generated_answer = self.generate_answer(question_text, context)

# Should: Ingest → Retrieve → Select
# 1. memory_system.ingest(haystack_sessions)
# 2. retrieved = memory_system.retrieve(question)
# 3. selected_choice = llm.select_choice(question, choices, retrieved)
```
**Fix Required:** Integrate actual memory retrieval, don't pass full context

### Issue 5: Wrong Context Structure
**Problem:** Confusing what context to use
```python
# Current: Using speaker summaries + full dialogue inconsistently
if use_summaries:
    context = haystack_session_summaries  # Summaries
else:
    context = haystack_sessions           # Full dialogue
# Both passed directly to LLM - no retrieval!

# Should: Choose evaluation approach deliberately
# - Baseline: Full context (for comparison)
# - Memory: Retrieve relevant turns first
# - RAG: Use memory system for retrieval
```
**Fix Required:** Clear separation of evaluation approaches

---

## 5. Data Flow Analysis

### Current (Wrong) Flow
```
Dataset Load
    ↓
For each Q:
  - Get question text
  - Get full haystack_sessions (all context)
  - Ignore choices field ❌
  - Pass (question, full_context) → DeepSeek
  - Get free-form text answer ❌
  - Compare with gold_answer using F1/BLEU/LLM ❌
  - Return scores
```

### Correct Flow (What Should Happen)
```
Dataset Load
    ↓
For each Q:
  - Get question text
  - Get choices (10 options) ✅
  - Get correct_choice_index ✅
  - Get haystack_sessions (if using memory)
  ├─ Approach 1 (Baseline):
  │  - Full context → LLM
  │  - Output: Selected choice index
  │  - Evaluate: accuracy = (predicted == correct)
  ├─ Approach 2 (Memory):
  │  - Ingest haystack_sessions into memory system
  │  - Retrieve relevant info for question
  │  - retrieved_info + choices → LLM select
  │  - Evaluate: accuracy = (predicted == correct)
  └─ Approach 3 (RAG):
    - Use RAG pipeline to retrieve turns
    - Measure Recall@k then answer with context
```

---

## 6. Root Cause Analysis

### Why This Happened
1. **Misunderstanding of LoCoMo-MC10:** Treated it as open-ended QA instead of MC
2. **Adapting from mem0 incorrectly:** mem0 evaluates open-ended generation, not MC
3. **Ignoring dataset schema:** The `choices` and `correct_choice_index` fields were overlooked
4. **No validation against official implementation:** Never compared with official SNAP Research code
5. **Focusing on LLM judge:** Chose metrics that don't apply to MC tasks

### What Needs to Change
1. **Parse choices field** from dataset
2. **Modify LLM prompts** to select from 10 options
3. **Change evaluation** to simple accuracy on choice index
4. **Integrate memory system** for retrieval before choice selection
5. **Support multiple approaches:** baseline, memory-augmented, RAG

---

## 7. References

### Official Resources
- **Official LoCoMo Benchmark:** https://github.com/snap-research/locomo
- **LoCoMo-MC10 Dataset:** https://huggingface.co/datasets/Percena/locomo-mc10
- **Project Website:** https://snap-research.github.io/locomo/

### Key Papers & Implementations
- SNAP Research's evaluation methodology (official)
- Percena's MC10 variant specification
- HuggingFace dataset card for LoCoMo-MC10

---

## 8. Next Steps

1. **Rewrite run_experiments.py** to handle multiple-choice selection
2. **Modify LLM prompt** to output choice indices 0-9
3. **Update evals.py** to evaluate accuracy on choice index matching
4. **Integrate ultrathink memory system** for retrieval
5. **Test against official benchmark** to verify correctness
6. **Document evaluation approaches** (baseline, memory, RAG)

