# Master Plan: Memory-Augmented LoCoMo-MC10 Benchmark

## 🎯 Objective

Properly evaluate **ultrathink memory system** on LoCoMo-MC10 by:
1. Ingesting conversation histories as memories
2. Retrieving relevant memories for each question
3. Using retrieved (filtered) context instead of full context
4. Comparing memory-augmented accuracy vs baseline (full context)
5. Measuring efficiency gains (token reduction, latency, cost)

---

## 📐 Architecture Overview

### Current Baseline (Full Context)
```
Question + Full Haystack Sessions (16,690 tokens)
    ↓
DeepSeek LLM
    ↓
Choice Index (72% accuracy)
```

### Target: Memory-Augmented (Ultrathink)
```
Conversation History
    ↓
[ultrathink] Memory Ingest
    ↓
Question
    ↓
[ultrathink] Memory Retrieve (top-k relevant)
    ↓
Retrieved Context (e.g., 2,000 tokens instead of 16,690)
    ↓
DeepSeek LLM
    ↓
Choice Index (expected: 65-75% accuracy, 88% cost reduction)
```

---

## 🔧 Integration Approach: Using Ultrathink

### Option 1: ✅ **REST API** (Recommended)
- **Pros**: No subprocess management, simple HTTP calls
- **Cons**: Need ultrathink REST server running
- **Use Case**: Flexible, scalable, supports concurrent requests

### Option 2: ⚠️ **CLI** (Alternative)
- **Pros**: Works standalone, no server needed
- **Cons**: Subprocess overhead, slower for large batches
- **Use Case**: Simpler development, testing

### Option 3: **MCP** (Advanced)
- **Pros**: Full integration with Claude ecosystem
- **Cons**: Requires MCP server setup
- **Use Case**: Future AI automation

**Decision: Use REST API** for production, CLI for testing.

---

## 📋 Implementation Plan (Phase-by-Phase)

### Phase 1: Set Up Ultrathink Integration Layer

**Goal**: Create clean interface to ultrathink

**Files to Create**:
- `ultrathink_client.py` - REST API client wrapper
  - `ingest_memories()` - Store conversation as memories
  - `retrieve_memories()` - Get top-k relevant memories
  - `delete_collection()` - Clean up after each question

**Implementation**:
```python
class UltrathinkClient:
    """Interface to ultrathink REST API."""

    def __init__(self, base_url: str = "http://localhost:8000"):
        self.base_url = base_url

    def ingest_memories(self, conversation: List[str], session_id: str) -> bool:
        """
        Store conversation history as memories.
        Each turn becomes a separate memory.
        """
        # POST /memories with conversation data
        pass

    def retrieve_memories(self, query: str, top_k: int = 10) -> List[str]:
        """
        Retrieve top-k relevant memories.
        Returns memory content ranked by relevance.
        """
        # POST /retrieve with query
        pass

    def clear_session(self, session_id: str) -> bool:
        """Clean up memories for this question."""
        # DELETE /memories/{session_id}
        pass
```

**Testing**: Create `test_ultrathink_integration.py`

---

### Phase 2: Update Experiment Runner

**Goal**: Integrate memory retrieval into benchmark

**Files to Modify**:
- `run_experiments.py` - Add memory-augmented pipeline

**New Pipeline**:
```python
def run_memory_augmented(self, output_path: str):
    """Run benchmark with memory-augmented retrieval."""

    for question in self.questions:
        # 1. INGEST: Store conversation as memories
        session_id = question["question_id"]
        self.memory_client.ingest_memories(
            conversation=question["haystack_sessions"],
            session_id=session_id
        )

        # 2. RETRIEVE: Get relevant context for question
        retrieved = self.memory_client.retrieve_memories(
            query=question["question"],
            top_k=10
        )

        # 3. CONTEXT: Format with retrieved memories (not full context)
        context = "\n".join(retrieved)

        # 4. LLM: Generate answer with reduced context
        predicted_idx, tokens = self.generate_answer(
            question=question["question"],
            context=context,  # <-- Retrieved, not full
            choices=question["choices"]
        )

        # 5. EVALUATE: Compare with ground truth
        correct_idx = question["correct_choice_index"]
        is_correct = predicted_idx == correct_idx

        # 6. CLEANUP: Remove session memories
        self.memory_client.clear_session(session_id)
```

**Key Changes**:
- Add `UltrathinkClient` parameter
- Ingest/retrieve/cleanup for each question
- Track retrieved context size vs full context
- Store both baseline and memory-augmented results

---

### Phase 3: Add Retrieval Metrics

**Goal**: Measure retrieval quality and efficiency

