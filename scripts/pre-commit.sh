#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running pre-commit checks...${NC}"
echo ""

# Step 1: Run linter
echo -e "${YELLOW}[1/3] Running lint...${NC}"
if ! golangci-lint run ./...; then
    echo ""
    echo -e "${RED}Lint failed! Please fix the issues above.${NC}"
    exit 1
fi
echo -e "${GREEN}Lint passed.${NC}"
echo ""

# Step 2: Run tests with coverage check
echo -e "${YELLOW}[2/3] Running tests with coverage check...${NC}"
if ! ./scripts/coverage.sh ./pkg/... ./internal/biz/... ./internal/data/...; then
    echo ""
    echo -e "${RED}Tests or coverage check failed!${NC}"
    exit 1
fi
echo ""

# Step 3: Format code
echo -e "${YELLOW}[3/3] Formatting code...${NC}"
go fmt ./...
goimports -w .

# Add formatted files to staging
git add -u

echo -e "${GREEN}Code formatted.${NC}"
echo ""

echo -e "${GREEN}All pre-commit checks passed!${NC}"
exit 0
