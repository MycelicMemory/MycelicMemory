"""
LLM Call Tracker - Real-Time Monitoring & Debugging

Provides detailed tracking of all LLM API interactions:
- Request/response logging
- Token counting
- Cost tracking
- Error handling
- Call history
- Performance analysis
"""

import time
import json
from typing import Any, Dict, Optional, Tuple
from dataclasses import dataclass
from datetime import datetime
import requests

from .logging_system import BenchmarkLogger, CallType


@dataclass
class APICallMetrics:
    """Metrics for a single API call."""
    call_id: str
    model: str
    timestamp: str
    request_time_ms: float
    response_time_ms: float
    total_time_ms: float
    input_tokens: int
    output_tokens: int
    total_tokens: int
    cost_usd: float
    http_status: int
    success: bool
    error_message: Optional[str] = None


class LLMCallTracker:
    """Track and monitor LLM API calls with full transparency."""

    # DeepSeek pricing (as of 2024)
    PRICING = {
        "input_tokens": 0.014 / 1_000_000,    # $0.014 per 1M tokens
        "output_tokens": 0.056 / 1_000_000,   # $0.056 per 1M tokens
    }

    def __init__(self, logger: Optional[BenchmarkLogger] = None):
        """Initialize tracker."""
        self.logger = logger
        self.call_history: list[APICallMetrics] = []
        self.call_count = 0
        self.total_cost = 0.0
        self.total_tokens = 0

    def call_llm_api(
        self,
        api_key: str,
        model: str,
        messages: list[Dict],
        temperature: float = 0.3,
        max_tokens: int = 100,
        **kwargs
    ) -> Tuple[str, APICallMetrics]:
        """
        Call LLM API with full tracking.

        Args:
            api_key: DeepSeek API key
            model: Model name
            messages: List of message dicts
            temperature: Temperature setting
            max_tokens: Max tokens to generate
            **kwargs: Additional parameters

        Returns:
            Tuple of (response_text, metrics)
        """
        self.call_count += 1
        call_id = f"call_{self.call_count}_{int(time.time()*1000)}"

        # Log request
        request_data = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "max_tokens": max_tokens,
            **kwargs
        }

        if self.logger:
            self.logger.log_benchmark_event(
                CallType.LLM_REQUEST,
                f"LLM call {call_id} initiated",
                {
                    "model": model,
                    "num_messages": len(messages),
                    "max_tokens": max_tokens,
                    "temperature": temperature,
                }
            )

        # Call API
        request_start = time.time()
        try:
            response = requests.post(
                "https://api.deepseek.com/chat/completions",
                headers={
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json"
                },
                json=request_data,
                timeout=30
            )

            request_time_ms = (time.time() - request_start) * 1000
            response.raise_for_status()
            response_data = response.json()

            # Extract response
            response_text = response_data["choices"][0]["message"]["content"]
            usage = response_data["usage"]

            # Calculate metrics
            input_tokens = usage.get("prompt_tokens", 0)
            output_tokens = usage.get("completion_tokens", 0)
            total_tokens = usage.get("total_tokens", 0)
            cost = (
                input_tokens * self.PRICING["input_tokens"] +
                output_tokens * self.PRICING["output_tokens"]
            )

            # Update totals
            self.total_cost += cost
            self.total_tokens += total_tokens

            # Create metrics
            metrics = APICallMetrics(
                call_id=call_id,
                model=model,
                timestamp=datetime.now().isoformat(),
                request_time_ms=request_time_ms,
                response_time_ms=(time.time() - request_start - request_time_ms/1000) * 1000,
                total_time_ms=(time.time() - request_start) * 1000,
                input_tokens=input_tokens,
                output_tokens=output_tokens,
                total_tokens=total_tokens,
                cost_usd=cost,
                http_status=response.status_code,
                success=True
            )

            self.call_history.append(metrics)

            # Log response
            if self.logger:
                self.logger.log_llm_call(
                    call_type=CallType.LLM_RESPONSE,
                    duration_ms=metrics.total_time_ms,
                    request={"model": model, "messages_count": len(messages)},
                    response={"tokens": total_tokens},
                    model=model,
                    tokens={
                        "input_tokens": input_tokens,
                        "output_tokens": output_tokens
                    },
                    cost_usd=cost,
                    status_code=response.status_code,
                    metadata={
                        "call_id": call_id,
                        "response_length": len(response_text),
                    }
                )

            return response_text, metrics

        except Exception as e:
            error_time_ms = (time.time() - request_start) * 1000

            metrics = APICallMetrics(
                call_id=call_id,
                model=model,
                timestamp=datetime.now().isoformat(),
                request_time_ms=error_time_ms,
                response_time_ms=0,
                total_time_ms=error_time_ms,
                input_tokens=0,
                output_tokens=0,
                total_tokens=0,
                cost_usd=0.0,
                http_status=0,
                success=False,
                error_message=str(e)
            )

            self.call_history.append(metrics)

            # Log error
            if self.logger:
                self.logger.log_llm_call(
                    call_type=CallType.LLM_ERROR,
                    duration_ms=error_time_ms,
                    request={"model": model},
                    error=str(e),
                    model=model,
                    status_code=0,
                    metadata={"call_id": call_id}
                )

            raise

    def get_call_history(self) -> list[Dict]:
        """Get all API calls."""
        return [
            {
                "call_id": m.call_id,
                "model": m.model,
                "timestamp": m.timestamp,
                "total_time_ms": m.total_time_ms,
                "input_tokens": m.input_tokens,
                "output_tokens": m.output_tokens,
                "total_tokens": m.total_tokens,
                "cost_usd": m.cost_usd,
                "success": m.success,
                "error": m.error_message,
            }
            for m in self.call_history
        ]

    def get_summary(self) -> Dict[str, Any]:
        """Get summary statistics."""
        successful_calls = [m for m in self.call_history if m.success]
        failed_calls = [m for m in self.call_history if not m.success]

        return {
            "total_calls": len(self.call_history),
            "successful_calls": len(successful_calls),
            "failed_calls": len(failed_calls),
            "total_tokens": self.total_tokens,
            "total_cost_usd": self.total_cost,
            "avg_cost_per_call": self.total_cost / len(self.call_history) if self.call_history else 0,
            "avg_tokens_per_call": self.total_tokens / len(self.call_history) if self.call_history else 0,
            "avg_time_per_call_ms": sum(m.total_time_ms for m in self.call_history) / len(self.call_history) if self.call_history else 0,
            "min_time_ms": min([m.total_time_ms for m in self.call_history], default=0),
            "max_time_ms": max([m.total_time_ms for m in self.call_history], default=0),
        }

    def save_trace(self, output_path: str):
        """Save detailed call trace."""
        trace = {
            "timestamp": datetime.now().isoformat(),
            "summary": self.get_summary(),
            "calls": self.get_call_history(),
        }

        with open(output_path, "w") as f:
            json.dump(trace, f, indent=2, default=str)

        return output_path

    def print_summary(self):
        """Print human-readable summary."""
        summary = self.get_summary()

        print("\n" + "="*70)
        print("LLM CALL SUMMARY")
        print("="*70)

        print(f"\nCall Statistics:")
        print(f"  Total Calls:        {summary['total_calls']}")
        print(f"  Successful:         {summary['successful_calls']}")
        print(f"  Failed:             {summary['failed_calls']}")

        print(f"\nToken Usage:")
        print(f"  Total Tokens:       {summary['total_tokens']:,}")
        print(f"  Avg per Call:       {summary['avg_tokens_per_call']:.0f}")

        print(f"\nCost:")
        print(f"  Total Cost:         ${summary['total_cost_usd']:.6f}")
        print(f"  Avg per Call:       ${summary['avg_cost_per_call']:.6f}")

        print(f"\nTiming:")
        print(f"  Avg Time/Call:      {summary['avg_time_per_call_ms']:.2f}ms")
        print(f"  Min/Max:            {summary['min_time_ms']:.2f}ms / {summary['max_time_ms']:.2f}ms")

        print("\n" + "="*70)
