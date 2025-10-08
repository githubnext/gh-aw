package workflow

import (
	"testing"
)

func TestDetectRuntimeFromCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected map[RuntimeType]string
	}{
		{
			name:    "npm install command",
			command: "npm install",
			expected: map[RuntimeType]string{
				RuntimeNode: "",
			},
		},
		{
			name:    "npx command",
			command: "npx playwright test",
			expected: map[RuntimeType]string{
				RuntimeNode: "",
			},
		},
		{
			name:    "python command",
			command: "python script.py",
			expected: map[RuntimeType]string{
				RuntimePython: "",
			},
		},
		{
			name:    "pip install",
			command: "pip install package",
			expected: map[RuntimeType]string{
				RuntimePython: "",
			},
		},
		{
			name:    "uv command",
			command: "uv pip install package",
			expected: map[RuntimeType]string{
				RuntimeUV: "",
			},
		},
		{
			name:    "uvx command",
			command: "uvx ruff check",
			expected: map[RuntimeType]string{
				RuntimeUV: "",
			},
		},
		{
			name:    "go command",
			command: "go build",
			expected: map[RuntimeType]string{
				RuntimeGo: "",
			},
		},
		{
			name:    "ruby command",
			command: "ruby script.rb",
			expected: map[RuntimeType]string{
				RuntimeRuby: "",
			},
		},
		{
			name:    "multiple commands",
			command: "npm install && python test.py",
			expected: map[RuntimeType]string{
				RuntimeNode:   "",
				RuntimePython: "",
			},
		},
		{
			name:     "no runtime commands",
			command:  "echo hello",
			expected: map[RuntimeType]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[RuntimeType]string)
			analyzeCommand(tt.command, requirements)

			if len(requirements) != len(tt.expected) {
				t.Errorf("Expected %d requirements, got %d", len(tt.expected), len(requirements))
			}

			for rt, expectedVersion := range tt.expected {
				actualVersion, exists := requirements[rt]
				if !exists {
					t.Errorf("Expected runtime %s to be detected", rt)
				} else if actualVersion != expectedVersion {
					t.Errorf("Expected version '%s' for %s, got '%s'", expectedVersion, rt, actualVersion)
				}
			}
		})
	}
}

func TestDetectFromCustomSteps(t *testing.T) {
	tests := []struct {
		name         string
		customSteps  string
		expected     []RuntimeType
		skipIfHasSetup bool
	}{
		{
			name: "detects npm from run command",
			customSteps: `steps:
  - name: Install deps
    run: npm install`,
			expected: []RuntimeType{RuntimeNode},
		},
		{
			name: "detects python from run command",
			customSteps: `steps:
  - name: Run script
    run: python test.py`,
			expected: []RuntimeType{RuntimePython},
		},
		{
			name: "detects uv from run command",
			customSteps: `steps:
  - name: Install with uv
    run: uv pip install pytest`,
			expected: []RuntimeType{RuntimeUV},
		},
		{
			name: "detects multiple runtimes",
			customSteps: `steps:
  - name: Install
    run: npm install
  - name: Test
    run: python test.py`,
			expected: []RuntimeType{RuntimeNode, RuntimePython},
		},
		{
			name: "skips detection if setup action exists",
			customSteps: `steps:
  - name: Setup Node
    uses: actions/setup-node@v4
    with:
      node-version: '20'
  - name: Install
    run: npm install`,
			expected: []RuntimeType{},
			skipIfHasSetup: true,
		},
		{
			name: "handles multi-line run commands",
			customSteps: `steps:
  - name: Build
    run: |
      npm install
      npm run build`,
			expected: []RuntimeType{RuntimeNode},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[RuntimeType]string)
			detectFromCustomSteps(tt.customSteps, requirements)

			if tt.skipIfHasSetup && len(requirements) != 0 {
				t.Errorf("Expected no requirements when setup action exists, got %v", requirements)
				return
			}

			if len(requirements) != len(tt.expected) {
				t.Errorf("Expected %d requirements, got %d: %v", len(tt.expected), len(requirements), requirements)
			}

			for _, expectedRuntime := range tt.expected {
				if _, exists := requirements[expectedRuntime]; !exists {
					t.Errorf("Expected runtime %s to be detected", expectedRuntime)
				}
			}
		})
	}
}

