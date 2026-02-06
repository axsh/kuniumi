# ビルドスクリプトのクリーンアップ

## 背景 (Background)
現在の `scripts/process/build.sh` は、バックエンド、Webview(React)、拡張機能(TypeScript)のビルドを含む多機能なスクリプトとなっていますが、本プロジェクトではGoのバックエンドビルドのみが必要とされています。不要な機能やコードが含まれていることで、メンテナンス性が低下し、誤解を招く恐れがあります。

## 要件 (Requirements)
以下の通り、`scripts/process/build.sh` から不要なコードと機能を削除し、整理を行う。

1.  **変数の整理**:
    *   `BACKEND_ONLY`, `VITE_ENABLE_TEST_TAB` などのフロントエンド/テストモード関連の変数を整理・削除する。
    *   `IDE_DIR`, `EXTENSION_DIR`, `WEBVIEW_DIR` などの不要なディレクトリ定義を削除する。
2.  **ヘルプメッセージの修正**:
    *   `Description` を本プロジェクト用に「Task Engineのバックエンド(Go)ビルドスクリプト」として修正する。
    *   不要なオプション `--backend-only`, `--production` を削除する。
    *   `--specify`, `--skip-test` は保持する。
3.  **処理の削除**:
    *   「Prepare Embedded Templates」セクションを削除。
    *   テンプレート変更検知 (`CHANGED_TEMPLATES`) を削除。
    *   「Deploying binary to extension」セクションを削除。
    *   フロントエンドビルド、拡張機能ビルド、フロントエンドテストのセクションを全削除。
4.  **動作**:
    *   スクリプト実行時、Goのバックエンドビルドと単体テストのみを実行する。

## 実現方針 (Implementation Approach)
*   `scripts/process/build.sh` を直接編集し、指定された行範囲の削除と修正を行う。
*   基本構造（引数解析、ビルド実行、エラーハンドリング）は維持する。

## 検証シナリオ (Verification Scenarios)
1.  **ヘルプメッセージの確認**:
    *   `./scripts/process/build.sh --help` を実行し、修正された説明とオプションが表示されることを確認する。
2.  **通常ビルドの実行**:
    *   `./scripts/process/build.sh` を実行し、Goのビルドとテストが正常に完了することを確認する。
    *   余計なステップ（テンプレート準備、Extensionへのコピーなど）が実行されていないことをログで確認する。
3.  **テストスキップの確認**:
    *   `./scripts/process/build.sh --skip-test` を実行し、テストがスキップされることを確認する。

## テスト項目 (Testing for the Requirements)
1.  **手動実行確認**:
    *   `bash scripts/process/build.sh --help`
    *   `bash scripts/process/build.sh`
    *   `bash scripts/process/build.sh --skip-test`
