"""
Real-time progress display for benchmark execution.
Provides colored, structured console output with running totals.
"""

from dataclasses import dataclass, field
from typing import Dict, List, Optional
from collections import defaultdict

from shared.display_base import Colors, DisplayBase


QUESTION_TYPE_COLORS = {
    "single_hop": Colors.BLUE,
    "multi_hop": Colors.MAGENTA,
    "temporal_reasoning": Colors.YELLOW,
    "open_domain": Colors.CYAN,
    "unknown": Colors.DIM,
}

QUESTION_TYPE_EMOJI = {
    "single_hop": "1\ufe0f\u20e3 ",
    "multi_hop": "\U0001f517",
    "temporal_reasoning": "\u23f0",
    "open_domain": "\U0001f310",
    "unknown": "\u2753",
}

QUESTION_TYPE_NAMES = {
    "single_hop": "Single Hop",
    "multi_hop": "Multi Hop",
    "temporal_reasoning": "Temporal",
    "open_domain": "Open Domain",
    "unknown": "Unknown",
}


@dataclass
class RunningTotals:
    """Track running totals during benchmark execution."""
    total_questions: int = 0
    correct_by_type: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    total_by_type: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    total_correct: int = 0
    total_input_tokens: int = 0
    total_output_tokens: int = 0
    total_cost: float = 0.0
    total_latency: float = 0.0
    latencies: List[float] = field(default_factory=list)


