# 010-MCP-Response-JSON-Format

> **Source Specification**: prompts/phases/000-firstrelease/ideas/main/010-MCP-Response-JSON-Format.md

## Goal Description

Unify response formats across all adapters (MCP, HTTP, CGI) by:
- Making MCP responses return JSON matching the OpenAPI response schema (`{"result": <value>}`)
- Converting all error responses (MCP, HTTP, CGI) to a standardized JSON format (`{"error": "<message>"}`)
- Adding error response schemas (400, 500) to the generated OpenAPI specification
- Extracting shared response construction logic to eliminate duplication

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req 1: MCP success response JSON | Proposed Changes > MCP Adapter (`adapter_mcp.go`) |
| Req 2: MCP error response JSON | Proposed Changes > MCP Adapter (`adapter_mcp.go`) |
| Req 3: HTTP error response JSON | Proposed Changes > HTTP Adapter (`adapter_http.go`) |
| Req 4: CGI error response JSON | Proposed Changes > CGI Adapter (`adapter_cgi.go`) |
| Req 5: OpenAPI error schema | Proposed Changes > OpenAPI Spec Generator (`openapi.go`) |
| Req 6: Backward compatibility | Preserved by using the same `buildSuccessResponse` for all adapters |
| Req 7: Shared response builder | Proposed Changes > Response Helpers (`response.go`) |

## Proposed Changes

### Response Helpers

#### [NEW] `response_test.go`(file:///c:/Users/yamya/myprog/kuniumi/response_test.go)

- **Description**: Unit tests for shared response builder functions. Written first per TDD.
- **Technical Design**:
  - Table-driven tests for `buildSuccessResponse` and `buildErrorResponse`
  - Tests are in `package kuniumi` (white-box, same as `reflection_test.go`)
- **Test Cases**:

```go
package kuniumi

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestBuildSuccessResponse(t *testing.T) {
    tests := []struct {
        name    string
        results []any
        want    map[string]any
    }{
        {
            name:    "single return value",
            results: []any{10},
            want:    map[string]any{"result": 10},
        },
        {
            name:    "multiple return values",
            results: []any{10, "hello"},
            want:    map[string]any{"result0": 10, "result1": "hello"},
        },
        {
            name:    "no return values (empty slice)",
            results: []any{},
            want:    map[string]any{},
        },
        {
            name:    "nil results",
            results: nil,
            want:    map[string]any{},
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := buildSuccessResponse(tt.results)
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestBuildErrorResponse(t *testing.T) {
    tests := []struct {
        name string
        msg  string
        want map[string]any
    }{
        {
            name: "simple error message",
            msg:  "something went wrong",
            want: map[string]any{"error": "something went wrong"},
        },
        {
            name: "empty message",
            msg:  "",
            want: map[string]any{"error": ""},
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := buildErrorResponse(tt.msg)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

#### [NEW] `response.go`(file:///c:/Users/yamya/myprog/kuniumi/response.go)

- **Description**: Shared response builder functions used by all adapters.
- **Technical Design**:

```go
package kuniumi

import (
    "encoding/json"
    "fmt"
    "net/http"
)

// buildSuccessResponse constructs the standard response map from function results.
//   - Single return: {"result": <value>}
//   - Multiple returns: {"result0": <value0>, "result1": <value1>, ...}
//   - No returns: {}
func buildSuccessResponse(results []any) map[string]any {
    response := make(map[string]any)
    if len(results) == 1 {
        response["result"] = results[0]
    } else {
        for i, res := range results {
            response[fmt.Sprintf("result%d", i)] = res
        }
    }
    return response
}

// buildErrorResponse constructs the standard error response map.
// Format: {"error": "<message>"}
func buildErrorResponse(msg string) map[string]any {
    return map[string]any{"error": msg}
}

// writeJSONError writes a JSON error response to an http.ResponseWriter.
func writeJSONError(w http.ResponseWriter, msg string, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(buildErrorResponse(msg))
}

