"""
Base metrics classes and utilities shared across benchmark variants.
"""

import time
import statistics
from typing import Dict, List
from dataclasses import dataclass, asdict


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
    total_latency: float = 0.0
    context_building_time: float = 0.0
    llm_response_time: float = 0.0

    def to_dict(self) -> Dict:
        return asdict(self)


class MetricsBase:
    """Base class for metrics trackers with shared utility methods."""

    # DeepSeek pricing (as of 2024)
    INPUT_PRICE_PER_MTOK = 0.014
    OUTPUT_PRICE_PER_MTOK = 0.056

    def __init__(self):
        self.start_time = time.time()

    @staticmethod
    def percentile(data: List[float], percentile: int) -> float:
        """Calculate percentile of data."""
        if not data:
            return 0.0
        sorted_data = sorted(data)
        index = int(len(sorted_data) * percentile / 100)
        return sorted_data[min(index, len(sorted_data) - 1)]

    @classmethod
    def estimate_cost(cls, input_tokens: int, output_tokens: int, num_questions: int) -> Dict:
        """
        Estimate API cost based on DeepSeek pricing.
        Pricing (as of 2024):
        - Input: $0.014 / 1M tokens
        - Output: $0.056 / 1M tokens
        """
        input_cost = (input_tokens / 1_000_000) * cls.INPUT_PRICE_PER_MTOK
        output_cost = (output_tokens / 1_000_000) * cls.OUTPUT_PRICE_PER_MTOK
        total_cost = input_cost + output_cost

        return {
            "input_cost_usd": round(input_cost, 6),
            "output_cost_usd": round(output_cost, 6),
            "total_cost_usd": round(total_cost, 6),
            "cost_per_question_usd": round(total_cost / num_questions, 6) if num_questions > 0 else 0.0,
        }

    def get_latency_stats(self, latencies: List[float]) -> Dict:
        """Calculate comprehensive latency statistics."""
        if not latencies:
            return {}

        return {
            "total_latency_seconds": sum(latencies),
            "mean_latency_seconds": statistics.mean(latencies),
            "median_latency_seconds": statistics.median(latencies),
            "p95_latency_seconds": self.percentile(latencies, 95),
            "p99_latency_seconds": self.percentile(latencies, 99),
            "min_latency_seconds": min(latencies),
            "max_latency_seconds": max(latencies),
            "stdev_latency_seconds": statistics.stdev(latencies) if len(latencies) > 1 else 0.0,
        }

    def get_token_stats(self, results: List, total_questions: int) -> Dict:
        """Calculate token usage statistics."""
        total_input = sum(r.tokens.input_tokens for r in results)
        total_output = sum(r.tokens.output_tokens for r in results)
        total = sum(r.tokens.total_tokens for r in results)

        return {
            "total_input_tokens": total_input,
            "total_output_tokens": total_output,
            "total_tokens": total,
            "mean_input_tokens": total_input / total_questions if total_questions > 0 else 0,
            "mean_output_tokens": total_output / total_questions if total_questions > 0 else 0,
            "mean_total_tokens": total / total_questions if total_questions > 0 else 0,
        }
