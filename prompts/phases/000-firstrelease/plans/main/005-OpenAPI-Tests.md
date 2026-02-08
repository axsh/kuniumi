# 005-OpenAPI-Tests

> **Source Specification**: [005-OpenAPI-Tests.md](file:///Users/yam/Axsh%20Dropbox/Yamazaki%20Yasuhiro/Works/myprog/kuniumi/prompts/phases/000-firstrelease/ideas/main/005-OpenAPI-Tests.md)

## Goal Description

CGI・Serveの両モードでOpenAPI仕様を取得し、そのフォーマットが正しいことを自動テストで検証する。CGIモードには現在OpenAPIサポートがないため、コード修正も含む。

## User Review Required

> [!IMPORTANT]
> CGIモードでの `PATH_INFO` 判定について: `openapi.json` を特殊パスとして扱い、関数ディスパッチよりも先に判定します。これにより `openapi.json` という名前の関数を登録することは事実上不可能になりますが、実用上問題ないと判断しています。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| CGIモードでのOpenAPI提供 (`PATH_INFO=/openapi.json`) | Proposed Changes > `adapter_cgi.go` |
| OpenAPI生成ロジックの共通化 (`generateOpenAPISpec`) | Proposed Changes > `openapi.go` (NEW) |
| Serveモードでの OpenAPIテスト独立化 | Proposed Changes > `kuniumi_test.go` |
| CGIモードでのOpenAPIテスト追加 | Proposed Changes > `kuniumi_test.go` |
| OpenAPIフォーマット検証（共通ヘルパー） | Proposed Changes > `kuniumi_test.go` > `assertValidOpenAPISpec` |
| `openapi`=`"3.0.0"` の検証 | `assertValidOpenAPISpec` 内のアサーション |
| `info.title`=`"Calculator"` の検証 | `assertValidOpenAPISpec` 内のアサーション |
| `info.version`=`"1.0.0"` の検証 | `assertValidOpenAPISpec` 内のアサーション |
| `paths./functions/Add.post` の存在確認 | `assertValidOpenAPISpec` 内のアサーション |
| `post.description` の検証 | `assertValidOpenAPISpec` 内のアサーション |
| `requestBody` スキーマの検証 (`x`, `y`) | `assertValidOpenAPISpec` 内のアサーション |
| 各プロパティの `description` 検証 | `assertValidOpenAPISpec` 内のアサーション |
| 各プロパティの `type` = `"integer"` 検証 | `assertValidOpenAPISpec` 内のアサーション |
| `responses.200` の存在確認 | `assertValidOpenAPISpec` 内のアサーション |
| 既存テストへの影響なし | Verification Plan > リグレッション確認 |

## Proposed Changes

### Kuniumi Core Package

#### [NEW] [openapi.go](file:///Users/yam/Axsh%20Dropbox/Yamazaki%20Yasuhiro/Works/myprog/kuniumi/openapi.go)

*   **Description**: OpenAPI仕様生成ロジックを独立ファイルに抽出する。`adapter_http.go` の `serveOpenAPI` 内のロジックと同一。
*   **Technical Design**:
    ```go
    package kuniumi

    import "fmt"

    // generateOpenAPISpec generates a simplified OpenAPI 3.0.0 specification
    // based on the registered functions.
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

---

#### [MODIFY] [adapter_http.go](file:///Users/yam/Axsh%20Dropbox/Yamazaki%20Yasuhiro/Works/myprog/kuniumi/adapter_http.go)

*   **Description**: `serveOpenAPI` メソッドのスペック生成ロジックを `generateOpenAPISpec()` の呼び出しに置き換える。
*   **Technical Design**:
    *   既存の `serveOpenAPI` メソッド（79-117行）を以下に修正:
    ```go
    func (a *App) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
        spec := a.generateOpenAPISpec()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(spec)
    }
    ```

---

#### [MODIFY] [adapter_cgi.go](file:///Users/yam/Axsh%20Dropbox/Yamazaki%20Yasuhiro/Works/myprog/kuniumi/adapter_cgi.go)

*   **Description**: `buildCgiCmd` の `RunE` 内で、関数ディスパッチの前に `openapi.json` パスを判定し、OpenAPI仕様をCGI形式で出力するロジックを追加する。
*   **Technical Design**:
    *   `path_info` の `TrimPrefix` 後（22行目以降）、関数検索ループの前に以下を挿入:
    ```go
    // OpenAPI spec request
    if pathInfo == "openapi.json" {
        fmt.Printf("Content-Type: application/json\r\nStatus: 200 OK\r\n\r\n")
        spec := a.generateOpenAPISpec()
        json.NewEncoder(os.Stdout).Encode(spec)
        return nil
    }
    ```
*   **Logic**:
    1. `PATH_INFO` から `/` プレフィックスを除去して `pathInfo` を取得（既存コード）。
    2. `pathInfo == "openapi.json"` の場合、関数ディスパッチをスキップして OpenAPI 仕様を出力する。
    3. CGIプロトコルに従い、`Content-Type` と `Status` ヘッダーを CRLF 区切りで出力した後、空行を挟んで JSON ボディを出力する。
    4. それ以外は既存の関数ディスパッチロジックに進む。

---

### Integration Tests

#### [MODIFY] [kuniumi_test.go](file:///Users/yam/Axsh%20Dropbox/Yamazaki%20Yasuhiro/Works/myprog/kuniumi/tests/kuniumi/kuniumi_test.go)

*   **Description**: OpenAPI検証ヘルパー関数を追加し、CGI/OpenAPIサブテストを新規追加、Serve内のOpenAPI検証をサブテストとして整理する。
*   **Technical Design**:

    ##### 1. `assertValidOpenAPISpec` ヘルパー関数の追加

    ファイル末尾（`httpGet` の後）に追加:

    ```go
    // assertValidOpenAPISpec validates the structure and content of an OpenAPI spec JSON.
    func assertValidOpenAPISpec(t *testing.T, specJSON []byte) {
        t.Helper()

        var spec map[string]interface{}
        err := json.Unmarshal(specJSON, &spec)
        require.NoError(t, err, "OpenAPI spec should be valid JSON")

        // Top-level fields
        assert.Equal(t, "3.0.0", spec["openapi"], "openapi version should be 3.0.0")

        info, ok := spec["info"].(map[string]interface{})
        require.True(t, ok, "info should be an object")
        assert.Equal(t, "Calculator", info["title"], "info.title should match app name")
        assert.Equal(t, "1.0.0", info["version"], "info.version should match app version")

        paths, ok := spec["paths"].(map[string]interface{})
        require.True(t, ok, "paths should be an object")

        // /functions/Add path
        pathAdd, ok := paths["/functions/Add"].(map[string]interface{})
        require.True(t, ok, "paths should contain /functions/Add")

        post, ok := pathAdd["post"].(map[string]interface{})
        require.True(t, ok, "/functions/Add should have post operation")

        // post.description
        assert.Equal(t, "Adds two integers together", post["description"],
            "post.description should match function description")

        // requestBody schema
        reqBody, ok := post["requestBody"].(map[string]interface{})
        require.True(t, ok, "post should have requestBody")

        content, ok := reqBody["content"].(map[string]interface{})
        require.True(t, ok, "requestBody should have content")

        appJson, ok := content["application/json"].(map[string]interface{})
        require.True(t, ok, "content should have application/json")

        schema, ok := appJson["schema"].(map[string]interface{})
        require.True(t, ok, "application/json should have schema")

        props, ok := schema["properties"].(map[string]interface{})
        require.True(t, ok, "schema should have properties")

        // Check property "x"
        propX, ok := props["x"].(map[string]interface{})
        require.True(t, ok, "properties should contain 'x'")
        assert.Equal(t, "First integer to add", propX["description"])
        assert.Equal(t, "integer", propX["type"])

        // Check property "y"
        propY, ok := props["y"].(map[string]interface{})
        require.True(t, ok, "properties should contain 'y'")
        assert.Equal(t, "Second integer to add", propY["description"])
        assert.Equal(t, "integer", propY["type"])

        // Check responses
        responses, ok := post["responses"].(map[string]interface{})
        require.True(t, ok, "post should have responses")

        resp200, ok := responses["200"].(map[string]interface{})
        require.True(t, ok, "responses should contain '200'")
        assert.NotEmpty(t, resp200["description"], "200 response should have description")
    }
    ```

    ##### 2. `CGI/OpenAPI` サブテストの追加

    既存の `Case 2: CGI Mode` テスト（102-117行）の後に追加:

    ```go
    // Case 2b: CGI OpenAPI
    t.Run("CGI/OpenAPI", func(t *testing.T) {
        cmd := exec.Command(binPath, "cgi")
        cmd.Env = append(os.Environ(), "PATH_INFO=/openapi.json", "REQUEST_METHOD=GET")
        cmd.Stdin = strings.NewReader("")

        var out bytes.Buffer
        cmd.Stdout = &out
        cmd.Stderr = os.Stderr

        require.NoError(t, cmd.Run())

        output := out.String()
        assert.Contains(t, output, "Status: 200 OK")
        assert.Contains(t, output, "Content-Type: application/json")

        // Parse JSON body (after CGI headers separated by \r\n\r\n)
        bodyIdx := strings.Index(output, "\r\n\r\n")
        require.Greater(t, bodyIdx, 0, "CGI output should contain header/body separator")
        body := output[bodyIdx+4:]

        assertValidOpenAPISpec(t, []byte(body))
    })
    ```

    ##### 3. `Serve` テスト内のOpenAPI検証をサブテストに整理

    既存の `Serve` テスト（120-191行）を再構成する。サーバー起動は親テスト `Serve` で行い、機能呼び出しを `Serve/FunctionCall`、OpenAPI検証を `Serve/OpenAPI` として分割する:

    ```go
    // Case 3: Serve Mode (HTTP)
    t.Run("Serve", func(t *testing.T) {
        // Run server in background
        cmd := exec.Command(binPath, "serve", "--port", "9999")
        var stdout, stderr bytes.Buffer
        cmd.Stdout = &stdout
        cmd.Stderr = &stderr

        require.NoError(t, cmd.Start())
        defer func() {
            cmd.Process.Kill()
            cmd.Wait()
        }()

        // Wait for server to start
        time.Sleep(1 * time.Second)

        t.Run("FunctionCall", func(t *testing.T) {
            // POST /functions/Add
            reqBody := []byte(`{"x": 5, "y": 5}`)
            resp, err := httpPost("http://localhost:9999/functions/Add", "application/json", bytes.NewReader(reqBody))
            require.NoError(t, err)
            defer resp.Body.Close()

            assert.Equal(t, 200, resp.StatusCode)

            var result map[string]interface{}
            json.NewDecoder(resp.Body).Decode(&result)
            assert.Equal(t, float64(10), result["result"])
        })

        t.Run("OpenAPI", func(t *testing.T) {
            respSpec, err := httpGet("http://localhost:9999/openapi.json")
            require.NoError(t, err)
            defer respSpec.Body.Close()

            assert.Equal(t, 200, respSpec.StatusCode)

            body, err := io.ReadAll(respSpec.Body)
            require.NoError(t, err)

            assertValidOpenAPISpec(t, body)
        })
    })
    ```

    *   **注意**: `io.ReadAll` を使用するため、import に `io` を追加する。既存の import リスト（5-20行）に `io` は既に含まれているが（`io` パッケージは8行目にある）、`io.ReadAll` は Go 1.16+ で利用可能。

## Step-by-Step Implementation Guide

### TDD アプローチ: テストを先に書き、失敗を確認してから実装する

- [x] **Step 1: テストコードの作成**
    1. `tests/kuniumi/kuniumi_test.go` に `assertValidOpenAPISpec` ヘルパー関数を追加する。
    2. `CGI/OpenAPI` サブテスト（Case 2b）を追加する。
    3. 既存の `Serve` テスト（Case 3）を `Serve/FunctionCall` と `Serve/OpenAPI` サブテストに再構成する。
    4. `Serve/OpenAPI` で `io.ReadAll` を使用するため、import を確認する。

- [x] **Step 2: テスト失敗を確認**
    1. `./scripts/process/build.sh` を実行しビルドが通ることを確認する。
    2. `./scripts/process/integration_test.sh --specify "CGI/OpenAPI"` で `CGI/OpenAPI` テストが失敗することを確認する（CGIモードにOpenAPI未実装のため）。

- [x] **Step 3: `openapi.go` の作成**
    1. プロジェクトルートに `openapi.go` を新規作成する。
    2. `generateOpenAPISpec()` メソッドを、上記 Proposed Changes の通り実装する。

- [x] **Step 4: `adapter_http.go` のリファクタリング**
    1. `serveOpenAPI` メソッド（79-117行）のスペック生成ロジックを削除し、`a.generateOpenAPISpec()` の呼び出しに置き換える。
    2. 不要になった import（`fmt`）があれば削除する。ただし `fmt` は `createHttpHandler` 内でも使用されているため残す。

- [x] **Step 5: `adapter_cgi.go` の修正**
    1. `buildCgiCmd` の `RunE` 内、22行目の `pathInfo = strings.TrimPrefix(pathInfo, "/")` の直後、`fnName` の計算の前に、OpenAPI判定ロジックを挿入する。
    2. `pathInfo == "openapi.json"` の場合に CGI ヘッダー + JSON出力して `return nil` する。

- [x] **Step 6: ビルドと全テスト実行**
    1. `./scripts/process/build.sh && ./scripts/process/integration_test.sh` を実行する。
    2. 全テスト（Help, CGI, CGI/OpenAPI, Serve/FunctionCall, Serve/OpenAPI, VirtualEnv）がパスすることを確認する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests (個別確認)**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --specify "CGI/OpenAPI"
    ```
    *   **Log Verification**: `CGI/OpenAPI` テストが `PASS` になること。出力に `Status: 200 OK` が含まれること。

    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --specify "Serve/OpenAPI"
    ```
    *   **Log Verification**: `Serve/OpenAPI` テストが `PASS` になること。

3.  **Integration Tests (全体リグレッション)**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: 既存テスト (`Help`, `CGI`, `Serve/FunctionCall`, `VirtualEnv`) を含む全テストが `PASS` になること。

## Documentation

本計画による変更は既存の仕様書やドキュメントへ大きな影響を与えない。`doc.go` には既にOpenAPI Generationへの言及があるため更新不要。
