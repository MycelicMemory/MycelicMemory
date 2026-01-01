"""
Comprehensive evaluation script with detailed metrics for LoCoMo-MC10 benchmark.
Generates detailed reports on accuracy, latency, token usage, and cost.
"""

import argparse
import json
from pathlib import Path
from metrics_tracker import MetricsTracker, TokenMetrics


def load_results(input_file: str) -> dict:
    """Load results from benchmark run."""
    with open(input_file, "r") as f:
        return json.load(f)


def build_metrics_from_results(results: dict) -> MetricsTracker:
    """
    Reconstruct MetricsTracker from saved results.
    This allows us to generate comprehensive metrics from any benchmark run.
    """
    tracker = MetricsTracker()

    for question_id, items in results.items():
        for item in items:
            # Extract token metrics
            tokens_data = item.get("tokens", {})
            token_metrics = TokenMetrics(
                input_tokens=tokens_data.get("input_tokens", 0),
                output_tokens=tokens_data.get("output_tokens", 0),
                total_tokens=tokens_data.get("total_tokens", 0),
            )

            # Add result to tracker
            tracker.add_result(
                question_id=question_id,
                question_text=item.get("question", ""),
                question_type=item.get("question_type", "unknown"),
                correct_choice_index=item.get("correct_choice_index", -1),
                predicted_choice_index=item.get("predicted_choice_index"),
                latency=item.get("latency_total", 0.0),
                tokens=token_metrics,
                llm_response_time=item.get("latency_llm_response", 0.0),
                context_building_time=item.get("latency_context_building", 0.0),
            )

    return tracker


