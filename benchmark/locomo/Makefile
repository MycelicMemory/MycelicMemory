# LoCoMo Benchmark Makefile
# Adapted from mem0ai/mem0 evaluation

.PHONY: setup download run-quick run-full evaluate scores clean help server

# Default DeepSeek API key (override with: make run DEEPSEEK_API_KEY=your_key)
export DEEPSEEK_API_KEY ?= REDACTED_API_KEY

help:
	@echo "LoCoMo Benchmark Commands:"
	@echo ""
	@echo "  make setup        - Install dependencies"
	@echo "  make download     - Download the LoCoMo-MC10 dataset"
	@echo "  make run-quick    - Run quick test (20 questions)"
	@echo "  make run-full     - Run full benchmark (all questions)"
	@echo "  make evaluate     - Evaluate results with LLM judge"
	@echo "  make scores       - Generate score summary"
	@echo "  make all          - Run full pipeline (download -> run -> evaluate -> scores)"
	@echo "  make server       - Start MCP bridge HTTP server (port 9876)"
	@echo "  make clean        - Remove results and cache"
	@echo ""

setup:
	pip install -r requirements.txt
	python -c "import nltk; nltk.download('punkt'); nltk.download('punkt_tab'); nltk.download('wordnet')"

download:
	python run_experiments.py --download --max_questions 1

run-quick:
	python run_experiments.py \
		--dataset_path dataset/locomo10.json \
		--output_path results/ultrathink_results_quick.json \
		--max_questions 20 \
		--download

run-full:
	python run_experiments.py \
		--dataset_path dataset/locomo10.json \
		--output_path results/ultrathink_results.json \
		--download

run-summaries:
	python run_experiments.py \
		--dataset_path dataset/locomo10.json \
		--output_path results/ultrathink_results_summaries.json \
		--use_summaries \
		--download

evaluate:
	python evals.py \
		--input_file results/ultrathink_results.json \
		--output_file results/evaluation_metrics.json

evaluate-quick:
	python evals.py \
		--input_file results/ultrathink_results_quick.json \
		--output_file results/evaluation_metrics_quick.json

scores:
	python generate_scores.py \
		--input_file results/evaluation_metrics.json \
		--output_file results/scores_summary.json

scores-quick:
	python generate_scores.py \
		--input_file results/evaluation_metrics_quick.json \
		--output_file results/scores_summary_quick.json

all: download run-full evaluate scores

quick: run-quick evaluate-quick scores-quick

# MCP Bridge Server (for Go integration)
server:
	python -m mcp_bridge.server --host localhost --port 9876

clean:
	rm -rf results/*.json
	rm -rf __pycache__ metrics/__pycache__
