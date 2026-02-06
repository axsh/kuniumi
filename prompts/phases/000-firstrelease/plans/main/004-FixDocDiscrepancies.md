# Fix Documentation Discrepancies

> **Source Specification**: [../../ideas/main/004-FixDocDiscrepancies.md](../../ideas/main/004-FixDocDiscrepancies.md)

## Goal Description
Align `README.md` and `prompts/specifications/kuniumu-architechture.md` with the current codebase (`options.go`, `virtual_env.go`, `app.go`, etc.). Specifically, unify the API usage to `WithParams` (deprecating `WithArgs`), reference `WithReturns`, and document the complete `VirtualEnvironment` API and Adapter details.

## User Review Required
None.

## Requirement Traceability

> **Traceability Check**:

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| API Consistency (`WithParams`, delete `WithArgs`) | Proposed Changes > README.md, kuniumu-architechture.md |
| Complete Code Examples | Proposed Changes > README.md |
| `VirtualEnvironment` Full API Coverage | Proposed Changes > kuniumu-architechture.md |
| Adapter Details (HTTP, Container) | Proposed Changes > kuniumu-architechture.md |

## Proposed Changes

### Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/kuniumi/README.md)
*   **Description**: Update Quick Start code to use `WithParams` and `WithReturns`.
*   **Technical Design**:
    *   Replace `WithArgs("x", "y")` with `WithParams(Param("x", "..."), Param("y", "..."))`.
    *   Add `WithReturns`.
    *   Add `kuniumi.GetVirtualEnv(ctx)` usage example.

**Content Update (Quick Start Section):**

```markdown
### Basic Implementation (`main.go`)

```go
package main

import (
    "context"
    "fmt"
    "github.com/axsh/kuniumi"
)

// Function to expose
// context.Context is required to access the VirtualEnvironment
func Add(ctx context.Context, x int, y int) (int, error) {
    // Example: Accessing Virtual Environment
    env := kuniumi.GetVirtualEnv(ctx)
    if env.Getenv("DEBUG") == "true" {
        fmt.Println("Debug mode enabled")
    }

    return x + y, nil
}

func main() {
    app := kuniumi.New(kuniumi.Config{
        Name:    "Calculator",
        Version: "1.0.0",
    })

    // Register function with parameters and return value description
    app.RegisterFunc(Add, "Add two integers",
        kuniumi.WithParams(
            kuniumi.Param("x", "First integer to add"),
            kuniumi.Param("y", "Second integer to add"),
        ),
        kuniumi.WithReturns("Sum of x and y"),
    )

    if err := app.Run(); err != nil {
        panic(err)
    }
}
```
```

#### [MODIFY] [doc.go](file:///c:/Users/yamya/myprog/kuniumi/doc.go)
*   **Description**: Update Usage example to match `README.md` style (include `WithParams`).
*   **Changes**:
    *   Update the `Usage` section code block to use `WithParams`.

#### [MODIFY] [prompts/specifications/kuniumu-architechture.md](file:///c:/Users/yamya/myprog/kuniumi/prompts/specifications/kuniumu-architechture.md)
*   **Description**: Update Quick Start, API Reference, and Adapter details.
*   **Changes**:
    1.  **Quick Start**: Sync with `README.md` changes (`WithParams`).
    2.  **API Reference (`RegisterFunc`)**:
        *   Replace `WithArgs` with `WithParams`.
        *   Add `WithReturns`.
    3.  **API Reference (`VirtualEnvironment`)**:
        *   Add `FindFile` and `ListFile` details.
        *   Add `FileInfo` struct definition.
    4.  **Adapters**:
        *   Add HTTP endpoint details (`POST /functions/{name}`, `GET /openapi.json`).
        *   Add Container Dockerfile details (Base image: `golang:1.24-alpine`).

**VirtualEnvironment Section Additions:**

