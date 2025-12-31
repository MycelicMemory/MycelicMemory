"""
Memory-Augmented LoCoMo Free-Response Benchmark

Runs LoCoMo free-response benchmark with ultrathink memory retrieval instead of full context.

Flow:
1. Load dataset (locomo10.json - 10 conversations with 1986 QA pairs)
2. For each question:
   - Ingest conversation history as memories
   - Retrieve relevant memories via semantic search
   - Generate FREE-TEXT answer using retrieved context
   - Evaluate with F1 score (category-specific)
   - Track metrics: F1, tokens saved, latency
   - Clean up memories
3. Generate results and comparison report
"""

import json
import time
import os
import sys
from typing import List, Dict, Tuple, Optional
from dataclasses import dataclass
from datetime import datetime
import requests

# Add parent directory to path for shared imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from shared.ultrathink_client import UltrathinkClient, RetrievalResult
from shared.logging_system import BenchmarkLogger, CallType, init_logger, get_logger
from shared.llm_call_tracker import LLMCallTracker
from shared.config import DEEPSEEK_API_KEY, DEEPSEEK_BASE_URL, DEEPSEEK_MODEL
from locomo10.metrics_tracker import FRMetricsTracker, TokenMetrics
from locomo10.progress_display import FRProgressDisplay
from locomo10.f1_evaluator import F1Evaluator, get_category_name, get_category_display


@dataclass
class RetrievalMetrics:
    """Metrics for memory retrieval efficiency."""
    tokens_baseline: int          # Full context baseline
    tokens_retrieved: int         # Retrieved context
    token_reduction_pct: float    # (baseline - retrieved) / baseline * 100
    num_memories_retrieved: int   # Count of memories used
    retrieval_latency: float      # Time to retrieve


