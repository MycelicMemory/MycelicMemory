"""
Evaluation script for LoCoMo-MC10 benchmark.
Evaluates multiple-choice predictions using simple accuracy metrics.
"""

import argparse
import json
from collections import defaultdict
from typing import Dict, List, Tuple


def evaluate_mc_results(results: Dict) -> Dict:
    """
    Evaluate multiple-choice results by comparing predicted vs correct choice indices.

    Args:
        results: Dictionary of question results from run_experiments.py

    Returns:
        Dictionary containing accuracy metrics overall and per question type
    """
    metrics = {
        "overall": {
            "total": 0,
            "correct": 0,
            "accuracy": 0.0,
            "by_type": {}
        }
    }

    type_metrics = defaultdict(lambda: {"total": 0, "correct": 0})

    # Process all results
    total_questions = 0
    total_correct = 0

    for question_id, items in results.items():
        for item in items:
            predicted_index = item.get("predicted_choice_index")
            correct_index = item.get("correct_choice_index")
            question_type = item.get("question_type", "unknown")

            # Handle missing predictions (None returned from LLM)
            if predicted_index is None:
                predicted_index = -1

            total_questions += 1
            type_metrics[question_type]["total"] += 1

            # Check if prediction matches ground truth
            if predicted_index == correct_index:
                total_correct += 1
                type_metrics[question_type]["correct"] += 1

    # Calculate overall accuracy
    metrics["overall"]["total"] = total_questions
    metrics["overall"]["correct"] = total_correct
    metrics["overall"]["accuracy"] = (
        (total_correct / total_questions * 100) if total_questions > 0 else 0.0
    )

    # Calculate per-type accuracy
    for question_type, counts in sorted(type_metrics.items()):
        accuracy = (
            (counts["correct"] / counts["total"] * 100)
            if counts["total"] > 0 else 0.0
        )
        metrics["overall"]["by_type"][question_type] = {
            "total": counts["total"],
            "correct": counts["correct"],
            "accuracy": accuracy
        }

    return metrics


def main():
    parser = argparse.ArgumentParser(
        description="Evaluate LoCoMo-MC10 benchmark results"
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
        default="results/evaluation_metrics.json",
        help="Path to save the evaluation results"
    )

    args = parser.parse_args()

    # Load results
    with open(args.input_file, "r") as f:
        data = json.load(f)

    # Evaluate
    metrics = evaluate_mc_results(data)

    # Save metrics
    with open(args.output_file, "w") as f:
        json.dump(metrics, f, indent=2)

    print(f"Evaluation metrics saved to {args.output_file}")

    # Print summary
    print("\n" + "=" * 60)
    print("OVERALL RESULTS")
    print("=" * 60)
    overall = metrics["overall"]
    print(f"Total Questions: {overall['total']}")
    print(f"Correct Predictions: {overall['correct']}")
    print(f"Accuracy: {overall['accuracy']:.2f}%")

    print("\n" + "=" * 60)
    print("RESULTS BY QUESTION TYPE")
    print("=" * 60)

    type_names = {
        "single_hop": "Single-Hop",
        "multi_hop": "Multi-Hop",
        "temporal_reasoning": "Temporal",
        "open_domain": "Open-Domain",
        "adversarial": "Adversarial"
    }

    if overall["by_type"]:
        for qtype in sorted(overall["by_type"].keys()):
            type_name = type_names.get(qtype, qtype)
            type_data = overall["by_type"][qtype]
            print(
                f"\n{type_name}:"
                f"\n  Total: {type_data['total']}"
                f"\n  Correct: {type_data['correct']}"
                f"\n  Accuracy: {type_data['accuracy']:.2f}%"
            )


if __name__ == "__main__":
    main()
