"""
Python Bridge Server for Ultrathink Benchmark Runner

This FastAPI server bridges the Go benchmark runner with the Python benchmark implementations.
It exposes REST endpoints that the Go runner calls to execute benchmarks.

Endpoints:
- GET  /health   - Health check
- POST /run      - Execute a benchmark run
- GET  /status   - Get progress of a running benchmark
- POST /cancel   - Cancel a running benchmark
"""

import asyncio
import json
import os
import sys
import time
import uuid
import random
from datetime import datetime
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, field
from concurrent.futures import ThreadPoolExecutor
from threading import Lock

from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
import uvicorn

# Add current directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

app = FastAPI(
    title="Ultrathink Benchmark Bridge",
    description="Bridge server for running LoCoMo benchmarks from Go",
    version="1.0.0"
)

# Thread pool for running benchmarks
executor = ThreadPoolExecutor(max_workers=2)

# Active runs tracking
active_runs: Dict[str, "RunState"] = {}
runs_lock = Lock()


@dataclass
class RunState:
    """State of a benchmark run."""
    run_id: str
    status: str = "pending"  # pending, running, completed, failed, cancelled
    progress: int = 0
    total: int = 0
    current_question: str = ""
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    results: Optional[Dict] = None
    error: Optional[str] = None


class BenchmarkRequest(BaseModel):
    """Request to run a benchmark."""
    run_id: str
    benchmark_type: str = "locomo10"  # "locomo10" or "locomo_mc10"
    max_questions: int = 0  # 0 = all questions
    categories: List[str] = []
    random_sample: bool = False
    seed: Optional[int] = None
    verbose: bool = False


class CancelRequest(BaseModel):
    """Request to cancel a benchmark."""
    run_id: str


# ==============================================================================
# Endpoints
# ==============================================================================

@app.get("/health")
async def health():
    """Health check endpoint."""
    return {"status": "ok", "timestamp": datetime.now().isoformat()}


@app.post("/run")
async def run_benchmark(request: BenchmarkRequest, background_tasks: BackgroundTasks):
    """
    Execute a benchmark run.

    This is a blocking endpoint that runs the full benchmark and returns results.
    For long-running benchmarks, consider using async mode (not yet implemented).
    """
    run_id = request.run_id or str(uuid.uuid4())

    # Create run state
    with runs_lock:
        if run_id in active_runs:
            raise HTTPException(status_code=409, detail=f"Run {run_id} already exists")
        active_runs[run_id] = RunState(run_id=run_id, status="running", started_at=datetime.now())

    try:
        # Run the appropriate benchmark
        if request.benchmark_type == "locomo10":
            results = await run_locomo10(
                run_id=run_id,
                max_questions=request.max_questions if request.max_questions > 0 else None,
                categories=request.categories,
                random_sample=request.random_sample,
                seed=request.seed,
                verbose=request.verbose
            )
        elif request.benchmark_type == "locomo_mc10":
            results = await run_locomo_mc10(
                run_id=run_id,
                max_questions=request.max_questions if request.max_questions > 0 else None,
                categories=request.categories,
                random_sample=request.random_sample,
                seed=request.seed,
                verbose=request.verbose
            )
        else:
            raise HTTPException(
                status_code=400,
                detail=f"Unknown benchmark type: {request.benchmark_type}"
            )

        # Update run state
        with runs_lock:
            if run_id in active_runs:
                active_runs[run_id].status = "completed"
                active_runs[run_id].completed_at = datetime.now()
                active_runs[run_id].results = results

        return {"success": True, "results": results}

    except Exception as e:
        with runs_lock:
            if run_id in active_runs:
                active_runs[run_id].status = "failed"
                active_runs[run_id].error = str(e)
                active_runs[run_id].completed_at = datetime.now()

        return {"success": False, "error": str(e)}


@app.get("/status")
async def get_status(run_id: str):
    """Get the status of a benchmark run."""
    with runs_lock:
        if run_id not in active_runs:
            raise HTTPException(status_code=404, detail=f"Run {run_id} not found")

        state = active_runs[run_id]
        return {
            "run_id": state.run_id,
            "status": state.status,
            "progress": state.progress,
            "total": state.total,
            "current_question": state.current_question,
            "started_at": state.started_at.isoformat() if state.started_at else None,
            "completed_at": state.completed_at.isoformat() if state.completed_at else None,
            "error": state.error
        }


