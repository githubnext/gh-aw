package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var ghHelperLog = logger.New("workflow:gh_helper")

// ExecGH wraps exec.Command for "gh" CLI calls and ensures proper token configuration.
// It sets GH_TOKEN from GITHUB_TOKEN if GH_TOKEN is not already set.
// This ensures gh CLI commands work in environments where GITHUB_TOKEN is set but GH_TOKEN is not.
//
// Usage:
//
//	cmd := ExecGH("api", "/user")
//	output, err := cmd.Output()
func ExecGH(args ...string) *exec.Cmd {
	cmd := exec.Command("gh", args...)

	// Check if GH_TOKEN is already set
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken != "" {
		ghHelperLog.Printf("GH_TOKEN is set, using it for gh CLI")
		return cmd
	}

	// Fall back to GITHUB_TOKEN if available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		ghHelperLog.Printf("GH_TOKEN not set, using GITHUB_TOKEN as fallback for gh CLI")
		// Set GH_TOKEN in the command's environment
		cmd.Env = append(os.Environ(), "GH_TOKEN="+githubToken)
	} else {
		ghHelperLog.Printf("Neither GH_TOKEN nor GITHUB_TOKEN is set, gh CLI will use default authentication")
	}

	return cmd
}

// ExecGHWithRESTFallback executes a gh CLI command with fallback to unauthenticated REST API.
// It is specifically designed for "gh api" commands.
//
// When gh CLI fails due to missing/invalid authentication, this function will attempt
// to make the same API call using direct HTTP REST API without authentication.
//
// Args:
//   - args: Command arguments (e.g., "api", "/repos/owner/repo/git/ref/tags/v1.0", "--jq", ".object.sha")
//
// Returns:
//   - output: The command output (either from gh CLI or REST API)
//   - fromREST: true if the output came from REST API fallback, false if from gh CLI
//   - err: Error if both gh CLI and REST API failed
//
// Usage:
//
//	output, fromREST, err := ExecGHWithRESTFallback("api", "/repos/actions/checkout/git/ref/tags/v4", "--jq", ".object.sha")
func ExecGHWithRESTFallback(args ...string) ([]byte, bool, error) {
	// First try with gh CLI
	cmd := ExecGH(args...)
	output, err := cmd.Output()
	
	if err == nil {
		ghHelperLog.Printf("gh CLI succeeded")
		return output, false, nil
	}

	// Check if this is a gh api command that failed due to authentication
	if len(args) < 2 || args[0] != "api" {
		ghHelperLog.Printf("Not a gh api command or insufficient args, returning original error")
		return nil, false, err
	}

	// Check if error is authentication-related
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		ghHelperLog.Printf("Not an exit error, returning original error")
		return nil, false, err
	}

	// Common authentication error exit codes:
	// - exit status 4: authentication error in gh CLI
	// - exit status 1: general error (could be auth-related)
	stderr := string(exitErr.Stderr)
	isAuthError := exitErr.ExitCode() == 4 ||
		strings.Contains(stderr, "authentication") ||
		strings.Contains(stderr, "HTTP 401") ||
		strings.Contains(stderr, "HTTP 403") ||
		strings.Contains(strings.ToLower(stderr), "unauthorized")

	if !isAuthError {
		ghHelperLog.Printf("Error is not authentication-related, returning original error: %v", stderr)
		return nil, false, err
	}

	ghHelperLog.Printf("Authentication error detected, attempting REST API fallback")

	// Extract API path and jq filter
	apiPath := args[1]
	var jqFilter string
	for i := 2; i < len(args); i++ {
		if args[i] == "--jq" && i+1 < len(args) {
			jqFilter = args[i+1]
			break
		}
	}

	// Attempt REST API call
	restOutput, err := callGitHubRESTAPI(apiPath, jqFilter)
	if err != nil {
		ghHelperLog.Printf("REST API fallback failed: %v", err)
		// Return original gh CLI error since REST API also failed
		return nil, false, exitErr
	}

	ghHelperLog.Printf("REST API fallback succeeded")
	return restOutput, true, nil
}

// callGitHubRESTAPI makes a direct HTTP call to the GitHub REST API without authentication.
// It handles jq filtering by parsing the JSON response and extracting the specified field.
func callGitHubRESTAPI(apiPath, jqFilter string) ([]byte, error) {
	// Normalize API path (remove leading slash if present)
	apiPath = strings.TrimPrefix(apiPath, "/")
	
	// Build API URL
	baseURL := "https://api.github.com"
	url := fmt.Sprintf("%s/%s", baseURL, apiPath)

	ghHelperLog.Printf("Making REST API call to: %s", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gh-aw")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// If jq filter is specified, extract the field
	if jqFilter != "" {
		filtered, err := applyJQFilter(body, jqFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to apply jq filter %s: %w", jqFilter, err)
		}
		return filtered, nil
	}

	return body, nil
}

// applyJQFilter applies a simple jq-like filter to JSON data.
// It only supports simple field extraction like ".object.sha" or ".name"
func applyJQFilter(jsonData []byte, filter string) ([]byte, error) {
	// Parse filter (e.g., ".object.sha" -> ["object", "sha"])
	filter = strings.TrimPrefix(filter, ".")
	fields := strings.Split(filter, ".")

	// Parse JSON
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Navigate through fields
	current := data
	for _, field := range fields {
		if field == "" {
			continue
		}

		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot access field %s: not an object", field)
		}

		value, exists := m[field]
		if !exists {
			return nil, fmt.Errorf("field %s not found", field)
		}

		current = value
	}

	// Convert result to string
	switch v := current.(type) {
	case string:
		return []byte(v + "\n"), nil
	case float64:
		return []byte(fmt.Sprintf("%v\n", v)), nil
	case bool:
		return []byte(fmt.Sprintf("%v\n", v)), nil
	default:
		// For complex types, return JSON
		result, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		return append(result, '\n'), nil
	}
}
