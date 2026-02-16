# 008-OpenAPI-ResponseSchema

> **Source Specification**: [008-OpenAPI-ResponseSchema.md](file:///c:/Users/yam/myprog/kuniumi/prompts/phases/000-firstrelease/ideas/main/008-OpenAPI-ResponseSchema.md)

## Goal Description

`generateOpenAPISpec` が生成するOpenAPI仕様において、`responses.200` にレスポンスボディのJSON Schemaが欠落しているバグを修正する。`FunctionMetadata.Returns` の型情報を活用し、HTTPアダプターの実レスポンス形式と整合するスキーマを生成する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| レスポンススキーマの生成 (`responses.200.content.application/json.schema`) | Proposed Changes > `reflection.go` (`GenerateOutputJSONSchema`) + `openapi.go` |
| 実際のレスポンス形式との整合性（単一: `result`, 複数: `result0`...） | Proposed Changes > `reflection.go` (`GenerateOutputJSONSchema`) |
| `ReturnMetadata.Description` の反映 | Proposed Changes > `reflection.go` (`GenerateOutputJSONSchema`) |
| 戻り値がない場合は `content` を含めない | Proposed Changes > `openapi.go` (`nil` チェック) |
| 統合テスト `assertValidOpenAPISpec` にレスポンススキーマ検証追加 | Proposed Changes > `kuniumi_test.go` |
| `GenerateOutputJSONSchema` 単体テスト | Proposed Changes > `reflection_test.go` |
| 任意: 再利用可能な関数切り出し | Proposed Changes > `reflection.go` (公開関数として実装) |

## Proposed Changes

### kuniumi (core package)

---

#### [MODIFY] [reflection_test.go](file:///c:/Users/yam/myprog/kuniumi/reflection_test.go)
*   **Description**: `GenerateOutputJSONSchema` の単体テストを追加する。
*   **Technical Design**:
    *   テーブル駆動テスト `TestGenerateOutputJSONSchema` を新設。
    *   テスト関数:
        ```go
        func singleReturn(ctx context.Context, x int) (string, error)   { return "", nil }
        func multiReturn(ctx context.Context) (int, string, error)      { return 0, "", nil }
        func noReturn(ctx context.Context) error                        { return nil }
        ```
*   **Logic**:
    *   **Case 1: 単一戻り値（description なし）**
        *   入力: `singleReturn` を `AnalyzeFunction` で解析した `FunctionMetadata`
        *   期待: `{"type": "object", "properties": {"result": {"type": "string"}}}`
    *   **Case 2: 単一戻り値（description あり）**
        *   入力: Case 1 の `meta` に `meta.Returns[0].Description = "test desc"` を設定
        *   期待: `{"type": "object", "properties": {"result": {"type": "string", "description": "test desc"}}}`
    *   **Case 3: 複数戻り値**
        *   入力: `multiReturn` を `AnalyzeFunction` で解析した `FunctionMetadata`
        *   期待: `{"type": "object", "properties": {"result0": {"type": "integer"}, "result1": {"type": "string"}}}`
    *   **Case 4: 戻り値なし（error のみ）**
        *   入力: `noReturn` を `AnalyzeFunction` で解析した `FunctionMetadata`
        *   期待: `nil`

---

#### [MODIFY] [reflection.go](file:///c:/Users/yam/myprog/kuniumi/reflection.go)
*   **Description**: `GenerateOutputJSONSchema` 関数を追加する。`GenerateJSONSchema` の直後（L158の後）に配置。
*   **Technical Design**:
    ```go
    // GenerateOutputJSONSchema generates a JSON Schema for the function return values.
    // The schema matches the response format used by the HTTP adapter:
    //   - Single return: {"type": "object", "properties": {"result": <schema>}}
    //   - Multiple returns: {"type": "object", "properties": {"result0": <schema>, "result1": <schema>, ...}}
    //   - No returns (error only): nil
    func GenerateOutputJSONSchema(meta *FunctionMetadata) map[string]interface{}
    ```
*   **Logic**:
    1. `len(meta.Returns) == 0` の場合、`nil` を返す。
    2. `properties := make(map[string]interface{})`
    3. `len(meta.Returns) == 1` の場合:
       - `schema := typeToSchema(meta.Returns[0].Type)`
       - `meta.Returns[0].Description != ""` ならば `schema["description"] = meta.Returns[0].Description`
       - `properties["result"] = schema`
    4. `len(meta.Returns) > 1` の場合:
       - 各 `i, ret := range meta.Returns` について:
         - `schema := typeToSchema(ret.Type)`
         - `ret.Description != ""` ならば `schema["description"] = ret.Description`
         - `properties[fmt.Sprintf("result%d", i)] = schema`
    5. `return map[string]interface{}{"type": "object", "properties": properties}`

---

#### [MODIFY] [openapi.go](file:///c:/Users/yam/myprog/kuniumi/openapi.go)
*   **Description**: `generateOpenAPISpec` のレスポンス定義にスキーマを追加する。
*   **Technical Design**: L33-38 のレスポンス部分を変更。
*   **Logic**:
    *   現在のコード:
        ```go
        "responses": map[string]any{
            "200": map[string]any{
                "description": "Successful execution",
            },
        },
        ```
    *   変更後:
        ```go
        // Build response definition
        responseDef := map[string]any{
            "description": "Successful execution",
        }
        outputSchema := GenerateOutputJSONSchema(fn.Meta)
        if outputSchema != nil {
            responseDef["content"] = map[string]any{
                "application/json": map[string]any{
                    "schema": outputSchema,
                },
            }
        }
        // ... "responses": map[string]any{"200": responseDef} をパスの定義に組み込む
        ```
    *   `outputSchema` が `nil`（error のみの関数）の場合は `content` を付与せず、現行動作を維持する。

---

### kuniumi (integration tests)

---

#### [MODIFY] [kuniumi_test.go](file:///c:/Users/yam/myprog/kuniumi/tests/kuniumi/kuniumi_test.go)
*   **Description**: `assertValidOpenAPISpec` にレスポンススキーマの検証を追加する。
*   **Technical Design**: L308-314（現在の `responses` 検証部分の後）にレスポンススキーマ検証を追加。
*   **Logic**:
    *   `resp200["description"]` の検証（既存、維持）の後に以下を追加:
        ```go
        // Check response body schema
        respContent, ok := resp200["content"].(map[string]interface{})
        require.True(t, ok, "200 response should have content")

        respAppJson, ok := respContent["application/json"].(map[string]interface{})
        require.True(t, ok, "response content should have application/json")

        respSchema, ok := respAppJson["schema"].(map[string]interface{})
        require.True(t, ok, "response application/json should have schema")

        assert.Equal(t, "object", respSchema["type"], "response schema type should be object")

        respProps, ok := respSchema["properties"].(map[string]interface{})
        require.True(t, ok, "response schema should have properties")

        resultProp, ok := respProps["result"].(map[string]interface{})
        require.True(t, ok, "response properties should contain 'result'")
        assert.Equal(t, "integer", resultProp["type"],
            "result type should be integer for Add function")
        assert.Equal(t, "Sum of x and y", resultProp["description"],
            "result description should match WithReturns value")
        ```
    *   テスト対象の `Add` 関数は `(int, error)` を返し、`WithReturns("Sum of x and y")` が設定されている。

## Step-by-Step Implementation Guide

1.  **単体テストの追加 (TDD: Red)**:
    *   `reflection_test.go` にテスト用関数 (`singleReturn`, `multiReturn`, `noReturn`) と `TestGenerateOutputJSONSchema` を追加する。
    *   `./scripts/process/build.sh` を実行し、コンパイルエラー（`GenerateOutputJSONSchema` 未定義）を確認する。

2.  **`GenerateOutputJSONSchema` の実装 (TDD: Green)**:
    *   `reflection.go` の L158（`GenerateJSONSchema` の直後）に `GenerateOutputJSONSchema` を追加する。
    *   `./scripts/process/build.sh` を実行し、単体テストがパスすることを確認する。

3.  **`openapi.go` の修正**:
    *   `generateOpenAPISpec` のレスポンス定義部分 (L33-38) を修正し、`GenerateOutputJSONSchema` を使用してレスポンススキーマを含めるようにする。
    *   `./scripts/process/build.sh` を実行し、ビルドが成功することを確認する。

4.  **統合テストの更新 (TDD: Red → Green)**:
    *   `tests/kuniumi/kuniumi_test.go` の `assertValidOpenAPISpec` にレスポンススキーマ検証を追加する。
    *   `./scripts/process/build.sh && ./scripts/process/integration_test.sh` を実行し、全テストがパスすることを確認する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `TestGenerateOutputJSONSchema` の全ケース（単一/複数/ゼロ戻り値、description有無）がパスすること。
    *   既存の `TestConvertStringToType`, `TestCallFunction_StringArgs` が引き続きパスすること。

2.  **Integration Tests**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   `TestKuniumiIntegration/CGI/OpenAPI`: CGIモードで取得したOpenAPI仕様にレスポンススキーマが含まれていること。
    *   `TestKuniumiIntegration/Serve/OpenAPI`: ServeモードでのOpenAPI仕様にレスポンススキーマが含まれていること。
    *   **Log Verification**: テスト出力に `FAIL` がないこと。特に `assertValidOpenAPISpec` でのアサーションエラーがないことを確認。

## Documentation

該当ドキュメントに影響なし。`prompts/specifications/kuniumu-architechture.md` にはOpenAPIの詳細な仕様記述がなく、更新不要。
