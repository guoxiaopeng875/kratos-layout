#!/bin/bash

set -e

# Default minimum coverage threshold (percentage)
MIN_COVERAGE=${MIN_COVERAGE:-60}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 <dir1> [dir2] [dir3] ..."
    echo ""
    echo "Check test coverage for specified directories."
    echo ""
    echo "Arguments:"
    echo "  dir1, dir2, ...  Directories to check (e.g., ./internal/... ./pkg/...)"
    echo ""
    echo "Environment variables:"
    echo "  MIN_COVERAGE     Minimum coverage threshold (default: 60)"
    echo ""
    echo "Examples:"
    echo "  $0 ./pkg/..."
    echo "  $0 ./internal/biz/... ./internal/data/..."
    echo "  MIN_COVERAGE=80 $0 ./pkg/..."
    exit 1
}

# Check if at least one directory is provided
if [ $# -eq 0 ]; then
    usage
fi

DIRS="$*"

# Run tests and generate coverage profile (capture output for errors)
TEST_OUTPUT=$(go test -coverprofile=coverage.out -covermode=atomic ${DIRS} 2>&1) || {
    echo -e "${RED}Tests failed:${NC}"
    echo ""
    echo "$TEST_OUTPUT" | grep -A 5 -E "(FAIL|Error|panic|error)" || echo "$TEST_OUTPUT"
    exit 1
}

echo -e "${YELLOW}Checking coverage threshold (minimum: ${MIN_COVERAGE}%)...${NC}"
echo ""

# Check total coverage
TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | tr -d '%')
FAILED=0

if [ -n "$TOTAL_COVERAGE" ]; then
    if [ "$(echo "$TOTAL_COVERAGE < $MIN_COVERAGE" | bc -l)" -eq 1 ]; then
        echo -e "${RED}[FAIL]${NC} Total coverage: ${TOTAL_COVERAGE}% < ${MIN_COVERAGE}%"
        FAILED=1
    else
        echo -e "${GREEN}[PASS]${NC} Total coverage: ${TOTAL_COVERAGE}% >= ${MIN_COVERAGE}%"
    fi
fi

echo ""

if [ $FAILED -eq 1 ]; then
    echo -e "${RED}Coverage check failed!${NC}"
    exit 1
else
    echo -e "${GREEN}Coverage check passed!${NC}"
    exit 0
fi
