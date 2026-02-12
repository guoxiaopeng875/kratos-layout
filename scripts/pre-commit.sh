#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Format code
echo "Formatting code..."
goimports -w .
gofmt -w .

# Run tests
echo "Running tests..."
go test -race $(go list ./... | grep -v /test/integration/)

# Run linter
echo "Running linter..."
golangci-lint run ./...

echo "Pre-commit checks passed!"
