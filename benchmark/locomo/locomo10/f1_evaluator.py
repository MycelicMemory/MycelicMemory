"""
F1 Score Evaluation for LoCoMo Free-Response Benchmark

Implements LoCoMo-compatible F1 scoring with token normalization and Porter stemming.
Based on the official LoCoMo evaluation methodology.

Categories:
- 1: Single-hop (entity/attribute facts)
- 2: Temporal (date/time reasoning)
- 3: Inferential (requires reasoning)
- 4: Open-domain (conversational context)
- 5: Adversarial (robustness testing)
"""

import re
import string
from collections import Counter
from typing import List, Tuple
from nltk.stem import PorterStemmer


# Category mappings
CATEGORY_NAMES = {
    1: "single_hop",
    2: "temporal",
    3: "inferential",
    4: "open_domain",
    5: "adversarial",
}

CATEGORY_DISPLAY = {
    1: "Single-hop",
    2: "Temporal",
    3: "Inferential",
    4: "Open-domain",
    5: "Adversarial",
}


class F1Evaluator:
    """F1 score evaluation matching LoCoMo methodology."""

    def __init__(self):
        self.stemmer = PorterStemmer()
        # Articles to remove during normalization
        self.articles = {"a", "an", "the", "and"}
        # Phrases indicating no information available (for adversarial questions)
        self.no_info_phrases = [
            "no information available",
            "not mentioned",
            "cannot be determined",
            "not specified",
            "no evidence",
            "unknown",
            "not provided",
            "no data",
            "cannot determine",
            "not stated",
        ]

    def normalize_answer(self, text: str) -> str:
        """
        Normalize answer for comparison.

        Steps:
        1. Remove commas
        2. Remove articles (a, an, the, and)
        3. Strip punctuation
        4. Lowercase
        5. Normalize whitespace
        """
        if text is None:
            return ""

        # Convert to string if not already
        text = str(text)

        if not text:
            return ""

        # Lowercase first
        text = text.lower()

        # Remove commas
        text = text.replace(",", " ")

        # Remove punctuation (except for alphanumeric and spaces)
        text = re.sub(r'[^\w\s]', ' ', text)

        # Split into words, remove articles
        words = text.split()
        words = [w for w in words if w not in self.articles]

        # Rejoin and normalize whitespace
        return " ".join(words)

    def tokenize_and_stem(self, text: str) -> List[str]:
        """
        Tokenize and apply Porter stemming.

        Returns list of stemmed tokens.
        """
        if not text:
            return []

        # Normalize first
        normalized = self.normalize_answer(text)

        # Split into tokens
        tokens = normalized.split()

        # Apply Porter stemming
        stemmed = [self.stemmer.stem(token) for token in tokens if token]

        return stemmed

    def f1_score(self, prediction: str, ground_truth: str) -> float:
        """
        Compute F1 score between prediction and ground truth.

        Uses token-level F1 with stemming.
        """
        pred_tokens = Counter(self.tokenize_and_stem(prediction))
        gold_tokens = Counter(self.tokenize_and_stem(ground_truth))

        if not pred_tokens or not gold_tokens:
            # If either is empty, F1 is 0 unless both are empty
            if not pred_tokens and not gold_tokens:
                return 1.0
            return 0.0

        # Count common tokens (intersection)
        common = sum((pred_tokens & gold_tokens).values())

        # Precision: common / predicted
        precision = common / sum(pred_tokens.values())

        # Recall: common / gold
        recall = common / sum(gold_tokens.values())

        if precision + recall == 0:
            return 0.0

        # F1 = 2 * precision * recall / (precision + recall)
        return 2 * precision * recall / (precision + recall)

    def is_no_information_answer(self, prediction: str) -> bool:
        """
        Check if prediction indicates no information is available.

        Used for adversarial question evaluation.
        """
        if prediction is None:
            return False
        prediction_lower = str(prediction).lower()
        return any(phrase in prediction_lower for phrase in self.no_info_phrases)

    def evaluate_by_category(
        self,
        prediction: str,
        ground_truth: str,
        category: int
    ) -> Tuple[float, str]:
        """
        Category-specific evaluation.

        Returns:
            Tuple of (score, evaluation_method)
        """
        if category == 5:  # Adversarial
            # Binary: 1.0 if answer indicates "no information", else 0.0
            # Adversarial questions test if the model correctly identifies
            # when information is not available in the context
            score = 1.0 if self.is_no_information_answer(prediction) else 0.0
            return score, "adversarial_binary"

        elif category == 1:  # Single-hop (may have multiple valid answers)
            # For single-hop questions with potentially multiple answers,
            # we take max F1 across comma-separated predictions
            # This handles cases where multiple entities are valid answers
            if "," in prediction:
                predictions = [p.strip() for p in prediction.split(",")]
                scores = [self.f1_score(pred, ground_truth) for pred in predictions]
                return max(scores), "max_f1_multi"
            else:
                return self.f1_score(prediction, ground_truth), "f1"

        else:  # Categories 2, 3, 4 (Temporal, Inferential, Open-domain)
            return self.f1_score(prediction, ground_truth), "f1"

    def evaluate(
        self,
        prediction: str,
        ground_truth: str,
        category: int
    ) -> dict:
        """
        Full evaluation of a prediction.

        Returns dict with:
            - f1_score: The F1 score (or binary score for adversarial)
            - evaluation_method: How the score was computed
            - normalized_prediction: Normalized prediction text
            - normalized_ground_truth: Normalized ground truth text
            - category: Category number
            - category_name: Category name string
        """
        score, method = self.evaluate_by_category(prediction, ground_truth, category)

        return {
            "f1_score": score,
            "evaluation_method": method,
            "normalized_prediction": self.normalize_answer(prediction),
            "normalized_ground_truth": self.normalize_answer(ground_truth),
            "category": category,
            "category_name": CATEGORY_NAMES.get(category, "unknown"),
        }


def get_category_name(category: int) -> str:
    """Get the name for a category number."""
    return CATEGORY_NAMES.get(category, "unknown")


def get_category_display(category: int) -> str:
    """Get the display name for a category number."""
    return CATEGORY_DISPLAY.get(category, "Unknown")
