"""
Real-time progress display for free-response benchmark execution.
Provides colored, structured console output with running F1 totals.
"""

import sys
from dataclasses import dataclass, field
from typing import Dict, List
from collections import defaultdict
from datetime import datetime


class Colors:
    """ANSI color codes for terminal output."""
    RESET = "\033[0m"
    BOLD = "\033[1m"
    DIM = "\033[2m"

    # Categories
    SINGLE_HOP = "\033[94m"      # Blue
    TEMPORAL = "\033[93m"        # Yellow
    INFERENTIAL = "\033[95m"     # Magenta
    OPEN_DOMAIN = "\033[96m"     # Cyan
    ADVERSARIAL = "\033[91m"     # Red

    # F1 score thresholds
    HIGH_F1 = "\033[92m"         # Green (>= 0.7)
    MID_F1 = "\033[93m"          # Yellow (0.4-0.7)
    LOW_F1 = "\033[91m"          # Red (< 0.4)

    # Metrics
    METRIC = "\033[90m"          # Gray


# Category mappings (1-5)
CATEGORY_COLORS = {
    1: Colors.SINGLE_HOP,
    2: Colors.TEMPORAL,
    3: Colors.INFERENTIAL,
    4: Colors.OPEN_DOMAIN,
    5: Colors.ADVERSARIAL,
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


class FRProgressDisplay:
    """Real-time progress display for free-response benchmark execution."""

    # DeepSeek pricing (per 1M tokens)
    INPUT_PRICE_PER_MTOK = 0.014
    OUTPUT_PRICE_PER_MTOK = 0.056

    def __init__(self, total_questions: int, use_colors: bool = True):
        """
        Initialize progress display.

        Args:
            total_questions: Total number of questions in benchmark
            use_colors: Whether to use ANSI colors (auto-detects TTY)
        """
        self.total_questions = total_questions
        self.use_colors = use_colors and sys.stdout.isatty()
        self.totals = RunningTotals()
        self.start_time = datetime.now()

    def _color(self, text: str, color: str) -> str:
        """Apply color if enabled."""
        if self.use_colors:
            return f"{color}{text}{Colors.RESET}"
        return text

    def _f1_color(self, f1_score: float) -> str:
        """Get color based on F1 score threshold."""
        if f1_score >= 0.7:
            return Colors.HIGH_F1
        elif f1_score >= 0.4:
            return Colors.MID_F1
        else:
            return Colors.LOW_F1

    def _format_category(self, category: int) -> str:
        """Format category with color and emoji."""
        emoji = CATEGORY_EMOJI.get(category, "\u2753")
        color = CATEGORY_COLORS.get(category, Colors.DIM)
        name = CATEGORY_NAMES.get(category, "Unknown")
        return f"{emoji} {self._color(name, color)}"

    def _calculate_cost(self, input_tokens: int, output_tokens: int) -> float:
        """Calculate cost based on DeepSeek pricing."""
        input_cost = (input_tokens / 1_000_000) * self.INPUT_PRICE_PER_MTOK
        output_cost = (output_tokens / 1_000_000) * self.OUTPUT_PRICE_PER_MTOK
        return input_cost + output_cost

    def display_header(self, dataset_path: str, ultrathink_url: str):
        """Display benchmark header."""
        print()
        print(self._color("=" * 80, Colors.BOLD))
        print(self._color("     MEMORY-AUGMENTED LoCoMo FREE-RESPONSE BENCHMARK", Colors.BOLD))
        print(self._color("=" * 80, Colors.BOLD))
        print(f"  Questions:  {self.total_questions}")
        print(f"  Dataset:    {dataset_path}")
        print(f"  Ultrathink: {ultrathink_url}")
        print(self._color("=" * 80, Colors.BOLD))

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

    def display_memory_ops(
        self,
        num_ingested: int,
        ingest_time: float,
        num_retrieved: int,
        retrieval_time: float,
    ):
        """Display memory operation results."""
        inbox = "\U0001f4e5"
        search = "\U0001f50d"
        ingest_str = f"Ingested {num_ingested} memories ({ingest_time:.3f}s)"
        retrieve_str = f"Retrieved {num_retrieved} memories ({retrieval_time:.3f}s)"
        print(f"   {inbox} {ingest_str}")
        print(f"   {search} {retrieve_str}")

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

        # --- LATENCY STATS ---
        timer_emoji = "\u23f1\ufe0f"
        memo_emoji = "\U0001f4dd"
        money_emoji = "\U0001f4b0"
        hourglass_emoji = "\u23f3"

        latency = metrics.get('latency', {})
        if latency:
            print(f"\n{self._color(timer_emoji + '  LATENCY STATISTICS', Colors.BOLD)}")
            print(self._color(single_line, Colors.DIM))
            print(f"   Mean:    {latency.get('mean_latency_seconds', 0):.3f}s")
            print(f"   Median:  {latency.get('median_latency_seconds', 0):.3f}s")
            print(f"   P95:     {latency.get('p95_latency_seconds', 0):.3f}s")
            print(f"   P99:     {latency.get('p99_latency_seconds', 0):.3f}s")
            print(f"   Min/Max: {latency.get('min_latency_seconds', 0):.3f}s / {latency.get('max_latency_seconds', 0):.3f}s")
            print(f"   StdDev:  {latency.get('stdev_latency_seconds', 0):.3f}s")

        # --- TOKEN USAGE ---
        tokens = metrics.get('tokens', {})
        if tokens:
            print(f"\n{self._color(memo_emoji + ' TOKEN USAGE', Colors.BOLD)}")
            print(self._color(single_line, Colors.DIM))
            print(f"   Total Input:    {tokens.get('total_input_tokens', 0):,}")
            print(f"   Total Output:   {tokens.get('total_output_tokens', 0):,}")
            print(f"   Total:          {tokens.get('total_tokens', 0):,}")
            print(f"   Mean per Q:     {tokens.get('mean_input_tokens', 0):.0f} in / {tokens.get('mean_output_tokens', 0):.1f} out")

        # --- COST ESTIMATION ---
        cost = metrics.get('cost_estimation', {})
        if cost:
            print(f"\n{self._color(money_emoji + ' COST ESTIMATION (DeepSeek)', Colors.BOLD)}")
            print(self._color(single_line, Colors.DIM))
            print(f"   Input Cost:     ${cost.get('input_cost_usd', 0):.6f}")
            print(f"   Output Cost:    ${cost.get('output_cost_usd', 0):.6f}")
            print(f"   Total Cost:     ${cost.get('total_cost_usd', 0):.6f}")
            print(f"   Per Question:   ${cost.get('cost_per_question_usd', 0):.6f}")

        # --- DURATION ---
        print(f"\n{self._color(hourglass_emoji + ' DURATION', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))
        mins = int(duration_secs // 60)
        secs = duration_secs % 60
        if mins > 0:
            print(f"   Total Time:     {mins}m {secs:.1f}s ({duration_secs:.2f}s)")
        else:
            print(f"   Total Time:     {secs:.1f}s")

        print()
        print(self._color(double_line, Colors.BOLD))
