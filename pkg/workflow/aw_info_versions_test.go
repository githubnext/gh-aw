package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestCLIVersionInAwInfo(t *testing.T) {
	tests := []struct {
		name          string
		cliVersion    string
		engineID      string
		description   string
		shouldInclude bool
		isRelease     bool // Whether to mark as release build
	}{
		{
			name:          "Released CLI version is stored in aw_info.json",
			cliVersion:    "1.2.3",
			engineID:      "copilot",
			description:   "Should include cli_version field with correct value for released builds",
			shouldInclude: true,
			isRelease:     true,
		},
		{
			name:          "CLI version with semver prerelease",
			cliVersion:    "1.2.3-beta.1",
			engineID:      "claude",
			description:   "Should handle prerelease versions",
			shouldInclude: true,
			isRelease:     true,
		},
		{
			name:          "Development CLI version is excluded",
			cliVersion:    "dev",
			engineID:      "copilot",
			description:   "Should NOT include cli_version field for development builds",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Dirty CLI version is excluded",
			cliVersion:    "1.2.3-dirty",
			engineID:      "copilot",
			description:   "Should NOT include cli_version field for dirty builds",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Test CLI version is excluded",
			cliVersion:    "1.0.0-test",
			engineID:      "claude",
			description:   "Should NOT include cli_version field for test builds",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Git hash with dirty suffix is excluded",
			cliVersion:    "708d3ee-dirty",
			engineID:      "copilot",
			description:   "Should NOT include cli_version field for git hash with dirty suffix",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Git commit hash is excluded",
			cliVersion:    "e63fd5a",
			engineID:      "copilot",
			description:   "Should NOT include cli_version field for git commit hash",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Short git hash is excluded",
			cliVersion:    "abc123",
			engineID:      "claude",
			description:   "Should NOT include cli_version field for short git hash",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Version starting with v is excluded",
			cliVersion:    "v1.2.3",
			engineID:      "copilot",
			description:   "Should NOT include cli_version field for version with v prefix",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Version with only major number is excluded",
			cliVersion:    "1",
			engineID:      "copilot",
			description:   "Should NOT include cli_version field for version with only major number",
			shouldInclude: false,
			isRelease:     false,
		},
		{
			name:          "Version with only major.minor is included",
			cliVersion:    "1.2",
			engineID:      "copilot",
			description:   "Should include cli_version field for version with major.minor",
			shouldInclude: true,
			isRelease:     true,
		},
		{
			name:          "Version with build metadata is included",
			cliVersion:    "1.2.3+build.456",
			engineID:      "claude",
			description:   "Should include cli_version field for version with build metadata",
			shouldInclude: true,
			isRelease:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original state
			originalIsRelease := isReleaseBuild
			defer func() { isReleaseBuild = originalIsRelease }()

			// Set the release flag for this test
			SetIsRelease(tt.isRelease)

			compiler := NewCompiler(false, "", tt.cliVersion)
			registry := GetGlobalEngineRegistry()
			engine, err := registry.GetEngine(tt.engineID)
			if err != nil {
				t.Fatalf("Failed to get %s engine: %v", tt.engineID, err)
			}

			workflowData := &WorkflowData{
				Name: "Test Workflow",
			}

			var yaml strings.Builder
			compiler.generateCreateAwInfo(&yaml, workflowData, engine)
			output := yaml.String()

			expectedLine := `cli_version: "` + tt.cliVersion + `"`
			containsVersion := strings.Contains(output, expectedLine)

			if tt.shouldInclude {
				if !containsVersion {
					t.Errorf("%s: Expected output to contain '%s', got:\n%s",
						tt.description, expectedLine, output)
				}
			} else {
				// For dev builds, cli_version should not appear at all
				if strings.Contains(output, "cli_version:") {
					t.Errorf("%s: Expected output to NOT contain 'cli_version:' field, got:\n%s",
						tt.description, output)
				}
			}
		})
	}
}

// TestAwfVersionInAwInfo has been disabled because it tests the deprecated network.firewall field
// TODO: Rewrite or remove this test once firewall configuration is solely handled via sandbox.agent
func TestAwfVersionInAwInfo(t *testing.T) {
	t.Skip("Test disabled - network.firewall field has been deprecated")
}

// TestBothVersionsInAwInfo has been disabled because it tests the deprecated network.firewall field
// TODO: Rewrite or remove this test once firewall configuration is solely handled via sandbox.agent
func TestBothVersionsInAwInfo(t *testing.T) {
	t.Skip("Test disabled - network.firewall field has been deprecated")
}
