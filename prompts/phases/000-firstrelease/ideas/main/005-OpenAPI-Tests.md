# 005: OpenAPI取得テストの追加（CGI / Serve両対応）

## 背景 (Background)

Kuniumiフレームワークは、登録されたGo関数をHTTP（Serveモード）やCGIモードなど複数のインターフェースで公開する。Serveモードでは `GET /openapi.json` エンドポイントでOpenAPI仕様を提供しており、`tests/kuniumi/kuniumi_test.go` の `Serve` サブテスト内でも既にOpenAPIの基本的な検証が行われている。

しかし、以下の課題がある：

1. **CGIモードにOpenAPIサポートがない**: `adapter_cgi.go` は`PATH_INFO`に基づいて関数をディスパッチするのみで、OpenAPI仕様を取得する手段が提供されていない。
2. **テストの構造**: OpenAPIの検証が`Serve`サブテスト内にインラインで埋め込まれており、独立したテストケースとして管理されていない。
3. **検証の網羅性**: 現在のテストではスキーマの一部（引数のdescription）のみを検証しており、OpenAPIフォーマット全体の正しさを十分に確認していない。

## 要件 (Requirements)

### 必須要件

1. **CGIモードでのOpenAPI提供**
   - CGIモードで `PATH_INFO=/openapi.json` かつ `REQUEST_METHOD=GET` を指定した場合、OpenAPI仕様がJSON形式で出力されること。
   - 出力にはCGIヘッダー（`Content-Type: application/json`, `Status: 200 OK`）が含まれること。

2. **Serveモードでの OpenAPIテスト**
   - `Serve` テスト内のOpenAPI検証を独立したサブテスト `Serve/OpenAPI` として分離する。
   - OpenAPIレスポンスのフォーマットが正しいことを検証する。

3. **CGIモードでのOpenAPIテスト**
   - `CGI/OpenAPI` サブテストを新規追加する。
   - CGIモードで取得したOpenAPIレスポンスのフォーマットが正しいことを検証する。

4. **OpenAPIフォーマットの検証項目**（CGI/Serve共通）
   - トップレベルフィールド: `openapi` (= `"3.0.0"`), `info`, `paths` が存在すること。
   - `info.title` がアプリ名（`"Calculator"`）と一致すること。
   - `info.version` がバージョン（`"1.0.0"`）と一致すること。
   - `paths` に `/functions/Add` が含まれること。
   - `/functions/Add` の `post` オペレーションが存在すること。
   - `post.description` が関数の説明文と一致すること。
   - `post.requestBody.content.application/json.schema` が存在し、`properties` に `x`, `y` が含まれること。
   - 各プロパティの `description` が正しいこと（`x`: `"First integer to add"`, `y`: `"Second integer to add"`）。
   - 各プロパティの `type` が `"integer"` であること。
   - `post.responses.200` が存在し、`description` フィールドがあること。

### 任意要件

- 将来的に他のアダプター（MCP等）にもOpenAPI相当の仕様出力を追加する際の参考となる設計にする。

## 実現方針 (Implementation Approach)

### 1. `adapter_cgi.go` の修正

CGIコマンドのRunE内で、`PATH_INFO` が `/openapi.json`（または `openapi.json`）の場合に、既存の `serveOpenAPI` と同等のOpenAPI仕様をCGI形式で出力するロジックを追加する。

```go
// adapter_cgi.go の RunE 内（関数ディスパッチの前）
pathInfo := os.Getenv("PATH_INFO")
pathInfo = strings.TrimPrefix(pathInfo, "/")

// OpenAPI spec request
if pathInfo == "openapi.json" {
    fmt.Printf("Content-Type: application/json\r\nStatus: 200 OK\r\n\r\n")
    spec := a.generateOpenAPISpec()
    json.NewEncoder(os.Stdout).Encode(spec)
    return nil
}
```

### 2. OpenAPI生成ロジックの共通化

`adapter_http.go` の `serveOpenAPI` メソッド内にあるOpenAPI仕様生成ロジックを、`App` のメソッド `generateOpenAPISpec() map[string]any` として抽出する。CGIとHTTPの両アダプターからこのメソッドを呼び出す。

```go
// app.go または adapter_http.go に追加
func (a *App) generateOpenAPISpec() map[string]any {
    spec := map[string]any{
        "openapi": "3.0.0",
        "info": map[string]string{
            "title":   a.config.Name,
            "version": a.config.Version,
        },
        "paths": map[string]any{},
    }
    paths := spec["paths"].(map[string]any)
    for _, fn := range a.functions {
        path := fmt.Sprintf("/functions/%s", fn.Name)
        schema := GenerateJSONSchema(fn.Meta)
        paths[path] = map[string]any{
            "post": map[string]any{
                "description": fn.Description,
                "requestBody": map[string]any{
                    "content": map[string]any{
                        "application/json": map[string]any{
                            "schema": schema,
                        },
                    },
                },
                "responses": map[string]any{
                    "200": map[string]any{
                        "description": "Successful execution",
                    },
                },
            },
        }
    }
    return spec
}
```

