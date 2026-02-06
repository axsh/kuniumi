# README更新仕様書: Kuniumiイントロダクション

## 1. 背景 (Background)

現在の `README.md` は非常に簡素な説明（1行のみ）しか記述されていません。
プロジェクトの顔として、訪問者が「Kuniumiとは何か」「どのような特徴があるか」「どのように使い始めるか」を一目で理解できるような、充実したイントロダクションが必要です。
また、アーキテクチャの詳細なドキュメント (`prompts/specifications/kuniumu-architechture.md`) が既に存在するため、そこへの適切な誘導も求められています。

## 2. 要件 (Requirements)

`README.md` を以下の構成で全面的に書き換えます。英語で記述します。

1.  **プロジェクト名とキャッチコピー**:
    *   `kuniumi` のロゴ（あれば）またはタイトル。
    *   簡潔で強力な説明文。
2.  **概要 (Overview)**:
    *   Go言語の関数をポータブルなWebサービスとして公開するフレームワークであることを説明。
    *   1回書けばどこでも動く ("Write Once, Run Anywhere") コンセプトの提示。
3.  **主な特徴 (Key Features)**:
    *   **Multi-Interface**: HTTP, MCP, CGI, Docker をサポート。
    *   **Virtual Environment**: ファイルシステムと環境変数の抽象化による安全性とポータビリティ。
    *   **Type Safety**: Goの型システムを活用した堅牢性。
4.  **クイックスタート (Quick Start)**:
    *   インストール方法 (`go get ...`)。
    *   最小限のコード例 (`main.go`)。
    *   実行コマンド例（HTTPサーバー起動、MCPモードなど）。
5.  **ドキュメントへのリンク (Documentation)**:
    *   より詳細な技術情報として `prompts/specifications/kuniumu-architechture.md` (Architecture Overview) へのリンクを明記。
6.  **ライセンス (License)**:
    *   ライセンス情報の記載（既存の `LICENSE` ファイルへの言及）。

## 3. 実現方針 (Implementation Approach)

*   既存の `README.md` を上書きします。
*   内容は `prompts/specifications/kuniumu-architechture.md` の「概要」や「クイックスタート」セクションをベースに、GitHubのREADMEとして見やすい形式（バッジ、コードハイライトなど）に整形します。
*   詳細な仕様やAPIリファレンスは `README.md` には含めず、リンク先 (`prompts/specifications/kuniumu-architechture.md`) に誘導することで、ドキュメントの二重管理を防ぎます。

## 4. 検証シナリオ (Verification Scenarios)

1.  **レンダリング確認**:
    *   VS Codeのプレビュー機能 (`Markdown: Open Preview`) を使用し、レイアウト、リンク、コードブロックが正しく表示されることを確認する。
    *   特に `prompts/specifications/kuniumu-architechture.md` へのリンクが正しく機能するか確認する（相対パス）。

## 5. テスト項目 (Testing for the Requirements)

*   **自動ビルド**:
    *   `scripts/process/build.sh` を実行し、ドキュメントの変更がビルドプロセスに悪影響を与えないこと（通常はありませんが念のため）を確認します。
*   **リンクチェック**:
    *   (手動) VSCode上でリンクをクリックし、対象ファイルが開くことを確認します。