class FreeResponseExperiment:
    """Run LoCoMo free-response benchmark with memory-augmented retrieval."""

    def __init__(
        self,
        dataset_path: str = "data/locomo10.json",
        max_questions: Optional[int] = None,
        ultrathink_url: str = "http://localhost:3099/api/v1",
        enable_logging: bool = True,
        log_dir: str = "logs",
        random_sample: bool = False,
        seed: Optional[int] = None
    ):
        """
        Initialize free-response experiment.

        Args:
            dataset_path: Path to LoCoMo free-response dataset (locomo10.json)
            max_questions: Max questions to process (None = all)
            ultrathink_url: Ultrathink server URL
            enable_logging: Enable comprehensive logging
            log_dir: Directory for log files
            random_sample: If True, randomly sample questions instead of taking first N
            seed: Random seed for reproducible sampling (None = random)
        """
        self.dataset_path = dataset_path
        self.max_questions = max_questions
        self.random_sample = random_sample
        self.seed = seed
        self.deepseek_api_key = DEEPSEEK_API_KEY
        self.ultrathink_url = ultrathink_url

        # Initialize logging
        self.enable_logging = enable_logging
        if enable_logging:
            self.logger = init_logger(
                name="fr-memory-augmented-benchmark",
                log_dir=log_dir
            )
        else:
            self.logger = None

        # Initialize clients
        self.memory_client = UltrathinkClient(base_url=ultrathink_url)
        self.llm_client = requests.Session()
        self.llm_tracker = LLMCallTracker(logger=self.logger)

        # Initialize F1 evaluator
        self.evaluator = F1Evaluator()

        # Initialize metrics
        self.metrics_tracker = FRMetricsTracker()

        # Load dataset
        self.questions = self._load_dataset()

    def _load_dataset(self) -> List[Dict]:
        """
        Load and flatten LoCoMo free-response dataset.

        The dataset has structure:
        [
            {
                "sample_id": "conv-1",
                "conversation": [...],
                "session_summary": [...],
                "qa": [
                    {"question": "...", "answer": "...", "evidence": [...], "category": 2}
                ]
            }
        ]

        Returns flattened list of questions with conversation context.
        """
        if not os.path.exists(self.dataset_path):
            print(f"Dataset not found at {self.dataset_path}")
            print("   Attempting to download...")
            self._download_dataset()

        with open(self.dataset_path, "r") as f:
            data = json.load(f)

        # Flatten conversations into individual QA pairs
        questions = []
        for conv in data:
            conv_id = conv.get("sample_id", "unknown")
            conversation = conv.get("conversation", [])

            for qa_idx, qa in enumerate(conv.get("qa", [])):
                question_id = f"{conv_id}_q{qa_idx}"
                questions.append({
                    "question_id": question_id,
                    "conversation_id": conv_id,
                    "question": qa.get("question", ""),
                    "answer": qa.get("answer", ""),
                    "evidence": qa.get("evidence", []),
                    "category": qa.get("category", 0),
                    "conversation": conversation,
                })

        if self.max_questions:
            if self.random_sample:
                import random
                rng = random.Random(self.seed) if self.seed is not None else random.Random()
                sample_size = min(self.max_questions, len(questions))
                questions = rng.sample(questions, sample_size)
            else:
                questions = questions[:self.max_questions]

        print(f"Loaded {len(questions)} questions from {len(data)} conversations")
        return questions

    def _download_dataset(self) -> None:
        """Download LoCoMo free-response dataset from GitHub."""
        url = "https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json"
        os.makedirs("data", exist_ok=True)

        try:
            print(f"   Downloading from {url}...")
            response = requests.get(url, timeout=120)
            response.raise_for_status()

            with open(self.dataset_path, "w") as f:
                f.write(response.text)
            print(f"   Downloaded to {self.dataset_path}")
        except Exception as e:
            print(f"   Download failed: {e}")
            raise

    def _flatten_conversation(self, conversation: Dict) -> List[Dict]:
        """
        Convert conversation dict into flat message list for memory ingestion.

        The conversation structure is:
        {
            "speaker_a": "Name1",
            "speaker_b": "Name2",
            "session_1_date_time": "...",
            "session_1": [
                {"speaker": "Name1", "dia_id": "D1:1", "text": "..."},
                ...
            ],
            "session_2_date_time": "...",
            "session_2": [...],
            ...
        }
        """
        messages = []
        speaker_a = conversation.get("speaker_a", "Speaker A")
        speaker_b = conversation.get("speaker_b", "Speaker B")

        # Find all session keys (session_1, session_2, etc.)
        session_keys = sorted([
            key for key in conversation.keys()
            if key.startswith("session_") and not key.endswith("_date_time")
        ], key=lambda x: int(x.split("_")[1]) if x.split("_")[1].isdigit() else 0)

        for session_key in session_keys:
            session = conversation.get(session_key, [])
            if not isinstance(session, list):
                continue

            # Get session date/time if available
            date_key = f"{session_key}_date_time"
            session_datetime = conversation.get(date_key, "")

            for turn in session:
                if not isinstance(turn, dict):
                    continue

                speaker = turn.get("speaker", "")
                text = turn.get("text", "")
                dia_id = turn.get("dia_id", "")

                if text.strip():
                    # Map speaker to role (alternating user/assistant)
                    role = "user" if speaker == speaker_a else "assistant"

                    # Include date context in first message of session
                    content = text
                    if session_datetime and turn == session[0]:
                        content = f"[{session_datetime}] {text}"

                    messages.append({
                        "role": role,
                        "content": content,
                        "metadata": {
                            "speaker": speaker,
                            "dia_id": dia_id,
                            "session": session_key
                        }
                    })

        return messages

    def _generate_answer(
        self,
        question: str,
        context: str,
        category: int
    ) -> Tuple[str, TokenMetrics, float]:
        """
        Generate FREE-TEXT answer using DeepSeek LLM with retrieved context.

        Args:
            question: The question text
            context: Retrieved context (from memories)
            category: Question category (1-5)

        Returns:
            Tuple of (predicted_answer, token_metrics, llm_latency)
        """
        # Category-specific prompting
        if category == 5:  # Adversarial
            system_prompt = """You are a helpful assistant. Answer questions based ONLY on the provided context.
If the answer is not found in the context, respond with "No information available" or similar."""
        else:
            system_prompt = """You are a helpful assistant. Answer questions based on the provided context.
Provide concise, direct answers without explanation."""

        prompt = f"""Based on the following conversation context, answer the question.

CONTEXT:
{context}

QUESTION: {question}

Provide a concise, direct answer. Do not explain your reasoning."""

        # Build messages for LLM
        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": prompt}
        ]

        try:
            # Use LLMCallTracker for full transparency
            response_text, metrics = self.llm_tracker.call_llm_api(
                api_key=self.deepseek_api_key,
                model="deepseek-chat",
                messages=messages,
                temperature=0,
                max_tokens=100,  # Longer for free-text answers
                top_p=1
            )

            # Create token metrics
            token_metrics = TokenMetrics(
                input_tokens=metrics.input_tokens,
                output_tokens=metrics.output_tokens,
                total_tokens=metrics.total_tokens
            )

            return response_text.strip(), token_metrics, metrics.total_time_ms / 1000

        except Exception as e:
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.BENCHMARK_END,
                    f"LLM error occurred",
                    {"error": str(e), "question": question[:100]}
                )
            print(f"LLM error: {e}")
            return "", TokenMetrics(0, 0, 0), 0

    def run(self, output_path: str = "results/memory_augmented_fr_results.json") -> Dict:
        """
        Run free-response memory-augmented benchmark.

        Args:
            output_path: Where to save results

        Returns:
            Dictionary of metrics
        """
        # Initialize progress display for live logging
        progress = FRProgressDisplay(total_questions=len(self.questions))
        progress.display_header(
            dataset_path=self.dataset_path,
            ultrathink_url=self.ultrathink_url
        )

        # Log benchmark start
        if self.logger:
            self.logger.log_benchmark_event(
                CallType.BENCHMARK_START,
                "Free-response memory-augmented LoCoMo benchmark started",
                {
                    "total_questions": len(self.questions),
                    "max_questions": self.max_questions,
                    "dataset": self.dataset_path,
                    "ultrathink_url": self.ultrathink_url
                }
            )

        # Verify ultrathink is running
        if not self.memory_client.health_check():
            print("Ultrathink server not running!")
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.BENCHMARK_END,
                    "Benchmark failed: Ultrathink server not available",
                    {"error": "Health check failed"}
                )
            sys.exit(1)

        results = {}
        start_time = time.time()

        # Track last conversation ID to avoid re-ingesting
        last_conv_id = None
        cached_memory_ids = []

        for idx, question_data in enumerate(self.questions):
            q_id = question_data["question_id"]
            conv_id = question_data["conversation_id"]
            category = question_data["category"]

            # Display question start with category
            progress.display_question_start(
                idx=idx,
                question_id=q_id,
                category=category,
                question_text=question_data["question"]
            )

            # Log question start
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.QUESTION_START,
                    f"Processing question {idx+1}/{len(self.questions)}",
                    {
                        "question_id": q_id,
                        "category": category,
                        "category_name": get_category_name(category),
                        "question_preview": question_data["question"][:100]
                    }
                )

            # Step 1: Ingest conversation as memories (cache by conversation)
            session_id = f"locomo-fr-{conv_id}"

            if conv_id != last_conv_id:
                # Clear previous session if exists
                if last_conv_id is not None:
                    prev_session_id = f"locomo-fr-{last_conv_id}"
                    self.memory_client.clear_session(prev_session_id)

                # Ingest new conversation
                messages = self._flatten_conversation(question_data["conversation"])
                ingest_start = time.time()
                cached_memory_ids, ingest_time = self.memory_client.ingest_conversation(
                    messages=messages,
                    session_id=session_id,
                    domain="locomo-fr-benchmark"
                )
                last_conv_id = conv_id

                # Log memory ingest
                if self.logger:
                    self.logger.log_memory_call(
                        call_type=CallType.MEMORY_INGEST,
                        operation="ingest_conversation",
                        duration_ms=ingest_time * 1000,
                        num_items=len(cached_memory_ids),
                        status="success",
                        metadata={
                            "question_id": q_id,
                            "session_id": session_id,
                            "num_messages": len(messages)
                        }
                    )
            else:
                # Use cached conversation
                ingest_time = 0.0

            # Step 2: Retrieve relevant memories
            retrieve_start = time.time()
            retrieved_results, retrieval_time = self.memory_client.retrieve_memories(
                query=question_data["question"],
                top_k=10,
                use_ai=True,
                min_similarity=0.0
            )

            # Display memory operations
            progress.display_memory_ops(
                num_ingested=len(cached_memory_ids),
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
                        "query_preview": question_data["question"][:100],
                        "top_k": 10,
                        "use_ai": True
                    }
                )

            # Step 3: Format retrieved context
            retrieved_context = self.memory_client.format_retrieved_as_context(retrieved_results)
            retrieved_tokens = len(retrieved_context.split())

            # Step 4: Generate FREE-TEXT answer
            prediction, token_metrics, llm_latency = self._generate_answer(
                question=question_data["question"],
                context=retrieved_context,
                category=category
            )

            # Step 5: Evaluate with F1 score
            ground_truth = question_data["answer"]
            eval_result = self.evaluator.evaluate(
                prediction=prediction,
                ground_truth=ground_truth,
                category=category
            )
            f1_score = eval_result["f1_score"]

            # Display result with F1 score
            progress.display_result(
                prediction=prediction,
                ground_truth=ground_truth,
                f1_score=f1_score,
                llm_latency=llm_latency,
                input_tokens=token_metrics.input_tokens,
                output_tokens=token_metrics.output_tokens,
                category=category
            )

            # Step 6: Track metrics
            baseline_tokens = 20000  # Approximate for full conversation
            total_latency = retrieval_time + llm_latency  # End-to-end latency

            # Record in metrics tracker
            self.metrics_tracker.add_result(
                question_id=q_id,
                question_text=question_data["question"],
                category=category,
                ground_truth=ground_truth,
                prediction=prediction,
                f1_score=f1_score,
                evaluation_method=eval_result["evaluation_method"],
                latency=total_latency,  # End-to-end for backwards compatibility
                tokens=token_metrics,
                llm_response_time=llm_latency,
                context_building_time=retrieval_time,
                memory_retrieval_latency=retrieval_time,  # Key metric for ultrathink performance
                end_to_end_latency=total_latency,  # Retrieval + LLM
            )

            # Store raw result
            result = {
                "question_id": q_id,
                "conversation_id": conv_id,
                "question": question_data["question"],
                "ground_truth": ground_truth,
                "prediction": prediction,
                "f1_score": f1_score,
                "evaluation_method": eval_result["evaluation_method"],
                "category": category,
                "category_name": get_category_name(category),
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
                    "token_reduction_pct": (baseline_tokens - retrieved_tokens) / baseline_tokens * 100 if baseline_tokens > 0 else 0,
                    "num_memories_retrieved": len(retrieved_results),
                    "retrieval_latency": retrieval_time,
                    "session_id": session_id
                }
            }

            results[q_id] = result

            # Log question end
            if self.logger:
                self.logger.log_benchmark_event(
                    CallType.QUESTION_END,
                    f"Question {idx+1} completed",
                    {
                        "question_id": q_id,
                        "f1_score": f1_score,
                        "category": category,
                        "tokens_used": token_metrics.total_tokens,
                        "total_latency": retrieval_time + llm_latency
                    }
                )

        # Cleanup final session
        if last_conv_id is not None:
            final_session_id = f"locomo-fr-{last_conv_id}"
            self.memory_client.clear_session(final_session_id)

        # Generate summary
        total_time = time.time() - start_time
        summary = {
            "benchmark": "locomo-fr-memory-augmented",
            "timestamp": datetime.now().isoformat(),
            "total_questions": len(self.questions),
            "total_time_seconds": total_time,
            "results": results
        }

        # Save results
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(summary, f, indent=2)

        print(f"\nSaved results to {output_path}")

        # Log benchmark end and generate report
        if self.logger:
            metrics = self.metrics_tracker.get_overall_metrics()
            self.logger.log_benchmark_event(
                CallType.BENCHMARK_END,
                "Free-response memory-augmented LoCoMo benchmark completed",
                {
                    "total_questions": len(self.questions),
                    "total_time_seconds": total_time,
                    "mean_f1": metrics.get('overall', {}).get('mean_f1', 0),
                    "results_file": output_path
                }
            )
            # Save comprehensive report
            report_path = self.logger.save_report()
            print(f"\nDetailed logs saved to {report_path}")

        # Display comprehensive final summary
        overall_metrics = self.metrics_tracker.get_overall_metrics()
        per_category_metrics = self.metrics_tracker.get_per_category_metrics()
        progress.display_final_summary(
            metrics=overall_metrics,
            per_category_metrics=per_category_metrics,
            duration_secs=total_time
        )

        return summary


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description="Run LoCoMo free-response benchmark with memory-augmented retrieval"
    )
    parser.add_argument(
        "--dataset",
        type=str,
        default="data/locomo10.json",
        help="Path to LoCoMo free-response dataset"
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
        default="results/memory_augmented_fr_results.json",
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
    parser.add_argument(
        "--random-sample",
        action="store_true",
        help="Randomly sample questions instead of taking first N"
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=None,
        help="Random seed for reproducible sampling"
    )

    args = parser.parse_args()

    # Create and run experiment
    experiment = FreeResponseExperiment(
        dataset_path=args.dataset,
        max_questions=args.max_questions,
        ultrathink_url=args.ultrathink_url,
        enable_logging=args.enable_logging,
        log_dir=args.log_dir,
        random_sample=args.random_sample,
        seed=args.seed
    )

    experiment.run(output_path=args.output)


if __name__ == "__main__":
    main()
