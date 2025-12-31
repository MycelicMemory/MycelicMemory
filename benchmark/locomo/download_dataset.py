#!/usr/bin/env python3
"""
Download LoCoMo-MC10 Dataset from HuggingFace

This script downloads the official LoCoMo-MC10 dataset from HuggingFace.
The dataset is 1,986 multiple-choice questions used for benchmarking.

Usage:
    python download_dataset.py
"""

import os
import json
import sys
from pathlib import Path


def download_dataset(output_path: str = "data/locomo-mc10-full.jsonl"):
    """Download LoCoMo-MC10 dataset from HuggingFace."""
    print("Downloading LoCoMo-MC10 dataset from HuggingFace...")
    print("This may take a minute or two...\n")

    try:
        from huggingface_hub import hf_hub_download
    except ImportError:
        print("❌ huggingface_hub not installed")
        print("   Install with: pip install huggingface_hub")
        sys.exit(1)

    try:
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)

        # Download the JSONL file
        json_file = hf_hub_download(
            repo_id="Percena/locomo-mc10",
            filename="data/locomo_mc10.json",
            repo_type="dataset"
        )
        print(f"✓ Downloaded: {json_file}")

        # Convert to JSONL format
        data = []
        with open(json_file) as f:
            for line in f:
                if line.strip():
                    data.append(json.loads(line))

        print(f"✓ Loaded {len(data)} questions")

        # Save locally
        with open(output_path, "w") as f:
            for q in data:
                f.write(json.dumps(q) + "\n")

        print(f"✓ Saved to {output_path}")

        # Show statistics
        print(f"\nDataset Statistics:")
        print(f"  Total Questions: {len(data)}")

        if data:
            q = data[0]
            print(f"\nSample Question:")
            print(f"  Question: {q.get('question', '')[:80]}...")
            print(f"  Type: {q.get('question_type', '')}")
            print(f"  Choices: {len(q.get('choices', []))}")

            # Count by type
            types = {}
            for q in data:
                qtype = q.get('question_type', 'unknown')
                types[qtype] = types.get(qtype, 0) + 1

            print(f"\nBreakdown by Type:")
            for qtype, count in sorted(types.items()):
                print(f"  {qtype}: {count}")

        print(f"\n✅ Dataset ready for benchmarking!")
        return True

    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = download_dataset()
    sys.exit(0 if success else 1)
