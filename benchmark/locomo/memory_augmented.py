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

# DeepSeek API configuration
DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY", "REDACTED_API_KEY")
DEEPSEEK_BASE_URL = "https://api.deepseek.com"
DEEPSEEK_MODEL = "deepseek-chat"


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
        dataset_path: str = "data/locomo10.json",
        max_questions: Optional[int] = None,
        ultrathink_url: str = "http://localhost:3002/api/v1"
    ):
        """
        Initialize memory-augmented experiment.

        Args:
            dataset_path: Path to LoCoMo-MC10 JSONL dataset
            max_questions: Max questions to process (None = all)
            ultrathink_url: Ultrathink server URL
        """
        self.dataset_path = dataset_path
        self.max_questions = max_questions
        self.deepseek_api_key = DEEPSEEK_API_KEY
        self.ultrathink_url = ultrathink_url

        # Initialize clients
        self.memory_client = UltrathinkClient(base_url=ultrathink_url)
        self.llm_client = requests.Session()

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
        url = "https://huggingface.co/datasets/Percena/locomo-mc10/resolve/main/raw/locomo10.json"
        os.makedirs("data", exist_ok=True)

        try:
            print(f"   Downloading from {url}...")
            response = requests.get(url, timeout=30)
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

        # Call DeepSeek API
        start_time = time.time()

        headers = {
            "Authorization": f"Bearer {self.deepseek_api_key}",
            "Content-Type": "application/json"
        }

        payload = {
            "model": "deepseek-chat",
            "messages": [
                {"role": "system", "content": "You are a helpful assistant. Answer questions based on provided context."},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0,
            "max_tokens": 20,  # Increased from 10 to allow for formatting
            "top_p": 1
        }

        try:
            response = self.llm_client.post(
                "https://api.deepseek.com/chat/completions",
                headers=headers,
                json=payload,
                timeout=30
            )
            response.raise_for_status()
            data = response.json()

            # Extract choice index from response
            response_text = data["choices"][0]["message"]["content"].strip()
            match = re.search(r'\b([0-9])\b', response_text)
            predicted_idx = int(match.group(1)) if match else None

            # Calculate tokens
            elapsed = time.time() - start_time
            token_metrics = TokenMetrics(
                input_tokens=data["usage"]["prompt_tokens"],
                output_tokens=data["usage"]["completion_tokens"],
                total_tokens=data["usage"]["total_tokens"]
            )

            return predicted_idx, (token_metrics, elapsed)

        except Exception as e:
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
        print("\n" + "="*70)
        print("MEMORY-AUGMENTED LoCoMo-MC10 BENCHMARK")
        print("="*70)

        # Verify ultrathink is running
        if not self.memory_client.health_check():
            print("❌ Ultrathink server not running!")
            sys.exit(1)

        results = {}
        start_time = time.time()

        for idx, question in enumerate(self.questions):
            q_id = question.get("question_id", f"q_{idx}")
            print(f"\n[{idx+1}/{len(self.questions)}] {q_id}: {question['question'][:50]}...")

            # Step 1: Ingest conversation as memories
            session_id = f"locomo-{q_id}"
            messages = self._flatten_haystack_sessions(question.get("haystack_sessions", []))

            ingest_start = time.time()
            memory_ids, ingest_time = self.memory_client.ingest_conversation(
                messages=messages,
                session_id=session_id,
                domain="locomo-benchmark"
            )
            print(f"   • Ingested {len(memory_ids)} memories in {ingest_time:.3f}s")

            # Step 2: Retrieve relevant memories
            retrieve_start = time.time()
            retrieved_results, retrieval_time = self.memory_client.retrieve_memories(
                query=question["question"],
                top_k=10,
                use_ai=True,
                min_similarity=0.3
            )
            print(f"   • Retrieved {len(retrieved_results)} memories in {retrieval_time:.3f}s")

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

            print(f"   • Predicted: {predicted_idx}, Correct: {correct_idx} {'✓' if is_correct else '❌'}")
            print(f"   • Tokens: {token_metrics.total_tokens} (retrieved context: ~{retrieved_tokens})")

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

        # Print metrics
        self._print_summary()

        return summary

    def _print_summary(self) -> None:
        """Print metrics summary."""
        print("\n" + "="*70)
        print("MEMORY-AUGMENTED BENCHMARK SUMMARY")
        print("="*70)

        metrics = self.metrics_tracker.get_overall_metrics()

        print(f"\nAccuracy: {metrics['overall_accuracy']:.1f}%")
        print(f"  Correct: {metrics['num_correct']}/{metrics['total_questions']}")

        if metrics.get('per_type_accuracy'):
            print("\nPer-Type Accuracy:")
            for q_type, acc_data in metrics['per_type_accuracy'].items():
                pct = acc_data['correct'] / acc_data['total'] * 100 if acc_data['total'] > 0 else 0
                print(f"  {q_type}: {pct:.1f}% ({acc_data['correct']}/{acc_data['total']})")

        print("\nLatency Metrics (seconds):")
        print(f"  Mean: {metrics['latency_mean']:.3f}s")
        print(f"  P95: {metrics['latency_p95']:.3f}s")
        print(f"  Min/Max: {metrics['latency_min']:.3f}s / {metrics['latency_max']:.3f}s")

        print("\nToken Usage:")
        print(f"  Total Input: {metrics['total_input_tokens']}")
        print(f"  Total Output: {metrics['total_output_tokens']}")
        print(f"  Mean per Q: {metrics['mean_input_tokens_per_q']:.0f} in, {metrics['mean_output_tokens_per_q']:.1f} out")

        print("\nCost Estimation (DeepSeek pricing):")
        print(f"  Input Cost: ${metrics['input_cost']:.6f}")
        print(f"  Output Cost: ${metrics['output_cost']:.6f}")
        print(f"  Total Cost: ${metrics['total_cost']:.6f}")
        print(f"  Per Question: ${metrics['cost_per_question']:.6f}")

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
        default="data/locomo10.json",
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
        default="http://localhost:3002/api/v1",
        help="Ultrathink server URL"
    )

    args = parser.parse_args()

    # Create and run experiment
    experiment = MemoryAugmentedExperiment(
        dataset_path=args.dataset,
        max_questions=args.max_questions,
        ultrathink_url=args.ultrathink_url
    )

    experiment.run(output_path=args.output)


if __name__ == "__main__":
    main()
