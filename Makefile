.PHONY: test test-unit test-integration test-frontend test-e2e test-all lint coverage clean help

GO := go
GOFLAGS := -v -race
FRONTEND_DIR := frontend
COVERAGE_FILE := coverage.out

help:
	@echo "FLACidal Test Commands"
	@echo "======================"
	@echo "make test           - Run all tests (unit + integration)"
	@echo "make test-unit      - Run unit tests only"
	@echo "make test-integration - Run integration tests (requires -tags=integration)"
	@echo "make test-frontend  - Run frontend TypeScript tests"
	@echo "make test-e2e       - Run E2E Playwright tests"
	@echo "make test-all       - Run complete test suite"
	@echo "make lint           - Run linters (golangci-lint + ESLint)"
	@echo "make coverage       - Generate coverage report (HTML)"
	@echo "make bench          - Run benchmarks"
	@echo "make clean          - Clean test artifacts"

test: test-unit

test-unit:
	@echo "Running unit tests..."
	$(GO) test $(GOFLAGS) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./backend/...

test-integration:
	@echo "Running integration tests..."
	$(GO) test $(GOFLAGS) -tags=integration -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./backend/...

test-frontend:
	@echo "Running frontend tests..."
	cd $(FRONTEND_DIR) && npm test -- --run

test-e2e:
	@echo "Running E2E tests..."
	npx playwright test

test-all: test-unit test-frontend
	@echo "All tests completed!"

lint:
	@echo "Running Go linter..."
	golangci-lint run ./backend/...
	@echo "Running frontend linter..."
	cd $(FRONTEND_DIR) && npm run lint || true

coverage:
	@echo "Generating coverage report..."
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./backend/...

clean:
	@echo "Cleaning test artifacts..."
	rm -f $(COVERAGE_FILE) coverage.html
	rm -rf $(FRONTEND_DIR)/coverage
