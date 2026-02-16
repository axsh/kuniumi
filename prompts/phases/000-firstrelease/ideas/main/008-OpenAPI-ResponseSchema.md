# 008: OpenAPIレスポンススキーマの追加

## 背景 (Background)

Kuniumiフレームワークの `generateOpenAPISpec` メソッド（`openapi.go`）は、登録された関数に基づいてOpenAPI 3.0.0仕様を生成する。現在、リクエストボディのスキーマ（`requestBody.content.application/json.schema`）は `GenerateJSONSchema` を使用して正しく生成されている。

しかし、レスポンス定義（`responses.200`）には `description` フィールドのみが設定されており、**レスポンスボディのスキーマが含まれていない**。これはOpenAPI仕様の活用（クライアントコード生成、API仕様書としての利用、MCP連携時のスキーマ参照等）において大きな欠陥である。

一方で、必要な情報はすでに内部に存在している：

- `FunctionMetadata.Returns []ReturnMetadata` に戻り値の型情報（`reflect.Type`）と説明が保持されている
- `typeToSchema` 関数により `reflect.Type` → JSON Schema変換が可能
- HTTPアダプター（`adapter_http.go`）では戻り値が1個の場合 `{"result": value}` 、複数の場合 `{"result0": ..., "result1": ...}` 形式でレスポンスを返している

つまり、レスポンススキーマを正しく生成するための素材はすべて揃っているが、`openapi.go` でそれらを組み立てていないという実装漏れである。

## 要件 (Requirements)

### 必須要件

1. **レスポンススキーマの生成**
   - `generateOpenAPISpec` が生成するOpenAPI仕様の `responses.200` に、レスポンスボディのJSON Schemaを含めること。
   - スキーマは `responses.200.content.application/json.schema` に配置すること。

2. **実際のレスポンス形式との整合性**
   - HTTPアダプター (`adapter_http.go`) の `createHttpHandler` が返す実際のJSON形式と一致するスキーマを生成すること。
   - 戻り値が1つの場合: `{"type": "object", "properties": {"result": <typeSchema>}}`
   - 戻り値が複数の場合: `{"type": "object", "properties": {"result0": <typeSchema0>, "result1": <typeSchema1>, ...}}`
   - 戻り値の `ReturnMetadata.Description` が設定されている場合、対応するプロパティに `description` フィールドを含めること。

3. **戻り値がない場合の扱い**
   - 戻り値が error のみ（`Returns` が空）の関数の場合、`responses.200` には `description` のみを設定し、`content` は含めない（現行動作を維持）。

4. **テストの更新**
   - 統合テスト `assertValidOpenAPISpec` にレスポンススキーマの検証項目を追加すること。
   - テスト対象の `Add` 関数（`func Add(ctx, int, int) (int, error)`）のレスポンススキーマとして、`result` プロパティが `integer` 型であることを検証すること。

### 任意要件

- `GenerateOutputJSONSchema` のような戻り値専用のスキーマ生成関数を切り出すことで、MCP等の他アダプターでの再利用性を高める。

## 実現方針 (Implementation Approach)

### 1. レスポンススキーマ生成関数の追加 (`reflection.go`)

`GenerateOutputJSONSchema` 関数を新設し、`FunctionMetadata.Returns` に基づいてレスポンスボディのJSON Schemaを生成する。

```go
// GenerateOutputJSONSchema generates a JSON Schema for the function return values.
// The schema matches the response format used by the HTTP adapter.
func GenerateOutputJSONSchema(meta *FunctionMetadata) map[string]interface{} {
    if len(meta.Returns) == 0 {
        return nil
    }

    properties := make(map[string]interface{})
    if len(meta.Returns) == 1 {
        schema := typeToSchema(meta.Returns[0].Type)
        if meta.Returns[0].Description != "" {
            schema["description"] = meta.Returns[0].Description
        }
        properties["result"] = schema
    } else {
        for i, ret := range meta.Returns {
            schema := typeToSchema(ret.Type)
            if ret.Description != "" {
                schema["description"] = ret.Description
            }
            properties[fmt.Sprintf("result%d", i)] = schema
        }
    }

    return map[string]interface{}{
        "type":       "object",
        "properties": properties,
    }
}
```

### 2. `openapi.go` の修正

`generateOpenAPISpec` メソッドのレスポンス定義に、生成したスキーマを追加する。

```go
// responses 部分の変更
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

// "responses": map[string]any{"200": responseDef}
```

### 3. テストの更新 (`tests/kuniumi/kuniumi_test.go`)

`assertValidOpenAPISpec` 関数に、レスポンススキーマの検証を追加する。

```go
// Check response schema
respContent, ok := resp200["content"].(map[string]interface{})
require.True(t, ok, "200 response should have content")

respAppJson, ok := respContent["application/json"].(map[string]interface{})
require.True(t, ok, "response content should have application/json")

respSchema, ok := respAppJson["schema"].(map[string]interface{})
require.True(t, ok, "response application/json should have schema")

respProps, ok := respSchema["properties"].(map[string]interface{})
require.True(t, ok, "response schema should have properties")

resultProp, ok := respProps["result"].(map[string]interface{})
require.True(t, ok, "response properties should contain 'result'")
assert.Equal(t, "integer", resultProp["type"], "result type should be integer for Add function")
```

## 検証シナリオ (Verification Scenarios)

### シナリオ1: 単一戻り値関数のレスポンススキーマ

1. ビルドした `kuniumi_example` バイナリに対し、`GET /openapi.json` でOpenAPI仕様を取得する。
2. `/functions/Add` の `post.responses.200` を確認する。
3. `content.application/json.schema` が存在すること。
4. `schema.type` が `"object"` であること。
5. `schema.properties.result.type` が `"integer"` であること（Add関数の戻り値は `int`）。
6. `schema.properties.result.description` が `"Sum of x and y"` であること（`WithReturns` で設定した説明）。

### シナリオ2: CGIモードでも同じレスポンススキーマが含まれること

1. CGIモード (`PATH_INFO=/openapi.json`, `REQUEST_METHOD=GET`) でOpenAPI仕様を取得する。
2. シナリオ1と同じレスポンススキーマが含まれていることを確認する。

### シナリオ3: 既存テストへの影響がないこと

1. 全ビルド・単体テスト・統合テストがパスすること。

## テスト項目 (Testing for the Requirements)

| 要件 | テストケース | 検証方法 |
|------|------------|---------|
| レスポンススキーマの生成 | `assertValidOpenAPISpec` 内のレスポンススキーマ検証 | `scripts/process/integration_test.sh` |
| 実際のレスポンス形式との整合性 | `assertValidOpenAPISpec` で `result` プロパティが `integer` 型 | `scripts/process/integration_test.sh` |
| CGI/Serve両モード対応 | `CGI/OpenAPI` および `Serve/OpenAPI` サブテスト | `scripts/process/integration_test.sh` |
| 既存テストへの影響なし | 全テストケースのパス | `scripts/process/build.sh` + `scripts/process/integration_test.sh` |
| `GenerateOutputJSONSchema` 単体テスト | 単一/複数/ゼロ戻り値のケース | `scripts/process/build.sh` |

### 検証コマンド

```bash
# 全体ビルド & 単体テスト
./scripts/process/build.sh

# 統合テスト実行
./scripts/process/integration_test.sh
```
