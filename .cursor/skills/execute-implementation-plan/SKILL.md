---
name: execute-implementation-plan
description: Execute coding and testing based on an implementation plan document. Use when the user wants to implement code from a plan, execute an implementation plan, or start coding a planned feature.
---

# Execute Implementation Plan Workflow

This skill implements code based on an implementation plan (`.../plans/.../XXX.md`), following coding and testing rules.

## 1. Input and Rule Verification

1. **Identify input file**: Use the file specified by the user or the currently open file as the "implementation plan".
2. **Load rules**: Read and follow throughout the entire task:
   - `prompts/rules/coding-rules.md` (coding rules)
   - `prompts/rules/testing-rules.md` (testing rules)

## 2. Implementation Execution

1. **Read the plan**: Understand target files and specific changes. If the plan spans multiple files, read all of them.
2. **Track progress**:
   - Mark completed items: `[ ]` → `[x]`
   - Mark in-progress items: `[ ]` → `[/]`
   - Update checkboxes across all plan files if split.
3. **Coding**: Follow the plan's instructions. Strictly adhere to `coding-rules.md` style and design principles.

## 3. Testing and Verification

### 3.1 Test Execution Order

Follow the order defined in the implementation plan:

1. **Build & Unit Test (MANDATORY)**:
   - **Before all other tests**: Build the entire project and pass unit tests first. This ensures integration tests run against the latest binaries and assets.
   - Command:
     ```bash
     ./scripts/process/build.sh
     ```
   - **CRITICAL**: If this fails (Exit Code != 0), do **NOT** proceed to the next step. The code is broken.

2. **Integration Tests**:
   - Run only after Step 1 succeeds.
   - Integration test files go in `tests/kuniumi/`.
   - Command:
     ```bash
     ./scripts/process/integration_test.sh
     ```
   - **CRITICAL**: Check the **exit code**. If non-zero, do **NOT** proceed.

3. **Other tests**: Run additional tests (e.g. performance) as needed.

### 3.2 Mandatory Fix Loop

> **NEVER IGNORE FAILURES**: Ignoring build or test failures (Exit Code != 0) and marking a task as done is considered destructive to the project.

When a build or test fails, repeat this loop **until it succeeds**:

1. **Read Logs**: Examine error logs and stack traces to identify the root cause.
2. **Fix Code**: Correct the code or test to eliminate the cause.
3. **Retry**: Re-run the same command.

- "Fix it later" is **FORBIDDEN**. Fix it now.
- Update the implementation plan's checkboxes/progress with any fixes made.

### 3.3 Documentation Update

If the plan includes "update existing specs", update the relevant specification documents after implementation is complete. Verify accuracy.
