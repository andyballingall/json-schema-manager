.PHONY: all build run clean test test-race test-cover check-coverage cover-html lint fmt snapshot release-check setup

all: build

# Build the application
build:
	@go run scripts/build/main.go

# Run GoReleaser snapshot
snapshot:
	@go run scripts/snapshot/main.go

# Run the application
run:
	@go run cmd/jsm/main.go

# Clean build artifacts
clean:
	@go run scripts/clean/main.go
	@go clean

# Run tests
test:
	@go run scripts/tester/main.go ./... -v

# Run tests with race detection
test-race:
	@go run scripts/tester/main.go -race ./... -v

# Run tests with coverage
test-cover:
	@go run scripts/tester/main.go -count=1 -race -coverpkg=./internal/... ./... -coverprofile=coverage.out
	@go tool cover -func=coverage.out

# Check that all files (except main.go) have 100% coverage
check-coverage: test-cover
	@go run scripts/check_coverage/main.go coverage.out

# View coverage in browser
cover-html:
	@go run scripts/tester/main.go -count=1 -race -coverpkg=./internal/... ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out

# Run linter
lint:
	@go run scripts/lint/main.go

# Format code with gofumpt
fmt:
	@go run scripts/fmt/main.go

# Run GoReleaser configuration check
release-check:
	@go run scripts/release_check/main.go

# Setup development environment
setup:
	@go run scripts/setup/main.go
