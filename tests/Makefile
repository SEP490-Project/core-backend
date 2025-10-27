# Test Makefile for Campaign Suggestion Refactoring Tests
# Provides convenient commands for running tests with comprehensive reporting

.PHONY: test test-unit test-integration test-performance test-all test-coverage test-report clean-test

# Colors for output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
BLUE   := $(shell tput -Txterm setaf 4)
RESET  := $(shell tput -Txterm sgr0)

# Test directories
UNIT_TESTS := ./tests/unit_tests/...
INTEGRATION_TESTS := ./tests/integration_tests/...
PERFORMANCE_TESTS := ./tests/performance_tests/...
ALL_TESTS := ./tests/...

# Output directories
REPORTS_DIR := ./test-reports
COVERAGE_DIR := $(REPORTS_DIR)/coverage

# Create reports directory
$(REPORTS_DIR):
	@mkdir -p $(REPORTS_DIR)
	@mkdir -p $(COVERAGE_DIR)

## test-unit: Run unit tests with verbose output
test-unit: $(REPORTS_DIR)
	@echo "$(BLUE)Running Unit Tests...$(RESET)"
	@go test $(UNIT_TESTS) -v -count=1 | tee $(REPORTS_DIR)/unit-tests.log
	@echo "$(GREEN)✓ Unit tests completed$(RESET)"

## test-integration: Run integration tests
test-integration: $(REPORTS_DIR)
	@echo "$(BLUE)Running Integration Tests...$(RESET)"
	@go test $(INTEGRATION_TESTS) -v -count=1 | tee $(REPORTS_DIR)/integration-tests.log
	@echo "$(GREEN)✓ Integration tests completed$(RESET)"

## test-performance: Run performance benchmarks
test-performance: $(REPORTS_DIR)
	@echo "$(BLUE)Running Performance Tests...$(RESET)"
	@go test $(PERFORMANCE_TESTS) -bench=. -benchmem -benchtime=5s | tee $(REPORTS_DIR)/performance-tests.log
	@echo "$(GREEN)✓ Performance tests completed$(RESET)"

## test: Run all tests with basic reporting
test: $(REPORTS_DIR)
	@echo "$(BLUE)Running All Tests...$(RESET)"
	@go test $(ALL_TESTS) -v -count=1 | tee $(REPORTS_DIR)/all-tests.log
	@echo "$(GREEN)✓ All tests completed$(RESET)"

## test-coverage: Run all tests with coverage report
test-coverage: $(REPORTS_DIR)
	@echo "$(BLUE)Running Tests with Coverage...$(RESET)"
	@go test $(ALL_TESTS) \
		-coverprofile=$(COVERAGE_DIR)/coverage.out \
		-covermode=atomic \
		-v \
		-count=1 | tee $(REPORTS_DIR)/coverage-tests.log
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tee $(COVERAGE_DIR)/coverage-summary.txt
	@echo "$(GREEN)✓ Coverage report generated:$(RESET) $(COVERAGE_DIR)/coverage.html"
	@echo "$(GREEN)✓ Coverage summary:$(RESET) $(COVERAGE_DIR)/coverage-summary.txt"

