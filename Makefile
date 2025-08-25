.PHONY: test test-race test-cover lint fmt vet clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Tool versions
GOLANGCI_LINT_VERSION=v2.4.0
GOSEC_VERSION=latest

# Test parameters
TEST_PACKAGES=./...
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

## help: Display this help message
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## test: Run tests
test:
	$(GOTEST) -v $(TEST_PACKAGES)

## test-race: Run tests with race detector
test-race:
	$(GOTEST) -v -race $(TEST_PACKAGES)

## test-cover: Run tests with coverage
test-cover:
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(TEST_PACKAGES)
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

## bench: Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem $(TEST_PACKAGES)

## lint: Run linter
lint: install-tools
	$(GOLINT) run

## fmt: Format code
fmt:
	$(GOFMT) -s -w .

## vet: Run go vet
vet:
	$(GOCMD) vet $(TEST_PACKAGES)

## mod-tidy: Clean up go.mod and go.sum
mod-tidy:
	$(GOMOD) tidy

## mod-verify: Verify dependencies
mod-verify:
	$(GOMOD) verify

## clean: Clean build artifacts and coverage files
clean:
	$(GOCLEAN)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

## ci: Run all CI checks
ci: fmt vet lint test-race test-cover

## install-tools: Install development tools
install-tools:
	@echo "Checking and installing development tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1 || [ "$$(golangci-lint version 2>/dev/null | sed 's/version /v/' | grep -oE 'v([0-9]+\.[0-9]+\.[0-9]+)')" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	else \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) already installed"; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec $(GOSEC_VERSION)..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION); \
	else \
		echo "gosec already installed"; \
	fi
	@echo "Development tools check completed"

## docker-dev: Start development container
docker-dev:
	docker compose up -d dev
	docker compose exec dev bash

## docker-test: Run tests in container
docker-test:
	docker compose run --rm test

## docker-lint: Run linter in container
docker-lint:
	docker compose run --rm lint

## docker-security: Run security scan in container
docker-security:
	docker compose run --rm security

## docker-clean: Clean up Docker resources
docker-clean:
	docker compose down -v
	docker system prune -f

## ci: Run all CI checks in container
docker-ci: docker-lint docker-test docker-security

## check: Run all checks (formatting, linting, testing)
check: fmt vet lint test-cover
	@echo "All checks passed!"

## release-check: Run checks before release
release-check: clean mod-tidy ci
	@echo "Release checks completed successfully!"