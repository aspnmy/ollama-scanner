#!/bin/bash

# Test build script for ollama_scanner

# Check if required dependencies are installed
check_dependencies() {
    command -v go >/dev/null 2>&1 || { echo "Go is required but not installed"; exit 1; }
}

# Run tests
run_tests() {
    echo "Running tests..."
    go test ./... -v
}

# Main execution
main() {
    check_dependencies
    run_tests
}

main