@app.post("/cancel")
async def cancel_benchmark(request: CancelRequest):
    """Cancel a running benchmark."""
    with runs_lock:
        if request.run_id not in active_runs:
            raise HTTPException(status_code=404, detail=f"Run {request.run_id} not found")

        state = active_runs[request.run_id]
        if state.status == "running":
            state.status = "cancelled"
            state.completed_at = datetime.now()
            return {"success": True, "message": f"Run {request.run_id} cancelled"}
        else:
            return {"success": False, "message": f"Run {request.run_id} is not running (status: {state.status})"}


# ==============================================================================
# Benchmark Runners
# ==============================================================================

async def run_locomo10(
    run_id: str,
    max_questions: Optional[int],
    categories: List[str],
    random_sample: bool,
    seed: Optional[int],
    verbose: bool
) -> Dict[str, Any]:
    """Run the LoCoMo free-response benchmark."""
    from locomo10.main import FreeResponseExperiment
    from locomo10.f1_evaluator import get_category_name

    # Run in thread pool to not block event loop
    loop = asyncio.get_event_loop()

    def _run():
        # Create experiment with options
        experiment = FreeResponseExperiment(
            dataset_path="data/locomo10.json",
            max_questions=max_questions,
            ultrathink_url=os.getenv("ULTRATHINK_URL", "http://localhost:3099/api/v1"),
            enable_logging=verbose
        )

        # Apply random sampling if requested
        if random_sample and max_questions:
            rng = random.Random(seed) if seed else random.Random()
            total_available = len(experiment.questions)
            sample_size = min(max_questions, total_available)
            experiment.questions = rng.sample(experiment.questions, sample_size)

        # Update progress tracking
        with runs_lock:
            if run_id in active_runs:
                active_runs[run_id].total = len(experiment.questions)

        # Run the benchmark
        output_path = f"results/bridge_{run_id}.json"
        summary = experiment.run(output_path=output_path)

        # Convert to bridge response format
        return convert_fr_results(summary)

    return await loop.run_in_executor(executor, _run)


async def run_locomo_mc10(
    run_id: str,
    max_questions: Optional[int],
    categories: List[str],
    random_sample: bool,
    seed: Optional[int],
    verbose: bool
) -> Dict[str, Any]:
    """Run the LoCoMo multiple-choice benchmark."""
    from locomo_mc10.main import MemoryAugmentedExperiment

    loop = asyncio.get_event_loop()

    def _run():
        # Create experiment with options
        experiment = MemoryAugmentedExperiment(
            dataset_path="data/locomo-mc10-full.jsonl",
            max_questions=max_questions,
            ultrathink_url=os.getenv("ULTRATHINK_URL", "http://localhost:3099/api/v1"),
            enable_logging=verbose
        )

        # Apply random sampling if requested
        if random_sample and max_questions:
            rng = random.Random(seed) if seed else random.Random()
            total_available = len(experiment.questions)
            sample_size = min(max_questions, total_available)
            experiment.questions = rng.sample(experiment.questions, sample_size)

        # Update progress tracking
        with runs_lock:
            if run_id in active_runs:
                active_runs[run_id].total = len(experiment.questions)

        # Run the benchmark
        output_path = f"results/bridge_{run_id}.json"
        summary = experiment.run(output_path=output_path)

        # Convert to bridge response format
        return convert_mc_results(summary)

    return await loop.run_in_executor(executor, _run)


# ==============================================================================
# Result Converters
# ==============================================================================

