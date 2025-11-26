package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestHasPlaywrightMCPServer(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     bool
	}{
		{
			name:         "nil workflow data",
			workflowData: nil,
			expected:     false,
		},
		{
			name:         "empty workflow data",
			workflowData: &WorkflowData{},
			expected:     false,
		},
		{
			name: "playwright in ParsedTools",
			workflowData: &WorkflowData{
				ParsedTools: &Tools{
					Playwright: &PlaywrightToolConfig{},
				},
			},
			expected: true,
		},
		{
			name: "playwright in raw Tools map",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"playwright": nil,
				},
			},
			expected: true,
		},
		{
			name: "playwright in Tools map with config",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"playwright": map[string]any{
						"allowed_domains": []string{"example.com"},
					},
				},
			},
			expected: true,
		},
		{
			name: "no playwright tool",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"github": nil,
				},
			},
			expected: false,
		},
		{
			name: "empty Tools map",
			workflowData: &WorkflowData{
				Tools: map[string]any{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPlaywrightMCPServer(tt.workflowData)
			if result != tt.expected {
				t.Errorf("HasPlaywrightMCPServer() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestShouldPreinstallPlaywrightBrowsers(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     bool
	}{
		{
			name:         "nil workflow data",
			workflowData: nil,
			expected:     false,
		},
		{
			name: "playwright without firewall",
			workflowData: &WorkflowData{
				ParsedTools: &Tools{
					Playwright: &PlaywrightToolConfig{},
				},
				// No firewall configuration
			},
			expected: false,
		},
		{
			name: "playwright with firewall enabled",
			workflowData: &WorkflowData{
				ParsedTools: &Tools{
					Playwright: &PlaywrightToolConfig{},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "playwright with firewall disabled",
			workflowData: &WorkflowData{
				ParsedTools: &Tools{
					Playwright: &PlaywrightToolConfig{},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: false,
					},
				},
			},
			expected: false,
		},
		{
			name: "no playwright with firewall enabled",
			workflowData: &WorkflowData{
				ParsedTools: &Tools{
					GitHub: &GitHubToolConfig{},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
			},
			expected: false,
		},
		{
			name: "playwright in tools map with firewall enabled",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"playwright": nil,
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldPreinstallPlaywrightBrowsers(tt.workflowData)
			if result != tt.expected {
				t.Errorf("ShouldPreinstallPlaywrightBrowsers() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGeneratePlaywrightBrowserInstallStep(t *testing.T) {
	step := GeneratePlaywrightBrowserInstallStep()
	stepContent := strings.Join([]string(step), "\n")

	// Check step name
	if !strings.Contains(stepContent, "name: Pre-install Playwright browsers") {
		t.Error("Expected step name 'Pre-install Playwright browsers' in step content")
	}

	// Check for npx playwright install command with the correct version
	expectedVersion := string(constants.DefaultPlaywrightMCPVersion)
	expectedCommand := "npx playwright@" + expectedVersion + " install --with-deps chromium"
	if !strings.Contains(stepContent, expectedCommand) {
		t.Errorf("Expected command '%s' in step content:\n%s", expectedCommand, stepContent)
	}

	// Check for explanatory comments
	if !strings.Contains(stepContent, "before firewall starts") {
		t.Error("Expected comment about firewall in step content")
	}
}

func TestCopilotEngineInstallationStepsWithPlaywright(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with Playwright configured and firewall enabled
	workflowDataWithPlaywright := &WorkflowData{
		Name: "test-workflow",
		ParsedTools: &Tools{
			Playwright: &PlaywrightToolConfig{},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	steps := engine.GetInstallationSteps(workflowDataWithPlaywright)

	// Should have: secret validation + Node.js setup + AWF install + Copilot CLI install + Playwright browser install = 5 steps
	if len(steps) != 5 {
		t.Errorf("Expected 5 installation steps (secret validation + Node.js setup + AWF install + Copilot CLI install + Playwright browser install), got %d", len(steps))
	}

	// Check that Playwright browser install step is present
	found := false
	for _, step := range steps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "Pre-install Playwright browsers") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected Playwright browser pre-install step in installation steps")
	}
}

func TestCopilotEngineInstallationStepsWithoutPlaywright(t *testing.T) {
	engine := NewCopilotEngine()

	// Test without Playwright but with firewall enabled
	workflowDataWithoutPlaywright := &WorkflowData{
		Name: "test-workflow",
		ParsedTools: &Tools{
			GitHub: &GitHubToolConfig{},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	steps := engine.GetInstallationSteps(workflowDataWithoutPlaywright)

	// Should have: secret validation + Node.js setup + AWF install + Copilot CLI install = 4 steps (no Playwright)
	if len(steps) != 4 {
		t.Errorf("Expected 4 installation steps (no Playwright browser install), got %d", len(steps))
	}

	// Check that Playwright browser install step is NOT present
	for _, step := range steps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "Pre-install Playwright browsers") {
			t.Error("Did not expect Playwright browser pre-install step when Playwright is not configured")
		}
	}
}

func TestCopilotEngineInstallationStepsPlaywrightNoFirewall(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with Playwright but without firewall
	workflowData := &WorkflowData{
		Name: "test-workflow",
		ParsedTools: &Tools{
			Playwright: &PlaywrightToolConfig{},
		},
		// No firewall configuration
	}

	steps := engine.GetInstallationSteps(workflowData)

	// Should have: secret validation + Node.js setup + Copilot CLI install = 3 steps (no AWF, no Playwright pre-install)
	if len(steps) != 3 {
		t.Errorf("Expected 3 installation steps (no AWF, no Playwright browser pre-install), got %d", len(steps))
	}

	// Check that Playwright browser install step is NOT present (not needed when firewall is disabled)
	for _, step := range steps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "Pre-install Playwright browsers") {
			t.Error("Did not expect Playwright browser pre-install step when firewall is disabled")
		}
	}
}
