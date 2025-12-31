"""
Memory-Augmented LoCoMo-MC10 Benchmark

Runs LoCoMo-MC10 benchmark with ultrathink memory retrieval instead of full context.

Flow:
1. Load dataset
2. For each question:
   - Ingest conversation history as memories
   - Retrieve relevant memories via semantic search
   - Generate answer using retrieved context (not full context)
   - Track metrics: accuracy, tokens saved, latency
   - Clean up memories
3. Generate results and comparison report
"""

import json
import time
import os
import sys
import re
from typing import List, Dict, Tuple, Optional
from dataclasses import dataclass
from datetime import datetime
import requests

from ultrathink_client import UltrathinkClient, RetrievalResult
from metrics_tracker import MetricsTracker, TokenMetrics, QuestionResult
from logging_system import BenchmarkLogger, CallType, init_logger, get_logger
from llm_call_tracker import LLMCallTracker
from config import DEEPSEEK_API_KEY, DEEPSEEK_BASE_URL, DEEPSEEK_MODEL
from progress_display import ProgressDisplay


@dataclass
class RetrievalMetrics:
    """Metrics for memory retrieval efficiency."""
    tokens_baseline: int          # Full context baseline
    tokens_retrieved: int         # Retrieved context
    token_reduction_pct: float    # (baseline - retrieved) / baseline * 100
    num_memories_retrieved: int   # Count of memories used
    retrieval_latency: float      # Time to retrieve


