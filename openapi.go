package kuniumi

import "fmt"

// generateOpenAPISpec generates a simplified OpenAPI 3.0.0 specification
// based on the registered functions.
func (a *App) generateOpenAPISpec() map[string]any {
	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]string{
			"title":   a.config.Name,
			"version": a.config.Version,
		},
		"paths": map[string]any{},
	}

	paths := spec["paths"].(map[string]any)

	for _, fn := range a.functions {
		path := fmt.Sprintf("/functions/%s", fn.Name)
		schema := GenerateJSONSchema(fn.Meta)

		paths[path] = map[string]any{
			"post": map[string]any{
				"operationId": fn.OperationID(),
				"description": fn.Description,
				"requestBody": map[string]any{
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": schema,
						},
					},
				},
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
				},
			},
		}
	}

	return spec
}
