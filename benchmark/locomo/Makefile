# LoCoMo Memory-Augmented Benchmark
# Tests ultrathink's memory retrieval against the LoCoMo datasets

.PHONY: help setup download-mc download-fr run-mc run-mc-quick run-fr run-fr-quick evaluate-mc clean

# Configuration
PYTHON ?= python3
ULTRATHINK_URL ?= http://localhost:3099/api/v1
RESULTS_DIR ?= results

# Dataset paths
MC_DATASET ?= data/locomo-mc10-full.jsonl
FR_DATASET ?= data/locomo10.json

help:
	@echo "LoCoMo Memory-Augmented Benchmark"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - ultrathink server running on port 3099"
	@echo "  - DEEPSEEK_API_KEY set (copy .env.example to .env)"
	@echo ""
	@echo "Free Response (locomo10) Benchmark:"
	@echo "  make download-fr    - Download LoCoMo free-response dataset"
	@echo "  make run-fr-quick   - Run FR benchmark (20 questions)"
	@echo "  make run-fr         - Run full FR benchmark (1986 questions)"
	@echo ""
	@echo "Multiple Choice (locomo_mc10) Benchmark:"
	@echo "  make download-mc    - Download LoCoMo-MC10 dataset"
	@echo "  make run-mc-quick   - Run MC benchmark (10 questions)"
	@echo "  make run-mc         - Run full MC benchmark"
	@echo "  make evaluate-mc    - Generate MC evaluation metrics"
	@echo ""
	@echo "Common:"
	@echo "  make setup          - Install Python dependencies"
	@echo "  make clean          - Remove results and logs"
	@echo ""

setup:
	pip3 install -r requirements.txt
	$(PYTHON) -c "import nltk; nltk.download('punkt'); nltk.download('punkt_tab'); nltk.download('wordnet')"

# Free Response (locomo10) targets - Primary benchmark
download-fr:
	@mkdir -p data
	curl -o $(FR_DATASET) https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json
	@echo "Downloaded $(FR_DATASET)"

run-fr-quick:
	$(PYTHON) -m locomo10.main \
		--dataset $(FR_DATASET) \
		--output $(RESULTS_DIR)/locomo10_results.json \
		--max-questions 20 \
		--ultrathink-url $(ULTRATHINK_URL)

run-fr:
	$(PYTHON) -m locomo10.main \
		--dataset $(FR_DATASET) \
		--output $(RESULTS_DIR)/locomo10_results.json \
		--ultrathink-url $(ULTRATHINK_URL)

# Multiple Choice (locomo_mc10) targets
download-mc:
	$(PYTHON) -m locomo_mc10.download_dataset

run-mc-quick:
	$(PYTHON) -m locomo_mc10.main \
		--dataset $(MC_DATASET) \
		--output $(RESULTS_DIR)/locomo_mc10_results.json \
		--max-questions 10 \
		--ultrathink-url $(ULTRATHINK_URL)

run-mc:
	$(PYTHON) -m locomo_mc10.main \
		--dataset $(MC_DATASET) \
		--output $(RESULTS_DIR)/locomo_mc10_results.json \
		--ultrathink-url $(ULTRATHINK_URL)

evaluate-mc:
	$(PYTHON) -m locomo_mc10.comprehensive_evals \
		--input_file $(RESULTS_DIR)/locomo_mc10_results.json \
		--output_file $(RESULTS_DIR)/evaluation_mc_metrics.json

clean:
	rm -rf $(RESULTS_DIR)/*.json
	rm -rf logs/*.log logs/*.jsonl logs/*.json
	rm -rf __pycache__ */__pycache__ */*/__pycache__
