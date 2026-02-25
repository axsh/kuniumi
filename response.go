package kuniumi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// buildSuccessResponse constructs the standard response map from function results.
//   - Single return: {"result": <value>}
//   - Multiple returns: {"result0": <value0>, "result1": <value1>, ...}
//   - No returns: {}
func buildSuccessResponse(results []any) map[string]any {
	response := make(map[string]any)
	if len(results) == 1 {
		response["result"] = results[0]
	} else {
		for i, res := range results {
			response[fmt.Sprintf("result%d", i)] = res
		}
	}
	return response
}

// buildErrorResponse constructs the standard error response map.
// Format: {"error": "<message>"}
func buildErrorResponse(msg string) map[string]any {
	return map[string]any{"error": msg}
}

// writeJSONError writes a JSON error response to an http.ResponseWriter.
func writeJSONError(w http.ResponseWriter, msg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(buildErrorResponse(msg))
}

// errorResponseSchema returns the OpenAPI schema definition for error responses.
func errorResponseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"error": map[string]any{"type": "string"},
		},
		"required": []string{"error"},
	}
}
