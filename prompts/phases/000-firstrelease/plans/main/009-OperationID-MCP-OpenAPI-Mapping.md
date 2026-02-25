# 009-OperationID-MCP-OpenAPI-Mapping

> **Source Specification**: `prompts/phases/000-firstrelease/ideas/main/009-OperationID-MCP-OpenAPI-Mapping.md`

## Goal Description

OpenAPI の `operationId` フィールドを MCP ツール名と完全一致させることで、両プロトコル間の明示的なマッピングを確立する。`RegisteredFunc` に `OperationID()` メソッドを追加し、OpenAPI 生成と MCP アダプターの両方がこの単一のソースから識別子を取得する構造に変更する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Add `operationId` to OpenAPI spec | Proposed Changes > openapi.go |
| Single source of truth (`OperationID()`) | Proposed Changes > app.go |
| MCP tool name uses `operationId` value | Proposed Changes > adapter_mcp.go |
| OpenAPI path `/functions/{Name}` unchanged | No changes to path construction in openapi.go / adapter_http.go |
| CGI routing `PATH_INFO=/Add` unchanged | No changes to adapter_cgi.go |
| MCP tool name format `functions.Add` preserved | OperationID() returns `"functions." + rf.Name` |
| Integration test verifies `operationId` matches MCP tool name | Proposed Changes > kuniumi_test.go |
| Remove `openAPIPathToMCPName` helper | Proposed Changes > adapter_mcp.go |

## Proposed Changes

### Core: RegisteredFunc OperationID method

#### [MODIFY] [app.go](file:///c:/Users/yamya/myprog/kuniumi/app.go)
- **Description**: Add `OperationID()` method to `RegisteredFunc`. This is the single source of truth for both OpenAPI and MCP.
- **Technical Design**:
  ```go
  // OperationID returns the canonical identifier for this function,
  // used as both the OpenAPI operationId and MCP tool name.
  // Format: "functions.{Name}" (e.g., "functions.Add").
  func (rf *RegisteredFunc) OperationID() string {
      return "functions." + rf.Name
  }
  ```
- **Logic**:
  - Concatenates the literal string `"functions."` with `rf.Name`.
  - The dot (`.`) separator is used instead of slash (`/`) because MCP tool names only allow `[a-zA-Z0-9_-.]` characters.
  - The `"functions."` prefix mirrors the OpenAPI path prefix `/functions/`, maintaining the semantic relationship.

### OpenAPI Generator

#### [MODIFY] [openapi.go](file:///c:/Users/yamya/myprog/kuniumi/openapi.go)
- **Description**: Add `operationId` field to each POST operation in the generated OpenAPI spec.
- **Technical Design**:
  - Current `"post"` map in `generateOpenAPISpec` (lines 24-49):
    ```go
    paths[path] = map[string]any{
        "post": map[string]any{
            "description": fn.Description,
            "requestBody": map[string]any{ ... },
            "responses":   map[string]any{ ... },
        },
    }
    ```
  - Changed `"post"` map:
    ```go
    paths[path] = map[string]any{
        "post": map[string]any{
            "operationId": fn.OperationID(),
            "description": fn.Description,
            "requestBody": map[string]any{ ... },
            "responses":   map[string]any{ ... },
        },
    }
    ```
- **Logic**:
  - Insert `"operationId": fn.OperationID()` as the first key in the `"post"` map.
  - For the `Add` function: `fn.OperationID()` returns `"functions.Add"`, so the OpenAPI spec will contain `"operationId": "functions.Add"`.
  - All other fields (`description`, `requestBody`, `responses`) remain unchanged.

### MCP Adapter

#### [MODIFY] [adapter_mcp.go](file:///c:/Users/yamya/myprog/kuniumi/adapter_mcp.go)
- **Description**: Replace `openAPIPathToMCPName` conversion with direct `fn.OperationID()` call. Remove the now-unnecessary helper function.
- **Technical Design**:
  - Current code (lines 25-31):
    ```go
    toolName := openAPIPathToMCPName(fmt.Sprintf("/functions/%s", fn.Name))
    tool := mcp.Tool{
        Name:        toolName,
        Description: fn.Description,
        InputSchema: GenerateJSONSchema(fn.Meta),
    }
    ```
  - Changed code:
    ```go
    tool := mcp.Tool{
        Name:        fn.OperationID(),
        Description: fn.Description,
        InputSchema: GenerateJSONSchema(fn.Meta),
    }
    ```
  - Remove the `openAPIPathToMCPName` function (lines 92-98):
    ```go
    // DELETE THIS ENTIRE FUNCTION:
    func openAPIPathToMCPName(path string) string {
        name := strings.TrimPrefix(path, "/")
        return strings.ReplaceAll(name, "/", ".")
    }
    ```
  - Remove unused imports: `"fmt"` and `"strings"` (if no longer used after deletion).