```markdown
#### `type FileInfo`

```go
type FileInfo struct {
    Name  string
    Size  int64
    IsDir bool
}
```

- **`func (v *VirtualEnvironment) ListFile(path string) ([]FileInfo, error)`**
    - 指定されたディレクトリ内のファイル一覧を取得します。

- **`func (v *VirtualEnvironment) FindFile(root string, pattern string, recursive bool) ([]string, error)`**
    - 指定されたパターン（Glob, 例: `*.go`）に一致するファイルを検索します。
    - `recursive` が `true` の場合、サブディレクトリも再帰的に検索します。
    - 戻り値は仮想パスのリストです。
```

**Adapters Section Additions:**

```markdown
### 4.1 HTTP アダプター (`serve`)

REST APIとして関数を公開します。

- **Endpoint**: `POST /functions/{function_name}`
    - Body: JSON Object (Arguments)
    - Response: JSON Object (`{"result": ...}`)
- **Metadata**: `GET /openapi.json`
    - OpenAPI 3.0.0 形式で API 定義を返します。

### 4.4 Container アダプター (`containerize`)

アプリケーションを Docker コンテナ化するための `Dockerfile` を生成、ビルド、プッシュします。

- **Base Image**: `golang:1.24-alpine` (Builder), `alpine:latest` (Runtime)
- **Commands**:
    - `docker build`
    - `docker push` (Optional)
```

## Step-by-Step Implementation Guide

1.  **Update README.md**:
    *   Replace the code block in "Basic Implementation" with the updated version using `WithParams`.
2.  **Update kuniumu-architechture.md**:
    *   Update "Quick Start" code block.
    *   Update "3.2 関数登録" section to usage `WithParams` and remove/deprecate `WithArgs`.
    *   Update "5. API リファレンス":
        *   Replace `WithArgs` entry with `WithParams` and `WithReturns`.
        *   Add `FileInfo` definition.
        *   Ensure all `VirtualEnvironment` methods (`ListFile`, `FindFile`) are described.
    *   Update "4. アダプター":
        *   Add specific details for HTTP and Container adapters.
3.  **Update doc.go**:
    *   Update the package comment to reflect correct API usage.
4.  **Verify Documentation Code**:
    *   Create a temporary file `tmp/doc_check.go` with the content of the updated "Basic Implementation".
    *   Run `scripts/process/build.sh` (or just `go build -o tmp/check.exe tmp/doc_check.go`) to verify it compiles.

## Verification Plan

### Automated Verification

1.  **Documentation Code Compilation**:
    Extract the code from `README.md` (conceptually) and compile it to ensure correctness.
    ```bash
    # Create temp file with the exact code from README
    cat > tmp/doc_check.go <<EOF
    package main
    import (
        "context"
        "fmt"
        "github.com/axsh/kuniumi"
    )
    func Add(ctx context.Context, x int, y int) (int, error) {
        env := kuniumi.GetVirtualEnv(ctx)
        if env.Getenv("DEBUG") == "true" {
            fmt.Println("Debug mode enabled")
        }
        return x + y, nil
    }
    func main() {
        app := kuniumi.New(kuniumi.Config{Name: "Calculator", Version: "1.0.0"})
        app.RegisterFunc(Add, "Desc",
            kuniumi.WithParams(kuniumi.Param("x", "Desc"), kuniumi.Param("y", "Desc")),
            kuniumi.WithReturns("Desc"),
        )
        // Skip Run() for check or keep it
    }
    EOF
    
    # Run build (using go build directly here for temp file is unavoidable as build.sh builds the project root, 
    # but strictly speaking I should use a script if available. 
    # However, for validating a temp file, direct command is often cleaner for verification steps in plans 
    # unless I create a specific test script. I will use a simple go build command here as it's a transient verification.)
    go build -o tmp/doc_check.exe tmp/doc_check.go
    ```

2.  **Project Build**:
    Ensure the main project still builds.
    ```bash
    ./scripts/process/build.sh
    ```
