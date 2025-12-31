# LoCoMo-MC10 Memory-Augmented Benchmark
# Tests ultrathink's memory retrieval against the LoCoMo-MC10 dataset

.PHONY: help setup download run run-quick evaluate clean

# Configuration
PYTHON ?= python3
ULTRATHINK_URL ?= http://localhost:3099/api/v1
DATASET_PATH ?= data/locomo-mc10-full.jsonl
RESULTS_DIR ?= results

help:
	@echo "LoCoMo-MC10 Memory-Augmented Benchmark"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - ultrathink server running on port 3099"
	@echo "  - DEEPSEEK_API_KEY set (copy .env.example to .env)"
	@echo ""
	@echo "Commands:"
	@echo "  make setup      - Install Python dependencies"
	@echo "  make download   - Download LoCoMo-MC10 dataset"
	@echo "  make run-quick  - Run benchmark (10 questions)"
	@echo "  make run        - Run full benchmark (all questions)"
	@echo "  make evaluate   - Generate evaluation metrics"
	@echo "  make clean      - Remove results and logs"
	@echo ""

setup:
	pip3 install -r requirements.txt
	$(PYTHON) -c "import nltk; nltk.download('punkt'); nltk.download('punkt_tab'); nltk.download('wordnet')"

download:
	$(PYTHON) download_dataset.py

run-quick:
	$(PYTHON) memory_augmented.py \
		--dataset $(DATASET_PATH) \
		--output $(RESULTS_DIR)/memory_augmented_results.json \
		--max-questions 10 \
		--ultrathink-url $(ULTRATHINK_URL)

run:
	$(PYTHON) memory_augmented.py \
		--dataset $(DATASET_PATH) \
		--output $(RESULTS_DIR)/memory_augmented_results.json \
		--ultrathink-url $(ULTRATHINK_URL)

evaluate:
	$(PYTHON) comprehensive_evals.py \
		--input_file $(RESULTS_DIR)/memory_augmented_results.json \
		--output_file $(RESULTS_DIR)/evaluation_metrics.json

clean:
	rm -rf $(RESULTS_DIR)/*.json
	rm -rf logs/*.log logs/*.jsonl logs/*.json
	rm -rf __pycache__ metrics/__pycache__
