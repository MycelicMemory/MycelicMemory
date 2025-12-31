"""
Robust Real-Time Logging & LLM Call Trail System

Provides comprehensive logging infrastructure for:
- LLM API calls with full request/response tracking
- Ultrathink memory operations
- Benchmark execution flow
- Performance metrics and timing
- Error tracking and debugging

Features:
- Structured logging (JSON format)
- Real-time console output with colors
- File-based persistence
- Call trail for debugging
- Performance profiling
- Async logging for high throughput
"""

import json
import time
import logging
import logging.handlers
import sys
import os
from dataclasses import dataclass, asdict, field
from datetime import datetime
from typing import Any, Dict, List, Optional
from enum import Enum
from pathlib import Path
import traceback
from functools import wraps


class LogLevel(Enum):
    """Log level enumeration."""
    DEBUG = logging.DEBUG
    INFO = logging.INFO
    WARNING = logging.WARNING
    ERROR = logging.ERROR
    CRITICAL = logging.CRITICAL


class CallType(Enum):
    """Type of call being logged."""
    LLM_REQUEST = "llm_request"
    LLM_RESPONSE = "llm_response"
    LLM_ERROR = "llm_error"
    MEMORY_INGEST = "memory_ingest"
    MEMORY_RETRIEVE = "memory_retrieve"
    MEMORY_DELETE = "memory_delete"
    BENCHMARK_START = "benchmark_start"
    BENCHMARK_END = "benchmark_end"
    QUESTION_START = "question_start"
    QUESTION_END = "question_end"


@dataclass
class LLMCallRecord:
    """Record of an LLM API call."""
    timestamp: str
    call_type: str
    duration_ms: float
    request: Dict[str, Any] = field(default_factory=dict)
    response: Dict[str, Any] = field(default_factory=dict)
    error: Optional[str] = None
    model: str = ""
    input_tokens: int = 0
    output_tokens: int = 0
    cost_usd: float = 0.0
    status_code: int = 0
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict:
        """Convert to dictionary."""
        return asdict(self)

    def to_json(self) -> str:
        """Convert to JSON string."""
        return json.dumps(self.to_dict(), default=str)


@dataclass
class MemoryCallRecord:
    """Record of a memory system call."""
    timestamp: str
    call_type: str
    duration_ms: float
    operation: str = ""
    num_items: int = 0
    status: str = "success"
    error: Optional[str] = None
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict:
        """Convert to dictionary."""
        return asdict(self)

    def to_json(self) -> str:
        """Convert to JSON string."""
        return json.dumps(self.to_dict(), default=str)