def print_metrics_summary(tracker: MetricsTracker) -> None:
    """Print comprehensive metrics summary."""
    overall_metrics = tracker.get_overall_metrics()
    per_type_metrics = tracker.get_per_type_metrics()
    error_analysis = tracker.get_error_analysis()

    print("\n" + "=" * 70)
    print("LOCOM0-MC10 BENCHMARK RESULTS - COMPREHENSIVE METRICS")
    print("=" * 70)

    # Overall metrics
    print("\n📊 OVERALL ACCURACY")
    print("-" * 70)
    print(f"Total Questions: {overall_metrics['overall']['total_questions']}")
    print(f"Correct Predictions: {overall_metrics['overall']['correct_predictions']}")
    print(f"Accuracy: {overall_metrics['overall']['accuracy']:.2f}%")
    print(f"Error Rate: {error_analysis['error_rate']:.2f}%")

    # Latency metrics
    print("\n⏱️  LATENCY METRICS (seconds)")
    print("-" * 70)
    latency = overall_metrics["latency"]
    print(f"Total Latency: {latency['total_latency_seconds']:.2f}s")
    print(f"Mean: {latency['mean_latency_seconds']:.3f}s")
    print(f"Median: {latency['median_latency_seconds']:.3f}s")
    print(f"P95: {latency['p95_latency_seconds']:.3f}s")
    print(f"P99: {latency['p99_latency_seconds']:.3f}s")
    print(f"Min: {latency['min_latency_seconds']:.3f}s")
    print(f"Max: {latency['max_latency_seconds']:.3f}s")
    print(f"StdDev: {latency['stdev_latency_seconds']:.3f}s")

    # Latency breakdown
    if overall_metrics.get("llm_latency"):
        print("\n⏱️  LLM LATENCY BREAKDOWN (seconds)")
        print("-" * 70)
        llm_latency = overall_metrics["llm_latency"]
        print(f"Mean LLM Response: {llm_latency['mean_llm_response_seconds']:.3f}s")
        print(f"Median LLM Response: {llm_latency['median_llm_response_seconds']:.3f}s")
        print(f"Total LLM Time: {llm_latency['total_llm_time_seconds']:.2f}s")

    if overall_metrics.get("context_latency"):
        print("\n⏱️  CONTEXT BUILDING LATENCY (seconds)")
        print("-" * 70)
        ctx_latency = overall_metrics["context_latency"]
        print(f"Mean Context Building: {ctx_latency['mean_context_building_seconds']:.3f}s")
        print(f"Total Context Building: {ctx_latency['total_context_building_seconds']:.2f}s")

    # Token metrics
    print("\n📝 TOKEN USAGE METRICS")
    print("-" * 70)
    tokens = overall_metrics["tokens"]
    print(f"Total Input Tokens: {tokens['total_input_tokens']:,}")
    print(f"Total Output Tokens: {tokens['total_output_tokens']:,}")
    print(f"Total Tokens: {tokens['total_tokens']:,}")
    print(f"Mean Input Tokens/Question: {tokens['mean_input_tokens']:.1f}")
    print(f"Mean Output Tokens/Question: {tokens['mean_output_tokens']:.1f}")
    print(f"Mean Total Tokens/Question: {tokens['mean_total_tokens']:.1f}")

    # Cost estimation
    print("\n💰 COST ESTIMATION (DeepSeek API)")
    print("-" * 70)
    cost = overall_metrics["cost_estimation"]
    print(f"Input Cost: ${cost['input_cost_usd']:.6f}")
    print(f"Output Cost: ${cost['output_cost_usd']:.6f}")
    print(f"Total Cost: ${cost['total_cost_usd']:.6f}")
    print(f"Cost per Question: ${cost['cost_per_question_usd']:.6f}")

    # Per-type breakdown (category names match LoCoMo benchmark)
    print("\n🎯 ACCURACY BY QUESTION TYPE")
    print("-" * 70)
    type_names = {
        "multi_hop": "Multi-Hop (Category 1 - Aggregates multiple evidence pieces)",
        "single_hop": "Single-Hop (Category 2 - Direct fact recall)",
        "temporal": "Temporal (Category 3 - Date/Time reasoning)",
        "temporal_reasoning": "Temporal (Category 3 - Date/Time reasoning)",  # Alias
        "open_domain": "Open-Domain (Category 4 - Conversational context)",
        "adversarial": "Adversarial (Category 5 - Robustness testing)",
    }

    for qtype in sorted(per_type_metrics.keys()):
        type_name = type_names.get(qtype, qtype)
        metrics = per_type_metrics[qtype]
        print(f"\n{type_name}")
        print(f"  Total: {metrics['total']}")
        print(f"  Correct: {metrics['correct']}")
        print(f"  Accuracy: {metrics['accuracy']:.2f}%")
        if metrics.get("latency"):
            print(f"  Mean Latency: {metrics['latency']['mean_seconds']:.3f}s")
            print(f"  P95 Latency: {metrics['latency']['p95_seconds']:.3f}s")
        if metrics.get("tokens"):
            print(f"  Mean Tokens/Q: {metrics['tokens']['mean_input'] + metrics['tokens']['mean_output']:.1f}")

    # Error analysis
    if error_analysis.get("errors_by_type"):
        print("\n❌ ERROR BREAKDOWN BY TYPE")
        print("-" * 70)
        for qtype, errors in sorted(error_analysis["errors_by_type"].items()):
            print(f"{type_names.get(qtype, qtype)}: {len(errors)} errors")


def save_metrics_report(tracker: MetricsTracker, output_file: str) -> None:
    """Save comprehensive metrics to JSON file."""
    metrics = {
        "overall": tracker.get_overall_metrics(),
        "per_type": tracker.get_per_type_metrics(),
        "error_analysis": tracker.get_error_analysis(),
    }

    with open(output_file, "w") as f:
        json.dump(metrics, f, indent=2)

    print(f"\nMetrics saved to {output_file}")


def main():
    parser = argparse.ArgumentParser(
        description="Generate comprehensive evaluation metrics for LoCoMo-MC10 benchmark"
    )
    parser.add_argument(
        "--input_file",
        type=str,
        default="results/ultrathink_results.json",
        help="Path to the input results file from run_experiments.py"
    )
    parser.add_argument(
        "--output_file",
        type=str,
        default="results/comprehensive_metrics.json",
        help="Path to save the comprehensive metrics report"
    )

    args = parser.parse_args()

    # Load results and build metrics
    results = load_results(args.input_file)
    tracker = build_metrics_from_results(results)

    # Save and display metrics
    save_metrics_report(tracker, args.output_file)
    print_metrics_summary(tracker)


if __name__ == "__main__":
    main()
