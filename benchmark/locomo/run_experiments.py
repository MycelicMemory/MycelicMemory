"""
Experiment runner for LoCoMo benchmark on ultrathink.
Adapted from mem0ai/mem0 evaluation code.
"""

import argparse
import json
import os
import time
from collections import defaultdict
from typing import Dict, List, Optional

import requests
from openai import OpenAI
from tqdm import tqdm

from prompts import ANSWER_PROMPT_ULTRATHINK, ANSWER_PROMPT_SUMMARY

# DeepSeek API configuration
DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY", "sk-265369bfd7534590a7e02be4f1026fe4")
DEEPSEEK_BASE_URL = "https://api.deepseek.com"
DEEPSEEK_MODEL = "deepseek-chat"

# LoCoMo dataset URL
DATASET_URL = "https://huggingface.co/datasets/Percena/locomo-mc10/resolve/main/data/locomo_mc10.json"

# Question type mapping
CATEGORY_MAP = {
    "single_hop": "1",
    "multi_hop": "2",
    "temporal_reasoning": "3",
    "open_domain": "4",
    "adversarial": "5"
}


def download_dataset(output_path: str = "dataset/locomo10.json") -> str:
    """Download the LoCoMo-MC10 dataset from HuggingFace."""
    if os.path.exists(output_path):
        print(f"Dataset already exists at {output_path}")
        return output_path

    print(f"Downloading dataset from {DATASET_URL}...")
    response = requests.get(DATASET_URL)
    response.raise_for_status()

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w") as f:
        f.write(response.text)

    print(f"Dataset saved to {output_path}")
    return output_path


def load_dataset(path: str, max_questions: Optional[int] = None) -> List[Dict]:
    """Load LoCoMo dataset from JSONL file."""
    questions = []
    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                q = json.loads(line)
                questions.append(q)
                if max_questions and len(questions) >= max_questions:
                    break
            except json.JSONDecodeError as e:
                print(f"Error parsing line: {e}")
                continue

    print(f"Loaded {len(questions)} questions")
    return questions


def build_context_from_sessions(question_data: Dict, use_summaries: bool = False) -> str:
    """Build context string from session data."""
    if use_summaries:
        summaries = question_data.get("haystack_session_summaries", [])
        session_ids = question_data.get("haystack_session_ids", [])
        datetimes = question_data.get("haystack_session_datetimes", [])

        context_parts = []
        for i, summary in enumerate(summaries):
            if summary:
                sid = session_ids[i] if i < len(session_ids) else f"Session {i}"
                dt = datetimes[i] if i < len(datetimes) else ""
                context_parts.append(f"[{sid} - {dt}]\n{summary}")

        return "\n\n".join(context_parts)
    else:
        # Use full dialogue turns
        sessions = question_data.get("haystack_sessions", [])
        context_parts = []

        for session_idx, session in enumerate(sessions):
            if session_idx > 0:
                context_parts.append("--- New Session ---")
            for turn in session:
                if isinstance(turn, dict):
                    content = turn.get("content", "")
                    role = turn.get("role", "")
                    if content:
                        context_parts.append(f"{role}: {content}" if role else content)
                elif isinstance(turn, str):
                    context_parts.append(turn)

        return "\n".join(context_parts)


def format_choices(choices: List[str]) -> str:
    """Format answer choices as numbered list for the prompt."""
    formatted = []
    for i, choice in enumerate(choices):
        formatted.append(f"{i}. {choice}")
    return "\n".join(formatted)


def extract_choice_index(response: str) -> Optional[int]:
    """Extract choice index (0-9) from LLM response."""
    response = response.strip()
    # Try to find a number 0-9 in the response
    import re
    match = re.search(r'\b([0-9])\b', response)
    if match:
        return int(match.group(1))
    return None


