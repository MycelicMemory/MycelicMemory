"""
Memory-Augmented LoCoMo-MC10 Benchmark using Ultrathink MCP

Uses ultrathink memory system via MCP interface for:
- Storing conversation history as memories
- Semantic retrieval for relevant context
- Avoiding full-context overhead

Flow:
1. Load dataset
2. For each question:
   - Store conversation as memories (MCP)
   - Retrieve relevant memories via semantic search (MCP)
   - Generate answer using retrieved context
   - Track metrics: accuracy, tokens saved, latency
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

from metrics_tracker import MetricsTracker, TokenMetrics


# DeepSeek API configuration
DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY", "REDACTED_API_KEY")
DEEPSEEK_BASE_URL = "https://api.deepseek.com"
DEEPSEEK_MODEL = "deepseek-chat"


class MemoryAugmentedMCPExperiment:
    """Run LoCoMo-MC10 benchmark with memory-augmented retrieval via MCP."""

    def __init__(
        self,
        dataset_path: str = "data/locomo10.json",
        max_questions: Optional[int] = None
    ):
        """
        Initialize memory-augmented MCP experiment.

        Args:
            dataset_path: Path to LoCoMo-MC10 JSONL dataset
            max_questions: Max questions to process (None = all)
        """
        self.dataset_path = dataset_path
        self.max_questions = max_questions
        self.deepseek_api_key = DEEPSEEK_API_KEY

        # Initialize metrics
        self.metrics_tracker = MetricsTracker()

        # Load dataset
        self.questions = self._load_dataset()

    def _load_dataset(self) -> List[Dict]:
        """Load LoCoMo-MC10 dataset from JSONL file."""
        questions = []

        if not os.path.exists(self.dataset_path):
            print(f"⚠️  Dataset not found at {self.dataset_path}")
            print("   Attempting to download...")
            self._download_dataset()

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

    def _store_memories_mcp(self, messages: List[Dict], session_id: str, domain: str = "locomo-benchmark") -> int:
        """
        Store conversation messages as memories using MCP.

        Uses Claude Code's ultrathink MCP integration.
        """
        num_stored = 0

        for i, msg in enumerate(messages):
            try:
                content = msg.get("content", "").strip()
                if not content:
                    continue

                role = msg.get("role", "user")

                # Store memory using MCP
                from mcp__ultrathink__store_memory import store_memory

                memory_id = store_memory(
                    content=content,
                    importance=5,  # Default importance
                    tags=[role, "conversation-turn", f"position-{i}", f"session-{session_id}"],
                    domain=domain,
                    source=f"locomo-{session_id}-turn-{i}"
                )

                if memory_id:
                    num_stored += 1

            except Exception as e:
                print(f"⚠️  Failed to store message {i}: {e}")
                continue

        return num_stored

    def _retrieve_memories_mcp(self, query: str, top_k: int = 10) -> Tuple[List[Dict], float]:
        """
        Retrieve relevant memories using MCP semantic search.

        Returns: (results, retrieval_time)
        """
        start_time = time.time()

        try:
            from mcp__ultrathink__search import search

            results = search(
                query=query,
                limit=top_k,
                search_type="semantic",
                use_ai=True,
                response_format="concise"
            )

            elapsed = time.time() - start_time

            # Format results into consistent structure
            formatted_results = []
            if results:
                for item in results:
                    formatted_results.append({
                        "id": item.get("id", ""),
                        "content": item.get("content", item.get("summary", "")),
                        "relevance_score": item.get("relevance_score", item.get("similarity_score", 0.0)),
                        "tags": item.get("tags", [])
                    })

            return formatted_results, elapsed

        except Exception as e:
            print(f"❌ Retrieval error: {e}")
            return [], time.time() - start_time

    def _format_retrieved_context(self, results: List[Dict]) -> str:
        """Format retrieved memories into context string."""
        if not results:
            return ""

        parts = []
        for i, result in enumerate(results, 1):
            score = result.get("relevance_score", 0.0)
            content = result.get("content", "")
            if content:
                parts.append(f"[Memory {i}] {content}")
                if score:
                    parts.append(f"(Relevance: {score:.2f})")

        return "\n".join(parts)

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
            "max_tokens": 20,
            "top_p": 1
        }

        try:
            response = requests.post(
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
                total_tokens=data["usage"]["total_tokens"],
                latency_total=elapsed,
                latency_llm_response=elapsed,
                latency_context_building=0
            )

            return predicted_idx, token_metrics

        except Exception as e:
            print(f"❌ LLM error: {e}")
            return None, TokenMetrics(0, 0, 0, 0, 0, 0)

    def run(self, output_path: str = "results/memory_augmented_mcp_results.json") -> Dict:
        """
        Run memory-augmented benchmark using MCP.

        Args:
            output_path: Where to save results

        Returns:
            Dictionary of metrics
        """
        print("\n" + "="*70)
        print("MEMORY-AUGMENTED LoCoMo-MC10 BENCHMARK (MCP)")
        print("="*70)

        results = {}
        start_time = time.time()

        for idx, question in enumerate(self.questions):
            q_id = question.get("question_id", f"q_{idx}")
            print(f"\n[{idx+1}/{len(self.questions)}] {q_id}: {question['question'][:50]}...")

            # Step 1: Store conversation as memories via MCP
            session_id = f"locomo-{q_id}"
            messages = self._flatten_haystack_sessions(question.get("haystack_sessions", []))

            store_start = time.time()
            num_stored = self._store_memories_mcp(messages, session_id)
            store_time = time.time() - store_start
            print(f"   • Stored {num_stored} memories in {store_time:.3f}s")

            # Step 2: Retrieve relevant memories via MCP
            retrieve_start = time.time()
            retrieved_results, retrieval_time = self._retrieve_memories_mcp(
                query=question["question"],
                top_k=10
            )
            print(f"   • Retrieved {len(retrieved_results)} memories in {retrieval_time:.3f}s")

            # Step 3: Format retrieved context
            retrieved_context = self._format_retrieved_context(retrieved_results)
            retrieved_tokens = len(retrieved_context.split())  # Approximate

            # Step 4: Generate answer with retrieved context
            predicted_idx, token_metrics = self._generate_answer(
                question=question["question"],
                context=retrieved_context,
                choices=question["choices"]
            )

            # Step 5: Evaluate
            correct_idx = question.get("correct_choice_index")
            is_correct = predicted_idx == correct_idx

            print(f"   • Predicted: {predicted_idx}, Correct: {correct_idx} {'✓' if is_correct else '❌'}")
            print(f"   • Tokens: {token_metrics.total_tokens} (retrieved: ~{retrieved_tokens})")

            # Step 6: Track metrics
            baseline_tokens = 16690  # Known baseline

            result = {
                "question_id": q_id,
                "question": question["question"],
                "predicted_choice_index": predicted_idx,
                "correct_choice_index": correct_idx,
                "is_correct": is_correct,
                "question_type": question.get("question_type", "unknown"),
                "latency_total": token_metrics.latency_total,
                "latency_context_building": token_metrics.latency_context_building,
                "latency_llm_response": token_metrics.latency_llm_response,
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

            self.metrics_tracker.add_result(
                question_id=q_id,
                question_text=question["question"],
                question_type=question.get("question_type", "unknown"),
                correct_choice_index=correct_idx,
                predicted_choice_index=predicted_idx,
                latency=token_metrics.latency_total,
                tokens=token_metrics,
                llm_response_time=token_metrics.latency_llm_response,
                context_building_time=token_metrics.latency_context_building
            )

        # Generate summary
        total_time = time.time() - start_time
        summary = {
            "benchmark": "locomo-mc10-memory-augmented-mcp",
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
        print("MEMORY-AUGMENTED (MCP) BENCHMARK SUMMARY")
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
        description="Run LoCoMo-MC10 benchmark with memory-augmented retrieval via MCP"
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
        default="results/memory_augmented_mcp_results.json",
        help="Output file for results"
    )

    args = parser.parse_args()

    # Create and run experiment
    experiment = MemoryAugmentedMCPExperiment(
        dataset_path=args.dataset,
        max_questions=args.max_questions
    )

    experiment.run(output_path=args.output)


if __name__ == "__main__":
    main()
