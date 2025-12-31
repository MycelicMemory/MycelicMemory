"""
Simple Ultrathink-Augmented LoCoMo Benchmark Demo

Demonstrates using ultrathink memory system with the actual LoCoMo dataset.
"""

import json
import time
import os
import re
import requests
from typing import Dict, List, Tuple
from datetime import datetime


DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY", "sk-265369bfd7534590a7e02be4f1026fe4")


class UltrathinkBenchmarkDemo:
    """Demonstrate ultrathink-augmented benchmark."""

    def __init__(self, dataset_path: str = "data/locomo10.json", num_questions: int = 10):
        self.dataset_path = dataset_path
        self.num_questions = num_questions
        self.deepseek_api_key = DEEPSEEK_API_KEY
        self.results = []

    def load_dataset(self) -> List[Tuple[str, str, str]]:
        """Load dataset and extract questions/answers."""
        questions = []

        with open(self.dataset_path) as f:
            data = json.load(f)

        count = 0
        for obj_idx, obj in enumerate(data):
            for qa in obj.get("qa", []):
                if count >= self.num_questions:
                    return questions

                question = qa.get("question", "")
                answer = qa.get("answer", "")
                category = str(qa.get("category", "unknown"))

                if question:
                    # Get context from the conversation dict
                    conversation = obj.get("conversation", {})
                    context_parts = []
                    for key, val in conversation.items():
                        if isinstance(val, str) and len(val) < 1000:
                            context_parts.append(val)

                    context = " ".join(context_parts[:3])  # First 3 conversation parts
                    questions.append((question, str(answer), context))
                    count += 1

        return questions

    def store_context(self, context: str, session_id: str) -> bool:
        """Store context in ultrathink using REST API."""
        try:
            from ultrathink_client import UltrathinkClient

            client = UltrathinkClient()
            messages = [{"role": "context", "content": context}]
            ids, elapsed = client.ingest_conversation(messages, session_id)
            return len(ids) > 0
        except Exception as e:
            print(f"⚠️  Context storage failed: {e}")
            return False

    def retrieve_context(self, query: str) -> str:
        """Retrieve relevant context from ultrathink."""
        try:
            from ultrathink_client import UltrathinkClient

            client = UltrathinkClient()
            results, elapsed = client.retrieve_memories(query, top_k=5)

            context_parts = []
            for result in results:
                if result.memory.content:
                    context_parts.append(result.memory.content)

            return "\n".join(context_parts)
        except Exception as e:
            print(f"⚠️  Retrieval failed: {e}")
            return ""

    def generate_answer(self, question: str, context: str) -> Tuple[str, Dict]:
        """Generate answer using DeepSeek."""
        prompt = f"""Answer the following question based on the provided context. Keep the answer concise.

CONTEXT:
{context}

QUESTION: {question}

ANSWER:"""

        start_time = time.time()

        headers = {
            "Authorization": f"Bearer {self.deepseek_api_key}",
            "Content-Type": "application/json"
        }

        payload = {
            "model": "deepseek-chat",
            "messages": [{"role": "user", "content": prompt}],
            "temperature": 0.3,
            "max_tokens": 50,
            "top_p": 1
        }

        try:
            response = requests.post(
                "https://api.deepseek.com/chat/completions",
                headers=headers,
                json=payload,
                timeout=30
            )
            response.raise_for_status()
            data = response.json()

            answer = data["choices"][0]["message"]["content"].strip()
            elapsed = time.time() - start_time

            return answer, {
                "tokens": data["usage"]["total_tokens"],
                "latency": elapsed
            }
        except Exception as e:
            print(f"❌ LLM error: {e}")
            return "", {"error": str(e)}

    def run(self, output_path: str = "results/ultrathink_demo_results.json"):
        """Run benchmark demo."""
        print("\n" + "="*70)
        print("ULTRATHINK-AUGMENTED LoCoMo BENCHMARK DEMO")
        print("="*70)

        questions = self.load_dataset()
        print(f"✓ Loaded {len(questions)} questions\n")

        correct = 0
        total_tokens = 0
        total_time = 0
        start_time = time.time()

        for idx, (question, expected_answer, context) in enumerate(questions):
            q_id = f"q_{idx}"
            print(f"[{idx+1}/{len(questions)}] {question[:60]}...")

            # Store context in ultrathink
            session_id = f"demo-q{idx}"
            self.store_context(context, session_id)

            # Retrieve enhanced context
            retrieved_context = self.retrieve_context(question)
            enhanced_context = f"{context}\n\n[RETRIEVED]\n{retrieved_context}"

            # Generate answer
            answer, metrics = self.generate_answer(question, enhanced_context)

            # Check correctness (simple substring matching)
            is_correct = str(expected_answer).lower() in answer.lower() or answer.lower() in str(expected_answer).lower()
            if is_correct:
                correct += 1

            status = "✓" if is_correct else "❌"
            print(f"  {status} Generated: {answer[:50]}... | Expected: {expected_answer}")
            if "tokens" in metrics:
                print(f"     Tokens: {metrics['tokens']}, Latency: {metrics.get('latency', 0):.2f}s")

            total_tokens += metrics.get("tokens", 0)
            total_time += metrics.get("latency", 0)

            self.results.append({
                "id": q_id,
                "question": question,
                "expected": expected_answer,
                "generated": answer,
                "correct": is_correct,
                "metrics": metrics
            })

        # Save results
        total_time_elapsed = time.time() - start_time
        accuracy = correct / len(questions) * 100 if questions else 0

        summary = {
            "timestamp": datetime.now().isoformat(),
            "benchmark": "ultrathink-demo",
            "total_questions": len(questions),
            "correct": correct,
            "accuracy_percent": accuracy,
            "total_tokens": total_tokens,
            "total_time_seconds": total_time_elapsed,
            "avg_time_per_q": total_time_elapsed / len(questions) if questions else 0,
            "results": self.results
        }

        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(summary, f, indent=2)

        print(f"\n" + "="*70)
        print(f"RESULTS: {correct}/{len(questions)} correct ({accuracy:.1f}%)")
        print(f"Time: {total_time_elapsed:.1f}s ({total_time_elapsed/len(questions):.2f}s/q)")
        print(f"Tokens: {total_tokens} total ({total_tokens/len(questions) if questions else 0:.0f} avg)")
        print(f"Saved to: {output_path}")
        print("="*70)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Ultrathink-Augmented LoCoMo Benchmark Demo")
    parser.add_argument("--dataset", default="data/locomo10.json", help="Dataset path")
    parser.add_argument("--num-questions", type=int, default=10, help="Number of questions")
    parser.add_argument("--output", default="results/ultrathink_demo_results.json", help="Output file")

    args = parser.parse_args()

    demo = UltrathinkBenchmarkDemo(
        dataset_path=args.dataset,
        num_questions=args.num_questions
    )

    demo.run(output_path=args.output)


if __name__ == "__main__":
    main()