class ProgressDisplay(DisplayBase):
    """Real-time progress display for benchmark execution."""

    def __init__(self, total_questions: int, use_colors: bool = True):
        """
        Initialize progress display.

        Args:
            total_questions: Total number of questions in benchmark
            use_colors: Whether to use ANSI colors (auto-detects TTY)
        """
        super().__init__(total_questions, use_colors)
        self.totals = RunningTotals()

    def _format_question_type(self, qtype: str) -> str:
        """Format question type with color and emoji."""
        emoji = QUESTION_TYPE_EMOJI.get(qtype, QUESTION_TYPE_EMOJI["unknown"])
        color = QUESTION_TYPE_COLORS.get(qtype, Colors.DIM)
        type_name = QUESTION_TYPE_NAMES.get(qtype, qtype.replace("_", " ").title())
        return f"{emoji} {self._color(type_name, color)}"

    # Note: _calculate_cost() inherited from DisplayBase

    def display_header(self, dataset_path: str, ultrathink_url: str):
        """Display benchmark header."""
        super().display_header(
            "MEMORY-AUGMENTED LoCoMo-MC10 BENCHMARK",
            dataset_path,
            ultrathink_url
        )

    def display_question_start(
        self,
        idx: int,
        question_id: str,
        question_type: str,
        question_text: str,
    ):
        """Display question header with type prominently shown."""
        print()
        print(self._color("\u2500" * 80, Colors.DIM))

        # Question number and type
        progress = f"[{idx + 1}/{self.total_questions}]"
        qtype_display = self._format_question_type(question_type)
        print(f"{self._color(progress, Colors.BOLD)} {qtype_display} | ID: {question_id}")

        # Question text (truncated to 75 chars)
        truncated = question_text[:75] + "..." if len(question_text) > 75 else question_text
        print(f"   {self._color('Q:', Colors.DIM)} {truncated}")

    # Note: display_memory_ops() inherited from DisplayBase

    def display_result(
        self,
        predicted_idx: Optional[int],
        correct_idx: int,
        is_correct: bool,
        llm_latency: float,
        input_tokens: int,
        output_tokens: int,
        question_type: str,
    ):
        """Display question result and update running totals."""
        # Update totals
        self.totals.total_questions += 1
        self.totals.total_by_type[question_type] += 1
        if is_correct:
            self.totals.correct_by_type[question_type] += 1
            self.totals.total_correct += 1
        self.totals.total_input_tokens += input_tokens
        self.totals.total_output_tokens += output_tokens
        self.totals.total_latency += llm_latency
        self.totals.latencies.append(llm_latency)

        # Calculate cost
        cost = self._calculate_cost(input_tokens, output_tokens)
        self.totals.total_cost += cost

        # Result line
        status = "\u2713 CORRECT" if is_correct else "\u2717 INCORRECT"
        status_color = Colors.HIGH if is_correct else Colors.LOW
        pred_str = str(predicted_idx) if predicted_idx is not None else "None"
        print(f"   {self._color(status, status_color)} | Predicted: {pred_str} vs Correct: {correct_idx}")

        # Metrics line
        tokens_str = f"{input_tokens:,}in/{output_tokens:,}out"
        latency_str = f"{llm_latency:.3f}s"
        cost_str = f"${cost:.6f}"
        print(f"   \U0001f4ca Tokens: {tokens_str} | Latency: {latency_str} | Cost: {cost_str}")

        # Running totals
        self._display_running_totals(question_type)

    def _display_running_totals(self, current_type: str):
        """Display running accuracy totals."""
        overall_correct = self.totals.total_correct
        overall_total = self.totals.total_questions
        overall_acc = (overall_correct / overall_total * 100) if overall_total > 0 else 0

        # Type-specific accuracy
        type_correct = self.totals.correct_by_type[current_type]
        type_total = self.totals.total_by_type[current_type]
        type_acc = (type_correct / type_total * 100) if type_total > 0 else 0

        type_color = QUESTION_TYPE_COLORS.get(current_type, Colors.DIM)
        type_name = QUESTION_TYPE_NAMES.get(current_type, current_type.replace("_", " ").title())

        running = f"Running: {self._color(f'{overall_acc:.1f}%', Colors.BOLD)} overall"
        running += f" | {self._color(f'{type_acc:.1f}%', type_color)} {type_name}"
        running += f" | Cost: ${self.totals.total_cost:.4f}"
        print(f"   \U0001f4c8 {running}")

    def display_final_summary(self, metrics: Dict, per_type_metrics: Dict, duration_secs: float):
        """Display comprehensive final summary."""
        print()
        print()
        print(self._color("\u2550" * 80, Colors.BOLD))
        print(self._color("                    BENCHMARK RESULTS SUMMARY", Colors.BOLD))
        print(self._color("\u2550" * 80, Colors.BOLD))

        # --- OVERALL ACCURACY ---
        overall = metrics.get('overall', {})
        acc = overall.get('accuracy', 0)
        correct = overall.get('correct_predictions', 0)
        total = overall.get('total_questions', 0)

        chart_emoji = "\U0001f4ca"  # 📊
        target_emoji = "\U0001f3af"  # 🎯
        print(f"\n{self._color(chart_emoji + ' OVERALL ACCURACY', Colors.BOLD)}")
        print(self._color("\u2500" * 40, Colors.DIM))
        acc_color = Colors.HIGH if acc >= 50 else Colors.LOW
        print(f"   Accuracy: {self._color(f'{acc:.1f}%', acc_color)}")
        print(f"   Correct:  {correct}/{total}")

        # --- ACCURACY BY QUESTION TYPE ---
        print(f"\n{self._color(target_emoji + ' ACCURACY BY QUESTION TYPE', Colors.BOLD)}")
        print(self._color("\u2500" * 40, Colors.DIM))

        type_order = ["single_hop", "multi_hop", "temporal_reasoning", "open_domain"]
        for qtype in type_order:
            if qtype in per_type_metrics:
                tm = per_type_metrics[qtype]
                type_display = self._format_question_type(qtype)
                type_acc = tm.get('accuracy', 0)
                type_correct = tm.get('correct', 0)
                type_total = tm.get('total', 0)
                acc_color = Colors.HIGH if type_acc >= 50 else Colors.LOW
                print(f"   {type_display}: {self._color(f'{type_acc:.1f}%', acc_color)} ({type_correct}/{type_total})")

        # Check for any other types not in the standard list
        for qtype in per_type_metrics:
            if qtype not in type_order:
                tm = per_type_metrics[qtype]
                type_display = self._format_question_type(qtype)
                type_acc = tm.get('accuracy', 0)
                type_correct = tm.get('correct', 0)
                type_total = tm.get('total', 0)
                acc_color = Colors.HIGH if type_acc >= 50 else Colors.LOW
                print(f"   {type_display}: {self._color(f'{type_acc:.1f}%', acc_color)} ({type_correct}/{type_total})")

        # Use shared display methods for common sections
        self.display_latency_stats(metrics.get('latency', {}))
        self.display_token_stats(metrics.get('tokens', {}))
        self.display_cost_stats(metrics.get('cost_estimation', {}))
        self.display_duration(duration_secs)

        print()
        print(self._color("\u2550" * 80, Colors.BOLD))