func TestDetectFromMCPConfigs(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]any
		expected []RuntimeType
	}{
		{
			name: "detects node from MCP command",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"command": "node",
					"args":    []string{"server.js"},
				},
			},
			expected: []RuntimeType{RuntimeNode},
		},
		{
			name: "detects python from MCP command",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"command": "python",
					"args":    []string{"-m", "server"},
				},
			},
			expected: []RuntimeType{RuntimePython},
		},
		{
			name: "detects npx from MCP command",
			tools: map[string]any{
				"playwright": map[string]any{
					"command": "npx",
					"args":    []string{"@playwright/mcp"},
				},
			},
			expected: []RuntimeType{RuntimeNode},
		},
		{
			name: "no detection for non-runtime commands",
			tools: map[string]any{
				"docker-tool": map[string]any{
					"command": "docker",
					"args":    []string{"run"},
				},
			},
			expected: []RuntimeType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[RuntimeType]string)
			detectFromMCPConfigs(tt.tools, requirements)

			if len(requirements) != len(tt.expected) {
				t.Errorf("Expected %d requirements, got %d: %v", len(tt.expected), len(requirements), requirements)
			}

			for _, expectedRuntime := range tt.expected {
				if _, exists := requirements[expectedRuntime]; !exists {
					t.Errorf("Expected runtime %s to be detected", expectedRuntime)
				}
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.2.0", "1.1.0", 1},
		{"1.1.0", "1.2.0", -1},
		{"20", "18", 1},
		{"18", "20", -1},
		{"3.12", "3.11", 1},
		{"3.11.5", "3.11.4", 1},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestGenerateRuntimeSetupSteps(t *testing.T) {
	tests := []struct {
		name         string
		requirements []RuntimeRequirement
		expectSteps  int
		checkContent []string
	}{
		{
			name: "generates node setup",
			requirements: []RuntimeRequirement{
				{Type: RuntimeNode, Version: "20"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Node.js",
				"actions/setup-node@v4",
				"node-version: '20'",
			},
		},
		{
			name: "generates python setup",
			requirements: []RuntimeRequirement{
				{Type: RuntimePython, Version: "3.11"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Python",
				"actions/setup-python@v5",
				"python-version: '3.11'",
			},
		},
		{
			name: "generates uv setup",
			requirements: []RuntimeRequirement{
				{Type: RuntimeUV, Version: ""},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup uv",
				"astral-sh/setup-uv@v5",
			},
		},
		{
			name: "generates multiple setups",
			requirements: []RuntimeRequirement{
				{Type: RuntimeNode, Version: "24"},
				{Type: RuntimePython, Version: "3.12"},
			},
			expectSteps: 2,
			checkContent: []string{
				"Setup Node.js",
				"Setup Python",
			},
		},
		{
			name: "uses default versions",
			requirements: []RuntimeRequirement{
				{Type: RuntimeNode, Version: ""},
			},
			expectSteps: 1,
			checkContent: []string{
				"node-version: '24'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := GenerateRuntimeSetupSteps(tt.requirements)

			if len(steps) != tt.expectSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectSteps, len(steps))
			}

			// Convert steps to string for content checking
			allContent := ""
			for _, step := range steps {
				for _, line := range step {
					allContent += line + "\n"
				}
			}

			for _, expected := range tt.checkContent {
				if !runtimeContainsString(allContent, expected) {
					t.Errorf("Expected to find '%s' in generated steps:\n%s", expected, allContent)
				}
			}
		})
	}
}

func TestShouldSkipRuntimeSetup(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     bool
	}{
		{
			name: "skip when setup-node exists",
			workflowData: &WorkflowData{
				CustomSteps: `steps:
  - uses: actions/setup-node@v4`,
			},
			expected: true,
		},
		{
			name: "skip when setup-python exists",
			workflowData: &WorkflowData{
				CustomSteps: `steps:
  - uses: actions/setup-python@v5`,
			},
			expected: true,
		},
		{
			name: "skip when setup-uv exists",
			workflowData: &WorkflowData{
				CustomSteps: `steps:
  - uses: astral-sh/setup-uv@v5`,
			},
			expected: true,
		},
		{
			name: "don't skip when no setup actions",
			workflowData: &WorkflowData{
				CustomSteps: `steps:
  - run: npm install`,
			},
			expected: false,
		},
		{
			name: "don't skip when empty",
			workflowData: &WorkflowData{
				CustomSteps: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipRuntimeSetup(tt.workflowData)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func runtimeContainsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		runtimeFindSubstring(s, substr)))
}

func runtimeFindSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