### 3. テストの追加・修正 (`tests/kuniumi/kuniumi_test.go`)

#### 3.1 OpenAPI検証ヘルパー関数の作成

CGI/Serve両方で同じ検証を行うため、検証ロジックをヘルパー関数に切り出す。

```go
func assertValidOpenAPISpec(t *testing.T, specJSON []byte) {
    t.Helper()
    var spec map[string]interface{}
    err := json.Unmarshal(specJSON, &spec)
    require.NoError(t, err, "OpenAPI spec should be valid JSON")

    // Check top-level fields
    assert.Equal(t, "3.0.0", spec["openapi"])
    // ... (info, paths, etc.)
}
```

#### 3.2 `CGI/OpenAPI` サブテストの追加

```go
t.Run("CGI/OpenAPI", func(t *testing.T) {
    cmd := exec.Command(binPath, "cgi")
    cmd.Env = append(os.Environ(), "PATH_INFO=/openapi.json", "REQUEST_METHOD=GET")
    // CGI doesn't read stdin for GET, but provide empty reader
    cmd.Stdin = strings.NewReader("")
    var out bytes.Buffer
    cmd.Stdout = &out
    require.NoError(t, cmd.Run())

    output := out.String()
    assert.Contains(t, output, "Status: 200 OK")

    // Parse JSON body (after headers)
    bodyIdx := strings.Index(output, "\r\n\r\n")
    require.Greater(t, bodyIdx, 0)
    body := output[bodyIdx+4:]

    assertValidOpenAPISpec(t, []byte(body))
})
```

#### 3.3 `Serve/OpenAPI` サブテストの独立化

既存の`Serve`テスト内のOpenAPI検証は残しつつ、サーバーの起動を共有する形で `t.Run("Serve", func(t *testing.T) { ... })` の内側にネストしたサブテストとして整理する。

## 検証シナリオ (Verification Scenarios)

### シナリオ1: CGIモードでOpenAPIを取得する

1. ビルドした `kuniumi_example` バイナリに対し、`PATH_INFO=/openapi.json` と `REQUEST_METHOD=GET` を環境変数に設定してCGIモードで実行する。
2. 標準出力にCGIヘッダー（`Content-Type: application/json`, `Status: 200 OK`）が出力される。
3. ヘッダーの後にJSON形式のOpenAPI仕様が出力される。
4. JSONをパースし、以下を確認する：
   - `openapi` が `"3.0.0"` である。
   - `info.title` が `"Calculator"` である。
   - `info.version` が `"1.0.0"` である。
   - `paths./functions/Add.post` が存在する。
   - `post.requestBody.content.application/json.schema.properties` に `x` と `y` が存在する。
   - `x.description` が `"First integer to add"` である。
   - `y.description` が `"Second integer to add"` である。
   - `x.type` と `y.type` が `"integer"` である。
   - `post.responses.200` が存在する。

### シナリオ2: ServeモードでOpenAPIを取得する

1. `kuniumi_example serve --port 9999` でHTTPサーバーを起動する。
2. `GET http://localhost:9999/openapi.json` にHTTPリクエストを送信する。
3. レスポンスのステータスコードが`200`であることを確認する。
4. レスポンスボディのJSONをパースし、シナリオ1の手順4と同じ項目を検証する。

### シナリオ3: CGIモードで存在しない関数に対するOpenAPIリクエスト以外の動作が壊れていないこと

1. 既存の `CGI` テスト（`PATH_INFO=/Add` での関数呼び出し）が引き続きパスすること。
2. 既存の `VirtualEnv` テスト（CGIモード＋マウント）が引き続きパスすること。

## テスト項目 (Testing for the Requirements)

| 要件 | テストケース | 検証方法 |
|------|------------|---------|
| CGIモードでのOpenAPI提供 | `TestKuniumiIntegration/CGI/OpenAPI` | `scripts/process/integration_test.sh` |
| ServeモードでのOpenAPIテスト | `TestKuniumiIntegration/Serve/OpenAPI` | `scripts/process/integration_test.sh` |
| OpenAPIフォーマットの正しさ（CGI） | `assertValidOpenAPISpec` ヘルパーによる検証 | `scripts/process/integration_test.sh` |
| OpenAPIフォーマットの正しさ（Serve） | `assertValidOpenAPISpec` ヘルパーによる検証 | `scripts/process/integration_test.sh` |
| 既存テストへの影響なし | 全テストケースのパス | `scripts/process/build.sh` + `scripts/process/integration_test.sh` |

### 検証コマンド

```bash
# 全体ビルド & 単体テスト
./scripts/process/build.sh

# 統合テスト実行
./scripts/process/integration_test.sh
```
