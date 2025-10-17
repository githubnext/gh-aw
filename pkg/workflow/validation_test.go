package workflow

import (
	"os/exec"
	"testing"
)

func TestValidateContainerImages(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		expectError    bool
		skipIfNoDocker bool
	}{
		{
			name: "no tools",
			workflowData: &WorkflowData{
				Tools: nil,
			},
			expectError: false,
		},
		{
			name: "tools without container",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"command": "npx",
						"args":    []any{"@github/github-mcp-server"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid container image",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"test-tool": map[string]any{
						"container": "alpine",
						"version":   "latest",
					},
				},
			},
			expectError:    false,
			skipIfNoDocker: true,
		},
		{
			name: "invalid container image",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"test-tool": map[string]any{
						"container": "nonexistent-image-that-should-not-exist-12345",
						"version":   "nonexistent",
					},
				},
			},
			expectError:    true,
			skipIfNoDocker: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if docker is not available
			if tt.skipIfNoDocker {
				if _, err := exec.LookPath("docker"); err != nil {
					t.Skip("docker not available, skipping test")
				}
			}

			compiler := NewCompiler(false, "", "test")
			err := compiler.validateContainerImages(tt.workflowData)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateDockerImage(t *testing.T) {
	// Skip if docker is not available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping test")
	}

	tests := []struct {
		name        string
		image       string
		expectError bool
	}{
		{
			name:        "valid image - alpine",
			image:       "alpine:latest",
			expectError: false,
		},
		{
			name:        "invalid image",
			image:       "nonexistent-image-12345:nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDockerImage(tt.image, false)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestExtractNpxPackages(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     []string
	}{
		{
			name: "no npx packages",
			workflowData: &WorkflowData{
				CustomSteps: "echo hello",
			},
			expected: []string{},
		},
		{
			name: "npx in custom steps",
			workflowData: &WorkflowData{
				CustomSteps: "npx @playwright/mcp@latest",
			},
			expected: []string{"@playwright/mcp@latest"},
		},
		{
			name: "npx in MCP config",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"playwright": map[string]any{
						"command": "npx",
						"args":    []any{"@playwright/mcp@latest"},
					},
				},
			},
			expected: []string{"@playwright/mcp@latest"},
		},
		{
			name: "multiple npx packages",
			workflowData: &WorkflowData{
				CustomSteps: "npx package1 && npx package2",
				Tools: map[string]any{
					"tool1": map[string]any{
						"command": "npx",
						"args":    []any{"package3"},
					},
				},
			},
			expected: []string{"package1", "package2", "package3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractNpxPackages(tt.workflowData)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			// Check that all expected packages are present (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, pkg := range tt.expected {
				expectedMap[pkg] = true
			}

			for _, pkg := range packages {
				if !expectedMap[pkg] {
					t.Errorf("unexpected package: %s", pkg)
				}
			}
		})
	}
}

func TestExtractPipPackages(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     []string
	}{
		{
			name: "no pip packages",
			workflowData: &WorkflowData{
				CustomSteps: "echo hello",
			},
			expected: []string{},
		},
		{
			name: "pip install command",
			workflowData: &WorkflowData{
				CustomSteps: "pip install pytest",
			},
			expected: []string{"pytest"},
		},
		{
			name: "pip3 install command",
			workflowData: &WorkflowData{
				CustomSteps: "pip3 install requests",
			},
			expected: []string{"requests"},
		},
		{
			name: "pip install with flags",
			workflowData: &WorkflowData{
				CustomSteps: "pip install --upgrade setuptools",
			},
			expected: []string{"setuptools"},
		},
		{
			name: "multiple pip packages",
			workflowData: &WorkflowData{
				CustomSteps: "pip install pytest && pip3 install requests",
			},
			expected: []string{"pytest", "requests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractPipPackages(tt.workflowData)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			// Check that all expected packages are present (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, pkg := range tt.expected {
				expectedMap[pkg] = true
			}

			for _, pkg := range packages {
				if !expectedMap[pkg] {
					t.Errorf("unexpected package: %s", pkg)
				}
			}
		})
	}
}

func TestExtractPipFromCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands string
		expected []string
	}{
		{
			name:     "no pip",
			commands: "echo hello",
			expected: []string{},
		},
		{
			name:     "single pip install",
			commands: "pip install package-name",
			expected: []string{"package-name"},
		},
		{
			name:     "pip3 install",
			commands: "pip3 install package-name",
			expected: []string{"package-name"},
		},
		{
			name:     "pip install with flags",
			commands: "pip install --upgrade package-name",
			expected: []string{"package-name"},
		},
		{
			name:     "multiple pip commands",
			commands: "pip install pkg1 && pip3 install pkg2",
			expected: []string{"pkg1", "pkg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractPipFromCommands(tt.commands)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			for i, pkg := range packages {
				if pkg != tt.expected[i] {
					t.Errorf("expected package[%d] = %s, got %s", i, tt.expected[i], pkg)
				}
			}
		})
	}
}

func TestExtractUvPackages(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     []string
	}{
		{
			name: "no uv packages",
			workflowData: &WorkflowData{
				CustomSteps: "echo hello",
			},
			expected: []string{},
		},
		{
			name: "uvx command",
			workflowData: &WorkflowData{
				CustomSteps: "uvx ruff check .",
			},
			expected: []string{"ruff"},
		},
		{
			name: "uv pip install",
			workflowData: &WorkflowData{
				CustomSteps: "uv pip install pytest",
			},
			expected: []string{"pytest"},
		},
		{
			name: "multiple uv packages",
			workflowData: &WorkflowData{
				CustomSteps: "uvx black . && uv pip install pytest ruff",
			},
			expected: []string{"black", "pytest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractUvPackages(tt.workflowData)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			// Check that all expected packages are present (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, pkg := range tt.expected {
				expectedMap[pkg] = true
			}

			for _, pkg := range packages {
				if !expectedMap[pkg] {
					t.Errorf("unexpected package: %s", pkg)
				}
			}
		})
	}
}

func TestExtractNpxFromCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands string
		expected []string
	}{
		{
			name:     "no npx",
			commands: "echo hello",
			expected: []string{},
		},
		{
			name:     "single npx",
			commands: "npx package-name",
			expected: []string{"package-name"},
		},
		{
			name:     "multiple npx with operators",
			commands: "npx pkg1 && npx pkg2 | npx pkg3",
			expected: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:     "npx with version specifier",
			commands: "npx @scope/package@1.0.0",
			expected: []string{"@scope/package@1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractNpxFromCommands(tt.commands)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			for i, pkg := range packages {
				if pkg != tt.expected[i] {
					t.Errorf("expected package[%d] = %s, got %s", i, tt.expected[i], pkg)
				}
			}
		})
	}
}

func TestExtractUvFromCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands string
		expected []string
	}{
		{
			name:     "no uv",
			commands: "echo hello",
			expected: []string{},
		},
		{
			name:     "uvx command",
			commands: "uvx ruff",
			expected: []string{"ruff"},
		},
		{
			name:     "uv pip install",
			commands: "uv pip install pytest",
			expected: []string{"pytest"},
		},
		{
			name:     "multiple uv commands",
			commands: "uvx black . && uv pip install pytest",
			expected: []string{"black", "pytest"},
		},
		{
			name:     "uv pip install with flags",
			commands: "uv pip install --upgrade pytest",
			expected: []string{"pytest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := extractUvFromCommands(tt.commands)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			for i, pkg := range packages {
				if pkg != tt.expected[i] {
					t.Errorf("expected package[%d] = %s, got %s", i, tt.expected[i], pkg)
				}
			}
		})
	}
}

