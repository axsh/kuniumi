# 統合テストスクリプトのクリーンアップ 実装計画

> **Source Specification**: prompts/phases/000-firstrelease/ideas/main/002-IntegrationTestCleanup.md

## Goal Description
`scripts/process/integration_test.sh` から、本プロジェクト（Kuniumi バックエンド）に不要なカテゴリ分け機能やフロントエンド/GUIテスト機能を削除し、バックエンドの統合テスト専用にスクリプトを最適化します。

## User Review Required
None.

## Requirement Traceability

> **Traceability Check**:

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| カテゴリ機能(`--categories`, `CATEGORIES`)の削除 | Scripts > integration_test.sh |
| デフォルトで `kuniumi` パッケージのみ実行 | Scripts > integration_test.sh |
| GUI/Frontend関連(`--ui`, `--headed`)の削除 | Scripts > integration_test.sh |
| GUIテスト実行ブロックの完全削除 | Scripts > integration_test.sh |
| ヘルプメッセージ(`usage`)の修正 | Scripts > integration_test.sh |
| 引数解析(`while`ループ)の簡素化 | Scripts > integration_test.sh |

## Proposed Changes

### Scripts

#### [MODIFY] [integration_test.sh](file:///c:/Users/yamya/myprog/kuniumi/scripts/process/integration_test.sh)
*   **Description**: 不要な機能（カテゴリ、GUIモード）を削除し、Goのテスト実行のみに特化する。
*   **Logic**:
    1.  **変数定義の削除**: `CATEGORIES`, `UI_MODE`, `HEADED` を削除。
    2.  **`usage` 関数の更新**:
        *   `--categories`, `--ui`, `--headed` の説明を削除。
        *   Examples を `$(basename "$0")` やフィルタリング例のみに更新。
    3.  **引数解析の簡素化**:
        *   `--categories`, `--ui`, `--headed` の `case` 分岐を削除。
        *   `--specify` (or `-run`) は保持。
    4.  **実行ロジックの変更**:
        *   カテゴリごとのループ処理 (`for CATEGORY in "${categories[@]}"`) を削除。
        *   直接 `kuniumi` パッケージのテストを実行するロジックに変更（または既存の `kuniumi` ブロックをメインストリームにし、条件分岐を外す）。
    5.  **GUIテストブロックの削除**:
        *   VSCode起動、Playwright実行、クリーンアップ処理の全ブロックを削除。

## Step-by-Step Implementation Guide

1.  **Remove Variables and Usage Info**:
    *   Edit `scripts/process/integration_test.sh` to remove `CATEGORIES`, `UI_MODE`, `HEADED` variable initializations.
    *   Update `usage()` function to remove help text for removed options.
2.  **Simplify Argument Parsing**:
    *   Edit `scripts/process/integration_test.sh` argument parsing loop to remove cases for `--categories`, `--ui`, `--headed`.
3.  **Refactor Test Execution Logic**:
    *   Edit `scripts/process/integration_test.sh` to remove the category loop and the GUI test execution block.
    *   Ensure the script executes `go test -tags=integration ./tests/kuniumi/...` (or equivalent current logic) unconditionally.

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    Run standard build process to ensure environment is sane.
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    Run the refactored integration test script. It should run the backend tests without error.
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: Ensure output shows "Running Kuniumi Integration Tests" (or similar) and **does not** show "Running GUI Tests".

3.  **Help Message Check**:
    Verify help message no longer lists removed options.
    ```bash
    ./scripts/process/integration_test.sh --help
    ```

## Documentation

#### [MODIFY] [integration_test.sh](file:///c:/Users/yamya/myprog/kuniumi/scripts/process/integration_test.sh)
*   **更新内容**: ヘルプメッセージ自体がドキュメントとなるため、上記変更にて対応。
