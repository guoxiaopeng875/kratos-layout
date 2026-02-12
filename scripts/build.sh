#!/bin/bash
set -e

APP_NAME=$1
VERSION=${2:-"dev"}

if [ -z "$APP_NAME" ]; then
    echo "Usage: $0 <app_name> [version]"
    exit 1
fi

# Find the main.go file
MAIN_PATH=$(find ./cmd -type f -name "main.go" -path "*/${APP_NAME}/*" 2>/dev/null | head -1)

if [ -z "$MAIN_PATH" ]; then
    echo "Error: main.go not found for app: $APP_NAME"
    exit 1
fi

CMD_DIR=$(dirname "$MAIN_PATH")

echo "Building $APP_NAME from $CMD_DIR ..."

mkdir -p bin/
go build -ldflags "-X main.Version=$VERSION -X main.Name=$APP_NAME" -o ./bin/$APP_NAME $CMD_DIR

echo "Built: ./bin/$APP_NAME"
