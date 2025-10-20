#!/bin/bash
# Generate Go code from protobuf definitions

set -e

# Install protoc-gen-go and protoc-gen-go-grpc if not present
echo "Installing protobuf Go plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc not found. Install Protocol Buffers compiler."
    echo "  macOS: brew install protobuf"
    echo "  Linux: sudo apt install protobuf-compiler"
    exit 1
fi

echo "Generating protobuf code..."

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    raft.proto

echo "✅ Protocol buffer code generated successfully"
ls -lh *.pb.go
