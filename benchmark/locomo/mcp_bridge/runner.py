"""
Benchmark Runner for MCP Bridge.

Wraps the existing benchmark evaluation code with progress callbacks
and structured result output.
"""

import json
import os
import sys
import threading
from collections import defaultdict
from typing import Any, Callable, Dict, List, Optional

# Add parent directory for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from run_experiments import download_dataset, load_dataset, build_context_from_sessions
from run_experiments import DEEPSEEK_API_KEY, DEEPSEEK_BASE_URL, DEEPSEEK_MODEL
from prompts import ANSWER_PROMPT_ULTRATHINK
from metrics.llm_judge import evaluate_llm_judge
from metrics.utils import calculate_bleu_scores, calculate_metrics

from openai import OpenAI


# Category mapping for the dataset
CATEGORY_MAP = {
    "single_hop": "1",
    "multi_hop": "2",
    "temporal_reasoning": "3",
    "open_domain": "4",
    "adversarial": "5"
}

CATEGORY_NAMES = {
    "1": "Single-Hop",
    "2": "Multi-Hop",
    "3": "Temporal",
    "4": "Open-Domain",
    "5": "Adversarial"
}


class BenchmarkRunner:
    """Runs LoCoMo benchmark with progress callbacks."""

    def __init__(self, dataset_path: str = None):
        self.dataset_path = dataset_path or os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "dataset", "locomo10.json"
        )
        self.client = OpenAI(
            api_key=DEEPSEEK_API_KEY,
            base_url=DEEPSEEK_BASE_URL
        )

    def generate_answer(self, question: str, context: str) -> str:
        """Generate an answer using DeepSeek."""
        prompt = ANSWER_PROMPT_ULTRATHINK.format(
            memories=context,
            question=question
        )

        try:
            response = self.client.chat.completions.create(
                model=DEEPSEEK_MODEL,
                messages=[{"role": "user", "content": prompt}],
                temperature=0.0,
                max_tokens=100,
            )
            return response.choices[0].message.content.strip()
        except Exception as e:
            print(f"Error generating answer: {e}")
            return ""

    def run(
        self,
        max_questions: int = 0,
        categories: List[str] = None,
        verbose: bool = False,
        on_progress: Callable[[int, int, str], None] = None
    ) -> Dict[str, Any]:
        """
        Run the benchmark.

        Args:
            max_questions: Maximum questions to process (0 = all)
            categories: Filter to specific categories
            verbose: Enable verbose output
            on_progress: Callback(current, total, question_text)

        Returns:
            Results dict with overall and by_category scores
        """
        # Ensure dataset exists
        if not os.path.exists(self.dataset_path):
            download_dataset(self.dataset_path)

        # Load questions
        questions = load_dataset(self.dataset_path, max_questions if max_questions > 0 else None)

        # Filter by category if specified
        if categories:
            cat_ids = set()
            for cat in categories:
                if cat in CATEGORY_MAP:
                    cat_ids.add(CATEGORY_MAP[cat])
                elif cat in CATEGORY_NAMES:
                    cat_ids.add(cat)
            if cat_ids:
                questions = [q for q in questions if CATEGORY_MAP.get(q.get("question_type"), "0") in cat_ids]

        total = len(questions)
        if verbose:
            print(f"Running benchmark on {total} questions")

        # Process each question
        results_by_id = defaultdict(list)
        all_results = []

        for idx, q in enumerate(questions):
            question_id = q.get("question_id", f"q{idx}")
            question_text = q.get("question", "")
            gold_answer = q.get("answer", "")
            question_type = q.get("question_type", "unknown")
            category = CATEGORY_MAP.get(question_type, "0")

            # Skip adversarial (category 5) by default
            if category == "5":
                continue

            # Report progress
            if on_progress:
                on_progress(idx + 1, total, question_text[:50] + "...")

            # Build context
            context = build_context_from_sessions(q)

            # Generate answer
            generated_answer = self.generate_answer(question_text, context)

            # Evaluate
            metrics = calculate_metrics(generated_answer, gold_answer)
            bleu_scores = calculate_bleu_scores(generated_answer, gold_answer)
            llm_score = evaluate_llm_judge(question_text, gold_answer, generated_answer)

            result = {
                "id": question_id,
                "category": CATEGORY_NAMES.get(category, f"Category {category}"),
                "question": question_text,
                "gold_answer": gold_answer,
                "generated_answer": generated_answer,
                "llm_judge_label": llm_score,
                "f1_score": metrics["f1"],
                "bleu1_score": bleu_scores["bleu1"]
            }

            results_by_id[question_id].append(result)
            all_results.append(result)

            if verbose:
                status = "✓" if llm_score == 1 else "✗"
                print(f"  [{idx+1}/{total}] {status} {question_text[:40]}...")

        # Aggregate results
        overall_llm = [r["llm_judge_label"] for r in all_results]
        overall_f1 = [r["f1_score"] for r in all_results]
        overall_bleu = [r["bleu1_score"] for r in all_results]

        by_category = defaultdict(lambda: {"llm": [], "f1": [], "bleu": []})
        for r in all_results:
            cat = r["category"]
            by_category[cat]["llm"].append(r["llm_judge_label"])
            by_category[cat]["f1"].append(r["f1_score"])
            by_category[cat]["bleu"].append(r["bleu1_score"])

        # Build output
        output = {
            "overall": {
                "llm_judge_accuracy": (sum(overall_llm) / len(overall_llm) * 100) if overall_llm else 0,
                "f1_score": sum(overall_f1) / len(overall_f1) if overall_f1 else 0,
                "bleu1_score": sum(overall_bleu) / len(overall_bleu) if overall_bleu else 0,
                "total_questions": len(all_results)
            },
            "by_category": {},
            "questions": all_results
        }

        for cat, scores in by_category.items():
            output["by_category"][cat] = {
                "llm_judge_accuracy": (sum(scores["llm"]) / len(scores["llm"]) * 100) if scores["llm"] else 0,
                "f1_score": sum(scores["f1"]) / len(scores["f1"]) if scores["f1"] else 0,
                "bleu1_score": sum(scores["bleu"]) / len(scores["bleu"]) if scores["bleu"] else 0,
                "count": len(scores["llm"])
            }

        if verbose:
            print(f"\nOverall LLM Judge Accuracy: {output['overall']['llm_judge_accuracy']:.1f}%")
            for cat, scores in output["by_category"].items():
                print(f"  {cat}: {scores['llm_judge_accuracy']:.1f}% ({scores['count']} questions)")

        return output


if __name__ == "__main__":
    # Quick test
    runner = BenchmarkRunner()
    results = runner.run(max_questions=5, verbose=True)
    print(json.dumps(results["overall"], indent=2))
