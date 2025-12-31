"""
Comprehensive metrics tracking for LoCoMo benchmark.
Tracks latency, token usage, cost, and performance metrics.
"""

import time
from typing import Dict, List, Optional
from dataclasses import dataclass, asdict
from collections import defaultdict
import statistics


@dataclass
class TokenMetrics:
    """Token usage metrics."""
    input_tokens: int = 0
    output_tokens: int = 0
    total_tokens: int = 0

    def to_dict(self) -> Dict:
        return asdict(self)


@dataclass
class LatencyMetrics:
    """Latency metrics in seconds."""
    total_latency: float = 0.0  # E2EL - End-to-End Latency
    ttft: Optional[float] = None  # Time to First Token (not tracked in our simple case)
    tpot: Optional[float] = None  # Time per Output Token

    def to_dict(self) -> Dict:
        return asdict(self)


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


class MetricsTracker:
    """Comprehensive metrics tracker for benchmark runs."""

    def __init__(self):
        self.results: List[QuestionResult] = []
        self.start_time = time.time()

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

        # Token metrics
        total_input_tokens = sum(r.tokens.input_tokens for r in self.results)
        total_output_tokens = sum(r.tokens.output_tokens for r in self.results)
        total_tokens = sum(r.tokens.total_tokens for r in self.results)

        return {
            "overall": {
                "total_questions": total_questions,
                "correct_predictions": correct,
                "accuracy": (correct / total_questions * 100) if total_questions > 0 else 0.0,
            },
            "latency": {
                "total_latency_seconds": sum(latencies),
                "mean_latency_seconds": statistics.mean(latencies),
                "median_latency_seconds": statistics.median(latencies),
                "p95_latency_seconds": self._percentile(latencies, 95),
                "p99_latency_seconds": self._percentile(latencies, 99),
                "min_latency_seconds": min(latencies),
                "max_latency_seconds": max(latencies),
                "stdev_latency_seconds": statistics.stdev(latencies) if len(latencies) > 1 else 0.0,
            },
            "llm_latency": {
                "mean_llm_response_seconds": statistics.mean(llm_times) if llm_times else 0.0,
                "median_llm_response_seconds": statistics.median(llm_times) if llm_times else 0.0,
                "total_llm_time_seconds": sum(llm_times),
            } if llm_times else {},
            "context_latency": {
                "mean_context_building_seconds": statistics.mean(context_times) if context_times else 0.0,
                "total_context_building_seconds": sum(context_times),
            } if context_times else {},
            "tokens": {
                "total_input_tokens": total_input_tokens,
                "total_output_tokens": total_output_tokens,
                "total_tokens": total_tokens,
                "mean_input_tokens": total_input_tokens / total_questions if total_questions > 0 else 0,
                "mean_output_tokens": total_output_tokens / total_questions if total_questions > 0 else 0,
                "mean_total_tokens": total_tokens / total_questions if total_questions > 0 else 0,
            },
            "cost_estimation": self._estimate_cost(total_input_tokens, total_output_tokens, len(self.results)),
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
                    "p95_seconds": self._percentile(latencies, 95),
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

    @staticmethod
    def _percentile(data: List[float], percentile: int) -> float:
        """Calculate percentile of data."""
        if not data:
            return 0.0
        sorted_data = sorted(data)
        index = int(len(sorted_data) * percentile / 100)
        return sorted_data[min(index, len(sorted_data) - 1)]

    @staticmethod
    def _estimate_cost(input_tokens: int, output_tokens: int, num_questions: int) -> Dict:
        """
        Estimate API cost based on DeepSeek pricing.
        Pricing (as of 2024):
        - Input: $0.014 / 1M tokens
        - Output: $0.056 / 1M tokens
        """
        input_price_per_mtok = 0.014
        output_price_per_mtok = 0.056

        input_cost = (input_tokens / 1_000_000) * input_price_per_mtok
        output_cost = (output_tokens / 1_000_000) * output_price_per_mtok
        total_cost = input_cost + output_cost

        return {
            "input_cost_usd": round(input_cost, 6),
            "output_cost_usd": round(output_cost, 6),
            "total_cost_usd": round(total_cost, 6),
            "cost_per_question_usd": round(total_cost / num_questions, 6) if num_questions > 0 else 0.0,
        }
