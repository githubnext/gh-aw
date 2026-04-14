//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
)

// TestSetVersionInfo verifies that SetVersionInfo keeps both cli.GetVersion()
// and workflow.GetVersion() in sync via a single call.
func TestSetVersionInfo(t *testing.T) {
	// Save original state and restore it after the test
	originalCLIVersion := version
	originalWorkflowVersion := workflow.GetVersion()
	defer func() {
		version = originalCLIVersion
		workflow.SetVersion(originalWorkflowVersion)
	}()

	tests := []struct {
		name    string
		version string
	}{
		{"release version", "1.2.3"},
		{"dev version", "dev"},
		{"semver with v prefix", "v2.0.0"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetVersionInfo(tt.version)

			if got := GetVersion(); got != tt.version {
				t.Errorf("GetVersion() = %q, want %q", got, tt.version)
			}
			if got := workflow.GetVersion(); got != tt.version {
				t.Errorf("workflow.GetVersion() = %q, want %q after SetVersionInfo(%q)", got, tt.version, tt.version)
			}
		})
	}
}