class BenchmarkLogger:
    """
    Comprehensive logging system for benchmark runs.

    Tracks:
    - LLM API calls with full request/response
    - Memory operations
    - Benchmark execution flow
    - Performance metrics
    """

    def __init__(
        self,
        name: str = "benchmark",
        log_dir: str = "logs",
        level: LogLevel = LogLevel.INFO,
        enable_file: bool = True,
        enable_console: bool = True,
        enable_json: bool = True
    ):
        """
        Initialize logger.

        Args:
            name: Logger name
            log_dir: Directory for log files
            level: Logging level
            enable_file: Enable file logging
            enable_console: Enable console output
            enable_json: Enable JSON logging
        """
        self.name = name
        self.log_dir = Path(log_dir)
        self.level = level
        self.enable_file = enable_file
        self.enable_console = enable_console
        self.enable_json = enable_json

        # Create log directory
        self.log_dir.mkdir(parents=True, exist_ok=True)

        # Initialize loggers
        self.main_logger = logging.getLogger(name)
        self.main_logger.setLevel(level.value)
        self.main_logger.handlers.clear()

        # Call trail
        self.call_trail: List[Dict] = []
        self.llm_calls: List[LLMCallRecord] = []
        self.memory_calls: List[MemoryCallRecord] = []

        # Statistics
        self.stats = {
            "total_llm_calls": 0,
            "total_memory_calls": 0,
            "total_llm_tokens": 0,
            "total_cost": 0.0,
            "start_time": datetime.now().isoformat(),
        }

        # Setup handlers
        self._setup_handlers()

    def _setup_handlers(self):
        """Setup logging handlers."""
        formatter = logging.Formatter(
            '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )

        # Console handler
        if self.enable_console:
            console_handler = logging.StreamHandler(sys.stdout)
            console_handler.setLevel(self.level.value)
            console_handler.setFormatter(formatter)
            self.main_logger.addHandler(console_handler)

        # File handler (standard logs)
        if self.enable_file:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            log_file = self.log_dir / f"benchmark_{timestamp}.log"

            file_handler = logging.handlers.RotatingFileHandler(
                log_file,
                maxBytes=100 * 1024 * 1024,  # 100MB
                backupCount=10
            )
            file_handler.setLevel(self.level.value)
            file_handler.setFormatter(formatter)
            self.main_logger.addHandler(file_handler)

            self.log_file = log_file

        # JSON handler (structured logs)
        if self.enable_json:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            json_file = self.log_dir / f"calls_{timestamp}.jsonl"
            self.json_file = json_file

    def log_llm_call(
        self,
        call_type: CallType,
        duration_ms: float,
        request: Dict[str, Any],
        response: Dict[str, Any] = None,
        error: Optional[str] = None,
        model: str = "",
        tokens: Dict[str, int] = None,
        cost_usd: float = 0.0,
        status_code: int = 0,
        metadata: Dict[str, Any] = None
    ):
        """Log an LLM API call."""
        if tokens is None:
            tokens = {}
        if metadata is None:
            metadata = {}

        record = LLMCallRecord(
            timestamp=datetime.now().isoformat(),
            call_type=call_type.value,
            duration_ms=duration_ms,
            request=request,
            response=response or {},
            error=error,
            model=model,
            input_tokens=tokens.get("input_tokens", 0),
            output_tokens=tokens.get("output_tokens", 0),
            cost_usd=cost_usd,
            status_code=status_code,
            metadata=metadata
        )

        self.llm_calls.append(record)
        self.stats["total_llm_calls"] += 1
        self.stats["total_llm_tokens"] += tokens.get("input_tokens", 0) + tokens.get("output_tokens", 0)
        self.stats["total_cost"] += cost_usd

        # Log to file
        if self.enable_json:
            with open(self.json_file, "a") as f:
                f.write(record.to_json() + "\n")

        # Log to console
        self._log_llm_summary(record)

        # Add to call trail
        self.call_trail.append(record.to_dict())

    def log_memory_call(
        self,
        call_type: CallType,
        operation: str,
        duration_ms: float,
        num_items: int = 0,
        status: str = "success",
        error: Optional[str] = None,
        metadata: Dict[str, Any] = None
    ):
        """Log a memory system call."""
        if metadata is None:
            metadata = {}

        record = MemoryCallRecord(
            timestamp=datetime.now().isoformat(),
            call_type=call_type.value,
            operation=operation,
            duration_ms=duration_ms,
            num_items=num_items,
            status=status,
            error=error,
            metadata=metadata
        )

        self.memory_calls.append(record)
        self.stats["total_memory_calls"] += 1

        # Log to file
        if self.enable_json:
            with open(self.json_file, "a") as f:
                f.write(record.to_json() + "\n")

        # Log to console
        status_icon = "✓" if status == "success" else "❌"
        self.main_logger.info(
            f"{status_icon} Memory {operation}: {num_items} items, {duration_ms:.2f}ms"
        )

        # Add to call trail
        self.call_trail.append(record.to_dict())

    def log_benchmark_event(
        self,
        event_type: CallType,
        message: str,
        metadata: Dict[str, Any] = None
    ):
        """Log a benchmark event."""
        if metadata is None:
            metadata = {}

        log_entry = {
            "timestamp": datetime.now().isoformat(),
            "event_type": event_type.value,
            "message": message,
            "metadata": metadata
        }

        if self.enable_json:
            with open(self.json_file, "a") as f:
                f.write(json.dumps(log_entry) + "\n")

        self.main_logger.info(f"{event_type.value}: {message}")
        self.call_trail.append(log_entry)

    def _log_llm_summary(self, record: LLMCallRecord):
        """Log summary of LLM call."""
        tokens = record.input_tokens + record.output_tokens
        level_map = {
            CallType.LLM_ERROR.value: logging.ERROR,
            CallType.LLM_RESPONSE.value: logging.INFO,
            CallType.LLM_REQUEST.value: logging.DEBUG,
        }
        level = level_map.get(record.call_type, logging.INFO)

        msg = f"LLM ({record.model}): {record.duration_ms:.2f}ms, {tokens} tokens, ${record.cost_usd:.6f}"
        if record.error:
            msg += f" - ERROR: {record.error}"

        self.main_logger.log(level, msg)

    def get_call_trail(self) -> List[Dict]:
        """Get the complete call trail."""
        return self.call_trail.copy()

    def get_llm_calls(self) -> List[Dict]:
        """Get all LLM calls."""
        return [call.to_dict() for call in self.llm_calls]

    def get_memory_calls(self) -> List[Dict]:
        """Get all memory calls."""
        return [call.to_dict() for call in self.memory_calls]

    def get_stats(self) -> Dict:
        """Get execution statistics."""
        self.stats["end_time"] = datetime.now().isoformat()
        return self.stats.copy()

    def save_report(self, output_path: str = None):
        """Save comprehensive report."""
        if output_path is None:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_path = self.log_dir / f"report_{timestamp}.json"

        report = {
            "benchmark_name": self.name,
            "timestamp": datetime.now().isoformat(),
            "statistics": self.get_stats(),
            "llm_calls_summary": {
                "total": len(self.llm_calls),
                "by_type": self._count_by_type(self.llm_calls),
            },
            "memory_calls_summary": {
                "total": len(self.memory_calls),
                "by_type": self._count_by_type(self.memory_calls),
            },
            "call_trail": self.get_call_trail(),
        }

        with open(output_path, "w") as f:
            json.dump(report, f, indent=2, default=str)

        self.main_logger.info(f"Report saved to {output_path}")
        return output_path

    @staticmethod
    def _count_by_type(items: List) -> Dict:
        """Count items by type."""
        counts = {}
        for item in items:
            if hasattr(item, "call_type"):
                call_type = item.call_type
            else:
                call_type = item.get("call_type", "unknown")
            counts[call_type] = counts.get(call_type, 0) + 1
        return counts


def log_llm_call(logger: BenchmarkLogger, model: str = "deepseek-chat"):
    """
    Decorator to automatically log LLM API calls.

    Usage:
        @log_llm_call(logger)
        def call_llm(...):
            ...
    """

    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            start_time = time.time()
            try:
                result = func(*args, **kwargs)
                duration_ms = (time.time() - start_time) * 1000

                # Extract tokens if available
                tokens = {}
                cost = 0.0
                if isinstance(result, tuple) and len(result) > 1:
                    # Check if second element contains metrics
                    if isinstance(result[1], dict):
                        tokens = result[1].get("tokens", {})
                        cost = (
                            (tokens.get("input_tokens", 0) * 0.014 / 1_000_000) +
                            (tokens.get("output_tokens", 0) * 0.056 / 1_000_000)
                        )

                logger.log_llm_call(
                    call_type=CallType.LLM_RESPONSE,
                    duration_ms=duration_ms,
                    request={},
                    response={},
                    model=model,
                    tokens=tokens,
                    cost_usd=cost,
                    status_code=200,
                    metadata={"function": func.__name__}
                )

                return result

            except Exception as e:
                duration_ms = (time.time() - start_time) * 1000
                logger.log_llm_call(
                    call_type=CallType.LLM_ERROR,
                    duration_ms=duration_ms,
                    request={},
                    error=str(e),
                    model=model,
                    status_code=500,
                    metadata={"function": func.__name__, "traceback": traceback.format_exc()}
                )
                raise

        return wrapper

    return decorator


# Global logger instance
_global_logger: Optional[BenchmarkLogger] = None


def get_logger(name: str = "benchmark") -> BenchmarkLogger:
    """Get or create global logger instance."""
    global _global_logger
    if _global_logger is None:
        _global_logger = BenchmarkLogger(name)
    return _global_logger


def init_logger(
    name: str = "benchmark",
    log_dir: str = "logs",
    level: LogLevel = LogLevel.INFO
) -> BenchmarkLogger:
    """Initialize and return logger."""
    global _global_logger
    _global_logger = BenchmarkLogger(name, log_dir, level)
    return _global_logger
