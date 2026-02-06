# テストのルール (Testing Rules)

本プロジェクトでは、テストの実行方法はシェルスクリプト `scripts/` に集約されています。

## 1. テスト実行の標準手順 (Test Execution Matrix)
 
コンポーネントとテストレベルに応じて、適切なスクリプトを使用してください。
 
| Target Component | Test Level | Command | Purpose |
| :--- | :--- | :--- | :--- |
| **Kuniumi** | **Unit** | `scripts/process/build.sh` | ロジックの正当性確認。Fail Fast。 |
| **Kuniumi** | **Integration** | `scripts/process/integration_test.sh` | コンテナやAPIとの連携確認。 |
| **Full Stack** | **Pipeline** | `.agent/workflows/build-pipeline.md` | PR/コミット前の全体健全性確認。 |

## 2. 実行順序のルール (Execution Order Rule)

> [!CRITICAL]
> **Always Build Before Integration Test**
>
> 統合テスト (`scripts/process/integration_test.sh`) を実行する前には、**必ず** ビルド (`scripts/process/build.sh`) を成功させてください。

## 3. エラー修正フロー

テストエラーが発生した場合、以下のフローで修正を行ってください。

1.  **Fail Fast**: `scripts/process/build.sh` が失敗した場合、直ちに修正し再実行する。
2.  **Filter Execution**: 特定の統合テストのみ失敗した場合、`--specify` オプションを使用して**該当テストのみ**を再実行する。
    ```bash
    ./scripts/process/integration_test.sh --specify "TestAuthentication"
    ```
    これにより、全テスト実行（数分）を待つことなく高速にデバッグが可能。
3.  **Full Verification**: 修正後の確認ができたら、最後にオプションなしでスクリプトを実行し、リグレッションがないか確認する。

## 4. 統合テストの構成と命名

> [!WARNING]
> **実行コマンドの制限**:
> `go test` コマンドを直接実行しないでください（特定ディレクトリでの開発作業を除く）。
> 検証やPR前の確認では、必ず `scripts/process/integration_test.sh` または `scripts/process/build.sh` を使用してください。

統合テストは `tests/kuniumi/` 配下に配置します。

*   **タグ**: ファイル先頭に `// +build integration` (または `//go:build integration`) を記述し、単体テストから除外すること。

## 5. テストスキップの禁止

**テストのスキップは厳格に禁止します。** 以下のルールを遵守してください。

### 5.1. スキップの禁止

*   `t.Skip()`, `t.Skipf()`, `t.SkipNow()` の使用は**一切禁止**します。
*   条件分岐でテストを回避することも禁止します。

### 5.2. 必須の対応方法

テストの前提条件が満たされていない場合は、**必ずエラーとして扱う**こと：

**❌ 禁止 (スキップ)**
```go
if !found {
    t.Skip("No Google/Gemini profile found")
}
```

**✅ 推奨 (エラー)**
```go
if !found {
    t.Fatalf("No Google/Gemini profile found in model_profiles.yaml")
}
```

### 5.3. 理由

*   **設定の明示**: テストがスキップされると、必要な設定が欠けていることが見逃される可能性があります。
*   **CI/CD の健全性**: 全テストが実行可能な状態を維持することで、パイプラインの信頼性が向上します。
*   **責任の明確化**: スキップではなくエラーにすることで、問題の解決責任が明確になります。

テストの前提条件（設定ファイル、環境変数、プロファイルなど）が不足している場合は、それらを整備してからテストを実行してください。
