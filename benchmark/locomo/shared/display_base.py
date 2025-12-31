"""
Base display classes and utilities shared across benchmark variants.
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

    # Score/accuracy thresholds
    HIGH = "\033[92m"         # Green (good performance)
    MID = "\033[93m"          # Yellow (moderate performance)
    LOW = "\033[91m"          # Red (poor performance)

    # Question types/categories
    BLUE = "\033[94m"
    MAGENTA = "\033[95m"
    YELLOW = "\033[93m"
    CYAN = "\033[96m"
    RED = "\033[91m"

    # Metrics
    METRIC = "\033[90m"       # Gray


class DisplayBase:
    """Base class for progress displays with shared utility methods."""

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
        self.start_time = datetime.now()

    def _color(self, text: str, color: str) -> str:
        """Apply color if enabled."""
        if self.use_colors:
            return f"{color}{text}{Colors.RESET}"
        return text

    def _calculate_cost(self, input_tokens: int, output_tokens: int) -> float:
        """Calculate cost based on DeepSeek pricing."""
        input_cost = (input_tokens / 1_000_000) * self.INPUT_PRICE_PER_MTOK
        output_cost = (output_tokens / 1_000_000) * self.OUTPUT_PRICE_PER_MTOK
        return input_cost + output_cost

    def display_header(self, title: str, dataset_path: str, ultrathink_url: str):
        """Display benchmark header."""
        print()
        print(self._color("=" * 80, Colors.BOLD))
        print(self._color(f"     {title}", Colors.BOLD))
        print(self._color("=" * 80, Colors.BOLD))
        print(f"  Questions:  {self.total_questions}")
        print(f"  Dataset:    {dataset_path}")
        print(f"  Ultrathink: {ultrathink_url}")
        print(self._color("=" * 80, Colors.BOLD))

    def display_memory_ops(
        self,
        num_ingested: int,
        ingest_time: float,
        num_retrieved: int,
        retrieval_time: float,
    ):
        """Display memory operation results."""
        ingest_str = f"Ingested {num_ingested} memories ({ingest_time:.3f}s)"
        retrieve_str = f"Retrieved {num_retrieved} memories ({retrieval_time:.3f}s)"
        print(f"   \U0001f4e5 {ingest_str}")
        print(f"   \U0001f50d {retrieve_str}")

    def display_latency_stats(self, latency: Dict):
        """Display latency statistics section."""
        if not latency:
            return

        timer_emoji = "\u23f1\ufe0f"
        single_line = "\u2500" * 40

        print(f"\n{self._color(timer_emoji + '  LATENCY STATISTICS', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))
        print(f"   Mean:    {latency.get('mean_latency_seconds', 0):.3f}s")
        print(f"   Median:  {latency.get('median_latency_seconds', 0):.3f}s")
        print(f"   P95:     {latency.get('p95_latency_seconds', 0):.3f}s")
        print(f"   P99:     {latency.get('p99_latency_seconds', 0):.3f}s")
        print(f"   Min/Max: {latency.get('min_latency_seconds', 0):.3f}s / {latency.get('max_latency_seconds', 0):.3f}s")
        print(f"   StdDev:  {latency.get('stdev_latency_seconds', 0):.3f}s")

    def display_token_stats(self, tokens: Dict):
        """Display token usage section."""
        if not tokens:
            return

        memo_emoji = "\U0001f4dd"
        single_line = "\u2500" * 40

        print(f"\n{self._color(memo_emoji + ' TOKEN USAGE', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))
        print(f"   Total Input:    {tokens.get('total_input_tokens', 0):,}")
        print(f"   Total Output:   {tokens.get('total_output_tokens', 0):,}")
        print(f"   Total:          {tokens.get('total_tokens', 0):,}")
        print(f"   Mean per Q:     {tokens.get('mean_input_tokens', 0):.0f} in / {tokens.get('mean_output_tokens', 0):.1f} out")

    def display_cost_stats(self, cost: Dict):
        """Display cost estimation section."""
        if not cost:
            return

        money_emoji = "\U0001f4b0"
        single_line = "\u2500" * 40

        print(f"\n{self._color(money_emoji + ' COST ESTIMATION (DeepSeek)', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))
        print(f"   Input Cost:     ${cost.get('input_cost_usd', 0):.6f}")
        print(f"   Output Cost:    ${cost.get('output_cost_usd', 0):.6f}")
        print(f"   Total Cost:     ${cost.get('total_cost_usd', 0):.6f}")
        print(f"   Per Question:   ${cost.get('cost_per_question_usd', 0):.6f}")

    def display_duration(self, duration_secs: float):
        """Display duration section."""
        hourglass_emoji = "\u23f3"
        single_line = "\u2500" * 40

        print(f"\n{self._color(hourglass_emoji + ' DURATION', Colors.BOLD)}")
        print(self._color(single_line, Colors.DIM))
        mins = int(duration_secs // 60)
        secs = duration_secs % 60
        if mins > 0:
            print(f"   Total Time:     {mins}m {secs:.1f}s ({duration_secs:.2f}s)")
        else:
            print(f"   Total Time:     {secs:.1f}s")
