"""
Phoenix (Arize) integration for ultrathink benchmarks.
Local-first LLM observability with web UI.

Phoenix provides:
- Trace visualization at localhost:6006
- Latency breakdown (memory retrieval vs LLM)
- Token analytics
- Eval score filtering
- No cloud required, fully local

Usage:
    tracer = PhoenixTracer(launch_ui=True)

    with tracer.trace("benchmark-run") as ctx:
        with tracer.span(ctx, "question-1") as q_ctx:
            tracer.log_retrieval(q_ctx, query="...", num_results=10, latency_ms=50)
            tracer.log_llm_call(q_ctx, model="deepseek", prompt="...", response="...", ...)
            tracer.log_eval(q_ctx, "f1_score", 0.85)
"""
import os
import time
from typing import Dict, Optional, Any
from contextlib import contextmanager
from dataclasses import dataclass

# Phoenix imports with graceful fallback
try:
    import phoenix as px
    from opentelemetry import trace as otel_trace
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import SimpleSpanProcessor
    PHOENIX_AVAILABLE = True
except ImportError:
    PHOENIX_AVAILABLE = False
    px = None
    otel_trace = None

# OpenInference semantic conventions (optional)
try:
    from openinference.semconv.trace import SpanAttributes
    OPENINFERENCE_AVAILABLE = True
except ImportError:
    OPENINFERENCE_AVAILABLE = False
    SpanAttributes = None


@dataclass
class TraceContext:
    """Context for a trace with spans."""
    trace_id: Any
    span: Any
    start_time: float
    name: str