// errorResponseSchema returns the OpenAPI schema definition for error responses.
func errorResponseSchema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "error": map[string]any{"type": "string"},
        },
        "required": []string{"error"},
    }
}
```

- **Logic**:
  - `buildSuccessResponse`: Identical logic to current HTTP/CGI adapters (lines 66–73 of `adapter_http.go`, lines 75–82 of `adapter_cgi.go`). Handles `nil` and empty slices by returning empty map `{}`.
  - `buildErrorResponse`: Returns `map[string]any{"error": msg}`.
  - `writeJSONError`: Combines `Content-Type` header, status code write, and JSON encoding. Replaces `http.Error()` calls in the HTTP adapter.
  - `errorResponseSchema`: Returns the reusable error schema for OpenAPI 400/500 response definitions.

### MCP Adapter

#### [MODIFY] `adapter_mcp.go`(file:///c:/Users/yamya/myprog/kuniumi/adapter_mcp.go)

- **Description**: Replace plain text responses with JSON-encoded responses using shared helpers.
- **Technical Design**:
  - Success path (lines 66–77): Use `buildSuccessResponse` + `json.Marshal`
  - Error path 1 — argument parse error (lines 42–47): Use `buildErrorResponse` + `json.Marshal`
  - Error path 2 — function execution error (lines 58–63): Use `buildErrorResponse` + `json.Marshal`

- **Logic — Success path** (replaces lines 66–77):
```go
response := buildSuccessResponse(results)
jsonBytes, marshalErr := json.Marshal(response)
if marshalErr != nil {
    return &mcp.CallToolResult{
        IsError: true,
        Content: []mcp.Content{
            &mcp.TextContent{Text: `{"error":"failed to marshal response"}`},
        },
    }, nil
}
return &mcp.CallToolResult{
    Content: []mcp.Content{
        &mcp.TextContent{Text: string(jsonBytes)},
    },
}, nil
```

- **Logic — Error path 1** (replaces lines 42–47, argument parse error):
```go
errJSON, _ := json.Marshal(buildErrorResponse(fmt.Sprintf("Invalid arguments format: %v", err)))
return &mcp.CallToolResult{
    IsError: true,
    Content: []mcp.Content{
        &mcp.TextContent{Text: string(errJSON)},
    },
}, nil
```

- **Logic — Error path 2** (replaces lines 58–63, function execution error):
```go
errJSON, _ := json.Marshal(buildErrorResponse(err.Error()))
return &mcp.CallToolResult{
    IsError: true,
    Content: []mcp.Content{
        &mcp.TextContent{Text: string(errJSON)},
    },
}, nil
```

### HTTP Adapter

#### [MODIFY] `adapter_http.go`(file:///c:/Users/yamya/myprog/kuniumi/adapter_http.go)

- **Description**: Replace `http.Error` calls with `writeJSONError`, refactor success response to use `buildSuccessResponse`.
- **Technical Design**:
  - Error path 1 — invalid JSON (line 47): Replace `http.Error(w, "Invalid JSON body", http.StatusBadRequest)` with `writeJSONError(w, "Invalid JSON body", http.StatusBadRequest)`
  - Error path 2 — missing metadata (line 53): Replace `http.Error(w, "Function metadata missing", http.StatusInternalServerError)` with `writeJSONError(w, "Function metadata missing", http.StatusInternalServerError)`
  - Error path 3 — function error (line 59): Replace `http.Error(w, fmt.Sprintf("Function error: %v", err), http.StatusInternalServerError)` with `writeJSONError(w, fmt.Sprintf("Function error: %v", err), http.StatusInternalServerError)`
  - Success path (lines 66–73): Replace inline response construction with `buildSuccessResponse(results)`

- **Logic — Updated handler** (replaces lines 40–76 of `createHttpHandler`):
```go
return func(w http.ResponseWriter, r *http.Request) {
    ctx := a.ContextWithEnv(r.Context())

    var args map[string]any
    if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
        writeJSONError(w, "Invalid JSON body", http.StatusBadRequest)
        return
    }

    if fn.Meta == nil {
        writeJSONError(w, "Function metadata missing", http.StatusInternalServerError)
        return
    }

    results, err := CallFunction(ctx, fn.Meta, args)
    if err != nil {
        writeJSONError(w, fmt.Sprintf("Function error: %v", err), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(buildSuccessResponse(results))
}
```

- **Import cleanup**: Remove `"fmt"` if no longer used. After the change, `fmt` is still needed for `fmt.Sprintf` in error path 3 and `fmt.Sprintf` in `buildServeCmd` (line 30).

### CGI Adapter

#### [MODIFY] `adapter_cgi.go`(file:///c:/Users/yamya/myprog/kuniumi/adapter_cgi.go)

- **Description**: Replace plain text error output with JSON error responses, refactor success response to use `buildSuccessResponse`.
- **Technical Design**:
  - Error path 1 — function not found (line 46): Change from `fmt.Printf("Status: 404 Not Found\r\n\r\nFunction not found: %s", fnName)` to JSON output with `Content-Type: application/json` header
  - Error path 2 — invalid JSON (line 58): Change from `fmt.Printf("Status: 400 Bad Request\r\n\r\nInvalid JSON: %v", err)` to JSON output
  - Error path 3 — function error (line 68): Change from `fmt.Printf("Status: 500 Internal Server Error\r\n\r\nError: %v", err)` to JSON output
  - Success path (lines 75–82): Replace inline response construction with `buildSuccessResponse(results)`

- **Logic — CGI error helper** (local to the closure or a new helper):
```go
// Within the RunE closure, define a helper or inline the pattern:
// Pattern for CGI JSON error:
fmt.Printf("Content-Type: application/json\r\nStatus: %s\r\n\r\n", statusLine)
json.NewEncoder(os.Stdout).Encode(buildErrorResponse(msg))
```

- **Logic — Error path 1** (replaces line 46):
```go
if targetFn == nil {
    fmt.Printf("Content-Type: application/json\r\nStatus: 404 Not Found\r\n\r\n")
    json.NewEncoder(os.Stdout).Encode(buildErrorResponse(
        fmt.Sprintf("Function not found: %s", fnName)))
    return nil
}
```

- **Logic — Error path 2** (replaces line 58):
```go
if err := json.NewDecoder(os.Stdin).Decode(&inputArgs); err != nil && err != io.EOF {
    fmt.Printf("Content-Type: application/json\r\nStatus: 400 Bad Request\r\n\r\n")
    json.NewEncoder(os.Stdout).Encode(buildErrorResponse(
        fmt.Sprintf("Invalid JSON: %v", err)))
    return nil
}
```

- **Logic — Error path 3** (replaces line 68):
```go
if err != nil {
    fmt.Printf("Content-Type: application/json\r\nStatus: 500 Internal Server Error\r\n\r\n")
    json.NewEncoder(os.Stdout).Encode(buildErrorResponse(
        fmt.Sprintf("Error: %v", err)))
    return nil
}
```

- **Logic — Success path** (replaces lines 75–82):
```go
fmt.Printf("Content-Type: application/json\r\nStatus: 200 OK\r\n\r\n")
json.NewEncoder(os.Stdout).Encode(buildSuccessResponse(results))
```

### OpenAPI Spec Generator

#### [MODIFY] `openapi.go`(file:///c:/Users/yamya/myprog/kuniumi/openapi.go)

- **Description**: Add 400 and 500 error response schemas to the generated OpenAPI specification.
- **Technical Design**:
  - Modify the `"responses"` section within the `for _, fn := range a.functions` loop (lines 34–49)
  - Add `"400"` and `"500"` response definitions using the `errorResponseSchema()` helper from `response.go`

- **Logic — Updated responses block** (replaces lines 34–49):
```go
"responses": map[string]any{
    "200": func() map[string]any {
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
        return responseDef
    }(),
    "400": map[string]any{
        "description": "Invalid request",
        "content": map[string]any{
            "application/json": map[string]any{
                "schema": errorResponseSchema(),
            },
        },
    },
    "500": map[string]any{
        "description": "Internal server error",
        "content": map[string]any{
            "application/json": map[string]any{
                "schema": errorResponseSchema(),
            },
        },
    },
},
```

### Integration Tests

#### [MODIFY] `tests/kuniumi/kuniumi_test.go`(file:///c:/Users/yamya/myprog/kuniumi/tests/kuniumi/kuniumi_test.go)

- **Description**: Update existing MCP test assertions and add new test cases for JSON error responses.
- **Technical Design**:
  - Modify `MCP/CallTool` test (lines 286–301): Change expected text from `"10"` to parsed JSON `{"result": 10}`
  - Add `MCP/CallToolError` test: Call with invalid args, verify JSON error response
  - Add `CGI/ErrorResponse` test: Call with non-existent function, verify JSON error
  - Add `Serve/ErrorResponse` test: POST invalid JSON, verify JSON error with status code
  - Modify `assertValidOpenAPISpec` (lines 316–406): Add assertions for 400 and 500 response schemas

- **Logic — Updated `MCP/CallTool`** (replaces lines 286–301):
```go
t.Run("CallTool", func(t *testing.T) {
    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name: "functions.Add",
        Arguments: map[string]any{
            "x": 7,
            "y": 3,
        },
    })
    require.NoError(t, err)
    require.False(t, result.IsError, "tool call should not return an error")
    require.Greater(t, len(result.Content), 0, "result should have content")

    textContent, ok := result.Content[0].(*mcp.TextContent)
    require.True(t, ok, "content should be TextContent")

    var parsed map[string]any
    err = json.Unmarshal([]byte(textContent.Text), &parsed)
    require.NoError(t, err, "MCP response text should be valid JSON")
    assert.Equal(t, float64(10), parsed["result"],
        "Add(7, 3) should return {\"result\": 10}")
})
```

- **Logic — New `MCP/CallToolError`** (insert after `CallTool` test):
```go
t.Run("CallToolError", func(t *testing.T) {
    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name: "functions.Add",
        Arguments: map[string]any{
            "x": "not_a_number",
            "y": 3,
        },
    })
    require.NoError(t, err)
    require.True(t, result.IsError, "tool call should return an error")
    require.Greater(t, len(result.Content), 0, "error result should have content")

    textContent, ok := result.Content[0].(*mcp.TextContent)
    require.True(t, ok, "content should be TextContent")

    var parsed map[string]any
    err = json.Unmarshal([]byte(textContent.Text), &parsed)
    require.NoError(t, err, "MCP error text should be valid JSON")
    assert.NotEmpty(t, parsed["error"], "error response should contain 'error' field")
})
```

- **Logic — New `CGI/ErrorResponse`** (insert after `CGI/StringArgs` test):
```go
t.Run("CGI/ErrorResponse", func(t *testing.T) {
    input := `{"x": 10, "y": 20}`
    cmd := exec.Command(binPath, "cgi")
    cmd.Env = append(os.Environ(), "PATH_INFO=/NonExistent", "REQUEST_METHOD=POST")
    cmd.Stdin = strings.NewReader(input)

    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = os.Stderr

    require.NoError(t, cmd.Run())

    output := out.String()
    assert.Contains(t, output, "Status: 404 Not Found")
    assert.Contains(t, output, "Content-Type: application/json")

    bodyIdx := strings.Index(output, "\r\n\r\n")
    require.Greater(t, bodyIdx, 0, "CGI output should contain header/body separator")
    body := output[bodyIdx+4:]

    var parsed map[string]any
    err := json.Unmarshal([]byte(body), &parsed)
    require.NoError(t, err, "CGI error body should be valid JSON")
    assert.Contains(t, parsed["error"], "NonExistent",
        "error should mention the missing function name")
})
```

- **Logic — New `Serve/ErrorResponse`** (insert after `Serve/FunctionCall` test):
```go
t.Run("ErrorResponse", func(t *testing.T) {
    resp, err := httpPost("http://localhost:9999/functions/Add",
        "application/json", strings.NewReader("not json"))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, 400, resp.StatusCode)
    assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

    var parsed map[string]any
    json.NewDecoder(resp.Body).Decode(&parsed)
    assert.Equal(t, "Invalid JSON body", parsed["error"])
})
```

- **Logic — Updated `assertValidOpenAPISpec`** (add after the existing 200 response checks, before closing `}`):
```go
// Check 400 error response
resp400, ok := responses["400"].(map[string]interface{})
require.True(t, ok, "responses should contain '400'")
assert.Equal(t, "Invalid request", resp400["description"])