class UltrathinkExperiment:
    """Run LoCoMo benchmark experiments using ultrathink memory system."""

    def __init__(
        self,
        dataset_path: str = "dataset/locomo10.json",
        use_summaries: bool = False,
        max_questions: Optional[int] = None,
    ):
        self.dataset_path = dataset_path
        self.use_summaries = use_summaries
        self.max_questions = max_questions

        self.client = OpenAI(
            api_key=DEEPSEEK_API_KEY,
            base_url=DEEPSEEK_BASE_URL,
        )

        # Load dataset
        self.questions = load_dataset(dataset_path, max_questions)

    def generate_answer(self, question: str, context: str, choices: List[str]) -> Optional[int]:
        """Generate answer choice using DeepSeek with the provided context."""
        choices_str = format_choices(choices)

        if self.use_summaries:
            prompt = ANSWER_PROMPT_SUMMARY.format(
                summaries=context,
                question=question,
                choices=choices_str
            )
        else:
            prompt = ANSWER_PROMPT_ULTRATHINK.format(
                memories=context,
                question=question,
                choices=choices_str
            )

        try:
            response = self.client.chat.completions.create(
                model=DEEPSEEK_MODEL,
                messages=[{"role": "user", "content": prompt}],
                temperature=0.0,
                max_tokens=10,
            )
            response_text = response.choices[0].message.content.strip()
            choice_index = extract_choice_index(response_text)
            return choice_index
        except Exception as e:
            print(f"Error generating answer: {e}")
            return None

    def run(self, output_path: str = "results/ultrathink_results.json"):
        """Run the benchmark experiment."""
        os.makedirs(os.path.dirname(output_path), exist_ok=True)

        results = defaultdict(list)
        total_time = 0

        for q in tqdm(self.questions, desc="Processing questions"):
            question_id = q.get("question_id", "unknown")
            question_text = q.get("question", "")
            gold_answer = q.get("answer", "")
            question_type = q.get("question_type", "unknown")
            category = CATEGORY_MAP.get(question_type, "0")

            # Parse MC fields
            choices = q.get("choices", [])
            correct_choice_index = q.get("correct_choice_index", -1)

            # Build context from session data
            context = build_context_from_sessions(q, self.use_summaries)

            # Generate answer (returns choice index 0-9)
            start_time = time.time()
            predicted_choice_index = self.generate_answer(question_text, context, choices)
            elapsed = time.time() - start_time
            total_time += elapsed

            # Store result with both predicted and correct indices
            results[question_id].append({
                "question": question_text,
                "gold_answer": gold_answer,
                "choices": choices,
                "correct_choice_index": correct_choice_index,
                "predicted_choice_index": predicted_choice_index,
                "category": category,
                "question_type": question_type,
                "latency": elapsed,
            })

        # Save results
        with open(output_path, "w") as f:
            json.dump(results, f, indent=2)

        print(f"\nResults saved to {output_path}")
        print(f"Total questions: {len(self.questions)}")
        print(f"Total time: {total_time:.2f}s")
        print(f"Average latency: {total_time / len(self.questions):.2f}s per question")

        return results


def main():
    parser = argparse.ArgumentParser(description="Run LoCoMo benchmark experiments")
    parser.add_argument(
        "--dataset_path",
        type=str,
        default="dataset/locomo10.json",
        help="Path to the dataset file"
    )
    parser.add_argument(
        "--output_path",
        type=str,
        default="results/ultrathink_results.json",
        help="Path to save results"
    )
    parser.add_argument(
        "--use_summaries",
        action="store_true",
        default=False,
        help="Use session summaries instead of full dialogues"
    )
    parser.add_argument(
        "--max_questions",
        type=int,
        default=None,
        help="Maximum number of questions to process (for testing)"
    )
    parser.add_argument(
        "--download",
        action="store_true",
        default=False,
        help="Download the dataset if not present"
    )

    args = parser.parse_args()

    # Download dataset if requested
    if args.download or not os.path.exists(args.dataset_path):
        download_dataset(args.dataset_path)

    # Run experiment
    experiment = UltrathinkExperiment(
        dataset_path=args.dataset_path,
        use_summaries=args.use_summaries,
        max_questions=args.max_questions,
    )
    experiment.run(args.output_path)


if __name__ == "__main__":
    main()
