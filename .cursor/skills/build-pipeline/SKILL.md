---
name: build-pipeline
description: Run the full build and verification pipeline including unit tests and integration tests. Use when the user wants to build the project, run tests, verify code changes, or check for regressions.
---

# Build and Verification Pipeline

This skill runs the project's build and test pipeline to verify safety (tests pass) and correctness (build succeeds) after code changes.

## 1. Preparation: Check Status

1. Run `scripts/utils/show_current_status.sh`.
2. Extract `phase` from the JSON output → `[Phase]`.

## 2. Full Build & Unit Test

Build the entire project and run unit tests in one step. Integration tests require the latest artifacts, so this step must complete first.

```bash
./scripts/process/build.sh
```

**CRITICAL**: If this step fails, stop immediately and fix before proceeding.

## 3. Integration & E2E Tests

Run all integration tests.

> **Prerequisite**: Step 2 must have succeeded. Running tests without a fresh build produces unreliable results (tests run against stale binaries).

```bash
./scripts/process/integration_test.sh
```

### Selective Execution

To run specific tests only:

```bash
./scripts/process/integration_test.sh --specify "TestNameRegex"
```

## 4. Analyze Results & Feedback Loop

When tests fail or behave unexpectedly:

### 4.1 Check Reports

Read detailed test reports to identify error causes.

### 4.2 Debug and Fix

1. **Fix**: Correct the implementation or test code.
2. **Re-run**: Execute only the relevant tests to verify the fix:
   ```bash
   ./scripts/process/integration_test.sh --specify "TestNameRegex"
   ```

## 5. Final Check

All tests pass with no regressions → task complete.