resp400Content, ok := resp400["content"].(map[string]interface{})
require.True(t, ok, "400 response should have content")
resp400AppJson, ok := resp400Content["application/json"].(map[string]interface{})
require.True(t, ok, "400 content should have application/json")
resp400Schema, ok := resp400AppJson["schema"].(map[string]interface{})
require.True(t, ok, "400 application/json should have schema")
assert.Equal(t, "object", resp400Schema["type"])
resp400Props, ok := resp400Schema["properties"].(map[string]interface{})
require.True(t, ok, "400 schema should have properties")
_, ok = resp400Props["error"]
assert.True(t, ok, "400 schema properties should contain 'error'")

// Check 500 error response
resp500, ok := responses["500"].(map[string]interface{})
require.True(t, ok, "responses should contain '500'")
assert.Equal(t, "Internal server error", resp500["description"])
```

## Step-by-Step Implementation Guide

1. **Create unit tests for response helpers (TDD — fail first)**:
   - Create `response_test.go` with `TestBuildSuccessResponse` and `TestBuildErrorResponse` as defined above.
   - Run `./scripts/process/build.sh` — tests should fail (functions not yet defined).

2. **Create shared response helpers**:
   - Create `response.go` with `buildSuccessResponse`, `buildErrorResponse`, `writeJSONError`, and `errorResponseSchema` as defined above.
   - Run `./scripts/process/build.sh` — unit tests should now pass.

3. **Update MCP adapter**:
   - Edit `adapter_mcp.go`:
     - Replace the argument parse error block (lines 42–47) with JSON-encoded error using `buildErrorResponse`.
     - Replace the function execution error block (lines 58–63) with JSON-encoded error using `buildErrorResponse`.
     - Replace the success response block (lines 66–77) with `buildSuccessResponse` + `json.Marshal`.
   - Run `./scripts/process/build.sh` — verify compilation.

4. **Update HTTP adapter**:
   - Edit `adapter_http.go`:
     - Replace `http.Error(w, "Invalid JSON body", http.StatusBadRequest)` (line 47) with `writeJSONError(w, "Invalid JSON body", http.StatusBadRequest)`.
     - Replace `http.Error(w, "Function metadata missing", http.StatusInternalServerError)` (line 53) with `writeJSONError(w, "Function metadata missing", http.StatusInternalServerError)`.
     - Replace `http.Error(w, fmt.Sprintf("Function error: %v", err), http.StatusInternalServerError)` (line 59) with `writeJSONError(w, fmt.Sprintf("Function error: %v", err), http.StatusInternalServerError)`.
     - Replace inline response construction (lines 66–73) with `buildSuccessResponse(results)`.
   - Run `./scripts/process/build.sh` — verify compilation.

5. **Update CGI adapter**:
   - Edit `adapter_cgi.go`:
     - Replace the 404 error (line 46) with JSON-formatted CGI response including `Content-Type: application/json` header.
     - Replace the 400 error (line 58) with JSON-formatted CGI response.
     - Replace the 500 error (line 68) with JSON-formatted CGI response.
     - Replace inline response construction (lines 75–82) with `buildSuccessResponse(results)`.
   - Run `./scripts/process/build.sh` — verify compilation.

6. **Update OpenAPI spec generator**:
   - Edit `openapi.go`:
     - Add `"400"` and `"500"` response definitions to the `"responses"` map (lines 34–49) using `errorResponseSchema()`.
   - Run `./scripts/process/build.sh` — verify compilation.

7. **Update integration tests**:
   - Edit `tests/kuniumi/kuniumi_test.go`:
     - Modify `MCP/CallTool` (line 300): Change from `assert.Equal(t, "10", textContent.Text, ...)` to JSON parse + assert on `parsed["result"]`.
     - Add `MCP/CallToolError` test after `CallTool`.
     - Add `CGI/ErrorResponse` test after `CGI/StringArgs`.
     - Add `Serve/ErrorResponse` test inside `Serve` group after `FunctionCall`.
     - Extend `assertValidOpenAPISpec` to check for 400 and 500 response schemas.
   - Run `./scripts/process/build.sh && ./scripts/process/integration_test.sh` — all tests should pass.

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```
   - Confirms `response_test.go` passes (both `TestBuildSuccessResponse` and `TestBuildErrorResponse`).
   - Confirms all existing unit tests still pass (`reflection_test.go`, `version_test.go`).

2. **Integration Tests**:
   ```bash
   ./scripts/process/build.sh && ./scripts/process/integration_test.sh
   ```
   - **`MCP/CallTool`**: Verifies MCP success response is `{"result":10}` (Scenario 1).
   - **`MCP/CallToolError`**: Verifies MCP error response is `{"error":"..."}` (Scenario 2).
   - **`Serve/ErrorResponse`**: Verifies HTTP 400 with JSON body `{"error":"Invalid JSON body"}` (Scenario 3).
   - **`CGI/ErrorResponse`**: Verifies CGI 404 with JSON body containing `"error"` field (Scenario 4).
   - **`assertValidOpenAPISpec`**: Verifies 400 and 500 response schemas in OpenAPI spec (Scenario 5).
   - **`Serve/FunctionCall`**, **`CGI`**, **`CGI/StringArgs`**: Existing tests confirm backward compatibility (Scenario 6).

## Documentation

No documentation files need to be updated. The specification at `prompts/phases/000-firstrelease/ideas/main/010-MCP-Response-JSON-Format.md` already documents the design.
