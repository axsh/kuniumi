# 009: Explicit MCP-OpenAPI Mapping via operationId

## Background

Kuniumi exposes registered Go functions through multiple adapters: HTTP (OpenAPI), CGI, and MCP (Model Context Protocol). Each adapter represents the same function differently:

| Adapter | Identifier | Example |
|---------|-----------|---------|
| HTTP / OpenAPI | Path | `/functions/Add` |
| MCP | Tool name | `functions.Add` |

A key limitation of MCP is that it **cannot describe return types** in its tool schema — the `inputSchema` field covers request parameters, but there is no equivalent for responses. OpenAPI, on the other hand, fully supports response schemas (`responses.200.content.application/json.schema`). This makes OpenAPI a natural complement to MCP for tools that need to communicate return type information.

However, there is currently **no explicit link** between a given OpenAPI endpoint and its corresponding MCP tool. The relationship is only implicit:

- The OpenAPI path `/functions/Add` and the MCP tool name `functions.Add` are constructed independently in separate adapters (`openapi.go` and `adapter_mcp.go`).
- A consumer would have to guess the conversion rule (strip leading `/`, replace `/` with `.`) to find the mapping.
- There is no single field in the OpenAPI spec that states "this operation corresponds to MCP tool X".

OpenAPI provides a standard field for exactly this purpose: **`operationId`**. It is a unique string identifier for each operation, independent of the path, and is commonly used by code generators to name functions. By setting `operationId` to match the MCP tool name exactly, we create an explicit, spec-compliant bridge between the two protocols.

## Requirements

### Mandatory

1. **Add `operationId` to OpenAPI spec**
   - Each operation in the generated OpenAPI spec must include an `operationId` field.
   - The `operationId` value must exactly match the corresponding MCP tool name (e.g., `functions.Add`).

2. **Single source of truth for the tool identifier**
   - Both the MCP adapter and OpenAPI generator must derive the tool identifier from the same logic, ensuring they always stay in sync.
   - Introduce a method or field on `RegisteredFunc` (e.g., `OperationID() string`) that both adapters reference.

3. **MCP tool name uses `operationId` value**
   - The MCP adapter must use the same `operationId` value as the tool name (not construct it independently).

4. **Backward compatibility**
   - The OpenAPI path must remain `/functions/{Name}` (no change to HTTP routing).
   - Existing CGI routing (`PATH_INFO=/Add`) must remain unchanged.
   - The MCP tool name format `functions.Add` (already in place) is preserved.

5. **Test coverage**
   - Integration tests must verify that the OpenAPI `operationId` field exists and matches the MCP tool name.
   - Existing OpenAPI and MCP tests must continue to pass.

### Optional

- Add a `WithOperationID(id string)` function option to allow users to override the auto-generated `operationId`.

## Implementation Approach

### 1. Add `OperationID` to `RegisteredFunc` (`app.go`)

Add a computed method that derives the canonical operation identifier:

```go
// OperationID returns the canonical identifier for this function,
// used as both the OpenAPI operationId and MCP tool name.
// Format: "functions.{Name}" (e.g., "functions.Add").
func (rf *RegisteredFunc) OperationID() string {
    return "functions." + rf.Name
}
```

This replaces the ad-hoc `openAPIPathToMCPName` conversion function in `adapter_mcp.go`.

### 2. Update OpenAPI generator (`openapi.go`)

Add `operationId` to each operation object:

```go
paths[path] = map[string]any{
    "post": map[string]any{
        "operationId": fn.OperationID(),
        "description": fn.Description,
        // ... requestBody, responses unchanged
    },
}
```

### 3. Update MCP adapter (`adapter_mcp.go`)

Replace the `openAPIPathToMCPName` call with `fn.OperationID()`:

```go
tool := mcp.Tool{
    Name:        fn.OperationID(),
    Description: fn.Description,
    InputSchema: GenerateJSONSchema(fn.Meta),
}
```

Remove the now-unnecessary `openAPIPathToMCPName` helper function.

### 4. Update tests (`tests/kuniumi/kuniumi_test.go`)

Add `operationId` verification to `assertValidOpenAPISpec`:

```go
// Verify operationId matches MCP tool name
assert.Equal(t, "functions.Add", post["operationId"],
    "operationId should match MCP tool name")
```

### Architecture Diagram

```
RegisteredFunc
    ├── Name: "Add"
    └── OperationID(): "functions.Add"   ← single source of truth
            │
            ├──→ openapi.go:  operationId: "functions.Add"
            │                  path: "/functions/Add"
            │
            └──→ adapter_mcp.go:  tool.Name: "functions.Add"
```

## Verification Scenarios

### Scenario 1: OpenAPI spec includes operationId

1. Build the `kuniumi_example` binary.
2. Request `GET /openapi.json` via the Serve adapter.
3. Parse the response JSON.
4. Navigate to `paths["/functions/Add"]["post"]`.
5. Verify `operationId` field exists with value `"functions.Add"`.
6. Verify `description` field remains `"Adds two integers together"` (unchanged).

### Scenario 2: operationId matches MCP tool name exactly

1. Start the binary in MCP mode (`mcp` subcommand).
2. Connect an MCP client and call `tools/list`.
3. Retrieve the tool name for the Add function.
4. Request OpenAPI spec via `GET /openapi.json`.
5. Retrieve the `operationId` for the `/functions/Add` path.
6. Assert that the MCP tool name and OpenAPI `operationId` are identical (`"functions.Add"`).

### Scenario 3: CGI mode also includes operationId

1. Execute in CGI mode with `PATH_INFO=/openapi.json`, `REQUEST_METHOD=GET`.
2. Parse the CGI response body as JSON.
3. Verify `operationId` is present at `paths["/functions/Add"]["post"]["operationId"]`.
4. Verify the value is `"functions.Add"`.

### Scenario 4: Existing functionality is unchanged

1. Run `scripts/process/build.sh` — all unit tests pass.
2. Run `scripts/process/integration_test.sh` — all integration tests pass (Help, CGI, CGI/StringArgs, CGI/OpenAPI, Serve/FunctionCall, Serve/OpenAPI, VirtualEnv, MCP/ListTools, MCP/ToolSchema, MCP/CallTool).

## Testing for the Requirements

| Requirement | Test case | Verification method |
|---|---|---|
| `operationId` present in OpenAPI spec | `assertValidOpenAPISpec` checks `post["operationId"]` | `scripts/process/integration_test.sh` |
| `operationId` matches MCP tool name | `MCP/ListTools` verifies name is `functions.Add`; `assertValidOpenAPISpec` verifies `operationId` is `functions.Add` | `scripts/process/integration_test.sh` |
| Single source of truth | Both adapters call `fn.OperationID()` (code review) | `scripts/process/build.sh` (compilation) |
| HTTP path unchanged | `Serve/FunctionCall` POSTs to `/functions/Add` | `scripts/process/integration_test.sh` |
| CGI routing unchanged | `CGI` test uses `PATH_INFO=/Add` | `scripts/process/integration_test.sh` |
| All existing tests pass | Full test suite | `scripts/process/build.sh` + `scripts/process/integration_test.sh` |

### Verification Commands

```bash
# Build & unit tests
./scripts/process/build.sh

# Integration tests
./scripts/process/integration_test.sh
```
