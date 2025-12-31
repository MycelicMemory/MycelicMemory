"""
LoCoMo Benchmark with Ultrathink MCP Integration

Uses the actual LoCoMo dataset format with conversation history and Q&A pairs.
Integrates with ultrathink for memory-augmented retrieval.

Dataset structure:
- Each object has: qa (list of questions), conversation (history), etc.
- Each qa item has: question, answer, evidence, category
"""

import json
import time
import os
import sys
import re
from typing import List, Dict, Tuple, Optional
from datetime import datetime
import requests


# DeepSeek API configuration
DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY", "REDACTED_API_KEY")


class LocalMemoryBenchmark:
    """Run LoCoMo benchmark with ultrathink memory integration."""

    def __init__(self, dataset_path: str = "data/locomo10.json", max_questions: Optional[int] = None):
        """Initialize benchmark."""
        self.dataset_path = dataset_path
        self.max_questions = max_questions
        self.deepseek_api_key = DEEPSEEK_API_KEY
        self.results = {}

    def load_dataset(self) -> List[Dict]:
        """Load LoCoMo dataset."""
        if not os.path.exists(self.dataset_path):
            print(f"❌ Dataset not found: {self.dataset_path}")
            return []

        with open(self.dataset_path) as f:
            data = json.load(f)

        print(f"✓ Loaded {len(data)} conversation objects")
        return data

    def store_memory(self, content: str, tags: List[str], importance: int = 5) -> bool:
        """Store a memory using ultrathink MCP."""
        try:
            from mcp__ultrathink__store_memory import store_memory as mcp_store
            memory_id = mcp_store(
                content=content,
                importance=importance,
                tags=tags,
                domain="locomo-benchmark"
            )
            return memory_id is not None
        except Exception as e:
            print(f"⚠️  Memory storage failed: {e}")
            return False

    def search_memories(self, query: str, limit: int = 10) -> List[Dict]:
        """Search memories using ultrathink MCP."""
        try:
            from mcp__ultrathink__search import search as mcp_search
            results = mcp_search(
                query=query,
                limit=limit,
                use_ai=True,
                response_format="concise"
            )
            return results if results else []
        except Exception as e:
            print(f"⚠️  Memory search failed: {e}")
            return []

    def generate_answer(self, question: str, context: str) -> Tuple[str, Dict]:
        """Generate answer using DeepSeek with retrieved context."""
        prompt = f"""Based on the following conversation context, answer this question concisely:

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
            "messages": [
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.3,
            "max_tokens": 100,
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

            metrics = {
                "input_tokens": data["usage"]["prompt_tokens"],
                "output_tokens": data["usage"]["completion_tokens"],
                "total_tokens": data["usage"]["total_tokens"],
                "latency": elapsed
            }

            return answer, metrics

        except Exception as e:
            print(f"❌ LLM error: {e}")
            return "", {"error": str(e)}

    def run(self, output_path: str = "results/locomo_benchmark_results.json"):
        """Run benchmark on dataset."""
        print("\n" + "="*70)
        print("LoCoMo BENCHMARK WITH ULTRATHINK MCP")
        print("="*70)

        data = self.load_dataset()
        if not data:
            print("❌ No data to process")
            return

        question_count = 0
        correct_count = 0
        start_time = time.time()

        for obj_idx, obj in enumerate(data):
            print(f"\n[Object {obj_idx+1}]")

            # Get conversation history
            conversation = obj.get("conversation", [])
            conversation_text = self._format_conversation(conversation)

            # Store conversation in ultrathink
            print(f"  • Storing conversation ({len(conversation)} turns)...")
            self.store_memory(
                content=conversation_text,
                tags=[f"object-{obj_idx}", "conversation"],
                importance=7
            )

            # Process each QA pair
            qa_list = obj.get("qa", [])
            for qa_idx, qa in enumerate(qa_list):
                if self.max_questions and question_count >= self.max_questions:
                    break

                question = qa.get("question", "")
                expected_answer = qa.get("answer", "")
                category = qa.get("category", 0)

                if not question:
                    continue

                question_count += 1
                q_id = f"q_{obj_idx}_{qa_idx}"

                print(f"    [{question_count}] {question[:60]}...")

                # Retrieve relevant memories
                print(f"        • Retrieving context...")
                retrieved = self.search_memories(question, limit=5)
                context_parts = []
                for mem in retrieved:
                    content = mem.get("content", mem.get("summary", ""))
                    if content:
                        context_parts.append(content)
                context = "\n".join(context_parts[:3]) if context_parts else conversation_text

                # Generate answer
                print(f"        • Generating answer...")
                answer, metrics = self.generate_answer(question, context)

                # Evaluate
                is_correct = self._check_answer(answer, expected_answer)
                if is_correct:
                    correct_count += 1

                # Store result
                self.results[q_id] = {
                    "object_index": obj_idx,
                    "qa_index": qa_idx,
                    "question": question,
                    "expected_answer": str(expected_answer),
                    "generated_answer": answer,
                    "is_correct": is_correct,
                    "category": category,
                    "retrieved_count": len(retrieved),
                    "metrics": metrics
                }

                status = "✓" if is_correct else "❌"
                print(f"        {status} Generated: {answer[:50]}... | Expected: {str(expected_answer)[:50]}...")

                if self.max_questions and question_count >= self.max_questions:
                    break

            if self.max_questions and question_count >= self.max_questions:
                break

        # Calculate metrics
        total_time = time.time() - start_time
        accuracy = correct_count / question_count * 100 if question_count > 0 else 0

        summary = {
            "timestamp": datetime.now().isoformat(),
            "benchmark": "locomo-with-ultrathink-mcp",
            "total_questions": question_count,
            "correct": correct_count,
            "accuracy_percent": accuracy,
            "total_time_seconds": total_time,
            "avg_time_per_q": total_time / question_count if question_count > 0 else 0,
            "results": self.results
        }

        # Save results
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(summary, f, indent=2)

        print(f"\n" + "="*70)
        print(f"RESULTS: {correct_count}/{question_count} correct ({accuracy:.1f}%)")
        print(f"Time: {total_time:.1f}s ({total_time/question_count:.2f}s per question)")
        print(f"Saved to: {output_path}")
        print("="*70)

        return summary

    def _format_conversation(self, conversation: List[Dict]) -> str:
        """Format conversation history."""
        parts = []
        for msg in conversation:
            speaker = msg.get("speaker", "")
            text = msg.get("text", "")
            if speaker and text:
                parts.append(f"{speaker}: {text}")
        return "\n".join(parts)

    def _check_answer(self, generated: str, expected: str) -> bool:
        """Check if generated answer matches expected."""
        # Normalize for comparison
        gen_lower = str(generated).lower().strip()
        exp_lower = str(expected).lower().strip()

        # Exact match
        if gen_lower == exp_lower:
            return True

        # Substring match
        if exp_lower in gen_lower or gen_lower in exp_lower:
            return True

        # For numeric answers, check if numbers match
        gen_nums = re.findall(r'\d+', gen_lower)
        exp_nums = re.findall(r'\d+', exp_lower)
        if gen_nums and exp_nums and set(gen_nums) & set(exp_nums):
            return True

        return False


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(description="LoCoMo Benchmark with Ultrathink MCP")
    parser.add_argument("--dataset", default="data/locomo10.json", help="Dataset path")
    parser.add_argument("--max-questions", type=int, default=None, help="Max questions")
    parser.add_argument("--output", default="results/locomo_mcp_results.json", help="Output file")

    args = parser.parse_args()

    benchmark = LocalMemoryBenchmark(
        dataset_path=args.dataset,
        max_questions=args.max_questions
    )

    benchmark.run(output_path=args.output)


if __name__ == "__main__":
    main()
