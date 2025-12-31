"""
Main evaluation script for LoCoMo benchmark.
Adapted from mem0ai/mem0 evaluation code.
"""

import argparse
import concurrent.futures
import json
import threading
from collections import defaultdict

from metrics.llm_judge import evaluate_llm_judge
from metrics.utils import calculate_bleu_scores, calculate_metrics
from tqdm import tqdm


def process_item(item_data):
    """Process a single conversation's questions and evaluate responses."""
    k, v = item_data
    local_results = defaultdict(list)

    for item in v:
        gt_answer = str(item["answer"])
        pred_answer = str(item["response"])
        category = str(item["category"])
        question = str(item["question"])

        # Skip category 5 (adversarial - optional)
        if category == "5":
            continue

        metrics = calculate_metrics(pred_answer, gt_answer)
        bleu_scores = calculate_bleu_scores(pred_answer, gt_answer)
        llm_score = evaluate_llm_judge(question, gt_answer, pred_answer)

        local_results[k].append(
            {
                "question": question,
                "answer": gt_answer,
                "response": pred_answer,
                "category": category,
                "bleu_score": bleu_scores["bleu1"],
                "f1_score": metrics["f1"],
                "llm_score": llm_score,
            }
        )

    return local_results


def main():
    parser = argparse.ArgumentParser(description="Evaluate LoCoMo benchmark results")
    parser.add_argument(
        "--input_file",
        type=str,
        default="results/ultrathink_results.json",
        help="Path to the input results file"
    )
    parser.add_argument(
        "--output_file",
        type=str,
        default="results/evaluation_metrics.json",
        help="Path to save the evaluation results"
    )
    parser.add_argument(
        "--max_workers",
        type=int,
        default=10,
        help="Maximum number of worker threads"
    )

    args = parser.parse_args()

    with open(args.input_file, "r") as f:
        data = json.load(f)

    results = defaultdict(list)
    results_lock = threading.Lock()

    # Use ThreadPoolExecutor with specified workers
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.max_workers) as executor:
        futures = [executor.submit(process_item, item_data) for item_data in data.items()]

        for future in tqdm(concurrent.futures.as_completed(futures), total=len(futures)):
            local_results = future.result()
            with results_lock:
                for k, items in local_results.items():
                    results[k].extend(items)

    # Save results to JSON file
    with open(args.output_file, "w") as f:
        json.dump(results, f, indent=4)

    print(f"Results saved to {args.output_file}")

    # Print summary
    all_llm_scores = []
    all_f1_scores = []
    all_bleu_scores = []
    by_category = defaultdict(lambda: {"llm": [], "f1": [], "bleu": []})

    for k, items in results.items():
        for item in items:
            all_llm_scores.append(item["llm_score"])
            all_f1_scores.append(item["f1_score"])
            all_bleu_scores.append(item["bleu_score"])
            cat = item["category"]
            by_category[cat]["llm"].append(item["llm_score"])
            by_category[cat]["f1"].append(item["f1_score"])
            by_category[cat]["bleu"].append(item["bleu_score"])

    print("\n" + "=" * 50)
    print("OVERALL RESULTS")
    print("=" * 50)
    if all_llm_scores:
        print(f"LLM Judge Accuracy: {sum(all_llm_scores) / len(all_llm_scores) * 100:.2f}%")
        print(f"F1 Score: {sum(all_f1_scores) / len(all_f1_scores):.4f}")
        print(f"BLEU-1 Score: {sum(all_bleu_scores) / len(all_bleu_scores):.4f}")

    print("\n" + "=" * 50)
    print("RESULTS BY CATEGORY")
    print("=" * 50)
    category_names = {
        "1": "Single-Hop",
        "2": "Multi-Hop",
        "3": "Temporal",
        "4": "Open-Domain",
        "5": "Adversarial"
    }
    for cat in sorted(by_category.keys()):
        cat_name = category_names.get(cat, f"Category {cat}")
        llm_scores = by_category[cat]["llm"]
        f1_scores = by_category[cat]["f1"]
        if llm_scores:
            print(f"\n{cat_name}:")
            print(f"  LLM Judge: {sum(llm_scores) / len(llm_scores) * 100:.2f}% ({sum(llm_scores)}/{len(llm_scores)})")
            print(f"  F1 Score: {sum(f1_scores) / len(f1_scores):.4f}")


if __name__ == "__main__":
    main()