class MemoryAugmentedExperiment:
    """Run LoCoMo-MC10 benchmark with memory-augmented retrieval."""

    def __init__(
        self,
        dataset_path: str = "dataset/locomo10.json",
        max_questions: Optional[int] = None,
        ultrathink_url: str = "http://localhost:3099/api/v1",
        enable_logging: bool = True,
        log_dir: str = "logs"
    ):
        """
        Initialize memory-augmented experiment.

        Args:
            dataset_path: Path to LoCoMo-MC10 JSONL dataset
            max_questions: Max questions to process (None = all)
            ultrathink_url: Ultrathink server URL
            enable_logging: Enable comprehensive logging
            log_dir: Directory for log files
        """
        self.dataset_path = dataset_path
        self.max_questions = max_questions
        self.deepseek_api_key = DEEPSEEK_API_KEY
        self.ultrathink_url = ultrathink_url

        # Initialize logging
        self.enable_logging = enable_logging
        if enable_logging:
            self.logger = init_logger(
                name="memory-augmented-benchmark",
                log_dir=log_dir
            )
        else:
            self.logger = None

        # Initialize clients
        self.memory_client = UltrathinkClient(base_url=ultrathink_url)
        self.llm_client = requests.Session()
        self.llm_tracker = LLMCallTracker(logger=self.logger)

        # Initialize metrics
        self.metrics_tracker = MetricsTracker()

        # Load dataset
        self.questions = self._load_dataset()

    def _load_dataset(self) -> List[Dict]:
        """Load LoCoMo-MC10 dataset from JSON or JSONL file."""
        questions = []

        if not os.path.exists(self.dataset_path):
            print(f"⚠️  Dataset not found at {self.dataset_path}")
            print("   Attempting to download...")
            self._download_dataset()

        with open(self.dataset_path, "r") as f:
            content = f.read().strip()

        # Try to load as JSON array first
        if content.startswith("["):
            try:
                data = json.loads(content)
                if isinstance(data, list):
                    questions = data
                    if self.max_questions:
                        questions = questions[:self.max_questions]
            except json.JSONDecodeError:
                pass

        # If not loaded as array, try JSONL format
        if not questions:
            with open(self.dataset_path, "r") as f:
                for i, line in enumerate(f):
                    if self.max_questions and i >= self.max_questions:
                        break
                    try:
                        q = json.loads(line.strip())
                        questions.append(q)
                    except json.JSONDecodeError:
                        continue

        print(f"✓ Loaded {len(questions)} questions")
        return questions

    def _download_dataset(self) -> None:
        """Download LoCoMo-MC10 dataset from HuggingFace."""
        # This is the correct MC10 format (flattened with question/choices/haystack_sessions)
        url = "https://huggingface.co/datasets/Percena/locomo-mc10/resolve/main/data/locomo_mc10.json"
        os.makedirs("data", exist_ok=True)

        try:
            print(f"   Downloading from {url}...")
            response = requests.get(url, timeout=120)
            response.raise_for_status()

            with open(self.dataset_path, "w") as f:
                f.write(response.text)
            print(f"   ✓ Downloaded to {self.dataset_path}")
        except Exception as e:
            print(f"   ❌ Download failed: {e}")
            raise

    def _flatten_haystack_sessions(self, haystack_sessions: List[List[Dict]]) -> List[Dict]:
        """Convert nested sessions into flat message list."""
        messages = []
        for session in haystack_sessions:
            for turn in session:
                role = turn.get("role", "user")
                content = turn.get("content", "")
                if content.strip():
                    messages.append({
                        "role": role,
                        "content": content
                    })
        return messages

    def _generate_answer(
        self,
        question: str,
        context: str,
        choices: List[str]
    ) -> Tuple[Optional[int], TokenMetrics]:
        """
        Generate answer using DeepSeek LLM with retrieved context.

        Args:
            question: The question text
            context: Retrieved context (from memories)
            choices: List of answer choices (0-9)

        Returns:
            Tuple of (predicted_choice_index, token_metrics)
        """
        # Build prompt
        choices_text = "\n".join([f"{i}. {choice}" for i, choice in enumerate(choices)])
        prompt = f"""You have been given a conversation with specific information. Using that information, answer the following multiple-choice question by returning ONLY the choice index (0-9).

CONTEXT:
{context}

QUESTION: {question}

CHOICES:
{choices_text}

Return ONLY the choice index (0-9), nothing else."""

        # Build messages for LLM
        messages = [
            {"role": "system", "content": "You are a helpful assistant. Answer questions based on provided context."},
            {"role": "user", "content": prompt}
        ]

        try:
            # Use LLMCallTracker for full transparency
            response_text, metrics = self.llm_tracker.call_llm_api(
                api_key=self.deepseek_api_key,
                model="deepseek-chat",
                messages=messages,
                temperature=0,
                max_tokens=20,
                top_p=1
            )

            # Extract choice index from response
            match = re.search(r'\b([0-9])\b', response_text)
            predicted_idx = int(match.group(1)) if match else None

            # Create token metrics
            token_metrics = TokenMetrics(
                input_tokens=metrics.input_tokens,
                output_tokens=metrics.output_tokens,
                total_tokens=metrics.total_tokens
            )

            return predicted_idx, (token_metrics, metrics.total_time_ms / 1000)

        except Exception as e:
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.BENCHMARK_END,
                    f"LLM error occurred",
                    {"error": str(e), "question": question[:100]}
                )
            print(f"❌ LLM error: {e}")
            return None, (TokenMetrics(0, 0, 0), 0)

    def run(self, output_path: str = "results/memory_augmented_results.json") -> Dict:
        """
        Run memory-augmented benchmark.

        Args:
            output_path: Where to save results

        Returns:
            Dictionary of metrics
        """
        # Initialize progress display for live logging
        progress = ProgressDisplay(total_questions=len(self.questions))
        progress.display_header(
            dataset_path=self.dataset_path,
            ultrathink_url=self.ultrathink_url
        )

        # Log benchmark start
        if self.logger:
            self.logger.log_benchmark_event(
                CallType.BENCHMARK_START,
                "Memory-augmented LoCoMo-MC10 benchmark started",
                {
                    "total_questions": len(self.questions),
                    "max_questions": self.max_questions,
                    "dataset": self.dataset_path,
                    "ultrathink_url": self.ultrathink_url
                }
            )

        # Verify ultrathink is running
        if not self.memory_client.health_check():
            print("❌ Ultrathink server not running!")
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.BENCHMARK_END,
                    "Benchmark failed: Ultrathink server not available",
                    {"error": "Health check failed"}
                )
            sys.exit(1)

        results = {}
        start_time = time.time()

        for idx, question in enumerate(self.questions):
            q_id = question.get("question_id", f"q_{idx}")
            q_type = question.get("question_type", "unknown")

            # Display question start with type prominently
            progress.display_question_start(
                idx=idx,
                question_id=q_id,
                question_type=q_type,
                question_text=question['question']
            )

            # Step 1: Log question start
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.QUESTION_START,
                    f"Processing question {idx+1}/{len(self.questions)}",
                    {
                        "question_id": q_id,
                        "question_type": question.get("question_type", "unknown"),
                        "question_preview": question["question"][:100]
                    }
                )

            # Step 2: Ingest conversation as memories
            session_id = f"locomo-{q_id}"
            messages = self._flatten_haystack_sessions(question.get("haystack_sessions", []))

            ingest_start = time.time()
            memory_ids, ingest_time = self.memory_client.ingest_conversation(
                messages=messages,
                session_id=session_id,
                domain="locomo-benchmark"
            )

            # Log memory ingest
            if self.logger:
                self.logger.log_memory_call(
                    call_type=CallType.MEMORY_INGEST,
                    operation="ingest_conversation",
                    duration_ms=ingest_time * 1000,
                    num_items=len(memory_ids),
                    status="success",
                    metadata={
                        "question_id": q_id,
                        "session_id": session_id,
                        "num_messages": len(messages)
                    }
                )

            # Step 3: Retrieve relevant memories
            retrieve_start = time.time()
            retrieved_results, retrieval_time = self.memory_client.retrieve_memories(
                query=question["question"],
                top_k=10,
                use_ai=True,  # Use semantic search with Ollama embeddings
                min_similarity=0.0
            )

            # Display memory operations
            progress.display_memory_ops(
                num_ingested=len(memory_ids),
                ingest_time=ingest_time,
                num_retrieved=len(retrieved_results),
                retrieval_time=retrieval_time
            )

            # Log memory retrieval
            if self.logger:
                self.logger.log_memory_call(
                    call_type=CallType.MEMORY_RETRIEVE,
                    operation="retrieve_memories",
                    duration_ms=retrieval_time * 1000,
                    num_items=len(retrieved_results),
                    status="success",
                    metadata={
                        "question_id": q_id,
                        "query_preview": question["question"][:100],
                        "top_k": 10,
                        "use_ai": True
                    }
                )

            # Step 3: Format retrieved context
            retrieved_context = self.memory_client.format_retrieved_as_context(retrieved_results)
            retrieved_tokens = len(retrieved_context.split())  # Approximate token count

            # Step 4: Generate answer with retrieved context
            predicted_idx, (token_metrics, llm_latency) = self._generate_answer(
                question=question["question"],
                context=retrieved_context,
                choices=question["choices"]
            )

            # Step 5: Evaluate
            correct_idx = question.get("correct_choice_index")
            is_correct = predicted_idx == correct_idx

            # Display result with running totals
            progress.display_result(
                predicted_idx=predicted_idx,
                correct_idx=correct_idx,
                is_correct=is_correct,
                llm_latency=llm_latency,
                input_tokens=token_metrics.input_tokens,
                output_tokens=token_metrics.output_tokens,
                question_type=q_type
            )

            # Step 6: Track metrics
            baseline_tokens = 16690  # Known baseline from run_experiments.py

            # Record in metrics tracker
            self.metrics_tracker.add_result(
                question_id=q_id,
                question_text=question["question"],
                question_type=question.get("question_type", "unknown"),
                correct_choice_index=correct_idx,
                predicted_choice_index=predicted_idx,
                latency=llm_latency,
                tokens=token_metrics,
                llm_response_time=llm_latency,
                context_building_time=retrieval_time
            )

            # Also store raw result with retrieval metadata
            result = {
                "question_id": q_id,
                "question": question["question"],
                "predicted_choice_index": predicted_idx,
                "correct_choice_index": correct_idx,
                "is_correct": is_correct,
                "question_type": question.get("question_type", "unknown"),
                "latency_total": retrieval_time + llm_latency,
                "latency_context_building": retrieval_time,
                "latency_llm_response": llm_latency,
                "tokens": {
                    "input_tokens": token_metrics.input_tokens,
                    "output_tokens": token_metrics.output_tokens,
                    "total_tokens": token_metrics.total_tokens
                },
                "retrieval_metadata": {
                    "tokens_baseline": baseline_tokens,
                    "tokens_retrieved": retrieved_tokens,
                    "token_reduction_pct": (baseline_tokens - retrieved_tokens) / baseline_tokens * 100,
                    "num_memories_retrieved": len(retrieved_results),
                    "retrieval_latency": retrieval_time,
                    "session_id": session_id
                }
            }

            results[q_id] = result

            # Step 7: Cleanup memories
            cleanup_start = time.time()
            deleted, cleanup_time = self.memory_client.clear_session(session_id)
            cleanup_elapsed = time.time() - cleanup_start
            # Note: deleted may be 0 if tag-based search doesn't work

            # Log memory cleanup
            if self.logger:
                self.logger.log_memory_call(
                    call_type=CallType.MEMORY_DELETE,
                    operation="clear_session",
                    duration_ms=cleanup_elapsed * 1000,
                    num_items=deleted,
                    status="success",
                    metadata={
                        "question_id": q_id,
                        "session_id": session_id
                    }
                )

            # Log question end
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.QUESTION_END,
                    f"Question {idx+1} completed",
                    {
                        "question_id": q_id,
                        "is_correct": is_correct,
                        "accuracy": 1.0 if is_correct else 0.0,
                        "tokens_used": token_metrics.total_tokens,
                        "token_reduction_pct": (baseline_tokens - retrieved_tokens) / baseline_tokens * 100,
                        "total_latency": retrieval_time + llm_latency
                    }
                )

        # Generate summary
        total_time = time.time() - start_time
        summary = {
            "benchmark": "locomo-mc10-memory-augmented",
            "timestamp": datetime.now().isoformat(),
            "total_questions": len(self.questions),
            "total_time_seconds": total_time,
            "results": results
        }

        # Save results
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(summary, f, indent=2)

        print(f"\n✓ Saved results to {output_path}")

        # Log benchmark end and generate report
        if self.logger:
            metrics = self.metrics_tracker.get_overall_metrics()
            self.logger.log_benchmark_event(
                CallType.BENCHMARK_END,
                "Memory-augmented LoCoMo-MC10 benchmark completed",
                {
                    "total_questions": len(self.questions),
                    "total_time_seconds": total_time,
                    "overall_accuracy": metrics.get('overall_accuracy', 0),
                    "total_cost": metrics.get('total_cost', 0),
                    "results_file": output_path
                }
            )
            # Save comprehensive report
            report_path = self.logger.save_report()
            print(f"\n✓ Detailed logs saved to {report_path}")

        # Display comprehensive final summary with all metrics
        overall_metrics = self.metrics_tracker.get_overall_metrics()
        per_type_metrics = self.metrics_tracker.get_per_type_metrics()
        progress.display_final_summary(
            metrics=overall_metrics,
            per_type_metrics=per_type_metrics,
            duration_secs=total_time
        )

        return summary

    def _print_summary(self) -> None:
        """Print metrics summary."""
        print("\n" + "="*70)
        print("MEMORY-AUGMENTED BENCHMARK SUMMARY")
        print("="*70)

        metrics = self.metrics_tracker.get_overall_metrics()

        # Overall Accuracy
        overall = metrics.get('overall', {})
        print(f"\nAccuracy: {overall.get('accuracy', 0):.1f}%")
        print(f"  Correct: {overall.get('correct_predictions', 0)}/{overall.get('total_questions', 0)}")

        # Latency Metrics
        latency = metrics.get('latency', {})
        if latency:
            print("\nLatency Metrics (seconds):")
            print(f"  Mean: {latency.get('mean_latency_seconds', 0):.3f}s")
            print(f"  Median: {latency.get('median_latency_seconds', 0):.3f}s")
            print(f"  P95: {latency.get('p95_latency_seconds', 0):.3f}s")
            print(f"  Min/Max: {latency.get('min_latency_seconds', 0):.3f}s / {latency.get('max_latency_seconds', 0):.3f}s")

        # LLM Latency
        llm_latency = metrics.get('llm_latency', {})
        if llm_latency:
            print("\nLLM Response Latency:")
            print(f"  Mean: {llm_latency.get('mean_llm_response_seconds', 0):.3f}s")
            print(f"  Total: {llm_latency.get('total_llm_time_seconds', 0):.3f}s")

        # Context Building Latency
        ctx_latency = metrics.get('context_latency', {})
        if ctx_latency:
            print("\nContext Building Latency:")
            print(f"  Mean: {ctx_latency.get('mean_context_building_seconds', 0):.3f}s")
            print(f"  Total: {ctx_latency.get('total_context_building_seconds', 0):.3f}s")

        # Token Usage
        tokens = metrics.get('tokens', {})
        if tokens:
            print("\nToken Usage:")
            print(f"  Total Input: {tokens.get('total_input_tokens', 0):,}")
            print(f"  Total Output: {tokens.get('total_output_tokens', 0):,}")
            print(f"  Total: {tokens.get('total_tokens', 0):,}")
            print(f"  Mean per Q: {tokens.get('mean_input_tokens', 0):.0f} in, {tokens.get('mean_output_tokens', 0):.1f} out")

        # Cost Estimation
        cost = metrics.get('cost_estimation', {})
        if cost:
            print("\nCost Estimation (DeepSeek pricing):")
            print(f"  Input Cost: ${cost.get('input_cost_usd', 0):.6f}")
            print(f"  Output Cost: ${cost.get('output_cost_usd', 0):.6f}")
            print(f"  Total Cost: ${cost.get('total_cost_usd', 0):.6f}")
            print(f"  Per Question: ${cost.get('cost_per_question_usd', 0):.6f}")

        print("\n" + "="*70)


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description="Run LoCoMo-MC10 benchmark with memory-augmented retrieval"
    )
    parser.add_argument(
        "--dataset",
        type=str,
        default="dataset/locomo10.json",
        help="Path to LoCoMo-MC10 JSONL dataset"
    )
    parser.add_argument(
        "--max-questions",
        type=int,
        default=None,
        help="Max questions to process"
    )
    parser.add_argument(
        "--output",
        type=str,
        default="results/memory_augmented_results.json",
        help="Output file for results"
    )
    parser.add_argument(
        "--ultrathink-url",
        type=str,
        default="http://localhost:3099/api/v1",
        help="Ultrathink server URL"
    )
    parser.add_argument(
        "--enable-logging",
        type=bool,
        default=True,
        help="Enable comprehensive logging (default: True)"
    )
    parser.add_argument(
        "--log-dir",
        type=str,
        default="logs",
        help="Directory for log files (default: logs)"
    )

    args = parser.parse_args()

    # Create and run experiment
    experiment = MemoryAugmentedExperiment(
        dataset_path=args.dataset,
        max_questions=args.max_questions,
        ultrathink_url=args.ultrathink_url,
        enable_logging=args.enable_logging,
        log_dir=args.log_dir
    )

    experiment.run(output_path=args.output)


if __name__ == "__main__":
    main()
