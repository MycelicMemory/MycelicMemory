"""
Baseline vs Memory-Augmented Comparison Evaluation

Compares results from:
- run_experiments.py (baseline: full context)
- memory_augmented.py (optimized: retrieved context)

Generates comprehensive comparison report showing:
- Accuracy impact
- Token/cost reduction
- Latency improvements
- Per-type analysis
"""

import json
import os
import sys
from typing import Dict, List, Tuple, Optional
from dataclasses import dataclass
from datetime import datetime


@dataclass
class ComparisonMetrics:
    """Side-by-side comparison of baseline vs memory-augmented."""
    # Accuracy
    baseline_accuracy: float
    memory_accuracy: float
    accuracy_delta: float

    # Tokens
    baseline_tokens_total: int
    memory_tokens_total: int
    token_reduction_abs: int
    token_reduction_pct: float

    # Cost
    baseline_cost: float
    memory_cost: float
    cost_savings: float
    cost_savings_pct: float

    # Latency
    baseline_latency_mean: float
    memory_latency_mean: float
    latency_improvement: float
    latency_improvement_pct: float

    # Retrieval stats
    avg_memories_retrieved: int
    avg_token_reduction_per_q: float


class ComparisonEvaluator:
    """Evaluate baseline vs memory-augmented results."""

    def __init__(
        self,
        baseline_file: str = "results/ultrathink_results.json",
        memory_file: str = "results/memory_augmented_results.json"
    ):
        """
        Initialize comparison evaluator.

        Args:
            baseline_file: Path to baseline results
            memory_file: Path to memory-augmented results
        """
        self.baseline_file = baseline_file
        self.memory_file = memory_file
        self.baseline_data = None
        self.memory_data = None

    def load_results(self) -> bool:
        """Load both result files."""
        print("\nLoading results...")

        # Load baseline
        if not os.path.exists(self.baseline_file):
            print(f"❌ Baseline file not found: {self.baseline_file}")
            return False

        try:
            with open(self.baseline_file, "r") as f:
                self.baseline_data = json.load(f)
            print(f"✓ Loaded baseline ({len(self.baseline_data.get('results', {}))} questions)")
        except Exception as e:
            print(f"❌ Error loading baseline: {e}")
            return False

        # Load memory-augmented
        if not os.path.exists(self.memory_file):
            print(f"⚠️  Memory-augmented file not found: {self.memory_file}")
            print("   This is OK if not running memory-augmented yet")
            return False

        try:
            with open(self.memory_file, "r") as f:
                self.memory_data = json.load(f)
            print(f"✓ Loaded memory-augmented ({len(self.memory_data.get('results', {}))} questions)")
        except Exception as e:
            print(f"❌ Error loading memory-augmented: {e}")
            return False

        return True

    def _extract_results(self, data: Dict, label: str) -> Dict:
        """Extract standardized results from either baseline or memory-augmented format."""
        results = {}

        # Handle different result formats
        for q_id, item in data.get("results", {}).items():
            # Get the result (may be a dict or a list)
            if isinstance(item, list) and len(item) > 0:
                item = item[0]
            elif isinstance(item, list):
                continue

            # Extract common fields
            correct_idx = item.get("correct_choice_index")
            predicted_idx = item.get("predicted_choice_index")

            # Handle tokens - could be nested in "tokens" dict
            if isinstance(item.get("tokens"), dict):
                tokens = item["tokens"]
                input_tokens = tokens.get("input_tokens", 0)
                output_tokens = tokens.get("output_tokens", 0)
                total_tokens = tokens.get("total_tokens", 0)
            else:
                input_tokens = item.get("tokens_input", item.get("input_tokens", 0))
                output_tokens = item.get("tokens_output", item.get("output_tokens", 0))
                total_tokens = item.get("tokens_total", item.get("total_tokens", 0))

            # Determine latency
            latency = item.get("latency_total", item.get("latency", 0))

            results[q_id] = {
                "is_correct": predicted_idx == correct_idx if predicted_idx is not None else False,
                "predicted": predicted_idx,
                "correct": correct_idx,
                "tokens": {
                    "input": input_tokens,
                    "output": output_tokens,
                    "total": total_tokens
                },
                "latency": latency,
                "question_type": item.get("question_type", "unknown"),
                "metadata": item.get("retrieval_metadata", {})  # Memory-augmented specific
            }

        return results

    def calculate_accuracy(self, results: Dict) -> Tuple[float, Dict]:
        """Calculate overall and per-type accuracy."""
        if not results:
            return 0.0, {}

        total = len(results)
        correct = sum(1 for r in results.values() if r["is_correct"])
        overall_acc = (correct / total * 100) if total > 0 else 0.0

        # Per-type breakdown
        by_type = {}
        for q_id, result in results.items():
            q_type = result["question_type"]
            if q_type not in by_type:
                by_type[q_type] = {"correct": 0, "total": 0}
            by_type[q_type]["total"] += 1
            if result["is_correct"]:
                by_type[q_type]["correct"] += 1

        per_type = {}
        for q_type, stats in by_type.items():
            per_type[q_type] = (stats["correct"] / stats["total"] * 100) if stats["total"] > 0 else 0.0

        return overall_acc, per_type

    def calculate_tokens(self, results: Dict) -> Tuple[int, int, int]:
        """Calculate total input, output, and total tokens."""
        input_total = sum(r["tokens"]["input"] for r in results.values())
        output_total = sum(r["tokens"]["output"] for r in results.values())
        total = sum(r["tokens"]["total"] for r in results.values())
        return input_total, output_total, total

    def calculate_latency(self, results: Dict) -> Tuple[float, float, float, float, float]:
        """Calculate latency statistics (mean, median, min, max, p95)."""
        if not results:
            return 0.0, 0.0, 0.0, 0.0, 0.0

        latencies = sorted([r["latency"] for r in results.values() if r["latency"] > 0])
        if not latencies:
            return 0.0, 0.0, 0.0, 0.0, 0.0

        mean = sum(latencies) / len(latencies)
        median = latencies[len(latencies) // 2]
        min_lat = min(latencies)
        max_lat = max(latencies)
        p95_idx = int(len(latencies) * 0.95)
        p95 = latencies[min(p95_idx, len(latencies) - 1)]

        return mean, median, min_lat, max_lat, p95

    def estimate_cost(self, input_tokens: int, output_tokens: int) -> float:
        """Estimate cost using DeepSeek pricing."""
        # DeepSeek pricing (v2024)
        input_cost = input_tokens * 0.014 / 1_000_000
        output_cost = output_tokens * 0.056 / 1_000_000
        return input_cost + output_cost

    def compare(self) -> Optional[ComparisonMetrics]:
        """Calculate comparison metrics."""
        # Load results
        if not self.load_results():
            return None

        if not self.baseline_data:
            print("No baseline data to compare")
            return None

        if not self.memory_data:
            print("⚠️  No memory-augmented data to compare (run memory_augmented.py first)")
            return None

        # Extract results in standard format
        baseline_results = self._extract_results(self.baseline_data, "baseline")
        memory_results = self._extract_results(self.memory_data, "memory")

        print(f"\nAnalyzing results...")
        print(f"Baseline: {len(baseline_results)} questions")
        print(f"Memory-Aug: {len(memory_results)} questions")

        # Calculate metrics
        baseline_acc, baseline_by_type = self.calculate_accuracy(baseline_results)
        memory_acc, memory_by_type = self.calculate_accuracy(memory_results)

        baseline_input, baseline_output, baseline_total = self.calculate_tokens(baseline_results)
        memory_input, memory_output, memory_total = self.calculate_tokens(memory_results)

        baseline_latency_mean, _, _, _, _ = self.calculate_latency(baseline_results)
        memory_latency_mean, _, _, _, _ = self.calculate_latency(memory_results)

        baseline_cost = self.estimate_cost(baseline_input, baseline_output)
        memory_cost = self.estimate_cost(memory_input, memory_output)

        # Calculate retrieval stats (from memory-augmented metadata)
        total_memories = 0
        total_token_reduction = 0
        count = 0

        for result in memory_results.values():
            if result.get("metadata"):
                total_memories += result["metadata"].get("num_memories_retrieved", 0)
                total_token_reduction += result["metadata"].get("token_reduction_pct", 0)
                count += 1

        avg_memories = total_memories / count if count > 0 else 0
        avg_token_reduction = total_token_reduction / count if count > 0 else 0

        return ComparisonMetrics(
            baseline_accuracy=baseline_acc,
            memory_accuracy=memory_acc,
            accuracy_delta=memory_acc - baseline_acc,
            baseline_tokens_total=baseline_total,
            memory_tokens_total=memory_total,
            token_reduction_abs=baseline_total - memory_total,
            token_reduction_pct=(baseline_total - memory_total) / baseline_total * 100 if baseline_total > 0 else 0,
            baseline_cost=baseline_cost,
            memory_cost=memory_cost,
            cost_savings=baseline_cost - memory_cost,
            cost_savings_pct=(baseline_cost - memory_cost) / baseline_cost * 100 if baseline_cost > 0 else 0,
            baseline_latency_mean=baseline_latency_mean,
            memory_latency_mean=memory_latency_mean,
            latency_improvement=baseline_latency_mean - memory_latency_mean,
            latency_improvement_pct=(baseline_latency_mean - memory_latency_mean) / baseline_latency_mean * 100
            if baseline_latency_mean > 0 else 0,
            avg_memories_retrieved=int(avg_memories),
            avg_token_reduction_per_q=avg_token_reduction
        )

    def print_comparison(self, metrics: ComparisonMetrics) -> None:
        """Print formatted comparison report."""
        print("\n" + "=" * 70)
        print("BASELINE vs MEMORY-AUGMENTED COMPARISON")
        print("=" * 70)

        print("\nAccuracy:")
        print(f"  Baseline:        {metrics.baseline_accuracy:6.1f}%")
        print(f"  Memory-Aug:      {metrics.memory_accuracy:6.1f}%")
        print(f"  Change:          {metrics.accuracy_delta:+6.1f}%")

        print("\nToken Usage:")
        print(f"  Baseline:        {metrics.baseline_tokens_total:,} tokens")
        print(f"  Memory-Aug:      {metrics.memory_tokens_total:,} tokens")
        print(f"  Reduction:       {metrics.token_reduction_abs:,} tokens ({metrics.token_reduction_pct:.1f}%)")

        print("\nCost Estimation (DeepSeek):")
        print(f"  Baseline:        ${metrics.baseline_cost:,.6f}")
        print(f"  Memory-Aug:      ${metrics.memory_cost:,.6f}")
        print(f"  Savings:         ${metrics.cost_savings:,.6f} ({metrics.cost_savings_pct:.1f}%)")

        print("\nLatency:")
        print(f"  Baseline Mean:   {metrics.baseline_latency_mean:6.3f}s")
        print(f"  Memory-Aug Mean: {metrics.memory_latency_mean:6.3f}s")
        if metrics.baseline_latency_mean > 0:
            print(f"  Improvement:     {metrics.latency_improvement:+6.3f}s ({metrics.latency_improvement_pct:+.1f}%)")

        print("\nRetrieval Statistics:")
        print(f"  Avg Memories Retrieved: {metrics.avg_memories_retrieved}")
        print(f"  Avg Token Reduction:    {metrics.avg_token_reduction_per_q:.1f}%")

        print("\n" + "=" * 70)

        # Show cost analysis for full dataset
        if metrics.baseline_tokens_total > 0:
            full_dataset_size = 1986
            baseline_full_cost = metrics.baseline_cost * (full_dataset_size / (metrics.baseline_tokens_total / 16690))
            memory_full_cost = metrics.memory_cost * (full_dataset_size / (metrics.memory_tokens_total / 2000))

            print("\nProjected Full Dataset Cost (1,986 questions):")
            print(f"  Baseline:        ${baseline_full_cost:,.2f}")
            print(f"  Memory-Aug:      ${memory_full_cost:,.2f}")
            print(f"  Savings:         ${baseline_full_cost - memory_full_cost:,.2f}")

    def save_comparison(self, metrics: ComparisonMetrics, output_path: str = "results/comparison_report.json") -> None:
        """Save comparison metrics to JSON."""
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)

        report = {
            "timestamp": datetime.now().isoformat(),
            "baseline_file": self.baseline_file,
            "memory_file": self.memory_file,
            "metrics": {
                "accuracy": {
                    "baseline": metrics.baseline_accuracy,
                    "memory_augmented": metrics.memory_accuracy,
                    "delta": metrics.accuracy_delta
                },
                "tokens": {
                    "baseline": metrics.baseline_tokens_total,
                    "memory_augmented": metrics.memory_tokens_total,
                    "reduction_absolute": metrics.token_reduction_abs,
                    "reduction_percent": metrics.token_reduction_pct
                },
                "cost": {
                    "baseline": metrics.baseline_cost,
                    "memory_augmented": metrics.memory_cost,
                    "savings": metrics.cost_savings,
                    "savings_percent": metrics.cost_savings_pct
                },
                "latency": {
                    "baseline_mean": metrics.baseline_latency_mean,
                    "memory_mean": metrics.memory_latency_mean,
                    "improvement": metrics.latency_improvement,
                    "improvement_percent": metrics.latency_improvement_pct
                },
                "retrieval": {
                    "avg_memories_retrieved": metrics.avg_memories_retrieved,
                    "avg_token_reduction_per_q": metrics.avg_token_reduction_per_q
                }
            }
        }

        with open(output_path, "w") as f:
            json.dump(report, f, indent=2)

        print(f"\n✓ Saved comparison report to {output_path}")


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(description="Compare baseline vs memory-augmented results")
    parser.add_argument(
        "--baseline",
        type=str,
        default="results/ultrathink_results.json",
        help="Path to baseline results"
    )
    parser.add_argument(
        "--memory",
        type=str,
        default="results/memory_augmented_results.json",
        help="Path to memory-augmented results"
    )
    parser.add_argument(
        "--output",
        type=str,
        default="results/comparison_report.json",
        help="Output path for comparison report"
    )

    args = parser.parse_args()

    # Run comparison
    evaluator = ComparisonEvaluator(
        baseline_file=args.baseline,
        memory_file=args.memory
    )

    metrics = evaluator.compare()

    if metrics:
        evaluator.print_comparison(metrics)
        evaluator.save_comparison(metrics, output_path=args.output)
    else:
        print("\n❌ Comparison failed - check file paths and ensure both files exist")
        sys.exit(1)


if __name__ == "__main__":
    main()
