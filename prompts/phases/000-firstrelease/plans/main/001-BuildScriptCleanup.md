# 001-BuildScriptCleanup

> **Source Specification**: prompts/phases/000-firstrelease/ideas/main/001-BuildScriptCleanup.md

## Goal Description
Clean up `scripts/process/build.sh` by removing unused frontend (React) and extension (TypeScript) build logic, as well as template embedding and file watching features that are no longer needed. The script should focus solely on building and testing the Go backend.

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 変数の整理 (Remove `BACKEND_ONLY`, `IDE_DIR`, etc.) | `Proposed Changes > scripts/process/build.sh` |
| ヘルプメッセージの修正 (Fix Description, remove prompts) | `Proposed Changes > scripts/process/build.sh` |
| 処理の削除 (Pre-build template steps) | `Proposed Changes > scripts/process/build.sh` |
| 処理の削除 (Extension deployment, Frontend build/test) | `Proposed Changes > scripts/process/build.sh` |
| 動作 (Go backend build & unit test only) | `Proposed Changes > scripts/process/build.sh` |

## Proposed Changes

### Scripts

#### [MODIFY] [build.sh](file:///c:/Users/yamya/myprog/kuniumi/scripts/process/build.sh)
*   **Description**: Remove unused variables, steps, and options.
*   **Logic**:
    *   **Variables**:
        *   Remove `BACKEND_ONLY`, `VITE_ENABLE_TEST_TAB`.
        *   Remove `IDE_DIR`, `EXTENSION_DIR`, `WEBVIEW_DIR`.
    *   **Help Message**:
        *   Update usage description to "Task Engine Backend (Go) Build Script".
        *   Remove options `--backend-only` and `--production`.
    *   **Argument Parsing**:
        *   Remove cases for `--backend-only` and `--production`.
    *   **Pre-Build Steps**:
        *   Remove `0. Prepare Embedded Templates (Pre-Build)` section (zipping templates).
    *   **Build Logic**:
        *   Remove `CHANGED_TEMPLATES` detection logic.
        *   Keep `CHANGED_FILES` (Go files) detection for incremental build.
    *   **Post-Build Steps**:
        *   Remove `Copy binary to extension directory` (lines 138-177).
        *   Remove `2. Frontend Build (Webview)` (lines 179-194).
        *   Remove `3. Extension Build (TypeScript)` (lines 196-206).
        *   Remove `4. Frontend Tests` (lines 208-217).
    *   **Final Output**:
        *   Change "=== Full Build Complete ===" to "=== Backend Build Complete ===".

## Step-by-Step Implementation Guide

1.  **Refactor build.sh**:
    *   Edit `scripts/process/build.sh` to remove valid sections and variables.
    *   Ensure line 2 references to `set -e` are kept.
    *   Keep logic for `unit test` execution.

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    Run the build script to ensure it builds the Go binary and runs unit tests.
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Skip Test Verification**:
    Run with `--skip-test` to ensure tests are skipped.
    ```bash
    ./scripts/process/build.sh --skip-test
    ```

3.  **Manual Help Verification**:
    Check the help message.
    ```bash
    ./scripts/process/build.sh --help
    ```

## Documentation
*   `prompts/phases/000-firstrelease/ideas/main/001-BuildScriptCleanup.md` (Already created)
