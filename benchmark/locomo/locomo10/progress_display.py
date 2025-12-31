"""
Real-time progress display for free-response benchmark execution.
Provides colored, structured console output with running F1 totals.
"""

from dataclasses import dataclass, field
from typing import Dict, List
from collections import defaultdict

from shared.display_base import Colors, DisplayBase


# Category mappings (1-5)
CATEGORY_COLORS = {
    1: Colors.BLUE,       # Single-hop
    2: Colors.YELLOW,     # Temporal
    3: Colors.MAGENTA,    # Inferential
    4: Colors.CYAN,       # Open-domain
    5: Colors.RED,        # Adversarial
}

CATEGORY_EMOJI = {
    1: "\U0001f4cd",      # Pin (Single-hop)
    2: "\u23f0",          # Clock (Temporal)
    3: "\U0001f9e0",      # Brain (Inferential)
    4: "\U0001f310",      # Globe (Open-domain)
    5: "\u2694\ufe0f",    # Swords (Adversarial)
}

CATEGORY_NAMES = {
    1: "Single-hop",
    2: "Temporal",
    3: "Inferential",
    4: "Open-domain",
    5: "Adversarial",
}


@dataclass
class RunningTotals:
    """Track running totals during benchmark execution."""
    total_questions: int = 0
    f1_scores_by_category: Dict[int, List[float]] = field(default_factory=lambda: defaultdict(list))
    all_f1_scores: List[float] = field(default_factory=list)
    total_input_tokens: int = 0
    total_output_tokens: int = 0
    total_cost: float = 0.0
    total_latency: float = 0.0
    latencies: List[float] = field(default_factory=list)


