//go:build integration

package kuniumi_test

import (
	"bytes"
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
}
