.PHONY: all build test clean proto run deps fmt lint test-race

# Variables
BINARY_NAME=raft-node
BUILD_DIR=bin
PROTO_DIR=pkg/proto

# Build everything
all: proto build

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	cd $(PROTO_DIR) && ./generate.sh

# Build the binary
build: proto
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/raft-node/main.go
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	go test -v ./...

# Run with race detector
test-race:
	go test -race -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(PROTO_DIR)/*.pb.go
	@echo "✅ Clean complete"

# Run the node (for development)
run: build
	$(BUILD_DIR)/$(BINARY_NAME) --config config/node_config.json

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Format complete"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run ./...
	@echo "✅ Lint complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Dependencies installed"

# Verify compilation without building binary
verify:
	@echo "Verifying code compiles..."
	go build ./...
	@echo "✅ Verification complete"

# Display help
help:
	@echo "Available targets:"
	@echo "  all        - Generate protobuf and build binary"
	@echo "  proto      - Generate protobuf code"
	@echo "  build      - Build the raft-node binary"
	@echo "  test       - Run tests"
	@echo "  test-race  - Run tests with race detector"
	@echo "  clean      - Clean build artifacts"
	@echo "  run        - Build and run the node"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code (requires golangci-lint)"
	@echo "  deps       - Install dependencies"
	@echo "  verify     - Verify code compiles"
	@echo "  help       - Display this help"
