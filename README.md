# Kuniumi (国産み)

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
