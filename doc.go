/*
Package kuniumi provides a toolkit for building portable functions that can run as MCP servers, Web APIs, CLIs, and CGI scripts.
The core philosophy of Kuniumi is "Write once, run anywhere". By defining a Go function once, you can expose it through multiple interfaces without changing the core logic.

# Key Features

  - **Multi-Protocol Support**: Automatically exposes registered functions as Model Context Protocol (MCP) tools, HTTP endpoints, CLI subcommands, and CGI scripts.
  - **Virtual Environment**: Provides a sandboxed environment (`VirtualEnvironment`) for file system access and environment variable management, ensuring consistent behavior across different deployment modes.
  - **Type-Safe Registration**: Functions can be registered with standard Go types, and Kuniumi handles argument parsing and response formatting.
  - **OpenAPI Generation**: Automatically generates OpenAPI definitions for exposed HTTP endpoints.

# Usage

To use Kuniumi, create an instance of `App`, register your functions, and then call `Run()`.

	package main

	import (
		"context"
		"github.com/yourusername/kuniumi"
	)

	func Hello(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}

	func main() {
		app := kuniumi.New(kuniumi.Config{
			Name:    "my-app",
			Version: "1.0.0",
		})
		app.RegisterFunc(Hello, "Returns a greeting")
		app.Run()
	}

Running the application:

	$ my-app serve --port 8080   # Start Web Server
	$ my-app mcp                 # Run as MCP Server
	$ my-app cgi                 # Run as CGI Script
*/
package kuniumi
