# 統合テストスクリプトのクリーンアップ

## 背景 (Background)
現在の `scripts/process/integration_test.sh` には、カテゴリ分け（`llm`, `taskengine`, `gui`）やフロントエンド（Playwright/VSCode Extension）のテストを実行する機能が含まれていますが、現在のプロジェクト構成では `kuniumi` パッケージのテストのみが対象となります。不要な機能を削除し、スクリプトを簡素化する必要があります。

## 要件 (Requirements)
以下の通り、`scripts/process/integration_test.sh` から不要なコードと機能を削除し、整理を行う。

1.  **カテゴリ機能の削除**:
    *   `--categories` オプションとその処理ロジックを削除する。
    *   `CATEGORIES` 変数に関連するループ処理やフィルタリングを削除する。
    *   デフォルトで `kuniumi`（バックエンド）のテストのみを実行するようにする。
2.  **GUI/Frontend関連の削除**:
    *   `--ui`, `--headed` オプションを削除する。
    *   GUIテスト実行ブロック（VSCode起動、Playwright実行、クリーンアップ）を完全に削除する。
3.  **ヘルプメッセージの修正**:
    *   `usage` 関数の出力を修正し、削除したオプション（`--categories`, `--ui`, `--headed`）への言及を削除する。
    *   Examples を現在の機能に合わせて更新する（「Run all tests」のみなど）。
4.  **コード整理**:
    *   引数解析（`while` ループ）を簡素化する。

## 実現方針 (Implementation Approach)
*   `scripts/process/integration_test.sh` を直接編集し、指定された行範囲の削除と修正を行う。

## 検証シナリオ (Verification Scenarios)
1.  **ヘルプメッセージの確認**:
    *   `./scripts/process/integration_test.sh --help` を実行し、オプションが整理されていることを確認する。
2.  **テスト実行**:
    *   `./scripts/process/integration_test.sh` を実行し、Goの統合テスト（`tests/kuniumi` など）が正常に実行されることを確認する。
    *   GUIテストが実行されないことを確認する。
