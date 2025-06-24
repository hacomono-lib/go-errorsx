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
lint:
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
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Development tools installed successfully"

## docker-dev: Start development container
docker-dev:
	docker-compose up -d dev
	docker-compose exec dev bash

## docker-test: Run tests in container
docker-test:
	docker-compose run --rm test

## docker-lint: Run linter in container
docker-lint:
	docker-compose run --rm lint

## docker-security: Run security scan in container
docker-security:
	docker-compose run --rm security

## docker-clean: Clean up Docker resources
docker-clean:
	docker-compose down -v
	docker system prune -f

## check: Run all checks (formatting, linting, testing)
check: fmt vet lint test-cover
	@echo "All checks passed!"

## release-check: Run checks before release
release-check: clean mod-tidy ci
	@echo "Release checks completed successfully!"