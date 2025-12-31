# Accuracy Audit: Potential Issues in Our Implementation

## Executive Summary

Our benchmark implementation may have accuracy issues in the following areas. This document identifies 8 categories of potential problems and how to verify/fix them.

## Critical Issues (High Probability)

### 1. ❌ **No Memory System Integration (CRITICAL)**

**Problem**: We're passing full conversation context directly to LLM, bypassing the ultrathink memory system entirely.

**Current Implementation**:
```python
context = build_context_from_sessions(q, self.use_summaries)
predicted_choice_index, token_metrics = self.generate_answer(question_text, context, choices)
```

**What Should Happen (Per LoCoMo Spec)**:
```
1. memory_system.ingest(haystack_sessions)      # Store conversation as memories
2. retrieved = memory_system.retrieve(question) # Get relevant facts only
3. selected = llm.select_choice(question, choices, retrieved)
```

**Impact**:
- We're testing "full context" baseline, not the actual memory-augmented approach
- Results don't reflect what memory system actually achieves
- Retrieval optimization won't be visible

**Verification**:
- [ ] Check if `ultrathink` memory system is actually being used anywhere
- [ ] Verify memory.ingest() is called with conversation data
- [ ] Confirm retrieval returns subset of conversation

---

### 2. ⚠️ **Choice Index Extraction Robustness**

**Problem**: Our regex `\b([0-9])\b` might not catch all response formats

**Current Implementation**:
```python
match = re.search(r'\b([0-9])\b', response)
if match:
    return int(match.group(1))
return None
```

**Potential Failure Cases**:
```
LLM Response → Extracted Index
"Answer: 5" → 5 ✓
"Index 5" → 5 ✓
"5" → 5 ✓
"Five" → None ✗ (not caught)
"Option 5: ..." → 5 ✓
"5." → None ✗ (period breaks \b)
"5)" → None ✗ (paren breaks \b)
"Choice(5)" → 5 ✓ but might get wrong number if multiple digits
"The answer is 5, which is..." → 5 ✓
```

**Fix Needed**:
```python
# More robust extraction
match = re.search(r'(?:choice|index|answer|option)[\s:]*([0-9])', response.lower())
# Also try just finding first isolated digit
if not match:
    match = re.search(r'([0-9])', response)
```

**Verification**:
- [ ] Log raw LLM responses to see actual formats
- [ ] Test extraction against sample responses
- [ ] Count failures (when match returns None)

---

### 3. ⚠️ **Prompt Clarity - LLM May Not Follow Instructions**

**Problem**: Our prompt might not be clear enough for LLM to return ONLY the index

**Current Prompt**:
```
Return ONLY the choice index (0-9) that corresponds to the best answer.
Do not explain your choice, just return the index number.

Answer Index:
```

**LLM Might Return**:
```
"The answer is 5"        (has extra text)
"5. This is because..."  (follows with explanation)
"The correct choice is 5" (wrapped in sentence)
"5" (good)
"Index 5" (adds prefix)
```

