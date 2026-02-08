package kuniumi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func (a *App) buildServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Web API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")

			mux := http.NewServeMux()

			// Register Functions
			for _, fn := range a.functions {
				path := fmt.Sprintf("/functions/%s", fn.Name)
				mux.HandleFunc("POST "+path, a.createHttpHandler(fn))
				// Also create GET for metadata?
			}

			// Open API Endpoint
			mux.HandleFunc("GET /openapi.json", a.serveOpenAPI)

			addr := fmt.Sprintf(":%d", port)
			fmt.Printf("Serving on %s\n", addr)
			return http.ListenAndServe(addr, mux)
		},
	}
	cmd.Flags().Int("port", 8080, "Port to listen on")
	return cmd
}

func (a *App) createHttpHandler(fn *RegisteredFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create Context with Virtual Environment
		ctx := a.ContextWithEnv(r.Context())

		// Decode Body
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// Call Function
		if fn.Meta == nil {
			http.Error(w, "Function metadata missing", http.StatusInternalServerError)
			return
		}

		results, err := CallFunction(ctx, fn.Meta, args)
		if err != nil {
			http.Error(w, fmt.Sprintf("Function error: %v", err), http.StatusInternalServerError)
			return
		}

		// Encode Response
		w.Header().Set("Content-Type", "application/json")

		response := make(map[string]any)
		if len(results) == 1 {
			response["result"] = results[0]
		} else {
			for i, res := range results {
				response[fmt.Sprintf("result%d", i)] = res
			}
		}

		json.NewEncoder(w).Encode(response)
	}
}

func (a *App) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec := a.generateOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}
