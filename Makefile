.PHONY: help test test-coverage test-verbose build build-collector run run-bg stop clean install-tools fmt lint docker-build docker-run demo all

# Get Go paths
GOPATH := $(shell go env GOPATH)
BUILDER := $(GOPATH)/bin/builder

# Collector configuration
COLLECTOR_CONFIG := collector-config.yaml
COLLECTOR_BINARY := dist/glean-otelcol
EXAMPLE_PING := example-ping.json

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "Available targets:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Run tests, build collector, and run demo
all: test build-collector demo

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -cover -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -cover ./... 2>&1 | tee test.log

## build: Build the receiver module
build:
	@echo "Building receiver module..."
	@go build -v .

## build-collector: Build custom OpenTelemetry Collector
build-collector: install-tools
	@echo "Building custom collector..."
	@rm -rf dist
	@$(BUILDER) --config=builder-config.yaml

## run: Run the collector in foreground
run: build-collector
	@echo "Starting collector..."
	@$(COLLECTOR_BINARY) --config=$(COLLECTOR_CONFIG)

## run-bg: Run the collector in background
run-bg: build-collector
	@echo "Starting collector in background..."
	@$(COLLECTOR_BINARY) --config=$(COLLECTOR_CONFIG) > collector.log 2>&1 & echo $$! > .collector.pid
	@sleep 2
	@if [ -f .collector.pid ]; then \
		echo "Collector running with PID $$(cat .collector.pid)"; \
	fi

## stop: Stop background collector
stop:
	@if [ -f .collector.pid ]; then \
		echo "Stopping collector (PID $$(cat .collector.pid))..."; \
		kill $$(cat .collector.pid) 2>/dev/null || true; \
		rm -f .collector.pid; \
	else \
		echo "No collector PID file found"; \
		pkill -9 glean-otelcol 2>/dev/null || true; \
	fi

## send-ping: Send example ping to running collector
send-ping:
	@if [ ! -f $(EXAMPLE_PING) ]; then \
		echo "Error: $(EXAMPLE_PING) not found"; \
		exit 1; \
	fi
	@echo "Sending example ping..."
	@curl -X POST http://localhost:9888/submit/glean/metrics/1/test-doc-123 \
		-H "Content-Type: application/json" \
		-d @$(EXAMPLE_PING) \
		&& echo "\nPing sent successfully!"

## demo: Run the demo script
demo:
	@echo "Running demo..."
	@./demo.sh

## clean: Clean build artifacts and logs
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist
	@rm -f coverage.out coverage.html test.log collector.log .collector.pid
	@go clean

## install-tools: Install required tools (builder)
install-tools:
	@echo "Installing OpenTelemetry Collector Builder..."
	@go install go.opentelemetry.io/collector/cmd/builder@v0.144.0

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

## lint: Run linters (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running linters..."; \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## tidy: Tidy Go modules
tidy:
	@echo "Tidying Go modules..."
	@go mod tidy

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t glean-otelcol:latest .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -d --name glean-collector \
		-p 9888:9888 -p 4317:4317 -p 4318:4318 \
		glean-otelcol:latest
	@echo "Collector container started!"

## docker-stop: Stop and remove Docker container
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop glean-collector 2>/dev/null || true
	@docker rm glean-collector 2>/dev/null || true

## docker-logs: View Docker container logs
docker-logs:
	@docker logs -f glean-collector

## verify: Run all verification checks (test, build, lint)
verify: test build build-collector
	@echo "✅ All verification checks passed!"

## quick-test: Quick integration test (build, run, send ping, stop)
quick-test: build-collector
	@echo "Starting quick integration test..."
	@$(MAKE) run-bg
	@sleep 3
	@$(MAKE) send-ping || ($(MAKE) stop && exit 1)
	@sleep 2
	@$(MAKE) stop
	@echo "✅ Quick test completed successfully!"

## logs: Show collector logs
logs:
	@if [ -f collector.log ]; then \
		tail -f collector.log; \
	else \
		echo "No collector log file found"; \
	fi