class FRProgressDisplay(DisplayBase):
    """Real-time progress display for free-response benchmark execution."""

    def __init__(self, total_questions: int, use_colors: bool = True):
        """
        Initialize progress display.

        Args:
            total_questions: Total number of questions in benchmark
            use_colors: Whether to use ANSI colors (auto-detects TTY)
        """
        super().__init__(total_questions, use_colors)
        self.totals = RunningTotals()

    def _f1_color(self, f1_score: float) -> str:
        """Get color based on F1 score threshold."""
        if f1_score >= 0.7:
            return Colors.HIGH
        elif f1_score >= 0.4:
            return Colors.MID
        else:
            return Colors.LOW

    def _format_category(self, category: int) -> str:
        """Format category with color and emoji."""
        emoji = CATEGORY_EMOJI.get(category, "\u2753")
        color = CATEGORY_COLORS.get(category, Colors.DIM)
        name = CATEGORY_NAMES.get(category, "Unknown")
        return f"{emoji} {self._color(name, color)}"

    # Note: _calculate_cost() inherited from DisplayBase

    def display_header(self, dataset_path: str, ultrathink_url: str):
        """Display benchmark header."""
        super().display_header(
            "MEMORY-AUGMENTED LoCoMo FREE-RESPONSE BENCHMARK",
            dataset_path,
            ultrathink_url
        )

    def display_question_start(
        self,
        idx: int,
        question_id: str,
        category: int,
        question_text: str,
    ):
        """Display question header with category prominently shown."""
        print()
        line = "\u2500" * 80
        print(self._color(line, Colors.DIM))

        # Question number and category
        progress = f"[{idx + 1}/{self.total_questions}]"
        category_display = self._format_category(category)
        print(f"{self._color(progress, Colors.BOLD)} {category_display} | ID: {question_id}")

        # Question text (truncated to 75 chars)
        truncated = question_text[:75] + "..." if len(question_text) > 75 else question_text
        q_label = self._color("Q:", Colors.DIM)
        print(f"   {q_label} {truncated}")

    # Note: display_memory_ops() inherited from DisplayBase

    def display_result(
        self,
        prediction: str,
        ground_truth: str,
        f1_score: float,
        llm_latency: float,
        input_tokens: int,
        output_tokens: int,
        category: int,
    ):
        """Display question result and update running totals."""
        # Update totals
        self.totals.total_questions += 1
        self.totals.f1_scores_by_category[category].append(f1_score)
        self.totals.all_f1_scores.append(f1_score)
        self.totals.total_input_tokens += input_tokens
        self.totals.total_output_tokens += output_tokens
        self.totals.total_latency += llm_latency
        self.totals.latencies.append(llm_latency)

        # Calculate cost
        cost = self._calculate_cost(input_tokens, output_tokens)
        self.totals.total_cost += cost

        # F1 score display with color
        f1_color = self._f1_color(f1_score)
        f1_str = self._color(f"{f1_score:.3f}", f1_color)

        # Convert to strings and truncate for display
        pred_str = str(prediction) if prediction else ""
        gold_str = str(ground_truth) if ground_truth else ""
        pred_short = pred_str[:40] + "..." if len(pred_str) > 40 else pred_str
        gold_short = gold_str[:40] + "..." if len(gold_str) > 40 else gold_str

        print(f"   F1: {f1_str} | Pred: \"{pred_short}\" vs Gold: \"{gold_short}\"")

        # Metrics line
        chart = "\U0001f4ca"
        tokens_str = f"{input_tokens:,}in/{output_tokens:,}out"
        latency_str = f"{llm_latency:.3f}s"
        cost_str = f"${cost:.6f}"
        print(f"   {chart} Tokens: {tokens_str} | Latency: {latency_str} | Cost: {cost_str}")

        # Running totals
        self._display_running_totals(category)

    def _display_running_totals(self, current_category: int):
        """Display running F1 totals."""
        overall_f1 = sum(self.totals.all_f1_scores) / len(self.totals.all_f1_scores) if self.totals.all_f1_scores else 0.0

        # Category-specific F1
        cat_scores = self.totals.f1_scores_by_category[current_category]
        cat_f1 = sum(cat_scores) / len(cat_scores) if cat_scores else 0.0

        cat_color = CATEGORY_COLORS.get(current_category, Colors.DIM)
        cat_name = CATEGORY_NAMES.get(current_category, "Unknown")

        chart = "\U0001f4c8"
        running = f"Running: {self._color(f'{overall_f1:.3f}', Colors.BOLD)} F1 overall"
        running += f" | {self._color(f'{cat_f1:.3f}', cat_color)} F1 {cat_name}"
        running += f" | Cost: ${self.totals.total_cost:.4f}"
        print(f"   {chart} {running}")

    def display_final_summary(self, metrics: Dict, per_category_metrics: Dict, duration_secs: float):
        """Display comprehensive final summary."""
        print()
        print()
        double_line = "\u2550" * 80
        print(self._color(double_line, Colors.BOLD))
        print(self._color("                    BENCHMARK RESULTS SUMMARY", Colors.BOLD))
        print(self._color(double_line, Colors.BOLD))

        # --- OVERALL F1 ---
        overall = metrics.get('overall', {})
        mean_f1 = overall.get('mean_f1', 0)
        median_f1 = overall.get('median_f1', 0)
        total = overall.get('total_questions', 0)

        chart_emoji = "\U0001f4ca"
        target_emoji = "\U0001f3af"
        single_line = "\u2500" * 40

        print(f"\n{self._color(chart_emoji + ' OVERALL F1 SCORE', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))
        f1_color = self._f1_color(mean_f1)
        print(f"   Mean F1:   {self._color(f'{mean_f1:.3f}', f1_color)}")
        print(f"   Median F1: {median_f1:.3f}")
        print(f"   Questions: {total}")

        # --- F1 BY CATEGORY ---
        print(f"\n{self._color(target_emoji + ' F1 BY QUESTION CATEGORY', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))

        category_order = [1, 2, 3, 4, 5]  # Single-hop, Temporal, Inferential, Open-domain, Adversarial
        for cat_id in category_order:
            cat_name = CATEGORY_NAMES.get(cat_id, "unknown").lower().replace("-", "_")
            if cat_name in per_category_metrics:
                cm = per_category_metrics[cat_name]
                cat_display = self._format_category(cat_id)
                cat_f1 = cm.get('f1', {}).get('mean', 0)
                cat_total = cm.get('total', 0)
                f1_color = self._f1_color(cat_f1)
                print(f"   {cat_display}: {self._color(f'{cat_f1:.3f}', f1_color)} F1 ({cat_total} questions)")

        # Check for any other categories not in the standard list
        for cat_name, cm in per_category_metrics.items():
            cat_id = cm.get('category_id', 0)
            if cat_id not in category_order:
                cat_display = self._format_category(cat_id)
                cat_f1 = cm.get('f1', {}).get('mean', 0)
                cat_total = cm.get('total', 0)
                f1_color = self._f1_color(cat_f1)
                print(f"   {cat_display}: {self._color(f'{cat_f1:.3f}', f1_color)} F1 ({cat_total} questions)")

        # Use shared display methods for common sections
        self.display_latency_stats(metrics.get('latency', {}))
        self.display_token_stats(metrics.get('tokens', {}))
        self.display_cost_stats(metrics.get('cost_estimation', {}))
        self.display_duration(duration_secs)

        print()
        print(self._color("\u2550" * 80, Colors.BOLD))