**Files to Modify**:
- `metrics_tracker.py` - Add retrieval metrics

**New Metrics**:
```python
@dataclass
class RetrievalMetrics:
    """Metrics for memory retrieval."""
    full_context_tokens: int  # Baseline full context
    retrieved_context_tokens: int  # Actual retrieved context
    context_reduction_ratio: float  # % reduction
    retrieval_time: float  # Time to retrieve
    relevant_memories_returned: int  # Count

def measure_retrieval_quality(
    question: str,
    retrieved: List[str],
    full_context: str,
    correct_answer: str
) -> RetrievalMetrics:
    """Measure how good the retrieval was."""
    # Return metrics about retrieval
```

---

### Phase 4: Create Comprehensive Evaluation Suite

**Goal**: Compare baseline vs memory-augmented

**Files to Create**:
- `comparison_evals.py` - Baseline vs memory-augmented comparison
  - Side-by-side accuracy
  - Cost comparison
  - Token usage reduction
  - Latency comparison
  - Per-type breakdown

**Output**:
```
COMPARISON REPORT
===================================
                    Baseline    Memory-Aug    Improvement
Accuracy:           72.0%       68.5%         -3.5%
Cost per Q:         $0.000234   $0.000027     -88.5%
Tokens per Q:       16,690      1,930         -88.4%
Latency:            1.839s      0.921s        -50%
P95 Latency:        2.852s      1.456s        -49%

Cost to get 1000 Qs:
  Baseline: $234
  Memory-Aug: $27
  Savings: $207 (88.5%)
===================================
```

---

### Phase 5: Implement Different Retrieval Strategies

**Goal**: Optimize retrieval for accuracy

**Strategies to Test**:

1. **Strategy A: Simple Keyword Retrieval**
   - Match question keywords against memories
   - Expected: 65-70% accuracy

2. **Strategy B: Semantic Retrieval**
   - Use embeddings to find similar memories
   - Expected: 68-72% accuracy
   - Cost: Slightly higher (embedding compute)

3. **Strategy C: Temporal Retrieval**
   - Weight recent memories higher
   - Expected: 70-75% accuracy
   - Good for temporal questions

4. **Strategy D: Multi-hop Retrieval**
   - Retrieve chains of related memories
   - Expected: 75-80% accuracy
   - Cost: Higher (more calls)

**Test Matrix**:
```
Question Type   Baseline   Keyword   Semantic   Temporal   Multi-hop
Single-Hop      100%       98%       99%        100%       100%
Multi-Hop       45.8%      42%       48%        45%        65%
Temporal        85.7%      60%       65%        88%        85%
Open-Domain     100%       95%       98%        100%       100%
Overall         72.0%      63%       67%        70%        75%
```

---

## 🎯 Evaluation Methodology

### Setup Phase
1. **Start ultrathink server**: `ultrathink serve` or REST API
2. **Load dataset**: 100-200 questions for testing
3. **Split evaluation**:
   - 25% for sanity check (quick test)
   - 75% for full evaluation

### Benchmark Execution
1. **Run baseline**: Full context version (already done)
2. **Run memory-augmented**: With retrieval
3. **Capture metrics**: Accuracy, cost, tokens, latency, retrieval quality
4. **Generate comparison**: Show improvements

### Analysis Phase
1. **Per-type breakdown**: Single-hop vs multi-hop vs temporal
2. **Retrieval analysis**: What was retrieved? Was it relevant?
3. **Cost-benefit**: Is accuracy loss worth the cost savings?
4. **Optimization**: Which retrieval strategy works best?

---

## 📊 Expected Outcomes

### Accuracy Impact
```
Baseline (full context):    72.0%
Memory-Augmented (top-10): ~68-70%
Variance: -2% to -4%

Explanation:
- Single-hop: Minimal impact (still 100%)
- Multi-hop: Biggest impact (~5-10% drop)
  Reason: May miss connections across conversations
- Temporal: Small impact (good recency weighting)
- Open-domain: Minimal impact
```

### Efficiency Gains
```
Tokens per question:
  Baseline: 16,690 tokens (full context)
  Retrieved: 1,500-2,500 tokens (top-10 memories)
  Reduction: 85-90%

Cost savings:
  Baseline: $0.000234/question
  Memory: $0.000027/question
  Savings: 88.5%

For full dataset (1,986 questions):
  Baseline: $466
  Memory: $54
  Total savings: $412
```

### Latency Impact
```
Latency breakdown:
  Baseline: 1.839s
    - LLM: 1.839s
    - Context: 0.000s

  Memory-Augmented: 0.921s
    - Retrieval: 0.200s
    - LLM (smaller context): 0.721s
    - Total: 50% faster
```

