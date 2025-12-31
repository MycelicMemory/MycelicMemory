"""
HTTP Bridge Server for MCP Benchmark Integration.

Provides HTTP endpoints for the Go MCP server to run benchmarks:
- GET /health - Health check
- POST /run - Start benchmark execution
- GET /status - Get progress of running benchmark
- POST /cancel - Cancel running benchmark
- GET /results/<run_id> - Get results for a specific run
"""

import json
import os
import sys
import threading
import time
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Dict, List, Optional
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import parse_qs, urlparse

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from runner import BenchmarkRunner


@dataclass
class RunState:
    """Tracks state of a running benchmark."""
    run_id: str
    status: str = "pending"  # pending, running, completed, failed, cancelled
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    total_questions: int = 0
    completed_count: int = 0
    current_question: str = ""
    results: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    cancelled: bool = False


class BenchmarkBridge:
    """Manages benchmark execution state."""

    def __init__(self):
        self.runs: Dict[str, RunState] = {}
        self.active_run_id: Optional[str] = None
        self.runner = BenchmarkRunner()
        self.lock = threading.Lock()

    def start_run(self, run_id: str, max_questions: int = 0, categories: List[str] = None, verbose: bool = False) -> RunState:
        """Start a new benchmark run."""
        with self.lock:
            if self.active_run_id is not None:
                raise RuntimeError(f"A benchmark is already running: {self.active_run_id}")

            state = RunState(
                run_id=run_id,
                status="running",
                started_at=datetime.now()
            )
            self.runs[run_id] = state
            self.active_run_id = run_id

        # Start in background thread
        thread = threading.Thread(
            target=self._run_benchmark,
            args=(run_id, max_questions, categories, verbose)
        )
        thread.daemon = True
        thread.start()

        return state

    def _run_benchmark(self, run_id: str, max_questions: int, categories: List[str], verbose: bool):
        """Run benchmark in background thread."""
        state = self.runs[run_id]

        try:
            # Progress callback
            def on_progress(current: int, total: int, question: str):
                state.completed_count = current
                state.total_questions = total
                state.current_question = question
                if state.cancelled:
                    raise InterruptedError("Benchmark cancelled")

            # Run the benchmark
            results = self.runner.run(
                max_questions=max_questions,
                categories=categories,
                verbose=verbose,
                on_progress=on_progress
            )

            state.results = results
            state.status = "completed"
            state.completed_at = datetime.now()

        except InterruptedError:
            state.status = "cancelled"
            state.completed_at = datetime.now()

        except Exception as e:
            state.status = "failed"
            state.error = str(e)
            state.completed_at = datetime.now()

        finally:
            with self.lock:
                self.active_run_id = None

    def get_status(self, run_id: str) -> Optional[RunState]:
        """Get status of a run."""
        return self.runs.get(run_id)

    def cancel(self, run_id: str) -> bool:
        """Cancel a running benchmark."""
        state = self.runs.get(run_id)
        if state and state.status == "running":
            state.cancelled = True
            return True
        return False


# Global bridge instance
bridge = BenchmarkBridge()


