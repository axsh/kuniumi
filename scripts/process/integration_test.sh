#!/bin/bash
set -e

# Default values
FILTER=""

# Ensure we are in project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." && pwd )"
cd "$ROOT_DIR"

# Help message
show_help() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Description:
  Runs integration tests for the Kuniumi backend.

Options:
  --specify "REGEX"    Run only tests matching the regex (passed to go test -run).
  --help               Show this help message.

Examples:
  # Run all tests
  $(basename "$0")

  # Run specific test
  $(basename "$0") --specify "TestAuthentication"
EOF
}

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --specify) FILTER="-run $2"; shift ;;
        --help) show_help; exit 0 ;;
        *) echo "Unknown parameter passed: $1"; show_help; exit 1 ;;
    esac
    shift
done

echo ""
echo "========================================="
echo ">> Running Kuniumi Integration Tests"
echo "========================================="
go test -v -failfast -tags=integration $FILTER ./tests/kuniumi/...

echo ""
echo ">> All Integration Tests Completed."
