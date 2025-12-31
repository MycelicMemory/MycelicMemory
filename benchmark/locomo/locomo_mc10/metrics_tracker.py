"""
Comprehensive metrics tracking for LoCoMo benchmark.
Tracks latency, token usage, cost, and performance metrics.
"""

from typing import Dict, List, Optional
from dataclasses import dataclass
from collections import defaultdict
import statistics

from shared.metrics_base import TokenMetrics, LatencyMetrics, MetricsBase

# Re-export for backwards compatibility
__all__ = ["TokenMetrics", "LatencyMetrics", "QuestionResult", "MetricsTracker"]


@dataclass
class QuestionResult:
    """Complete result for a single question."""
    question_id: str
    question_text: str
    question_type: str
    correct_choice_index: int
    predicted_choice_index: Optional[int]
    is_correct: bool
    latency: float
    tokens: TokenMetrics
    llm_response_time: float = 0.0
    context_building_time: float = 0.0


class MetricsTracker(MetricsBase):
    """Comprehensive metrics tracker for benchmark runs."""

    def __init__(self):
        super().__init__()
        self.results: List[QuestionResult] = []

    def add_result(
        self,
        question_id: str,
        question_text: str,
        question_type: str,
        correct_choice_index: int,
        predicted_choice_index: Optional[int],
        latency: float,
        tokens: TokenMetrics,
        llm_response_time: float = 0.0,
        context_building_time: float = 0.0,
    ) -> None:
        """Record a question result."""
        is_correct = predicted_choice_index == correct_choice_index

        result = QuestionResult(
            question_id=question_id,
            question_text=question_text,
            question_type=question_type,
            correct_choice_index=correct_choice_index,
            predicted_choice_index=predicted_choice_index,
            is_correct=is_correct,
            latency=latency,
            tokens=tokens,
            llm_response_time=llm_response_time,
            context_building_time=context_building_time,
        )
        self.results.append(result)

    def get_overall_metrics(self) -> Dict:
        """Get overall benchmark metrics."""
        if not self.results:
            return {}

        total_questions = len(self.results)
        correct = sum(1 for r in self.results if r.is_correct)

        # Latency metrics
        latencies = [r.latency for r in self.results]
        llm_times = [r.llm_response_time for r in self.results if r.llm_response_time > 0]
        context_times = [r.context_building_time for r in self.results if r.context_building_time > 0]

        # Token metrics using shared utility
        token_stats = self.get_token_stats(self.results, total_questions)
        total_input_tokens = token_stats["total_input_tokens"]
        total_output_tokens = token_stats["total_output_tokens"]

        return {
            "overall": {
                "total_questions": total_questions,
                "correct_predictions": correct,
                "accuracy": (correct / total_questions * 100) if total_questions > 0 else 0.0,
            },
            "latency": self.get_latency_stats(latencies),
            "llm_latency": {
                "mean_llm_response_seconds": statistics.mean(llm_times) if llm_times else 0.0,
                "median_llm_response_seconds": statistics.median(llm_times) if llm_times else 0.0,
                "total_llm_time_seconds": sum(llm_times),
            } if llm_times else {},
            "context_latency": {
                "mean_context_building_seconds": statistics.mean(context_times) if context_times else 0.0,
                "total_context_building_seconds": sum(context_times),
            } if context_times else {},
            "tokens": token_stats,
            "cost_estimation": self.estimate_cost(total_input_tokens, total_output_tokens, len(self.results)),
        }

    def get_per_type_metrics(self) -> Dict:
        """Get metrics broken down by question type."""
        by_type = defaultdict(lambda: {
            "total": 0,
            "correct": 0,
            "accuracy": 0.0,
            "latencies": [],
            "tokens": {"input": [], "output": [], "total": []},
        })

        for result in self.results:
            qtype = result.question_type
            by_type[qtype]["total"] += 1
            if result.is_correct:
                by_type[qtype]["correct"] += 1
            by_type[qtype]["latencies"].append(result.latency)
            by_type[qtype]["tokens"]["input"].append(result.tokens.input_tokens)
            by_type[qtype]["tokens"]["output"].append(result.tokens.output_tokens)
            by_type[qtype]["tokens"]["total"].append(result.tokens.total_tokens)

        # Calculate statistics for each type
        metrics = {}
        for qtype, data in sorted(by_type.items()):
            latencies = data["latencies"]
            metrics[qtype] = {
                "total": data["total"],
                "correct": data["correct"],
                "accuracy": (data["correct"] / data["total"] * 100) if data["total"] > 0 else 0.0,
                "latency": {
                    "mean_seconds": statistics.mean(latencies),
                    "median_seconds": statistics.median(latencies),
                    "p95_seconds": self.percentile(latencies, 95),
                    "min_seconds": min(latencies),
                    "max_seconds": max(latencies),
                } if latencies else {},
                "tokens": {
                    "total_input": sum(data["tokens"]["input"]),
                    "total_output": sum(data["tokens"]["output"]),
                    "total_tokens": sum(data["tokens"]["total"]),
                    "mean_input": sum(data["tokens"]["input"]) / data["total"] if data["total"] > 0 else 0,
                    "mean_output": sum(data["tokens"]["output"]) / data["total"] if data["total"] > 0 else 0,
                },
            }

        return metrics

    def get_error_analysis(self) -> Dict:
        """Analyze prediction errors by type."""
        errors_by_type = defaultdict(list)

        for result in self.results:
            if not result.is_correct:
                errors_by_type[result.question_type].append({
                    "question_id": result.question_id,
                    "correct_index": result.correct_choice_index,
                    "predicted_index": result.predicted_choice_index,
                })

        return {
            "total_errors": len([r for r in self.results if not r.is_correct]),
            "error_rate": (
                (len([r for r in self.results if not r.is_correct]) / len(self.results) * 100)
                if self.results else 0.0
            ),
            "errors_by_type": dict(errors_by_type),
        }

    # Note: percentile() and estimate_cost() inherited from MetricsBase
