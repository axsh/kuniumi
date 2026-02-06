package kuniumi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func (a *App) buildMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run as a Model Context Protocol (MCP) server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create MCP Server
			s := mcp.NewServer(&mcp.Implementation{
				Name:    a.config.Name,
				Version: a.config.Version,
			}, nil)

			// Register Tools
			for _, fn := range a.functions {
				tool := mcp.Tool{
					Name:        fn.Name,
					Description: fn.Description,
					InputSchema: GenerateJSONSchema(fn.Meta),
				}

				// Capture closure variables
				targetFn := fn

				s.AddTool(&tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					// req.Params.Arguments is json.RawMessage
					params := req.Params
					var toolArgs map[string]any

					// Handle nil or empty arguments
					if len(params.Arguments) > 0 {
						if err := json.Unmarshal(params.Arguments, &toolArgs); err != nil {
							return &mcp.CallToolResult{
								IsError: true,
								Content: []mcp.Content{
									&mcp.TextContent{Text: fmt.Sprintf("Invalid arguments format: %v", err)},
								},
							}, nil
						}
					} else {
						toolArgs = make(map[string]interface{})
					}

					// Create context with env
					appCtx := a.ContextWithEnv(ctx)

					results, err := CallFunction(appCtx, targetFn.Meta, toolArgs)
					if err != nil {
						return &mcp.CallToolResult{
							IsError: true,
							Content: []mcp.Content{
								&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)},
							},
						}, nil
					}

					var content string
					if len(results) == 1 {
						content = fmt.Sprintf("%v", results[0])
					} else {
						content = fmt.Sprintf("%v", results)
					}

					return &mcp.CallToolResult{
						Content: []mcp.Content{
							&mcp.TextContent{Text: content},
						},
					}, nil
				})
			}

			// Serve StdIO
			// Using StdioTransport
			transport := &mcp.StdioTransport{}
			return s.Run(cmd.Context(), transport)
		},
	}
	return cmd
}