## test-report: Run all tests and generate comprehensive JSON report
test-report: $(REPORTS_DIR)
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo "$(BLUE)   Campaign Suggestion Refactoring - Test Report$(RESET)"
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo ""
	@echo "$(YELLOW)Running Unit Tests...$(RESET)"
	@go test $(UNIT_TESTS) -v -json -count=1 > $(REPORTS_DIR)/unit-tests.json 2>&1 || true
	@go test $(UNIT_TESTS) -v -count=1 | tee $(REPORTS_DIR)/unit-tests.log
	@echo ""
	@echo "$(YELLOW)Running Integration Tests...$(RESET)"
	@go test $(INTEGRATION_TESTS) -v -json -count=1 > $(REPORTS_DIR)/integration-tests.json 2>&1 || true
	@go test $(INTEGRATION_TESTS) -v -count=1 | tee $(REPORTS_DIR)/integration-tests.log
	@echo ""
	@echo "$(YELLOW)Running Coverage Analysis...$(RESET)"
	@go test $(ALL_TESTS) \
		-coverprofile=$(COVERAGE_DIR)/coverage.out \
		-covermode=atomic \
		-count=1 > /dev/null 2>&1 || true
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage-summary.txt 2>&1 || true
	@echo ""
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo "$(GREEN)✓ Test Execution Complete$(RESET)"
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo ""
	@echo "$(YELLOW)Report Files Generated:$(RESET)"
	@echo "  • Unit Test Log:        $(REPORTS_DIR)/unit-tests.log"
	@echo "  • Unit Test JSON:       $(REPORTS_DIR)/unit-tests.json"
	@echo "  • Integration Log:      $(REPORTS_DIR)/integration-tests.log"
	@echo "  • Integration JSON:     $(REPORTS_DIR)/integration-tests.json"
	@echo "  • Coverage HTML:        $(COVERAGE_DIR)/coverage.html"
	@echo "  • Coverage Summary:     $(COVERAGE_DIR)/coverage-summary.txt"
	@echo ""
	@$(MAKE) --no-print-directory test-summary

## test-summary: Generate and display test summary
test-summary:
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo "$(BLUE)   Test Summary$(RESET)"
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo ""
	@if [ -f "$(REPORTS_DIR)/unit-tests.log" ]; then \
		echo "$(YELLOW)Unit Tests:$(RESET)"; \
		grep -E "^(PASS|FAIL|ok|FAIL)" $(REPORTS_DIR)/unit-tests.log | tail -5 || echo "  No results found"; \
		echo ""; \
	fi
	@if [ -f "$(REPORTS_DIR)/integration-tests.log" ]; then \
		echo "$(YELLOW)Integration Tests:$(RESET)"; \
		grep -E "^(PASS|FAIL|ok|FAIL)" $(REPORTS_DIR)/integration-tests.log | tail -5 || echo "  No results found"; \
		echo ""; \
	fi
	@if [ -f "$(COVERAGE_DIR)/coverage-summary.txt" ]; then \
		echo "$(YELLOW)Coverage Summary:$(RESET)"; \
		tail -1 $(COVERAGE_DIR)/coverage-summary.txt || echo "  No coverage data"; \
		echo ""; \
	fi
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo "$(GREEN)View detailed HTML coverage report:$(RESET)"
	@echo "  open $(COVERAGE_DIR)/coverage.html"
	@echo ""

## test-all: Run all test suites sequentially
test-all: test-unit test-integration test-coverage
	@echo "$(GREEN)✓ All test suites completed$(RESET)"

## test-watch: Run tests in watch mode (requires entr)
test-watch:
	@echo "$(BLUE)Watching for file changes...$(RESET)"
	@find . -name '*.go' | entr -c make test-unit

## test-quick: Run tests without cache for quick feedback
test-quick:
	@echo "$(BLUE)Running Quick Test (no cache)...$(RESET)"
	@go test $(ALL_TESTS) -count=1

## test-verbose: Run tests with maximum verbosity
test-verbose: $(REPORTS_DIR)
	@echo "$(BLUE)Running Tests (Verbose Mode)...$(RESET)"
	@go test $(ALL_TESTS) -v -count=1 -parallel=1 | tee $(REPORTS_DIR)/verbose-tests.log

## clean-test: Remove all test reports and cached results
clean-test:
	@echo "$(YELLOW)Cleaning test reports...$(RESET)"
	@rm -rf $(REPORTS_DIR)
	@go clean -testcache
	@echo "$(GREEN)✓ Test reports cleaned$(RESET)"

## help: Display this help message
help:
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo "$(BLUE)   Campaign Suggestion Test Suite$(RESET)"
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo ""
	@echo "$(YELLOW)Available Commands:$(RESET)"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':' | sed 's/^/  /'
	@echo ""
	@echo "$(BLUE)===========================================================$(RESET)"
	@echo "$(YELLOW)Quick Start:$(RESET)"
	@echo "  make test-report     # Generate comprehensive test report"
	@echo "  make test-coverage   # Run tests with coverage"
	@echo "  make test-unit       # Run only unit tests"
	@echo ""

# Default target
.DEFAULT_GOAL := help