class PhoenixTracer:
    """
    Wrapper for Phoenix tracing with graceful degradation.

    If Phoenix is not installed or disabled, all methods become no-ops.
    This allows the benchmark to run without Phoenix dependencies.
    """

    def __init__(self, enabled: bool = True, launch_ui: bool = False):
        """
        Initialize Phoenix tracer.

        Args:
            enabled: Whether to enable tracing (default True)
            launch_ui: Whether to launch the Phoenix UI on init (default False)
        """
        self.enabled = enabled and PHOENIX_AVAILABLE
        self.session = None
        self.tracer = None
        self._initialized = False
        self._phoenix_url = ""

        if self.enabled and launch_ui:
            self._initialize(launch_ui=True)

    def _is_phoenix_running(self, port: int = 6006) -> bool:
        """Check if Phoenix is already running on the given port."""
        import socket
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                s.settimeout(1)
                result = s.connect_ex(('localhost', port))
                return result == 0
        except Exception:
            return False

    def _initialize(self, launch_ui: bool = False):
        """Initialize Phoenix and OpenTelemetry."""
        if self._initialized:
            return

        try:
            if launch_ui:
                # Check if Phoenix is already running
                if self._is_phoenix_running():
                    print("Phoenix already running at http://localhost:6006/")
                    self._phoenix_url = "http://localhost:6006/"
                else:
                    # Launch Phoenix UI in background
                    self.session = px.launch_app()
                    self._phoenix_url = str(self.session.url) if self.session else ""

            # Set up OpenTelemetry tracer with Phoenix
            from phoenix.otel import register
            tracer_provider = register(project_name="ultrathink-benchmark")
            self.tracer = otel_trace.get_tracer("ultrathink.benchmark")
            self._initialized = True

        except Exception as e:
            print(f"Warning: Failed to initialize Phoenix: {e}")
            self.enabled = False

    def launch_ui(self) -> str:
        """
        Launch Phoenix UI and return URL.

        Returns:
            URL of Phoenix UI (e.g., "http://localhost:6006") or empty string
        """
        if not self.enabled:
            return ""

        if not self._initialized:
            self._initialize(launch_ui=True)

        # Return URL from session or cached URL (if Phoenix was already running)
        if self.session:
            return str(self.session.url)
        return self._phoenix_url

    @contextmanager
    def trace(self, name: str, metadata: Dict = None):
        """
        Create a root trace context.

        Args:
            name: Name of the trace (e.g., "locomo10-benchmark")
            metadata: Optional metadata dict

        Yields:
            TraceContext or None if disabled
        """
        if not self.enabled:
            yield None
            return

        if not self._initialized:
            self._initialize()

        if not self.tracer:
            yield None
            return

        with self.tracer.start_as_current_span(name) as span:
            # Set metadata as span attributes
            if metadata:
                for k, v in metadata.items():
                    span.set_attribute(f"metadata.{k}", str(v))

            ctx = TraceContext(
                trace_id=span.get_span_context().trace_id if span.get_span_context() else None,
                span=span,
                start_time=time.time(),
                name=name
            )
            try:
                yield ctx
            finally:
                # Record duration
                duration = time.time() - ctx.start_time
                span.set_attribute("duration_seconds", duration)

    @contextmanager
    def span(self, parent_ctx: Optional[TraceContext], name: str, metadata: Dict = None):
        """
        Create a child span within a trace.

        Args:
            parent_ctx: Parent trace context (can be None)
            name: Name of the span (e.g., "question-1", "memory-retrieval")
            metadata: Optional metadata dict

        Yields:
            TraceContext or None if disabled
        """
        if not self.enabled or not self.tracer:
            yield None
            return

        with self.tracer.start_as_current_span(name) as span:
            if metadata:
                for k, v in metadata.items():
                    span.set_attribute(f"metadata.{k}", str(v))

            ctx = TraceContext(
                trace_id=parent_ctx.trace_id if parent_ctx else None,
                span=span,
                start_time=time.time(),
                name=name
            )
            try:
                yield ctx
            finally:
                duration = time.time() - ctx.start_time
                span.set_attribute("duration_seconds", duration)

    def log_retrieval(self, ctx: Optional[TraceContext], query: str,
                      num_results: int, latency_ms: float):
        """
        Log a memory retrieval operation.

        Args:
            ctx: Trace context
            query: Search query
            num_results: Number of results retrieved
            latency_ms: Retrieval latency in milliseconds
        """
        if not self.enabled or not ctx or not ctx.span:
            return

        # Use OpenInference semantic conventions if available
        if OPENINFERENCE_AVAILABLE and SpanAttributes:
            ctx.span.set_attribute(SpanAttributes.INPUT_VALUE, query)
        else:
            ctx.span.set_attribute("retrieval.query", query)

        ctx.span.set_attribute("retrieval.num_results", num_results)
        ctx.span.set_attribute("retrieval.latency_ms", latency_ms)

    def log_llm_call(self, ctx: Optional[TraceContext], model: str, prompt: str,
                     response: str, input_tokens: int, output_tokens: int,
                     latency_ms: float, cost_usd: float = 0.0):
        """
        Log an LLM generation.

        Args:
            ctx: Trace context
            model: Model name (e.g., "deepseek-chat")
            prompt: Input prompt (truncated for storage)
            response: LLM response (truncated for storage)
            input_tokens: Number of input tokens
            output_tokens: Number of output tokens
            latency_ms: LLM latency in milliseconds
            cost_usd: Estimated cost in USD
        """
        if not self.enabled or not ctx or not ctx.span:
            return

        # Use OpenInference semantic conventions if available
        if OPENINFERENCE_AVAILABLE and SpanAttributes:
            ctx.span.set_attribute(SpanAttributes.LLM_MODEL_NAME, model)
            ctx.span.set_attribute(SpanAttributes.INPUT_VALUE, prompt[:2000])
            ctx.span.set_attribute(SpanAttributes.OUTPUT_VALUE, response[:2000])
            ctx.span.set_attribute(SpanAttributes.LLM_TOKEN_COUNT_PROMPT, input_tokens)
            ctx.span.set_attribute(SpanAttributes.LLM_TOKEN_COUNT_COMPLETION, output_tokens)
        else:
            ctx.span.set_attribute("llm.model", model)
            ctx.span.set_attribute("llm.prompt", prompt[:2000])
            ctx.span.set_attribute("llm.response", response[:2000])
            ctx.span.set_attribute("llm.input_tokens", input_tokens)
            ctx.span.set_attribute("llm.output_tokens", output_tokens)

        ctx.span.set_attribute("llm.latency_ms", latency_ms)
        ctx.span.set_attribute("llm.cost_usd", cost_usd)
        ctx.span.set_attribute("llm.total_tokens", input_tokens + output_tokens)

    def log_eval(self, ctx: Optional[TraceContext], name: str, value: float,
                 comment: str = None):
        """
        Log an evaluation score.

        Args:
            ctx: Trace context
            name: Eval name (e.g., "f1_score", "accuracy")
            value: Eval value (0.0 to 1.0)
            comment: Optional comment
        """
        if not self.enabled or not ctx or not ctx.span:
            return

        ctx.span.set_attribute(f"eval.{name}", value)
        if comment:
            ctx.span.set_attribute(f"eval.{name}.comment", comment)

    def log_error(self, ctx: Optional[TraceContext], error: str, error_type: str = None):
        """
        Log an error.

        Args:
            ctx: Trace context
            error: Error message
            error_type: Optional error type/category
        """
        if not self.enabled or not ctx or not ctx.span:
            return

        ctx.span.set_attribute("error", True)
        ctx.span.set_attribute("error.message", error[:1000])
        if error_type:
            ctx.span.set_attribute("error.type", error_type)

    def close(self):
        """Cleanup resources."""
        # Phoenix handles cleanup automatically
        pass

    @property
    def is_available(self) -> bool:
        """Check if Phoenix is available and enabled."""
        return self.enabled and PHOENIX_AVAILABLE