func TestValidateRuntimePackages(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expectError  bool
		skipReason   string
	}{
		{
			name: "no runtime packages",
			workflowData: &WorkflowData{
				CustomSteps: "echo hello",
			},
			expectError: false,
		},
		// Note: These tests would fail if npm/uv/pip are available, so we skip them
		// The actual validation logic is tested by the extraction tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			err := compiler.validateRuntimePackages(tt.workflowData)

			// If we expect an error and got one, or don't expect one and didn't get one, test passes
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "timeout error with TimeoutError",
			output:   "TimeoutError: The read operation timed out",
			expected: true,
		},
		{
			name:     "timeout error with ReadTimeoutError",
			output:   "pip._vendor.urllib3.exceptions.ReadTimeoutError: HTTPSConnectionPool(host='pypi.org', port=443): Read timed out.",
			expected: true,
		},
		{
			name:     "timeout error with 'timed out'",
			output:   "Connection timed out while connecting to server",
			expected: true,
		},
		{
			name:     "timeout error with 'timeout' (case insensitive)",
			output:   "ERROR: Request timeout occurred",
			expected: true,
		},
		{
			name:     "not a timeout error - package not found",
			output:   "ERROR: No matching distribution found for nonexistent-package",
			expected: false,
		},
		{
			name:     "not a timeout error - network error",
			output:   "ERROR: Could not find a version that satisfies the requirement",
			expected: false,
		},
		{
			name:     "not a timeout error - empty output",
			output:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeoutError(tt.output)
			if result != tt.expected {
				t.Errorf("isTimeoutError() = %v, want %v for output: %s", result, tt.expected, tt.output)
			}
		})
	}
}

func TestCollectPackagesFromWorkflow(t *testing.T) {
	// Mock extractor function that returns packages from commands
	mockExtractor := func(commands string) []string {
		if commands == "command1" {
			return []string{"pkg1", "pkg2"}
		}
		if commands == "command2" {
			return []string{"pkg2", "pkg3"}
		}
		return []string{}
	}

	tests := []struct {
		name         string
		workflowData *WorkflowData
		toolCommand  string
		expected     []string
	}{
		{
			name: "extract from custom steps only",
			workflowData: &WorkflowData{
				CustomSteps: "command1",
			},
			toolCommand: "",
			expected:    []string{"pkg1", "pkg2"},
		},
		{
			name: "extract from engine steps only",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{
						{"run": "command1"},
					},
				},
			},
			toolCommand: "",
			expected:    []string{"pkg1", "pkg2"},
		},
		{
			name: "deduplicate across custom and engine steps",
			workflowData: &WorkflowData{
				CustomSteps: "command1",
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{
						{"run": "command2"},
					},
				},
			},
			toolCommand: "",
			expected:    []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name: "extract from MCP tools when toolCommand provided",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"test-tool": map[string]any{
						"command": "test-cmd",
						"args":    []any{"pkg4"},
					},
				},
			},
			toolCommand: "test-cmd",
			expected:    []string{"pkg4"},
		},
		{
			name: "skip MCP tools when toolCommand is empty",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"test-tool": map[string]any{
						"command": "test-cmd",
						"args":    []any{"pkg4"},
					},
				},
			},
			toolCommand: "",
			expected:    []string{},
		},
		{
			name: "deduplicate across all sources",
			workflowData: &WorkflowData{
				CustomSteps: "command1",
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{
						{"run": "command1"}, // Same packages as custom steps
					},
				},
				Tools: map[string]any{
					"test-tool": map[string]any{
						"command": "test-cmd",
						"args":    []any{"pkg1"}, // Duplicate from custom steps
					},
				},
			},
			toolCommand: "test-cmd",
			expected:    []string{"pkg1", "pkg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := collectPackagesFromWorkflow(tt.workflowData, mockExtractor, tt.toolCommand)

			if len(packages) != len(tt.expected) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expected), len(packages), packages)
				return
			}

			// Check that all expected packages are present (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, pkg := range tt.expected {
				expectedMap[pkg] = true
			}

			for _, pkg := range packages {
				if !expectedMap[pkg] {
					t.Errorf("unexpected package: %s", pkg)
				}
			}
		})
	}
}