- **Logic**:
  - `fn.OperationID()` returns `"functions.Add"` — the same value that `openAPIPathToMCPName("/functions/Add")` returned.
  - The `openAPIPathToMCPName` function is no longer needed because the identifier is now computed centrally by `OperationID()`.
  - After removing `openAPIPathToMCPName`, the `"strings"` import is no longer used and must be removed. The `"fmt"` import is still used by `fmt.Sprintf` in the tool handler (line 45, 61, etc.).

### Integration Tests

#### [MODIFY] [tests/kuniumi/kuniumi_test.go](file:///c:/Users/yamya/myprog/kuniumi/tests/kuniumi/kuniumi_test.go)
- **Description**: Add `operationId` verification to `assertValidOpenAPISpec`. This ensures that the OpenAPI spec's `operationId` matches the MCP tool name.
- **Technical Design**:
  - Insert after the existing `post.description` assertion (after line 343):
    ```go
    // operationId must match MCP tool name
    assert.Equal(t, "functions.Add", post["operationId"],
        "operationId should match MCP tool name")
    ```
- **Logic**:
  - `assertValidOpenAPISpec` is called from two test paths: `CGI/OpenAPI` and `Serve/OpenAPI`.
  - Both paths will now verify that `operationId` is present and equals `"functions.Add"`.
  - The MCP tests (`MCP/ListTools`) already verify that the MCP tool name is `"functions.Add"`.
  - Combined, these tests prove that `operationId == MCP tool name`.

## Step-by-Step Implementation Guide

1. **Add `OperationID()` method to `RegisteredFunc`**:
   - Edit `app.go`.
   - Add the following method after the `RegisteredFunc` struct definition (after line 36):
     ```go
     func (rf *RegisteredFunc) OperationID() string {
         return "functions." + rf.Name
     }
     ```

2. **Add `operationId` assertion to integration test (TDD — test first)**:
   - Edit `tests/kuniumi/kuniumi_test.go`.
   - In `assertValidOpenAPISpec`, add after line 343 (after the `post.description` assertion):
     ```go
     assert.Equal(t, "functions.Add", post["operationId"],
         "operationId should match MCP tool name")
     ```
   - Run `./scripts/process/build.sh` — unit tests should pass (no test file changes to unit tests).
   - Run `./scripts/process/integration_test.sh` — the new assertion should **FAIL** because `openapi.go` does not yet emit `operationId`.

3. **Add `operationId` to OpenAPI spec**:
   - Edit `openapi.go`.
   - In the `"post"` map (line 24), add `"operationId": fn.OperationID()` as the first entry:
     ```go
     "post": map[string]any{
         "operationId": fn.OperationID(),
         "description": fn.Description,
         ...
     }
     ```
   - Run `./scripts/process/build.sh && ./scripts/process/integration_test.sh` — the `operationId` assertion should now **PASS**.

4. **Replace `openAPIPathToMCPName` with `OperationID()` in MCP adapter**:
   - Edit `adapter_mcp.go`.
   - Replace line 26 (`toolName := openAPIPathToMCPName(...)`) and line 28 (`Name: toolName`) with:
     ```go
     tool := mcp.Tool{
         Name:        fn.OperationID(),
         Description: fn.Description,
         InputSchema: GenerateJSONSchema(fn.Meta),
     }
     ```
   - Delete the `openAPIPathToMCPName` function (lines 92-98).
   - Remove the `"strings"` import from the import block (line 7).
   - Run `./scripts/process/build.sh && ./scripts/process/integration_test.sh` — all tests should **PASS** including `MCP/ListTools` which verifies the name is still `"functions.Add"`.

5. **Final verification**:
   - Run `./scripts/process/build.sh && ./scripts/process/integration_test.sh`.
   - Confirm all 11 test cases pass: Help, CGI, CGI/StringArgs, CGI/OpenAPI, Serve, Serve/FunctionCall, Serve/OpenAPI, VirtualEnv, MCP, MCP/ListTools, MCP/ToolSchema, MCP/CallTool.

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```
   - Verifies: compilation succeeds, all unit tests pass.

2. **Integration Tests**:
   ```bash
   ./scripts/process/build.sh && ./scripts/process/integration_test.sh
   ```
   - **Key test cases**:
     - `CGI/OpenAPI`: `assertValidOpenAPISpec` verifies `operationId` == `"functions.Add"` in the OpenAPI spec served via CGI.
     - `Serve/OpenAPI`: Same verification via HTTP server.
     - `MCP/ListTools`: Verifies MCP tool name == `"functions.Add"`.
     - `MCP/CallTool`: Verifies the tool is callable via `"functions.Add"`.
   - **Log Verification**: Confirm no `AddTool: invalid tool name` warnings in stderr (MCP SDK validation should pass since `functions.Add` only contains valid characters).

## Documentation

No documentation files require updates. The `operationId` field is a standard OpenAPI feature and is self-documenting within the generated spec.