**Test Cases**:
- [ ] Sample 10 LLM responses - what % are JUST "0-9"?
- [ ] How many have extra text that regex misses?
- [ ] Does temperature=0 help? (we're using 0)

---

### 4. ⚠️ **Max Tokens Too Low**

**Problem**: `max_tokens=10` might truncate meaningful responses

**Current**:
```python
response = self.client.chat.completions.create(
    ...
    max_tokens=10,
)
```

**Issue**:
- 10 tokens is very restrictive
- LLM might be forced to truncate mid-response
- "The answer is choice 5" = 6 tokens, leaves only 4 for explanation

**Fix**:
- Increase to `max_tokens=20` or `max_tokens=50`
- Or add explicit instruction: "Answer with only a number 0-9, nothing else"

---

### 5. ⚠️ **Full Context Inflation (Token Counting)**

**Problem**: We might be double-counting tokens or measuring wrong thing

**Current**:
```python
token_metrics.input_tokens = response.usage.prompt_tokens
```

**What This Includes**:
- The prompt (question + choices + context)
- All conversation history from haystack_sessions
- Sometimes formatting/padding tokens

**Not Measured**:
- Tokens lost to context truncation (if context > model max)
- Actual tokens used by retrieval system

**Verification**:
- [ ] Compare our token count with what API actually charged
- [ ] Check if context is being truncated (silently?)
- [ ] Verify token count increases linearly with conversation length

---

### 6. ⚠️ **Context Building Time Not Captured**

**Problem**: Context building shows ~0.000s, which is implausible

**Current**:
```python
context_start = time.time()
context = build_context_from_sessions(q, self.use_summaries)
context_building_time = time.time() - context_start
```

**Output**:
```
Mean Context Building: 0.000s
Total Context Building: 0.02s (for 50 questions!)
```

**Issue**:
- Building massive context strings should take measurable time
- Might be rounding/precision issue
- Or the function is too fast (string building in Python is fast)

**Verification**:
- [ ] Manually time `build_context_from_sessions()` on sample
- [ ] Add detailed logging to see actual elapsed time
- [ ] Check if time.time() has sufficient precision

---

## Questionable Assumptions (Medium Probability)

### 7. ⚠️ **Assumption: Accuracy = Simple Index Matching**

**Problem**: What if LoCoMo-MC10 expects different evaluation?

**Verification Needed**:
- [ ] Read official LoCoMo-MC10 specification carefully
- [ ] Check Percena's reference implementation
- [ ] Verify with mem0's evaluation (even if they used wrong metrics)
- [ ] Check official papers/leaderboards for expected accuracy ranges

**Current Assumption**:
```python
is_correct = predicted_choice_index == correct_choice_index
```

**Could Be Wrong If**:
- Soft matching is expected (e.g., semantically similar answers)
- Partial credit system exists
- Case-sensitive matching but we're doing case-insensitive
- Off-by-one indexing issue (0-indexed vs 1-indexed)

---

### 8. ⚠️ **No Validation Against Ground Truth**

**Problem**: We compare predicted index against `correct_choice_index`, but never verify this matches actual answer text

**Current**:
```python
correct_choice_index = q.get("correct_choice_index")  # Trust this
predicted_choice_index = extract_choice_index(response)  # Extract from LLM
is_correct = predicted_choice_index == correct_choice_index
```

**Should Verify**:
```python
# Check: correct_choice_index actually points to answer text
assert q["choices"][correct_choice_index] == q["answer"], \
    f"Mismatch: choices[{correct_choice_index}] != answer"
```

**Verification**:
- [ ] Sample 20 questions, verify correct_choice_index points to answer
- [ ] Check if any misalignments exist
- [ ] Validate data integrity

---

## How mem0 Differs (If They Use Open-Ended)

If mem0 is using original LoCoMo (not MC10):

| Aspect | mem0 (Open-Ended) | Ours (MC10) |
|--------|-------------------|-----------|
| Output Format | Free-form text answer | Choice index (0-9) |
| Evaluation | BLEU, F1, LLM Judge | Simple accuracy |
| Prompt | "Generate answer" | "Select from 10 options" |
| Context | Full conversations | Full conversations + choices |
| Metrics | Semantic similarity | Exact match |

**Our implementation is CORRECT for MC10** but NOT comparable to mem0's open-ended evaluation.

---

## Recommended Verification Steps

### Immediate (High Priority)
1. [ ] Log 10 raw LLM responses, check extraction accuracy
2. [ ] Verify `correct_choice_index` points to actual answer
3. [ ] Manual spot-check of 5 full questions
4. [ ] Compare accuracy across runs (should be stable)

### Short Term (Before Trusting Results)
1. [ ] Implement memory system integration (even if baseline for now)
2. [ ] Improve prompt clarity or add output constraints
3. [ ] Test extraction robustness with edge cases
4. [ ] Increase max_tokens from 10 to 50

### Long Term (Ongoing)
1. [ ] Compare results against published LoCoMo-MC10 benchmarks
2. [ ] Test with different LLM models
3. [ ] Implement retrieval-based evaluation
4. [ ] Add confidence/uncertainty tracking
5. [ ] Compare single-hop accuracy against multi-hop (should be significantly higher)

---

## Current Results: Plausibility Assessment

**Test Results (50 questions)**:
- Overall: 72.0%
- Single-Hop: 100.0% (19/19)
- Multi-Hop: 45.8% (11/24)
- Temporal: 85.7% (6/7)

**What Makes Sense**:
- ✓ Single-hop > multi-hop (expected)
- ✓ All single-hop correct (makes sense for easiest type)
- ✓ Multi-hop significantly harder (makes sense)

**What Might Be Wrong**:
- ? 72% overall - is this realistic for baseline?
- ? No test on open-domain/adversarial questions (only 50 in sample)
- ? All 19 single-hop correct - suspiciously perfect
- ? No variability in single-hop accuracy (0% error rate)

---

## Action Items

1. **Create debug script** to log raw LLM responses and extraction
2. **Validate dataset** - verify correct_choice_index mappings
3. **Implement memory integration** - don't just pass full context
4. **Test robustness** - vary prompts, test extraction edge cases
5. **Compare baselines** - find published LoCoMo-MC10 results for comparison
