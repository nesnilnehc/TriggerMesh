# Makefile for TriggerMesh

# Go variables
GO := go
GOFLAGS := -mod=mod
GOTEST := $(GO) test
GOBUILD := $(GO) build

# Binary name
BINARY := triggermesh

# Output directory
BIN_DIR := bin

# Main package path
MAIN_PACKAGE := ./cmd/triggermesh

# Test packages
TEST_PACKAGES := ./internal/...

# Default target
.DEFAULT_GOAL := build

# Build the binary
build:
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(GOFLAGS) -o $(BIN_DIR)/$(BINARY) $(MAIN_PACKAGE)

# Run the application
run:
	$(GO) run $(GOFLAGS) $(MAIN_PACKAGE) --config config.yaml

# Run all tests
test:
	$(GOTEST) $(GOFLAGS) $(TEST_PACKAGES) -v

# Run tests with coverage
coverage:
	$(GOTEST) $(GOFLAGS) $(TEST_PACKAGES) -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out

# Format code
fmt:
	$(GO) fmt $(GOFLAGS) ./...

# Vet code
vet:
	$(GO) vet $(GOFLAGS) ./...

# Clean up
clean:
	rm -f $(BINARY) coverage.out
	rm -rf $(BIN_DIR)

# Build Docker image
docker-build:
	docker build -t triggermesh .

# Run Docker container
docker-run:
	docker-compose up -d

# Stop Docker container
docker-stop:
	docker-compose down

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  run            - Run the application"
	@echo "  test           - Run all tests"
	@echo "  coverage       - Run tests with coverage"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code"
	@echo "  clean          - Clean up"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-stop    - Stop Docker container"
	@echo "  help           - Show this help"
