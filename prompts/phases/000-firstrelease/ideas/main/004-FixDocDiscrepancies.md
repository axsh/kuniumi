# ドキュメントと実装の乖離修正仕様書 (Comprehensive)

## 1. 背景 (Background)

ユーザーからの指摘により、`README.md` および `prompts/specifications/kuniumu-architechture.md` が最新のソースコード実装と乖離していることが判明しました。
全ソースコードの調査を実施した結果、以下の点において修正が必要であることが確認されました。これらを統一し、正確な技術ドキュメントを提供する必要があります。

## 2. 実装とドキュメントの乖離一覧 (Discrepancies)

調査の結果、以下の乖離が特定されました。

### 2.1 関数登録 API (Function Registration)
*   **Code**: `options.go` にて `WithParams(...ParamDef)` が定義され、`examples/basic/main.go` でも使用されている推奨APIです。`WithArgs` は `app.go` に存在しますが、パラメータの説明文 (`Description`) を付与できないため、実質的に下位互換/非推奨扱いです。
*   **Docs**: `README.md` および `kuniumu-architechture.md` は `WithArgs` を使用しています。
*   **Action**: 全ドキュメントのサンプルコードとAPIリファレンスを `WithParams` に統一します。

### 2.2 VirtualEnvironment API
*   **Code**: `virtual_env.go` には以下のメソッドが存在します。
    *   `Getenv`, `ListEnv`
    *   `WriteFile`, `ReadFile`, `RewriteFile`, `CopyFile`, `RemoveFile`, `Chmod`
    *   `ListFile`, `FindFile`, `ChangeCurrentDirectory`, `GetCurrentDirectory`
*   **Docs**: `kuniumu-architechture.md` (前回の簡易更新後) に一部が追加されましたが、`FindFile` や `ListFile` の詳細な仕様（戻り値の型 `FileInfo` など）や、`RewriteFile` の挙動詳細が不足している可能性があります。
*   **Action**: `kuniumu-architechture.md` の `VirtualEnvironment` セクションを更新し、すべてのパブリックメソッドを網羅します。

### 2.3 Adapter の仕様詳細
*   **Code (`adapter_container.go`)**: Dockerfile生成時に `golang:1.24-alpine` をベースイメージとして使用しています。
*   **Code (`adapter_http.go`)**: `POST /functions/{name}` で関数を実行し、`GET /openapi.json` でOpenAPIスペックを提供します。
*   **Docs**: これらの具体的なエンドポイント仕様や内部で使用されるバージョン情報が明記されていません。
*   **Action**: `kuniumu-architechture.md` のアダプターセクションに詳細を追記します。

## 3. 要件 (Requirements)

発見された乖離を解消するため、以下の修正を行います。

### 修正対象: `README.md`
1.  **Quick Start コードの修正**:
    *   `WithArgs` を `WithParams` に変更する。
    *   `main.go` の例として、`examples/basic/main.go` のエッセンス（`VirtualEnvironment` の取得、ログ出力のデモなど）をもう少し反映させ、機能の紹介として適切なものにする（ただし長くなりすぎないように）。

### 修正対象: `prompts/specifications/kuniumu-architechture.md`
1.  **クイックスタートの修正**:
    *   `README.md` と同様に `WithParams` を使用するコードに更新。
2.  **API リファレンスの修正 (`RegisterFunc` options)**:
    *   `WithParams` と `Param` の説明を詳細に追加。
    *   `WithArgs` は「非推奨」としてマークするか、削除する。
    *   `WithReturns` の説明を追加。
3.  **API リファレンスの修正 (`VirtualEnvironment`)**:
    *   `FindFile`, `ListFile` を含む全メソッドをリストアップ。
    *   `FileInfo` 構造体の定義を記載。
4.  **各アダプターの仕様詳細の追記**:
    *   HTTPアダプターのエンドポイント仕様。
    *   Containerアダプターの生成するDockerfileのベースイメージ情報など。

## 4. 実現方針 (Implementation Approach)

*   ソースコード (`examples/basic/main.go` 等) を「正」としてドキュメントを書き換えます。
*   `README.md` は、プロジェクトの顔として「動くコード」を提示することを最優先します。

## 5. 検証シナリオ (Verification Scenarios)

1.  **ドキュメントコードの動作確認**:
    *   修正後の `README.md` に記載するコードブロックを、実際に `tmp/verify_readme.go` として保存し、`go build` が成功することを確認します。
    *   これにより、ドキュメント上のコードが嘘（コンパイルエラーになるコード）でないことを保証します。

## 6. テスト項目 (Testing for the Requirements)

*   **Build Verification**:
    *   実装計画の一部として、修正後のドキュメントコードをコンパイルするステップを含めます。
