"""
Quick verification script to check for accuracy issues in our implementation.
"""

import json
import sys


def verify_dataset_integrity(data_path: str = "results/ultrathink_results.json") -> None:
    """Verify that correct_choice_index actually points to the answer text."""
    print("=" * 70)
    print("DATASET INTEGRITY CHECK")
    print("=" * 70)

    with open(data_path, "r") as f:
        data = json.load(f)

    mismatches = []
    total = 0

    for q_id, items in data.items():
        for item in items:
            total += 1
            correct_idx = item.get("correct_choice_index", -1)
            choices = item.get("choices", [])
            gold_answer = item.get("gold_answer", "")

            # Check if index is valid
            if correct_idx < 0 or correct_idx >= len(choices):
                mismatches.append({
                    "issue": "Invalid choice index",
                    "question_id": q_id,
                    "index": correct_idx,
                    "num_choices": len(choices),
                })
                continue

            # Check if choice matches answer
            choice_text = choices[correct_idx]
            if choice_text != gold_answer:
                mismatches.append({
                    "issue": "Choice text doesn't match answer",
                    "question_id": q_id,
                    "correct_index": correct_idx,
                    "choice_text": choice_text[:50],
                    "gold_answer": gold_answer[:50],
                })

    print(f"✓ Checked {total} questions")
    if mismatches:
        print(f"❌ Found {len(mismatches)} mismatches:")
        for i, mismatch in enumerate(mismatches[:5]):  # Show first 5
            print(f"\n  {i+1}. {mismatch['issue']}")
            print(f"     Question: {mismatch['question_id']}")
            for key, val in mismatch.items():
                if key != "issue" and key != "question_id":
                    print(f"     {key}: {val}")
    else:
        print("✓ All correct_choice_index values are valid")


def analyze_single_hop_performance(data_path: str = "results/ultrathink_results.json") -> None:
    """Check single-hop performance - should be nearly perfect."""
    print("\n" + "=" * 70)
    print("SINGLE-HOP ANALYSIS")
    print("=" * 70)

    with open(data_path, "r") as f:
        data = json.load(f)

    single_hop = []
    for q_id, items in data.items():
        for item in items:
            if item.get("question_type") == "single_hop":
                single_hop.append(item)

    if not single_hop:
        print("⚠️  No single-hop questions found")
        return

    correct = sum(1 for q in single_hop if q.get("predicted_choice_index") == q.get("correct_choice_index"))
    accuracy = correct / len(single_hop) * 100

    print(f"Single-Hop Questions: {len(single_hop)}")
    print(f"Correct: {correct}")
    print(f"Accuracy: {accuracy:.1f}%")

    if accuracy == 100.0:
        print("⚠️  WARNING: 100% accuracy is suspiciously high")
        print("   This might indicate:")
        print("   - Test set is too easy for these questions")
        print("   - Extraction is not working correctly")
        print("   - Model is overfitting to the answers")

    # Show some examples
    print("\nFirst 3 single-hop questions:")
    for i, q in enumerate(single_hop[:3]):
        pred = q.get("predicted_choice_index")
        correct = q.get("correct_choice_index")
        status = "✓" if pred == correct else "❌"
        print(f"{status} Q: {q['question'][:50]}...")
        print(f"   Predicted: {pred}, Correct: {correct}")


def analyze_choice_extraction(data_path: str = "results/ultrathink_results.json") -> None:
    """Check if any questions have None predicted index."""
    print("\n" + "=" * 70)
    print("CHOICE EXTRACTION ANALYSIS")
    print("=" * 70)

    with open(data_path, "r") as f:
        data = json.load(f)

    none_count = 0
    total = 0

    for q_id, items in data.items():
        for item in items:
            total += 1
            if item.get("predicted_choice_index") is None:
                none_count += 1

    print(f"Total Questions: {total}")
    print(f"Extraction Failures (None): {none_count}")
    print(f"Extraction Success Rate: {(total - none_count) / total * 100:.1f}%")

    if none_count > 0:
        print(f"❌ WARNING: {none_count} questions failed choice extraction")
        print("   This means those questions were marked as incorrect by default")


def compare_vs_mem0(data_path: str = "results/ultrathink_results.json") -> None:
    """Check if our results are in reasonable range."""
    print("\n" + "=" * 70)
    print("PLAUSIBILITY CHECK")
    print("=" * 70)

    with open(data_path, "r") as f:
        data = json.load(f)

    # Calculate stats
    total = len([i for items in data.values() for i in items])
    correct = sum(1 for items in data.values() for i in items
                  if i.get("predicted_choice_index") == i.get("correct_choice_index"))
    accuracy = correct / total * 100 if total > 0 else 0

    print(f"Overall Accuracy: {accuracy:.1f}%")
    print(f"Correct/Total: {correct}/{total}")

    print("\nExpected Ranges (LoCoMo-MC10):")
    print("  Random baseline: 10% (1 out of 10)")
    print("  Weak baseline: 30-40%")
    print("  Strong baseline: 50-70%")
    print("  State-of-the-art: 70-85%")

    if accuracy < 30:
        print(f"\n⚠️  WARNING: {accuracy:.1f}% is very low (below weak baseline)")
    elif accuracy > 85:
        print(f"\n⚠️  WARNING: {accuracy:.1f}% is very high (above SOTA)")
        print("   Verify that dataset or extraction is correct")
    else:
        print(f"\n✓ {accuracy:.1f}% is in expected range")


def main():
    """Run all verification checks."""
    print("\n🔍 BENCHMARK ACCURACY VERIFICATION\n")

    try:
        verify_dataset_integrity()
        analyze_single_hop_performance()
        analyze_choice_extraction()
        compare_vs_mem0()

        print("\n" + "=" * 70)
        print("VERIFICATION COMPLETE")
        print("=" * 70)
        print("\nNext steps:")
        print("1. Check ACCURACY_AUDIT.md for detailed issues")
        print("2. Implement memory system integration")
        print("3. Improve choice extraction robustness")
        print("4. Compare against published LoCoMo-MC10 results")

    except FileNotFoundError:
        print("Error: results/ultrathink_results.json not found")
        print("Run the benchmark first: python run_experiments.py --max_questions 50")
        sys.exit(1)


if __name__ == "__main__":
    main()
