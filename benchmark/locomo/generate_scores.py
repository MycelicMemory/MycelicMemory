"""
Generate aggregate scores from evaluation results.
Adapted from mem0ai/mem0 evaluation code.
"""

import argparse
import json

import pandas as pd


def main():
    parser = argparse.ArgumentParser(description="Generate scores from evaluation results")
    parser.add_argument(
        "--input_file",
        type=str,
        default="results/evaluation_metrics.json",
        help="Path to evaluation metrics file"
    )
    parser.add_argument(
        "--output_file",
        type=str,
        default="results/scores_summary.json",
        help="Path to save scores summary"
    )

    args = parser.parse_args()

    # Load the evaluation metrics data
    with open(args.input_file, "r") as f:
        data = json.load(f)

    # Flatten the data into a list of question items
    all_items = []
    for key in data:
        all_items.extend(data[key])

    if not all_items:
        print("No items found in evaluation data")
        return

    # Convert to DataFrame
    df = pd.DataFrame(all_items)

    # Convert category to numeric type
    df["category"] = pd.to_numeric(df["category"])

    # Calculate mean scores by category
    result = df.groupby("category").agg({
        "bleu_score": "mean",
        "f1_score": "mean",
        "llm_score": "mean"
    }).round(4)

    # Add count of questions per category
    result["count"] = df.groupby("category").size()

    # Category names
    category_names = {
        1: "Single-Hop",
        2: "Multi-Hop",
        3: "Temporal",
        4: "Open-Domain",
        5: "Adversarial"
    }

    print("\n" + "=" * 60)
    print("MEAN SCORES PER CATEGORY")
    print("=" * 60)
    print(f"{'Category':<20} {'LLM Judge':<12} {'F1':<10} {'BLEU-1':<10} {'Count':<8}")
    print("-" * 60)

    for cat in sorted(result.index):
        cat_name = category_names.get(cat, f"Cat {cat}")
        row = result.loc[cat]
        llm_pct = row["llm_score"] * 100
        print(f"{cat_name:<20} {llm_pct:>10.2f}% {row['f1_score']:>9.4f} {row['bleu_score']:>9.4f} {int(row['count']):>7}")

    # Calculate overall means
    overall_means = df.agg({
        "bleu_score": "mean",
        "f1_score": "mean",
        "llm_score": "mean"
    }).round(4)

    print("\n" + "=" * 60)
    print("OVERALL MEAN SCORES")
    print("=" * 60)
    print(f"LLM Judge Accuracy: {overall_means['llm_score'] * 100:.2f}%")
    print(f"F1 Score: {overall_means['f1_score']:.4f}")
    print(f"BLEU-1 Score: {overall_means['bleu_score']:.4f}")
    print(f"Total Questions: {len(all_items)}")

    # Save summary to JSON
    summary = {
        "overall": {
            "llm_judge_accuracy": round(float(overall_means["llm_score"]) * 100, 2),
            "f1_score": round(float(overall_means["f1_score"]), 4),
            "bleu1_score": round(float(overall_means["bleu_score"]), 4),
            "total_questions": len(all_items),
        },
        "by_category": {}
    }

    for cat in sorted(result.index):
        cat_name = category_names.get(cat, f"category_{cat}")
        row = result.loc[cat]
        summary["by_category"][cat_name] = {
            "llm_judge_accuracy": round(float(row["llm_score"]) * 100, 2),
            "f1_score": round(float(row["f1_score"]), 4),
            "bleu1_score": round(float(row["bleu_score"]), 4),
            "count": int(row["count"]),
        }

    with open(args.output_file, "w") as f:
        json.dump(summary, f, indent=2)

    print(f"\nScores summary saved to {args.output_file}")


if __name__ == "__main__":
    main()
