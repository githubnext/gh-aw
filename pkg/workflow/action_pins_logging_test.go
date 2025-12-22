package workflow

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestActionPinResolutionWithMismatchedVersions demonstrates the issue where
// action_pins.json has mismatches between keys and version fields
func TestActionPinResolutionWithMismatchedVersions(t *testing.T) {
	// This test demonstrates the problem: when requesting actions/ai-inference@v1,
	// if dynamic resolution fails, it falls back to the hardcoded pin which has
	// version v2.0.4, causing the wrong SHA to be returned

	tests := []struct {
		name           string
		repo           string
		requestedVer   string
		expectedPinVer string // The version in the hardcoded pin
		expectMismatch bool
	}{
		{
			name:           "ai-inference v1 resolves to v2.0.4 pin",
			repo:           "actions/ai-inference",
			requestedVer:   "v1",
			expectedPinVer: "v2.0.4",
			expectMismatch: true,
		},
		{
			name:           "setup-dotnet v4 resolves to v4.3.1 pin",
			repo:           "actions/setup-dotnet",
			requestedVer:   "v4",
			expectedPinVer: "v4.3.1",
			expectMismatch: true,
		},
		{
			name:           "github-script v7.0.1 resolves to v8.0.0 pin (latest version)",
			repo:           "actions/github-script",
			requestedVer:   "v7.0.1",
			expectedPinVer: "v8.0.0", // Returns latest version for this repo
			expectMismatch: true,
		},
		{
			name:           "checkout v5.0.1 exact match",
			repo:           "actions/checkout",
			requestedVer:   "v5.0.1",
			expectedPinVer: "v5.0.1",
			expectMismatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a WorkflowData without a resolver to force fallback to hardcoded pins
			data := &WorkflowData{
				StrictMode:     false, // Non-strict mode allows version mismatch
				ActionResolver: nil,   // No resolver to force hardcoded pin usage
			}

			// Capture stderr to check for warning messages
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			result, err := GetActionPinWithData(tt.repo, tt.requestedVer, data)

			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderr := buf.String()

			if err != nil {
				t.Errorf("GetActionPinWithData() error = %v", err)
				return
			}

			if result == "" {
				t.Errorf("GetActionPinWithData() returned empty result")
				return
			}

			// Check if the result contains the expected version
			if !strings.Contains(result, "# "+tt.expectedPinVer) {
				t.Errorf("GetActionPinWithData() = %s, expected to contain '# %s'", result, tt.expectedPinVer)
			}

			// For mismatched versions, we should see a warning
			if tt.expectMismatch {
				if !strings.Contains(stderr, "⚠") {
					t.Errorf("Expected warning message in stderr for version mismatch, got: %s", stderr)
				}
				// Verify the warning mentions both versions
				if !strings.Contains(stderr, tt.requestedVer) || !strings.Contains(stderr, tt.expectedPinVer) {
					t.Errorf("Warning should mention both requested version (%s) and hardcoded version (%s), got: %s",
						tt.requestedVer, tt.expectedPinVer, stderr)
				}
			}

			// Log the resolution for debugging
			t.Logf("Resolution: %s@%s → %s", tt.repo, tt.requestedVer, result)
			if stderr != "" {
				t.Logf("Stderr: %s", strings.TrimSpace(stderr))
			}
		})
	}
}

// TestActionPinResolutionWithStrictMode tests that strict mode prevents version mismatches
func TestActionPinResolutionWithStrictMode(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		requestedVer string
		expectError  bool
	}{
		{
			name:         "ai-inference v1 fails in strict mode",
			repo:         "actions/ai-inference",
			requestedVer: "v1",
			expectError:  true,
		},
		{
			name:         "checkout v5.0.1 succeeds in strict mode",
			repo:         "actions/checkout",
			requestedVer: "v5.0.1",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a WorkflowData in strict mode without a resolver
			data := &WorkflowData{
				StrictMode:     true, // Strict mode should error on version mismatch
				ActionResolver: nil,
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			result, err := GetActionPinWithData(tt.repo, tt.requestedVer, data)

			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error in strict mode for %s@%s, got nil", tt.repo, tt.requestedVer)
				}
				if result != "" {
					t.Errorf("Expected empty result on error, got: %s", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == "" {
					t.Errorf("Expected non-empty result")
				}
			}
		})
	}
}
