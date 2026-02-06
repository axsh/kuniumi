# 計画立案規範 (Planning Rules)

本規範は、計画立案プロセス、品質管理、およびビルド/テストの実行基準を定め、プロジェクトの円滑な進行を確保することを目的とします。

## 1. テストと品質に対する共通哲学 (General Philosophy)

コンポーネントと言語に関わらず、以下の原則を遵守してください。

### 1.1 TDD (Test Driven Development)
*   **Failed First**: いかなる機能も、「その機能が正しく動作することを確認するテスト」を先に書き、それが失敗することを確認してから実装を開始してください。
*   **Verification Oriented**: 「どう実装するか」の前に「どう検証するか（テストするか）」を計画してください。検証不可能な機能は実装してはいけません。

### 1.2 Fail Fast
*   フィードバックループを最小化するため、単体テスト（Unit Test）は高速に実行可能であることを維持してください。
*   統合テスト等の重いテストを回す前に、単体テストで論理的な誤りを排除してください。

### 1.3 Concretization (具体化)
*   **Abstraction is Logic Loss**: 仕様書から実装計画への変換プロセスは、抽象化（要約）ではなく、**具体化（詳細化）**のプロセスでなければなりません。
    *   **Data Structure is Mandatory**: 仕様書に記載された構造体定義やデータモデルは、必ず実装計画の `Proposed Changes` に含めてください。これらを「自明な既存コード」や「コンテキスト情報」として省略してはいけません。
*   **Traceability**: 前段のドキュメント（仕様書）に記載された具体的な手順、条件、シナリオ（例: 「(1) Aして (2) Bする」）は、実装計画においても個別の検証項目または実装手順として明確に追跡可能である必要があります。

## 2. 計画策定における要件 (Planning Requirements)

実装計画 (`prompts/phases/.../plans/...`) を作成する際は、**「これを読めば実装の実現方法が明確にわかる」** レベルの詳細さが求められます。

- どのファイルを新規に作成すべきか、もしくはどのファイルをどのように修正すべきかを明確に記述してください。
- コーディングする際に必要な情報を、具体的に記載してください。

*   **詳細な実現手順**:
    *   どのパッケージの、どの構造体/インターフェースに変更を加えるか。
    *   追加するAPIエンドポイントの定義（パス、メソッド、リクエスト/レスポンス）。
*   **テスト計画 (Unit & Integration)**:
    *   **Unit Tests**: テーブル駆動テスト (`tests := []struct{...}`) のケース設計。モックが必要な依存関係（DB, 外部API）。
    *   **Integration Tests**: `tests/` 配下のどのファイルに追加するか。実際のDBやDockerコンテナとの連携確認手順。
    *   記述順序: `Proposed Changes` では必ず `_test.go` を先に記述してください。

## 3. ビルドとテスト方針

本プロジェクトは `.agent/workflows/` および `scripts/` を活用した標準化されたプロセスを採用しています。

### 3.1 標準化された実行手段

全ての検証は `scripts/` 配下のスクリプトを通じて行ってください。

*   Unit: `scripts/process/build.sh`
*   Integration: `scripts/process/integration_test.sh`

**Note**: Do not use `cd` commands in plans. Use paths relative to the project root.

> [!WARNING]
> **Prohibition of Raw Toolchain Commands**:
> You are **PROHIBITED** from suggesting raw `go build`, or `go test` commands in implementation plans.
> You **MUST** use the provided scripts (`scripts/process/build.sh`, `scripts/process/integration_test.sh`).
> *Reason*: These scripts handle critical build steps (binary relocation, environment variables, incremental build logic) that raw commands miss.

### 3.2 ビルドと検証要件 (Build & Verification Requirement)

実装計画の「Verification Plan」には、検証に行うための**スクリプト実行コマンド**を必ず明記してください。

*   **推奨記述例**:
    1.  **Automated Verification**: Run `./scripts/process/build.sh && ./scripts/process/integration_test.sh` to verify the changes.

    > [!WARNING]
    > **Prohibition of Manual Verification**:
    > You are **PROHIBITED** from planning "Manual Verification" as the primary verification method.
    > Do not write steps like "Open VSCode and click button". Instead, write "Create E2E test scenario to click button".
    > *Exception*: Strictly visual aesthetics (e.g. "Check if the color is correct") can be manual, but functional logic MUST be automated.
    
    *   **Prohibited Commands**:
        *   ❌ `go build`, `go test`
        *   ❌ `cd ... && ...`
        *   ❌ `integration_test.sh` (STRICTLY PROHIBITED without preceding `build.sh`)
    *   **Required Commands**:
        *   ✅ `./scripts/process/build.sh && ./scripts/process/integration_test.sh`
