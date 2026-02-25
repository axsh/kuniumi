//go:build integration

package kuniumi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKuniumiIntegration verifies the Kuniumi framework by building and running the example app.
func TestKuniumiIntegration(t *testing.T) {
	// 1. Build the example binary
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Find project root
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	// Prepare temp build directory
	buildDir, err := os.MkdirTemp("", "kuniumi_build_test")
	require.NoError(t, err)
	defer os.RemoveAll(buildDir)

	// Initialize new module
	setupCmd := exec.Command("go", "mod", "init", "example.com/test-build")
	setupCmd.Dir = buildDir
	require.NoError(t, setupCmd.Run())

	// Add replace directive to local kuniumi
	kuniumiPath := projectRoot
	replaceCmd := exec.Command("go", "mod", "edit", "-replace", fmt.Sprintf("github.com/axsh/kuniumi=%s", kuniumiPath))
	replaceCmd.Dir = buildDir
	require.NoError(t, replaceCmd.Run())

	// Copy example main.go
	exampleSrc := filepath.Join(projectRoot, "examples", "basic", "main.go")
	input, err := os.ReadFile(exampleSrc)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(buildDir, "main.go"), input, 0644)
	require.NoError(t, err)

	// Tidy dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = buildDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	require.NoError(t, tidyCmd.Run(), "Failed to tidy dependencies")

	// Build
	binName := "kuniumi_example.exe"
	if os.PathSeparator == '/' {
		binName = "kuniumi_example"
	}
	binPath := filepath.Join(projectRoot, "tmp", binName)

	// Ensure tmp dir exists
	os.MkdirAll(filepath.Join(projectRoot, "tmp"), 0755)

	t.Logf("Building example in %s to %s", buildDir, binPath)
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = buildDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	require.NoError(t, buildCmd.Run(), "Failed to build example app")

	defer os.Remove(binPath) // Cleanup

	// --- Test Cases ---

	// Case 1: Help Command
	t.Run("Help", func(t *testing.T) {
		cmd := exec.Command(binPath, "--help")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)
		assert.Contains(t, string(out), "Calculator v1.0.0")
		assert.Contains(t, string(out), "based on kuniumi")
		assert.Contains(t, string(out), "Available Commands:")
	})

	// Case 2: CGI Mode
	t.Run("CGI", func(t *testing.T) {
		input := `{"x": 10, "y": 20}`
		cmd := exec.Command(binPath, "cgi")
		cmd.Env = append(os.Environ(), "PATH_INFO=/Add", "REQUEST_METHOD=POST")
		cmd.Stdin = strings.NewReader(input)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr // Debug info

		require.NoError(t, cmd.Run())

		output := out.String()
		assert.Contains(t, output, "Status: 200 OK")
		assert.Contains(t, output, `{"result":30}`)
	})

	// Case 2a-err: CGI Error Response (JSON format)
	t.Run("CGI/ErrorResponse", func(t *testing.T) {
		input := `{"x": 10, "y": 20}`
		cmd := exec.Command(binPath, "cgi")
		cmd.Env = append(os.Environ(), "PATH_INFO=/NonExistent", "REQUEST_METHOD=POST")
		cmd.Stdin = strings.NewReader(input)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr

		require.NoError(t, cmd.Run())

		output := out.String()
		assert.Contains(t, output, "Status: 404 Not Found")
		assert.Contains(t, output, "Content-Type: application/json")

		bodyIdx := strings.Index(output, "\r\n\r\n")
		require.Greater(t, bodyIdx, 0, "CGI output should contain header/body separator")
		body := output[bodyIdx+4:]

		var parsed map[string]any
		err := json.Unmarshal([]byte(body), &parsed)
		require.NoError(t, err, "CGI error body should be valid JSON")
		assert.Contains(t, parsed["error"], "NonExistent",
			"error should mention the missing function name")
	})

	// Case 2b: CGI Mode with string numeric values
	t.Run("CGI/StringArgs", func(t *testing.T) {
		input := `{"x": "10", "y": "20"}`
		cmd := exec.Command(binPath, "cgi")
		cmd.Env = append(os.Environ(), "PATH_INFO=/Add", "REQUEST_METHOD=POST")
		cmd.Stdin = strings.NewReader(input)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr

		require.NoError(t, cmd.Run())

		output := out.String()
		assert.Contains(t, output, "Status: 200 OK")
		assert.Contains(t, output, `{"result":30}`)
	})

	// Case 2c: CGI OpenAPI
	t.Run("CGI/OpenAPI", func(t *testing.T) {
		cmd := exec.Command(binPath, "cgi")
		cmd.Env = append(os.Environ(), "PATH_INFO=/openapi.json", "REQUEST_METHOD=GET")
		cmd.Stdin = strings.NewReader("")

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr

		require.NoError(t, cmd.Run())

		output := out.String()
		assert.Contains(t, output, "Status: 200 OK")
		assert.Contains(t, output, "Content-Type: application/json")

		// Parse JSON body (after CGI headers separated by \r\n\r\n)
		bodyIdx := strings.Index(output, "\r\n\r\n")
		require.Greater(t, bodyIdx, 0, "CGI output should contain header/body separator")
		body := output[bodyIdx+4:]

		assertValidOpenAPISpec(t, []byte(body))
	})

	// Case 3: Serve Mode (HTTP)
	t.Run("Serve", func(t *testing.T) {
		// Run server in background
		cmd := exec.Command(binPath, "serve", "--port", "9999")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Start())
		defer func() {
			cmd.Process.Kill()
			cmd.Wait()
		}()

		// Wait for server to start
		time.Sleep(1 * time.Second)

		t.Run("FunctionCall", func(t *testing.T) {
			// POST /functions/Add
			reqBody := []byte(`{"x": 5, "y": 5}`)
			resp, err := httpPost("http://localhost:9999/functions/Add", "application/json", bytes.NewReader(reqBody))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 200, resp.StatusCode)

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			assert.Equal(t, float64(10), result["result"])
		})

		t.Run("ErrorResponse", func(t *testing.T) {
			resp, err := httpPost("http://localhost:9999/functions/Add",
				"application/json", strings.NewReader("not json"))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 400, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

			var parsed map[string]any
			json.NewDecoder(resp.Body).Decode(&parsed)
			assert.Equal(t, "Invalid JSON body", parsed["error"])
		})

		t.Run("OpenAPI", func(t *testing.T) {
			respSpec, err := httpGet("http://localhost:9999/openapi.json")
			require.NoError(t, err)
			defer respSpec.Body.Close()

			assert.Equal(t, 200, respSpec.StatusCode)

			body, err := io.ReadAll(respSpec.Body)
			require.NoError(t, err)

			assertValidOpenAPISpec(t, body)
		})
	})

	// Case 4: Virtual Environment & File Write
	t.Run("VirtualEnv", func(t *testing.T) {
		// Prepare a temp dir for mounting
		tmpDir, err := os.MkdirTemp("", "kuniumi_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir) // Cleanup

		input := `{"x": 1, "y": 1}`
		cmd := exec.Command(binPath, "cgi",
			"--env", "DEBUG=true",
			"--mount", fmt.Sprintf("%s:/", tmpDir),
		)
		cmd.Env = append(os.Environ(), "PATH_INFO=/Add", "REQUEST_METHOD=POST")
		cmd.Stdin = strings.NewReader(input)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		require.NoError(t, err, "CGI execution failed")

		// Check if file exists on Host (tmpDir/debug.log)
		// Because "/" (Virtual) -> tmpDir (Host)
		// "debug.log" -> "/debug.log" -> "tmpDir/debug.log"

		content, err := os.ReadFile(filepath.Join(tmpDir, "debug.log"))
		if assert.NoError(t, err, "File should be created in mounted volume") {
			assert.Contains(t, string(content), "Adding 1 + 1")
		}
	})

	// Case 5: MCP Mode (Model Context Protocol over stdio)
	t.Run("MCP", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client := mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)

		transport := &mcp.CommandTransport{
			Command: exec.Command(binPath, "mcp"),
		}
		session, err := client.Connect(ctx, transport, nil)
		require.NoError(t, err)
		defer session.Close()

		t.Run("ListTools", func(t *testing.T) {
			result, err := session.ListTools(ctx, nil)
			require.NoError(t, err)
			require.NotNil(t, result)

			var toolNames []string
			for _, tool := range result.Tools {
				toolNames = append(toolNames, tool.Name)
			}
			assert.Contains(t, toolNames, "functions.Add",
				"tool name should follow OpenAPI path convention (dots instead of slashes)")
		})

		t.Run("ToolSchema", func(t *testing.T) {
			result, err := session.ListTools(ctx, nil)
			require.NoError(t, err)

			var addTool *mcp.Tool
			for _, tool := range result.Tools {
				if tool.Name == "functions.Add" {
					addTool = tool
					break
				}
			}
			require.NotNil(t, addTool, "should find functions.Add tool")
			assert.Equal(t, "Adds two integers together", addTool.Description)
		})

		t.Run("CallTool", func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "functions.Add",
				Arguments: map[string]any{
					"x": 7,
					"y": 3,
				},
			})
			require.NoError(t, err)
			require.False(t, result.IsError, "tool call should not return an error")
			require.Greater(t, len(result.Content), 0, "result should have content")

			textContent, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok, "content should be TextContent")

			var parsed map[string]any
			err = json.Unmarshal([]byte(textContent.Text), &parsed)
			require.NoError(t, err, "MCP response text should be valid JSON")
			assert.Equal(t, float64(10), parsed["result"],
				"Add(7, 3) should return {\"result\": 10}")
		})

		t.Run("CallToolError", func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "functions.Add",
				Arguments: map[string]any{
					"x": "not_a_number",
					"y": 3,
				},
			})
			require.NoError(t, err)
			require.True(t, result.IsError, "tool call should return an error")
			require.Greater(t, len(result.Content), 0, "error result should have content")

			textContent, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok, "content should be TextContent")

			var parsed map[string]any
			err = json.Unmarshal([]byte(textContent.Text), &parsed)
			require.NoError(t, err, "MCP error text should be valid JSON")
			assert.NotEmpty(t, parsed["error"], "error response should contain 'error' field")
		})
	})
}

