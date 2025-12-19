package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestCLIVersionInAwInfo(t *testing.T) {
	tests := []struct {
		name        string
		cliVersion  string
		engineID    string
		description string
	}{
		{
			name:        "CLI version is stored in aw_info.json",
			cliVersion:  "1.2.3",
			engineID:    "copilot",
			description: "Should include cli_version field with correct value",
		},
		{
			name:        "CLI version with semver prerelease",
			cliVersion:  "1.2.3-beta.1",
			engineID:    "claude",
			description: "Should handle prerelease versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if !strings.Contains(output, expectedLine) {
				t.Errorf("%s: Expected output to contain '%s', got:\n%s",
					tt.description, expectedLine, output)
			}
		})
	}
}

func TestAwfVersionInAwInfo(t *testing.T) {
	tests := []struct {
		name               string
		firewallEnabled    bool
		firewallVersion    string
		expectedAwfVersion string
		description        string
	}{
		{
			name:               "Firewall enabled with explicit version",
			firewallEnabled:    true,
			firewallVersion:    "v1.0.0",
			expectedAwfVersion: "v1.0.0",
			description:        "Should use explicit firewall version",
		},
		{
			name:               "Firewall enabled with default version",
			firewallEnabled:    true,
			firewallVersion:    "",
			expectedAwfVersion: string(constants.DefaultFirewallVersion),
			description:        "Should use default firewall version when not specified",
		},
		{
			name:               "Firewall disabled",
			firewallEnabled:    false,
			firewallVersion:    "",
			expectedAwfVersion: "",
			description:        "Should have empty awf_version when firewall is disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "1.0.0")
			registry := GetGlobalEngineRegistry()
			engine, err := registry.GetEngine("copilot")
			if err != nil {
				t.Fatalf("Failed to get copilot engine: %v", err)
			}

			workflowData := &WorkflowData{
				Name: "Test Workflow",
			}

			if tt.firewallEnabled {
				workflowData.NetworkPermissions = &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
						Version: tt.firewallVersion,
					},
				}
			}

			var yaml strings.Builder
			compiler.generateCreateAwInfo(&yaml, workflowData, engine)
			output := yaml.String()

			expectedLine := `awf_version: "` + tt.expectedAwfVersion + `"`
			if !strings.Contains(output, expectedLine) {
				t.Errorf("%s: Expected output to contain '%s', got:\n%s",
					tt.description, expectedLine, output)
			}
		})
	}
}

func TestBothVersionsInAwInfo(t *testing.T) {
	// Test that both CLI version and AWF version are present simultaneously
	compiler := NewCompiler(false, "", "2.0.0-test")
	registry := GetGlobalEngineRegistry()
	engine, err := registry.GetEngine("copilot")
	if err != nil {
		t.Fatalf("Failed to get copilot engine: %v", err)
	}

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
				Version: "v0.5.0",
			},
		},
	}

	var yaml strings.Builder
	compiler.generateCreateAwInfo(&yaml, workflowData, engine)
	output := yaml.String()

	// Check for cli_version
	expectedCLILine := `cli_version: "2.0.0-test"`
	if !strings.Contains(output, expectedCLILine) {
		t.Errorf("Expected output to contain cli_version '%s', got:\n%s", expectedCLILine, output)
	}

	// Check for awf_version
	expectedAwfLine := `awf_version: "v0.5.0"`
	if !strings.Contains(output, expectedAwfLine) {
		t.Errorf("Expected output to contain awf_version '%s', got:\n%s", expectedAwfLine, output)
	}
}
