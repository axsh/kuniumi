#!/bin/bash
set -e

# Default values
TARGET="./..."
SKIP_TEST=false

# Ensure we are in project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." && pwd )"

cd "$ROOT_DIR"

# Help message
show_help() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Description:
  Task Engine Backend (Go) Build Script.
  Builds the backend binary and runs unit tests.

Options:
  --specify "TARGET"   Specify the target package(s) for Go unit tests.
                       Default: "./..." (Recursively test all packages)
  --skip-test          Skip running tests.
  --help               Show this help message.
EOF
}

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --specify) TARGET="$2"; shift ;;
        --skip-test) SKIP_TEST=true ;;
        --help) show_help; exit 0 ;;
        *) echo "Unknown parameter passed: $1"; show_help; exit 1 ;;
    esac
    shift
done

echo "=== Building Task Engine Project (Backend Only) ==="

# ---------------------------------------------------------
# 1. Backend Build & Test (Go)
# ---------------------------------------------------------
echo ">> [Backend] Checking status..."

# Determine binary name based on OS
BINARY_NAME="basic"
if [[ "$OSTYPE" == "msys" ]]; then
    BINARY_NAME="basic.exe"
fi
TARGET_BINARY="bin/$BINARY_NAME"

# Check if we need to rebuild
NEEDS_REBUILD=true
if [ -f "$TARGET_BINARY" ]; then
    CHANGED_FILES=$(find . -name "*.go" -newer "$TARGET_BINARY" -print -quit)
    
    if [ -z "$CHANGED_FILES" ]; then
        echo "   No changes detected in .go files compared to $TARGET_BINARY."
        NEEDS_REBUILD=false
    else
         echo "   Changes detected in .go files. Rebuilding..."
    fi
else
    echo "   Binary not found. Building..."
fi

if [ "$NEEDS_REBUILD" = true ]; then
    # Run Unit Tests (Fail Fast)
    if [ "$SKIP_TEST" = false ]; then
        echo ">> [Backend] Running Unit Tests (Fail Fast)..."
        echo "   Target: $TARGET"
        go test -v -count=1 -failfast "$TARGET"
    else
        echo ">> [Backend] Skipping Unit Tests..."
    fi

    echo ">> [Backend] Building Binary..."
    mkdir -p bin
    go build -v -o bin/ ./examples/...
else
    echo ">> [Backend] Up-to-date. Skipping build & test."
fi

echo "=== Backend Build Complete ==="