// Helpers

func httpPost(url, contentType string, body io.Reader) (*http.Response, error) {
	return http.Post(url, contentType, body)
}

func httpGet(url string) (*http.Response, error) {
	return http.Get(url)
}

// assertValidOpenAPISpec validates the structure and content of an OpenAPI spec JSON.
func assertValidOpenAPISpec(t *testing.T, specJSON []byte) {
	t.Helper()

	var spec map[string]interface{}
	err := json.Unmarshal(specJSON, &spec)
	require.NoError(t, err, "OpenAPI spec should be valid JSON")

	// Top-level fields
	assert.Equal(t, "3.0.0", spec["openapi"], "openapi version should be 3.0.0")

	info, ok := spec["info"].(map[string]interface{})
	require.True(t, ok, "info should be an object")
	assert.Equal(t, "Calculator", info["title"], "info.title should match app name")
	assert.Equal(t, "1.0.0", info["version"], "info.version should match app version")

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok, "paths should be an object")

	// /functions/Add path
	pathAdd, ok := paths["/functions/Add"].(map[string]interface{})
	require.True(t, ok, "paths should contain /functions/Add")

	post, ok := pathAdd["post"].(map[string]interface{})
	require.True(t, ok, "/functions/Add should have post operation")

	// post.description
	assert.Equal(t, "Adds two integers together", post["description"],
		"post.description should match function description")

	// operationId must match MCP tool name
	assert.Equal(t, "functions.Add", post["operationId"],
		"operationId should match MCP tool name")

	// requestBody schema
	reqBody, ok := post["requestBody"].(map[string]interface{})
	require.True(t, ok, "post should have requestBody")

	content, ok := reqBody["content"].(map[string]interface{})
	require.True(t, ok, "requestBody should have content")

	appJson, ok := content["application/json"].(map[string]interface{})
	require.True(t, ok, "content should have application/json")

	schema, ok := appJson["schema"].(map[string]interface{})
	require.True(t, ok, "application/json should have schema")

	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "schema should have properties")

	// Check property "x"
	propX, ok := props["x"].(map[string]interface{})
	require.True(t, ok, "properties should contain 'x'")
	assert.Equal(t, "First integer to add", propX["description"])
	assert.Equal(t, "integer", propX["type"])

	// Check property "y"
	propY, ok := props["y"].(map[string]interface{})
	require.True(t, ok, "properties should contain 'y'")
	assert.Equal(t, "Second integer to add", propY["description"])
	assert.Equal(t, "integer", propY["type"])

	// Check responses
	responses, ok := post["responses"].(map[string]interface{})
	require.True(t, ok, "post should have responses")

	resp200, ok := responses["200"].(map[string]interface{})
	require.True(t, ok, "responses should contain '200'")
	assert.NotEmpty(t, resp200["description"], "200 response should have description")

	// Check response body schema
	respContent, ok := resp200["content"].(map[string]interface{})
	require.True(t, ok, "200 response should have content")

	respAppJson, ok := respContent["application/json"].(map[string]interface{})
	require.True(t, ok, "response content should have application/json")

	respSchema, ok := respAppJson["schema"].(map[string]interface{})
	require.True(t, ok, "response application/json should have schema")

	assert.Equal(t, "object", respSchema["type"], "response schema type should be object")

	respProps, ok := respSchema["properties"].(map[string]interface{})
	require.True(t, ok, "response schema should have properties")

	resultProp, ok := respProps["result"].(map[string]interface{})
	require.True(t, ok, "response properties should contain 'result'")
	assert.Equal(t, "integer", resultProp["type"],
		"result type should be integer for Add function")
	assert.Equal(t, "Sum of x and y", resultProp["description"],
		"result description should match WithReturns value")

	// Check 400 error response
	resp400, ok := responses["400"].(map[string]interface{})
	require.True(t, ok, "responses should contain '400'")
	assert.Equal(t, "Invalid request", resp400["description"])

	resp400Content, ok := resp400["content"].(map[string]interface{})
	require.True(t, ok, "400 response should have content")
	resp400AppJson, ok := resp400Content["application/json"].(map[string]interface{})
	require.True(t, ok, "400 content should have application/json")
	resp400Schema, ok := resp400AppJson["schema"].(map[string]interface{})
	require.True(t, ok, "400 application/json should have schema")
	assert.Equal(t, "object", resp400Schema["type"])
	resp400Props, ok := resp400Schema["properties"].(map[string]interface{})
	require.True(t, ok, "400 schema should have properties")
	_, ok = resp400Props["error"]
	assert.True(t, ok, "400 schema properties should contain 'error'")

	// Check 500 error response
	resp500, ok := responses["500"].(map[string]interface{})
	require.True(t, ok, "responses should contain '500'")
	assert.Equal(t, "Internal server error", resp500["description"])
}
