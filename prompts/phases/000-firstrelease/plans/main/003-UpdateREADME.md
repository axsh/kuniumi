# Update README to be a Proper Introduction

> **Source Specification**: [../../ideas/main/003-UpdateREADME.md](../../ideas/main/003-UpdateREADME.md)

## Goal Description
Rewrite `README.md` to provide a comprehensive introduction to the Kuniumi framework, including its concept ("Write Once, Run Anywhere"), key features, quick start guide, and links to detailed documentation. This aims to improve the project's accessibility for new users.

## User Review Required
None.

## Requirement Traceability

> **Traceability Check**:

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. Project name and catchphrase | Proposed Changes > README.md (Header) |
| 2. Overview (Framework description, "Write Once, Run Anywhere") | Proposed Changes > README.md (Overview) |
| 3. Key Features (Multi-Interface, Virtual Environment, Type Safety) | Proposed Changes > README.md (Key Features) |
| 4. Quick Start (Install, Code, Command examples) | Proposed Changes > README.md (Quick Start) |
| 5. Documentation Links (Architecture Overview) | Proposed Changes > README.md (Documentation) |
| 6. License (Mention LICENSE file) | Proposed Changes > README.md (License) |

## Proposed Changes

### Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/kuniumi/README.md)
*   **Description**: Completely rewrite the file to serve as the project's landing page.
*   **Content Draft**:

```markdown
# Kuniumi (国生み)

**Portable Function Framework for Go**

Kuniumi is a framework designed to expose functions written in Go as portable web services through various interfaces (HTTP, CGI, MCP, Docker containers).

> **Concept: Write Once, Run Anywhere**
> developers focus on implementing business logic (functions), while Kuniumi handles adaptation to execution environments (Adapter) and abstraction of file systems/environment variables (Virtual Environment).

## Key Features

- **Multi-Interface**: Generate HTTP servers, MCP servers, CGI scripts, and Docker containers from a single Go codebase.
- **Virtual Environment**: Secure and portable access to file systems and environment variables via a sandboxed environment.
- **Type Safety**: Automatically extracts function metadata using Go's reflection to ensure type-safe interfaces.

## Quick Start

### Installation

```bash
go get github.com/axsh/kuniumi
```

### Basic Implementation (`main.go`)

```go
package main

import (
    "context"
    "github.com/axsh/kuniumi"
)

// Function to expose
// context.Context is required to access the VirtualEnvironment
func Add(ctx context.Context, x int, y int) (int, error) {
    return x + y, nil
}

func main() {
    app := kuniumi.New(kuniumi.Config{
        Name:    "Calculator",
        Version: "1.0.0",
    })

    // Register function with argument names
    app.RegisterFunc(Add, "Add two integers", kuniumi.WithArgs("x", "y"))

    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

### Build and Run

```bash
# Build the application
go build -o calculator main.go

# Run as HTTP Server
./calculator serve --port 8080

# Run as MCP Server (Stdio)
./calculator mcp

# Run as CGI
export PATH_INFO="/Add"
echo '{"x": 10, "y": 20}' | ./calculator cgi
```

## Documentation

For more detailed technical information, please refer to the **[Architecture Overview](prompts/specifications/kuniumu-architechture.md)**.

## License

See [LICENSE](LICENSE) file.
```

## Step-by-Step Implementation Guide

1.  **Update README.md**:
    *   Edit `README.md` and replace its entire content with the draft provided in the "Proposed Changes" section.

## Verification Plan

### Automated Verification

1.  **Build Check**:
    Run the build script to ensure the documentation change does not negatively impact the build process (e.g. valid characters).
    ```bash
    ./scripts/process/build.sh
    ```

### Manual Verification

1.  **Preview Check**:
    *   Open `README.md` in VS Code.
    *   Use `Markdown: Open Preview` (Ctrl+Shift+V).
    *   Verify that the layout, code blocks, and badges (if any) are rendered correctly.
    *   Click on the **[Architecture Overview](prompts/specifications/kuniumu-architechture.md)** link and ensure it behaves as expected (opens the correct file).
