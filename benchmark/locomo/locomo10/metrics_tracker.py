"""
Metrics tracking for LoCoMo Free-Response benchmark.
Tracks F1 scores, latency, token usage, and cost metrics.
"""

from typing import Dict, List
from dataclasses import dataclass
from collections import defaultdict
import statistics

from shared.metrics_base import TokenMetrics, LatencyMetrics, MetricsBase
from .f1_evaluator import CATEGORY_NAMES, CATEGORY_DISPLAY

# Re-export for backwards compatibility
__all__ = ["TokenMetrics", "LatencyMetrics", "FRQuestionResult", "FRMetricsTracker"]


@dataclass
class FRQuestionResult:
    """Complete result for a single free-response question."""
    question_id: str
    question_text: str
    category: int
    category_name: str
    ground_truth: str
    prediction: str
    f1_score: float
    evaluation_method: str
    latency: float  # Alias for end_to_end_latency (kept for backwards compatibility)
    tokens: TokenMetrics
    llm_response_time: float = 0.0
    context_building_time: float = 0.0  # Alias for memory_retrieval_latency
    memory_retrieval_latency: float = 0.0  # Time to retrieve memories from ultrathink
    end_to_end_latency: float = 0.0  # Total time: retrieval + LLM response


class FRMetricsTracker(MetricsBase):
    """Metrics tracker for free-response benchmark runs."""

    def __init__(self):
        super().__init__()
        self.results: List[FRQuestionResult] = []

    def add_result(
        self,
        question_id: str,
        question_text: str,
        category: int,
        ground_truth: str,
        prediction: str,
        f1_score: float,
        evaluation_method: str,
        latency: float,
        tokens: TokenMetrics,
        llm_response_time: float = 0.0,
        context_building_time: float = 0.0,
        memory_retrieval_latency: float = 0.0,
        end_to_end_latency: float = 0.0,
    ) -> None:
        """Record a question result."""
        # If new fields not provided, derive from legacy fields
        if memory_retrieval_latency == 0.0 and context_building_time > 0.0:
            memory_retrieval_latency = context_building_time
        if end_to_end_latency == 0.0 and latency > 0.0:
            end_to_end_latency = latency

        result = FRQuestionResult(
            question_id=question_id,
            question_text=question_text,
            category=category,
            category_name=CATEGORY_NAMES.get(category, "unknown"),
            ground_truth=ground_truth,
            prediction=prediction,
            f1_score=f1_score,
            evaluation_method=evaluation_method,
            latency=latency,
            tokens=tokens,
            llm_response_time=llm_response_time,
            context_building_time=context_building_time,
            memory_retrieval_latency=memory_retrieval_latency,
            end_to_end_latency=end_to_end_latency,
        )
        self.results.append(result)

    def get_overall_metrics(self) -> Dict:
        """Get overall benchmark metrics."""
        if not self.results:
            return {}

        total_questions = len(self.results)
        f1_scores = [r.f1_score for r in self.results]
        mean_f1 = statistics.mean(f1_scores) if f1_scores else 0.0

        # Latency metrics - separate memory retrieval and end-to-end
        memory_retrieval_latencies = [r.memory_retrieval_latency for r in self.results if r.memory_retrieval_latency > 0]
        end_to_end_latencies = [r.end_to_end_latency for r in self.results if r.end_to_end_latency > 0]

        # Legacy latency fields for backwards compatibility
        latencies = [r.latency for r in self.results]
        llm_times = [r.llm_response_time for r in self.results if r.llm_response_time > 0]

        # Token metrics using shared utility
        token_stats = self.get_token_stats(self.results, total_questions)
        total_input_tokens = token_stats["total_input_tokens"]
        total_output_tokens = token_stats["total_output_tokens"]

        # Calculate accuracy by category (excluding adversarial - category 5)
        # Accuracy = proportion of questions with F1 >= 0.5 (considered "correct")
        accuracy_by_category = self._calculate_accuracy_by_category(threshold=0.5)

        return {
            "overall": {
                "total_questions": total_questions,
                "mean_f1": mean_f1,
                "median_f1": statistics.median(f1_scores) if f1_scores else 0.0,
                "min_f1": min(f1_scores) if f1_scores else 0.0,
                "max_f1": max(f1_scores) if f1_scores else 0.0,
                "stdev_f1": statistics.stdev(f1_scores) if len(f1_scores) > 1 else 0.0,
            },
            # Memory retrieval latency (ultrathink search time) - key metric to compare with Nebula (~125ms)
            "memory_retrieval_latency": {
                "p50_seconds": statistics.median(memory_retrieval_latencies) if memory_retrieval_latencies else 0.0,
                "p95_seconds": self.percentile(memory_retrieval_latencies, 95),
                "mean_seconds": statistics.mean(memory_retrieval_latencies) if memory_retrieval_latencies else 0.0,
                "min_seconds": min(memory_retrieval_latencies) if memory_retrieval_latencies else 0.0,
                "max_seconds": max(memory_retrieval_latencies) if memory_retrieval_latencies else 0.0,
            } if memory_retrieval_latencies else {},
            # End-to-end latency (retrieval + LLM response)
            "end_to_end_latency": {
                "p50_seconds": statistics.median(end_to_end_latencies) if end_to_end_latencies else 0.0,
                "p95_seconds": self.percentile(end_to_end_latencies, 95),
                "mean_seconds": statistics.mean(end_to_end_latencies) if end_to_end_latencies else 0.0,
                "min_seconds": min(end_to_end_latencies) if end_to_end_latencies else 0.0,
                "max_seconds": max(end_to_end_latencies) if end_to_end_latencies else 0.0,
            } if end_to_end_latencies else {},
            # Accuracy by category (excluding adversarial)
            "accuracy_by_category": accuracy_by_category,
            # Legacy latency fields for backwards compatibility
            "latency": self.get_latency_stats(latencies),
            "llm_latency": {
                "mean_llm_response_seconds": statistics.mean(llm_times) if llm_times else 0.0,
                "median_llm_response_seconds": statistics.median(llm_times) if llm_times else 0.0,
                "total_llm_time_seconds": sum(llm_times),
            } if llm_times else {},
            "tokens": token_stats,
            "cost_estimation": self.estimate_cost(total_input_tokens, total_output_tokens, len(self.results)),
        }

    def _calculate_accuracy_by_category(self, threshold: float = 0.5) -> Dict:
        """
        Calculate accuracy by category, excluding adversarial (category 5).

        Accuracy is defined as the proportion of questions with F1 >= threshold.
        """
        by_category = defaultdict(lambda: {"total": 0, "correct": 0})

        for result in self.results:
            # Skip adversarial questions (category 5)
            if result.category == 5:
                continue

            cat_name = result.category_name
            by_category[cat_name]["total"] += 1
            if result.f1_score >= threshold:
                by_category[cat_name]["correct"] += 1

        # Calculate accuracy percentages
        accuracy = {}
        total_correct = 0
        total_questions = 0

        for cat_name, data in by_category.items():
            if data["total"] > 0:
                acc = (data["correct"] / data["total"]) * 100
                accuracy[cat_name] = {
                    "accuracy_pct": round(acc, 1),
                    "correct": data["correct"],
                    "total": data["total"],
                }
                total_correct += data["correct"]
                total_questions += data["total"]

        # Overall accuracy (excluding adversarial)
        if total_questions > 0:
            accuracy["overall_excluding_adversarial"] = {
                "accuracy_pct": round((total_correct / total_questions) * 100, 1),
                "correct": total_correct,
                "total": total_questions,
            }

        return accuracy

    def get_per_category_metrics(self) -> Dict:
        """Get metrics broken down by question category."""
        by_category = defaultdict(lambda: {
            "total": 0,
            "f1_scores": [],
            "latencies": [],
            "tokens": {"input": [], "output": [], "total": []},
        })

        for result in self.results:
            cat = result.category
            by_category[cat]["total"] += 1
            by_category[cat]["f1_scores"].append(result.f1_score)
            by_category[cat]["latencies"].append(result.latency)
            by_category[cat]["tokens"]["input"].append(result.tokens.input_tokens)
            by_category[cat]["tokens"]["output"].append(result.tokens.output_tokens)
            by_category[cat]["tokens"]["total"].append(result.tokens.total_tokens)

        # Calculate statistics for each category
        metrics = {}
        for cat in sorted(by_category.keys()):
            data = by_category[cat]
            f1_scores = data["f1_scores"]
            latencies = data["latencies"]
            cat_name = CATEGORY_NAMES.get(cat, "unknown")

            metrics[cat_name] = {
                "category_id": cat,
                "category_display": CATEGORY_DISPLAY.get(cat, "Unknown"),
                "total": data["total"],
                "f1": {
                    "mean": statistics.mean(f1_scores) if f1_scores else 0.0,
                    "median": statistics.median(f1_scores) if f1_scores else 0.0,
                    "min": min(f1_scores) if f1_scores else 0.0,
                    "max": max(f1_scores) if f1_scores else 0.0,
                    "stdev": statistics.stdev(f1_scores) if len(f1_scores) > 1 else 0.0,
                },
                "latency": {
                    "mean_seconds": statistics.mean(latencies) if latencies else 0.0,
                    "median_seconds": statistics.median(latencies) if latencies else 0.0,
                    "p95_seconds": self.percentile(latencies, 95),
                    "min_seconds": min(latencies) if latencies else 0.0,
                    "max_seconds": max(latencies) if latencies else 0.0,
                },
                "tokens": {
                    "total_input": sum(data["tokens"]["input"]),
                    "total_output": sum(data["tokens"]["output"]),
                    "total_tokens": sum(data["tokens"]["total"]),
                    "mean_input": sum(data["tokens"]["input"]) / data["total"] if data["total"] > 0 else 0,
                    "mean_output": sum(data["tokens"]["output"]) / data["total"] if data["total"] > 0 else 0,
                },
            }

        return metrics

    def get_low_score_analysis(self, threshold: float = 0.5) -> Dict:
        """Analyze low-scoring predictions."""
        low_scores_by_category = defaultdict(list)

        for result in self.results:
            if result.f1_score < threshold:
                low_scores_by_category[result.category_name].append({
                    "question_id": result.question_id,
                    "question": result.question_text[:100],
                    "ground_truth": result.ground_truth,
                    "prediction": result.prediction,
                    "f1_score": result.f1_score,
                })

        low_score_count = sum(1 for r in self.results if r.f1_score < threshold)

        return {
            "threshold": threshold,
            "total_low_scores": low_score_count,
            "low_score_rate": (
                (low_score_count / len(self.results) * 100)
                if self.results else 0.0
            ),
            "low_scores_by_category": dict(low_scores_by_category),
        }

    # Note: percentile() and estimate_cost() inherited from MetricsBase
