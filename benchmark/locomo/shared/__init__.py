"""
Shared modules for LoCoMo benchmarks.

Contains common utilities used by both locomo_mc10 and locomo10 benchmarks.
"""

from .config import DEEPSEEK_API_KEY, DEEPSEEK_BASE_URL, DEEPSEEK_MODEL
from .ultrathink_client import UltrathinkClient, RetrievalResult
from .llm_call_tracker import LLMCallTracker
from .logging_system import BenchmarkLogger, CallType, init_logger, get_logger
from .metrics_base import TokenMetrics, LatencyMetrics, MetricsBase
from .display_base import Colors, DisplayBase

__all__ = [
    # Config
    "DEEPSEEK_API_KEY",
    "DEEPSEEK_BASE_URL",
    "DEEPSEEK_MODEL",
    # Ultrathink client
    "UltrathinkClient",
    "RetrievalResult",
    # Tracking
    "LLMCallTracker",
    # Logging
    "BenchmarkLogger",
    "CallType",
    "init_logger",
    "get_logger",
    # Metrics base
    "TokenMetrics",
    "LatencyMetrics",
    "MetricsBase",
    # Display base
    "Colors",
    "DisplayBase",
]
