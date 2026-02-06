package main

import (
	"context"
	"fmt"
	"os"

	"github.com/axsh/kuniumi"
)

// Add is a simple function to be exposed.
func Add(ctx context.Context, x int, y int) (int, error) {
	// Example of using VirtualEnvironment
	env := kuniumi.GetVirtualEnv(ctx)

	// Check for a debug flag or similar from Env
	if env.Getenv("DEBUG") == "true" {
		fmt.Println("Debug mode is on")
		// Try writing to a file if mount is present
		if err := env.WriteFile("debug.log", []byte(fmt.Sprintf("Adding %d + %d", x, y))); err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: WriteFile failed: %v\n", err)
		}
	}

	return x + y, nil
}

func main() {
	app := kuniumi.New(kuniumi.Config{
		Name:    "Calculator",
		Version: "1.0.0",
	})

	app.RegisterFunc(Add, "Adds two integers together",
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
