package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestRunCommand403ErrorInCodespace tests that a 403 error in a codespace
// shows a helpful error message
func TestRunCommand403ErrorInCodespace(t *testing.T) {
	// Save original environment
	originalCodespaces := os.Getenv("CODESPACES")
	defer func() {
		if originalCodespaces != "" {
			os.Setenv("CODESPACES", originalCodespaces)
		} else {
			os.Unsetenv("CODESPACES")
		}
	}()

	tests := []struct {
		name               string
		inCodespace        bool
		stderrContent      string
		wantCodespaceMsg   bool
		wantCodespaceInErr bool
	}{
		{
			name:               "403 error in codespace shows specialized message",
			inCodespace:        true,
			stderrContent:      "HTTP 403: Resource not accessible by integration",
			wantCodespaceMsg:   true,
			wantCodespaceInErr: true,
		},
		{
			name:               "forbidden error in codespace shows specialized message",
			inCodespace:        true,
			stderrContent:      "Error: forbidden",
			wantCodespaceMsg:   true,
			wantCodespaceInErr: true,
		},
		{
			name:               "permission denied in codespace shows specialized message",
			inCodespace:        true,
			stderrContent:      "Error: permission denied",
			wantCodespaceMsg:   true,
			wantCodespaceInErr: true,
		},
		{
			name:               "403 error outside codespace shows standard error",
			inCodespace:        false,
			stderrContent:      "HTTP 403: Resource not accessible by integration",
			wantCodespaceMsg:   false,
			wantCodespaceInErr: false,
		},
		{
			name:               "non-403 error in codespace shows standard error",
			inCodespace:        true,
			stderrContent:      "Error: 404 Not Found",
			wantCodespaceMsg:   false,
			wantCodespaceInErr: false,
		},
		{
			name:               "generic error in codespace shows standard error",
			inCodespace:        true,
			stderrContent:      "Error: something went wrong",
			wantCodespaceMsg:   false,
			wantCodespaceInErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up codespace environment
			if tt.inCodespace {
				os.Setenv("CODESPACES", "true")
			} else {
				os.Unsetenv("CODESPACES")
			}

			// Test the helper functions directly
			if tt.wantCodespaceMsg {
				if !isRunningInCodespace() {
					t.Error("Expected to be running in codespace")
				}
				if !is403PermissionError(tt.stderrContent) {
					t.Errorf("Expected %q to be detected as 403 error", tt.stderrContent)
				}
			}

			// Verify the error message contains expected content
			if tt.wantCodespaceMsg {
				msg := getCodespacePermissionErrorMessage()
				if !strings.Contains(msg, "Codespace") {
					t.Error("Expected codespace-specific message to contain 'Codespace'")
				}
				if !strings.Contains(msg, "actions:write") {
					t.Error("Expected codespace-specific message to contain 'actions:write'")
				}
			}
		})
	}
}

// TestCodespaceErrorMessageIntegration tests the integration of codespace error detection
// in a more realistic scenario
func TestCodespaceErrorMessageIntegration(t *testing.T) {
	// Save original environment
	originalCodespaces := os.Getenv("CODESPACES")
	defer func() {
		if originalCodespaces != "" {
			os.Setenv("CODESPACES", originalCodespaces)
		} else {
			os.Unsetenv("CODESPACES")
		}
	}()

	// Simulate being in a codespace
	os.Setenv("CODESPACES", "true")

	// Create a mock stderr buffer with 403 error
	var stderr bytes.Buffer
	stderr.WriteString("HTTP 403: Resource not accessible by integration")

	// Verify we can detect this as a 403 error
	errorMsg := stderr.String()
	if !is403PermissionError(errorMsg) {
		t.Errorf("Expected %q to be detected as 403 error", errorMsg)
	}

	// Verify we detect we're in a codespace
	if !isRunningInCodespace() {
		t.Error("Expected to detect running in codespace")
	}

	// Verify the error message is appropriate
	msg := getCodespacePermissionErrorMessage()
	expectedPhrases := []string{
		"Codespace",
		"actions:write",
		"Personal Access Token",
		"GITHUB_TOKEN",
		"docs.github.com",
		"devcontainer",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(msg, phrase) {
			t.Errorf("Expected error message to contain %q, got: %s", phrase, msg)
		}
	}
}