class BridgeHandler(BaseHTTPRequestHandler):
    """HTTP request handler for benchmark bridge."""

    def _send_json(self, data: Dict, status: int = 200):
        """Send JSON response."""
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode("utf-8"))

    def _parse_json_body(self) -> Dict:
        """Parse JSON from request body."""
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length == 0:
            return {}
        body = self.rfile.read(content_length)
        return json.loads(body.decode("utf-8"))

    def do_GET(self):
        """Handle GET requests."""
        parsed = urlparse(self.path)
        path = parsed.path

        if path == "/health":
            self._send_json({"status": "ok", "service": "benchmark-bridge"})

        elif path == "/status":
            query = parse_qs(parsed.query)
            run_id = query.get("run_id", [None])[0]

            if run_id is None:
                # Return active run status
                if bridge.active_run_id:
                    run_id = bridge.active_run_id
                else:
                    self._send_json({"error": "No active run"}, 404)
                    return

            state = bridge.get_status(run_id)
            if state is None:
                self._send_json({"error": "Run not found"}, 404)
                return

            self._send_json({
                "run_id": state.run_id,
                "status": state.status,
                "total_questions": state.total_questions,
                "completed_count": state.completed_count,
                "current_question": state.current_question,
                "percent_complete": (
                    state.completed_count / state.total_questions * 100
                    if state.total_questions > 0 else 0
                ),
                "elapsed_seconds": (
                    (state.completed_at or datetime.now()) - state.started_at
                ).total_seconds() if state.started_at else 0
            })

        elif path.startswith("/results/"):
            run_id = path.split("/")[-1]
            state = bridge.get_status(run_id)

            if state is None:
                self._send_json({"error": "Run not found"}, 404)
                return

            if state.results is None:
                self._send_json({"error": "Results not yet available"}, 404)
                return

            self._send_json(state.results)

        else:
            self._send_json({"error": "Not found"}, 404)

    def do_POST(self):
        """Handle POST requests."""
        parsed = urlparse(self.path)
        path = parsed.path

        if path == "/run":
            try:
                data = self._parse_json_body()
                run_id = data.get("run_id")
                max_questions = data.get("max_questions", 0)
                categories = data.get("categories")
                verbose = data.get("verbose", False)

                if not run_id:
                    self._send_json({"error": "run_id is required"}, 400)
                    return

                state = bridge.start_run(run_id, max_questions, categories, verbose)
                self._send_json({
                    "success": True,
                    "run_id": state.run_id,
                    "status": state.status
                })

            except RuntimeError as e:
                self._send_json({"success": False, "error": str(e)}, 409)

            except Exception as e:
                self._send_json({"success": False, "error": str(e)}, 500)

        elif path == "/run-sync":
            # Synchronous run - blocks until complete
            try:
                data = self._parse_json_body()
                run_id = data.get("run_id", f"sync-{datetime.now().isoformat()}")
                max_questions = data.get("max_questions", 0)
                categories = data.get("categories")
                verbose = data.get("verbose", False)

                # Run synchronously
                runner = BenchmarkRunner()
                results = runner.run(
                    max_questions=max_questions,
                    categories=categories,
                    verbose=verbose
                )

                self._send_json({
                    "success": True,
                    "run_id": run_id,
                    "results": results
                })

            except Exception as e:
                self._send_json({"success": False, "error": str(e)}, 500)

        elif path == "/cancel":
            try:
                data = self._parse_json_body()
                run_id = data.get("run_id")

                if not run_id:
                    run_id = bridge.active_run_id

                if not run_id:
                    self._send_json({"error": "No active run to cancel"}, 400)
                    return

                if bridge.cancel(run_id):
                    self._send_json({"success": True, "run_id": run_id})
                else:
                    self._send_json({"error": "Run not found or not running"}, 404)

            except Exception as e:
                self._send_json({"success": False, "error": str(e)}, 500)

        else:
            self._send_json({"error": "Not found"}, 404)

    def log_message(self, format, *args):
        """Suppress default logging."""
        pass


def run_server(host: str = "localhost", port: int = 9876):
    """Run the HTTP server."""
    server = HTTPServer((host, port), BridgeHandler)
    print(f"Benchmark bridge server running on http://{host}:{port}")
    print("Endpoints:")
    print("  GET  /health        - Health check")
    print("  POST /run           - Start async benchmark")
    print("  POST /run-sync      - Run benchmark synchronously")
    print("  GET  /status        - Get benchmark progress")
    print("  POST /cancel        - Cancel running benchmark")
    print("  GET  /results/<id>  - Get results for a run")
    print()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down...")
        server.shutdown()


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description="Benchmark Bridge Server")
    parser.add_argument("--host", default="localhost", help="Host to bind to")
    parser.add_argument("--port", type=int, default=9876, help="Port to listen on")
    args = parser.parse_args()

    run_server(args.host, args.port)