def convert_fr_results(summary: Dict) -> Dict[str, Any]:
    """Convert free-response results to bridge response format."""
    results = summary.get("results", {})

    # Calculate overall metrics
    all_f1_scores = []
    by_category: Dict[str, Dict] = {}
    questions = []

    total_input_tokens = 0
    total_output_tokens = 0
    latencies = []

    for q_id, result in results.items():
        f1 = result.get("f1_score", 0)
        all_f1_scores.append(f1)

        category = result.get("category_name", "unknown")
        if category not in by_category:
            by_category[category] = {"scores": [], "count": 0}
        by_category[category]["scores"].append(f1)
        by_category[category]["count"] += 1

        # Token tracking
        tokens = result.get("tokens", {})
        total_input_tokens += tokens.get("input_tokens", 0)
        total_output_tokens += tokens.get("output_tokens", 0)

        # Latency tracking
        latencies.append(result.get("latency_total", 0))

        questions.append({
            "id": q_id,
            "category": category,
            "question": result.get("question", ""),
            "gold_answer": result.get("ground_truth", ""),
            "generated_answer": result.get("prediction", ""),
            "llm_judge_label": 1 if f1 > 0.5 else 0,  # Approximate
            "f1_score": f1,
            "bleu1_score": 0.0  # Not computed for FR
        })

    # Compute overall stats
    mean_f1 = sum(all_f1_scores) / len(all_f1_scores) if all_f1_scores else 0

    # Build category breakdown
    category_results = {}
    for cat, data in by_category.items():
        cat_mean = sum(data["scores"]) / len(data["scores"]) if data["scores"] else 0
        category_results[cat] = {
            "llm_judge_accuracy": cat_mean * 100,  # Convert to percentage
            "f1_score": cat_mean,
            "bleu1_score": 0.0,
            "count": data["count"],
            "correct": sum(1 for s in data["scores"] if s > 0.5)
        }

    # Latency stats
    latency_stats = None
    if latencies:
        sorted_lat = sorted(latencies)
        latency_stats = {
            "mean_latency_seconds": sum(latencies) / len(latencies),
            "median_latency_seconds": sorted_lat[len(sorted_lat) // 2],
            "p95_latency_seconds": sorted_lat[int(len(sorted_lat) * 0.95)] if len(sorted_lat) > 20 else max(sorted_lat),
            "p99_latency_seconds": sorted_lat[int(len(sorted_lat) * 0.99)] if len(sorted_lat) > 100 else max(sorted_lat),
            "min_latency_seconds": min(latencies),
            "max_latency_seconds": max(latencies),
            "stdev_latency_seconds": _stdev(latencies)
        }

    # Token stats
    total_tokens = total_input_tokens + total_output_tokens
    num_questions = len(results)
    token_stats = {
        "total_input_tokens": total_input_tokens,
        "total_output_tokens": total_output_tokens,
        "total_tokens": total_tokens,
        "mean_input_tokens": total_input_tokens / num_questions if num_questions else 0,
        "mean_output_tokens": total_output_tokens / num_questions if num_questions else 0
    }

    # Cost estimation (DeepSeek pricing)
    cost_stats = {
        "input_cost_usd": total_input_tokens * 0.014 / 1_000_000,
        "output_cost_usd": total_output_tokens * 0.056 / 1_000_000,
        "total_cost_usd": (total_input_tokens * 0.014 + total_output_tokens * 0.056) / 1_000_000,
        "cost_per_question_usd": ((total_input_tokens * 0.014 + total_output_tokens * 0.056) / 1_000_000) / num_questions if num_questions else 0
    }

    return {
        "overall": {
            "llm_judge_accuracy": mean_f1 * 100,
            "f1_score": mean_f1,
            "bleu1_score": 0.0,
            "total_questions": num_questions,
            "total_correct": sum(1 for s in all_f1_scores if s > 0.5)
        },
        "by_category": category_results,
        "questions": questions,
        "latency": latency_stats,
        "tokens": token_stats,
        "cost_estimation": cost_stats,
        "duration_seconds": summary.get("total_time_seconds", 0)
    }


def convert_mc_results(summary: Dict) -> Dict[str, Any]:
    """Convert multiple-choice results to bridge response format."""
    results = summary.get("results", {})

    # Calculate metrics
    correct_count = 0
    by_category: Dict[str, Dict] = {}
    questions = []

    total_input_tokens = 0
    total_output_tokens = 0
    latencies = []

    for q_id, result in results.items():
        is_correct = result.get("is_correct", False)
        if is_correct:
            correct_count += 1

        category = result.get("question_type", "unknown")
        if category not in by_category:
            by_category[category] = {"correct": 0, "total": 0}
        by_category[category]["total"] += 1
        if is_correct:
            by_category[category]["correct"] += 1

        # Token tracking
        tokens = result.get("tokens", {})
        total_input_tokens += tokens.get("input_tokens", 0)
        total_output_tokens += tokens.get("output_tokens", 0)

        # Latency tracking
        latencies.append(result.get("latency_total", 0))

        questions.append({
            "id": q_id,
            "category": category,
            "question": result.get("question", ""),
            "gold_answer": str(result.get("correct_choice_index", "")),
            "generated_answer": str(result.get("predicted_choice_index", "")),
            "llm_judge_label": 1 if is_correct else 0,
            "f1_score": 1.0 if is_correct else 0.0,
            "bleu1_score": 0.0
        })

    num_questions = len(results)
    accuracy = correct_count / num_questions if num_questions else 0

    # Build category breakdown
    category_results = {}
    for cat, data in by_category.items():
        cat_acc = data["correct"] / data["total"] if data["total"] else 0
        category_results[cat] = {
            "llm_judge_accuracy": cat_acc * 100,
            "f1_score": cat_acc,
            "bleu1_score": 0.0,
            "count": data["total"],
            "correct": data["correct"]
        }

    # Latency stats
    latency_stats = None
    if latencies:
        sorted_lat = sorted(latencies)
        latency_stats = {
            "mean_latency_seconds": sum(latencies) / len(latencies),
            "median_latency_seconds": sorted_lat[len(sorted_lat) // 2],
            "p95_latency_seconds": sorted_lat[int(len(sorted_lat) * 0.95)] if len(sorted_lat) > 20 else max(sorted_lat),
            "p99_latency_seconds": sorted_lat[int(len(sorted_lat) * 0.99)] if len(sorted_lat) > 100 else max(sorted_lat),
            "min_latency_seconds": min(latencies),
            "max_latency_seconds": max(latencies),
            "stdev_latency_seconds": _stdev(latencies)
        }

    # Token stats
    total_tokens = total_input_tokens + total_output_tokens
    token_stats = {
        "total_input_tokens": total_input_tokens,
        "total_output_tokens": total_output_tokens,
        "total_tokens": total_tokens,
        "mean_input_tokens": total_input_tokens / num_questions if num_questions else 0,
        "mean_output_tokens": total_output_tokens / num_questions if num_questions else 0
    }

    # Cost estimation
    cost_stats = {
        "input_cost_usd": total_input_tokens * 0.014 / 1_000_000,
        "output_cost_usd": total_output_tokens * 0.056 / 1_000_000,
        "total_cost_usd": (total_input_tokens * 0.014 + total_output_tokens * 0.056) / 1_000_000,
        "cost_per_question_usd": ((total_input_tokens * 0.014 + total_output_tokens * 0.056) / 1_000_000) / num_questions if num_questions else 0
    }

    return {
        "overall": {
            "llm_judge_accuracy": accuracy * 100,
            "f1_score": accuracy,
            "bleu1_score": 0.0,
            "total_questions": num_questions,
            "total_correct": correct_count
        },
        "by_category": category_results,
        "questions": questions,
        "latency": latency_stats,
        "tokens": token_stats,
        "cost_estimation": cost_stats,
        "duration_seconds": summary.get("total_time_seconds", 0)
    }


def _stdev(values: List[float]) -> float:
    """Calculate standard deviation."""
    if len(values) < 2:
        return 0.0
    mean = sum(values) / len(values)
    variance = sum((x - mean) ** 2 for x in values) / (len(values) - 1)
    return variance ** 0.5


# ==============================================================================
# Main
# ==============================================================================

def main():
    """Run the bridge server."""
    import argparse

    parser = argparse.ArgumentParser(description="Ultrathink Benchmark Bridge Server")
    parser.add_argument("--host", default="0.0.0.0", help="Host to bind to")
    parser.add_argument("--port", type=int, default=9876, help="Port to listen on")
    parser.add_argument("--reload", action="store_true", help="Enable auto-reload")

    args = parser.parse_args()

    print(f"Starting Ultrathink Benchmark Bridge on {args.host}:{args.port}")
    uvicorn.run(
        "bridge_server:app",
        host=args.host,
        port=args.port,
        reload=args.reload
    )


if __name__ == "__main__":
    main()
