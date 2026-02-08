package kuniumi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) buildCgiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cgi",
		Short: "Execute a registered function directly (CGI mode)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Determine which function to call.
			// PATH_INFO usually contains the path. e.g. /Add
			pathInfo := os.Getenv("PATH_INFO")
			pathInfo = strings.TrimPrefix(pathInfo, "/")

			// OpenAPI spec request
			if pathInfo == "openapi.json" {
				fmt.Printf("Content-Type: application/json\r\nStatus: 200 OK\r\n\r\n")
				spec := a.generateOpenAPISpec()
				json.NewEncoder(os.Stdout).Encode(spec)
				return nil
			}

			// Normalize: if pathInfo is "functions/Add", handle it.
			// Or just "Add".
			fnName := pathInfo
			fnName = strings.TrimPrefix(fnName, "functions/")

			var targetFn *RegisteredFunc
			for _, fn := range a.functions {
				if fn.Name == fnName {
					targetFn = fn
					break
				}
			}

			if targetFn == nil {
				fmt.Printf("Status: 404 Not Found\r\n\r\nFunction not found: %s", fnName)
				return nil
			}

			// 2. Read Body
			// If POST, read Stdin.
			// We expect JSON body.
			var inputArgs map[string]interface{}

			// Basic check for content length if needed, but JSON decoder is enough
			if err := json.NewDecoder(os.Stdin).Decode(&inputArgs); err != nil && err != io.EOF {
				// Allow empty body if EOF
				fmt.Printf("Status: 400 Bad Request\r\n\r\nInvalid JSON: %v", err)
				return nil
			}

			// 3. Setup Context
			ctx := a.ContextWithEnv(context.Background())

			// 4. Call Function
			results, err := CallFunction(ctx, targetFn.Meta, inputArgs)
			if err != nil {
				fmt.Printf("Status: 500 Internal Server Error\r\n\r\nError: %v", err)
				return nil
			}

			// 5. Response
			fmt.Printf("Content-Type: application/json\r\nStatus: 200 OK\r\n\r\n")

			response := make(map[string]interface{})
			if len(results) == 1 {
				response["result"] = results[0]
			} else {
				for i, res := range results {
					response[fmt.Sprintf("result%d", i)] = res
				}
			}

			json.NewEncoder(os.Stdout).Encode(response)
			return nil
		},
	}
	return cmd
}