---

## ✅ Success Criteria

### Minimum (Phase 1-2)
- [ ] Ultrathink client implemented and tested
- [ ] Memory ingestion working (can store conversations)
- [ ] Memory retrieval working (returns relevant memories)
- [ ] Memory-augmented pipeline integrated
- [ ] Baseline test passes (100 questions)

### Target (Phase 3-4)
- [ ] Retrieval metrics tracked
- [ ] Accuracy within -5% of baseline (65%+)
- [ ] Token reduction > 80%
- [ ] Cost reduction > 80%
- [ ] Latency improvement > 30%

### Stretch (Phase 5)
- [ ] Multiple retrieval strategies tested
- [ ] Per-type optimization identified
- [ ] Semantic retrieval implemented
- [ ] Accuracy recovery to 70%+ with good strategy
- [ ] Documentation complete

---

## 🔍 Testing Strategy

### Unit Tests
- `test_ultrathink_client.py`
  - Ingest functionality
  - Retrieve functionality
  - Session cleanup

### Integration Tests
- `test_memory_pipeline.py`
  - Full pipeline with 5 questions
  - Verify accuracy, tokens, cost

### Validation Tests
- `test_vs_baseline.py`
  - Compare memory-aug vs baseline
  - Verify expected cost savings
  - Verify accuracy is reasonable

### Benchmark Tests
- `benchmark_full.py`
  - Run on 100-200 questions
  - Generate comparison report
  - Analyze per-type

---

## 📅 Implementation Timeline

### Week 1: Foundation (Phase 1-2)
- Day 1-2: ultrathink client implementation + testing
- Day 3-4: Update experiment runner
- Day 5: Integration testing, debug

### Week 2: Metrics & Optimization (Phase 3-5)
- Day 1-2: Add retrieval metrics, comparison evals
- Day 3-4: Test different retrieval strategies
- Day 5: Analysis and documentation

### Week 3: Validation & Refinement
- Days 1-3: Full benchmark run (200 questions)
- Days 4-5: Optimize based on results

---

## 📁 File Structure

```
benchmark/locomo/
├── run_experiments.py          # Main benchmark runner
├── ultrathink_client.py        # NEW: ultrathink integration
├── memory_augmented.py         # NEW: Memory-augmented pipeline
├── comparison_evals.py         # NEW: Baseline vs memory-aug comparison
├── metrics_tracker.py          # Track all metrics
├── comprehensive_evals.py      # Evaluation reporting
├── verify_accuracy.py          # Accuracy validation
├── test_ultrathink_integration.py  # NEW: Unit tests
├── test_memory_pipeline.py     # NEW: Integration tests
└── docs/
    ├── MASTER_PLAN.md          # This file
    ├── ACCURACY_AUDIT.md       # Audit findings
    ├── METRICS_GUIDE.md        # Metrics documentation
    ├── BENCHMARK_ANALYSIS.md   # Original issues
    └── PIPELINE_DIAGRAMS.md    # Architecture diagrams
```

---

## 🚀 Getting Started

### Prerequisites
```bash
# 1. Start ultrathink REST API server
cd /path/to/ultrathink
ultrathink serve --port 8000 --log-level debug

# 2. Verify it's running
curl http://localhost:8000/health
```

### Quick Start
```bash
cd benchmark/locomo

# Phase 1: Test integration
python test_ultrathink_integration.py

# Phase 2: Run baseline + memory-augmented (10 questions)
python run_experiments.py --max_questions 10 --run_baseline
python memory_augmented.py --max_questions 10

# Phase 3: Generate comparison
python comparison_evals.py

# Phase 4: Full benchmark (100 questions)
python run_experiments.py --max_questions 100 --run_baseline
python memory_augmented.py --max_questions 100
python comparison_evals.py
```

---

## 📝 Next Steps

1. **Review & Approve Plan** - Make sure approach aligns with vision
2. **Start Phase 1** - Implement ultrathink client
3. **Test Integration** - Verify retrieval works
4. **Run Baseline** - Get baseline metrics
5. **Run Memory-Aug** - Test memory system
6. **Compare** - Analyze results and optimize

---

## Key Insights

**Why This Matters**:
- Baseline (72%) uses full context - not realistic
- Real systems need retrieval - can't pass everything
- This benchmark shows **actual memory system value**
- Measures: Accuracy vs Efficiency trade-off
- Identifies: Which retrieval strategies work best

**What We'll Learn**:
- Does ultrathink retrieval preserve accuracy?
- How much cost/token reduction do we get?
- Which question types are hardest without full context?
- What's the optimal retrieval strategy?
- Can we achieve 70%+ accuracy with 90% cost reduction?
