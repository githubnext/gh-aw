package workflow

import (
	"strings"
	"testing"
)

// TestResolveAWFPath tests the resolveAWFPath helper function
func TestResolveAWFPath(t *testing.T) {
	tests := []struct {
		name       string
		customPath string
		expected   string
	}{
		{
			name:       "empty path returns default",
			customPath: "",
			expected:   "/usr/local/bin/awf",
		},
		{
			name:       "absolute path is returned as-is",
			customPath: "/custom/path/to/awf",
			expected:   "/custom/path/to/awf",
		},
		{
			name:       "relative path is resolved against GITHUB_WORKSPACE",
			customPath: "bin/awf",
			expected:   "${GITHUB_WORKSPACE}/bin/awf",
		},
		{
			name:       "relative path with subdirectory",
			customPath: "tools/binaries/awf-custom",
			expected:   "${GITHUB_WORKSPACE}/tools/binaries/awf-custom",
		},
		{
			name:       "root relative path",
			customPath: "awf",
			expected:   "${GITHUB_WORKSPACE}/awf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAWFPath(tt.customPath)
			if result != tt.expected {
				t.Errorf("resolveAWFPath(%q) = %q, expected %q", tt.customPath, result, tt.expected)
			}
		})
	}
}

// TestGetAWFBinaryPath tests the getAWFBinaryPath helper function
func TestGetAWFBinaryPath(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		expected       string
	}{
		{
			name:           "nil config returns default",
			firewallConfig: nil,
			expected:       "awf",
		},
		{
			name:           "empty path returns default",
			firewallConfig: &FirewallConfig{Path: ""},
			expected:       "awf",
		},
		{
			name:           "custom absolute path is resolved",
			firewallConfig: &FirewallConfig{Path: "/custom/path/to/awf"},
			expected:       "/custom/path/to/awf",
		},
		{
			name:           "custom relative path is resolved",
			firewallConfig: &FirewallConfig{Path: "bin/awf"},
			expected:       "${GITHUB_WORKSPACE}/bin/awf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAWFBinaryPath(tt.firewallConfig)
			if result != tt.expected {
				t.Errorf("getAWFBinaryPath() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestGenerateAWFPathValidationStep tests the generateAWFPathValidationStep function
func TestGenerateAWFPathValidationStep(t *testing.T) {
	tests := []struct {
		name               string
		customPath         string
		expectedContains   []string
		unexpectedContains []string
	}{
		{
			name:       "absolute path validation step",
			customPath: "/custom/path/to/awf",
			expectedContains: []string{
				"Validate custom AWF binary",
				"/custom/path/to/awf",
				"if [ ! -f",
				"if [ ! -x",
				"--version",
			},
		},
		{
			name:       "relative path validation step",
			customPath: "bin/awf",
			expectedContains: []string{
				"Validate custom AWF binary",
				"${GITHUB_WORKSPACE}/bin/awf",
				"if [ ! -f",
				"if [ ! -x",
				"--version",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := generateAWFPathValidationStep(tt.customPath)
			stepContent := strings.Join(step, "\n")

			for _, expected := range tt.expectedContains {
				if !strings.Contains(stepContent, expected) {
					t.Errorf("Expected step to contain %q, but it didn't.\nStep content:\n%s", expected, stepContent)
				}
			}

			for _, unexpected := range tt.unexpectedContains {
				if strings.Contains(stepContent, unexpected) {
					t.Errorf("Expected step NOT to contain %q, but it did.\nStep content:\n%s", unexpected, stepContent)
				}
			}
		})
	}
}

// TestCustomPathInstallationSteps tests that GetInstallationSteps generates the correct steps
// when a custom path is specified
func TestCustomPathInstallationSteps(t *testing.T) {
	t.Run("custom path generates validation step instead of install step", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Path:    "/custom/awf/binary",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Check that steps are generated
		if len(steps) == 0 {
			t.Fatal("Expected at least one installation step")
		}

		// Join all steps to check content
		var allStepsContent string
		for _, step := range steps {
			allStepsContent += strings.Join(step, "\n") + "\n"
		}

		// Should contain validation step
		if !strings.Contains(allStepsContent, "Validate custom AWF binary") {
			t.Error("Expected 'Validate custom AWF binary' step")
		}

		// Should NOT contain install step
		if strings.Contains(allStepsContent, "Install awf binary") {
			t.Error("Should NOT contain 'Install awf binary' step when custom path is specified")
		}

		// Should NOT contain curl download
		if strings.Contains(allStepsContent, "curl -L https://github.com/githubnext/gh-aw-firewall") {
			t.Error("Should NOT contain curl download when custom path is specified")
		}
	})

	t.Run("no custom path generates install step", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Join all steps to check content
		var allStepsContent string
		for _, step := range steps {
			allStepsContent += strings.Join(step, "\n") + "\n"
		}

		// Should contain install step
		if !strings.Contains(allStepsContent, "Install awf binary") {
			t.Error("Expected 'Install awf binary' step")
		}

		// Should NOT contain validation step
		if strings.Contains(allStepsContent, "Validate custom AWF binary") {
			t.Error("Should NOT contain 'Validate custom AWF binary' step when no custom path")
		}
	})
}

// TestCustomPathExecutionSteps tests that GetExecutionSteps uses the correct AWF binary path
func TestCustomPathExecutionSteps(t *testing.T) {
	t.Run("custom path uses resolved path in execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Path:    "/custom/awf/binary",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Should use the custom binary path
		if !strings.Contains(stepContent, "/custom/awf/binary") {
			t.Error("Expected command to use custom AWF binary path '/custom/awf/binary'")
		}
	})

	t.Run("relative path is resolved against GITHUB_WORKSPACE in execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Path:    "bin/awf",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Should use the resolved relative path
		if !strings.Contains(stepContent, "${GITHUB_WORKSPACE}/bin/awf") {
			t.Error("Expected command to use resolved path '${GITHUB_WORKSPACE}/bin/awf'")
		}
	})

	t.Run("no custom path uses default awf binary", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Should use default 'awf' command (from PATH installed in installation step)
		// The command is shell-escaped so it appears as 'awf' in the output
		if !strings.Contains(stepContent, "awf --env-all") {
			t.Error("Expected command to use default 'awf' binary from PATH")
		}
	})
}

// TestFirewallConfigPathExtraction tests that the path field is correctly extracted from frontmatter
func TestFirewallConfigPathExtraction(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name         string
		firewallObj  any
		expectedPath string
	}{
		{
			name: "path is extracted from object",
			firewallObj: map[string]any{
				"path": "/custom/path/to/awf",
			},
			expectedPath: "/custom/path/to/awf",
		},
		{
			name: "relative path is extracted",
			firewallObj: map[string]any{
				"path": "bin/awf",
			},
			expectedPath: "bin/awf",
		},
		{
			name: "path with other fields",
			firewallObj: map[string]any{
				"path":      "/custom/awf",
				"log-level": "debug",
				"version":   "v1.0.0",
			},
			expectedPath: "/custom/awf",
		},
		{
			name:         "no path field",
			firewallObj:  map[string]any{},
			expectedPath: "",
		},
		{
			name:         "boolean firewall (no path)",
			firewallObj:  true,
			expectedPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.extractFirewallConfig(tt.firewallObj)
			if config == nil {
				if tt.expectedPath != "" {
					t.Errorf("Expected config with path %q, got nil config", tt.expectedPath)
				}
				return
			}

			if config.Path != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, config.Path)
			}
		})
	}
}
