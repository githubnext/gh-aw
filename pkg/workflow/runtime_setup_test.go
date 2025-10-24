package workflow

import (
	"strings"
	"testing"
)

func TestDetectRuntimeFromCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string // Expected runtime IDs
	}{
		{
			name:     "npm install command",
			command:  "npm install",
			expected: []string{"node"},
		},
		{
			name:     "npx command",
			command:  "npx playwright test",
			expected: []string{"node"},
		},
		{
			name:     "python command",
			command:  "python script.py",
			expected: []string{"python"},
		},
		{
			name:     "pip install",
			command:  "pip install package",
			expected: []string{"python"},
		},
		{
			name:     "uv command",
			command:  "uv pip install package",
			expected: []string{"uv"},
		},
		{
			name:     "uvx command",
			command:  "uvx ruff check",
			expected: []string{"uv"},
		},
		{
			name:     "go command",
			command:  "go build",
			expected: []string{"go"},
		},
		{
			name:     "ruby command",
			command:  "ruby script.rb",
			expected: []string{"ruby"},
		},
		{
			name:     "dotnet command",
			command:  "dotnet build",
			expected: []string{"dotnet"},
		},
		{
			name:     "java command",
			command:  "java -jar app.jar",
			expected: []string{"java"},
		},
		{
			name:     "javac command",
			command:  "javac Main.java",
			expected: []string{"java"},
		},
		{
			name:     "maven command",
			command:  "mvn clean install",
			expected: []string{"java"},
		},
		{
			name:     "gradle command",
			command:  "gradle build",
			expected: []string{"java"},
		},
		{
			name:     "elixir command",
			command:  "elixir script.exs",
			expected: []string{"elixir"},
		},
		{
			name:     "mix command",
			command:  "mix deps.get",
			expected: []string{"elixir"},
		},
		{
			name:     "haskell ghc command",
			command:  "ghc Main.hs",
			expected: []string{"haskell"},
		},
		{
			name:     "cabal command",
			command:  "cabal build",
			expected: []string{"haskell"},
		},
		{
			name:     "stack command",
			command:  "stack build",
			expected: []string{"haskell"},
		},
		{
			name:     "multiple commands",
			command:  "npm install && python test.py",
			expected: []string{"node", "python"},
		},
		{
			name:     "no runtime commands",
			command:  "echo hello",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[string]*RuntimeRequirement)
			detectRuntimeFromCommand(tt.command, requirements)

			if len(requirements) != len(tt.expected) {
				t.Errorf("Expected %d runtime(s), got %d: %v", len(tt.expected), len(requirements), getRequirementIDs(requirements))
			}

			for _, expectedID := range tt.expected {
				if _, exists := requirements[expectedID]; !exists {
					t.Errorf("Expected runtime %s to be detected", expectedID)
				}
			}
		})
	}
}

func TestDetectFromCustomSteps(t *testing.T) {
	tests := []struct {
		name           string
		customSteps    string
		expected       []string
		skipIfHasSetup bool
	}{
		{
			name: "detects node from npm command",
			customSteps: `steps:
  - run: npm install`,
			expected: []string{"node"},
		},
		{
			name: "detects python from python command",
			customSteps: `steps:
  - run: python test.py`,
			expected: []string{"python"},
		},
		{
			name: "detects multiple runtimes",
			customSteps: `steps:
  - run: npm install
  - run: python test.py`,
			expected: []string{"node", "python"},
		},
		{
			name: "detects node even when setup-node exists (filtering happens later)",
			customSteps: `steps:
  - uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020
  - run: npm install`,
			expected: []string{"node"}, // Changed: now detects, filtering happens in DetectRuntimeRequirements
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[string]*RuntimeRequirement)
			detectFromCustomSteps(tt.customSteps, requirements)

			if len(requirements) != len(tt.expected) {
				t.Errorf("Expected %d requirements, got %d: %v", len(tt.expected), len(requirements), getRequirementIDs(requirements))
			}

			for _, expectedID := range tt.expected {
				if _, exists := requirements[expectedID]; !exists {
					t.Errorf("Expected runtime %s to be detected", expectedID)
				}
			}
		})
	}
}

func TestDetectFromMCPConfigs(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]any
		expected []string
	}{
		{
			name: "detects node from MCP command",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"command": "node",
					"args":    []string{"server.js"},
				},
			},
			expected: []string{"node"},
		},
		{
			name: "detects python from MCP command",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"command": "python",
					"args":    []string{"-m", "server"},
				},
			},
			expected: []string{"python"},
		},
		{
			name: "detects npx from MCP command",
			tools: map[string]any{
				"playwright": map[string]any{
					"command": "npx",
					"args":    []string{"@playwright/mcp"},
				},
			},
			expected: []string{"node"},
		},
		{
			name: "no detection for non-runtime commands",
			tools: map[string]any{
				"docker-tool": map[string]any{
					"command": "docker",
					"args":    []string{"run"},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[string]*RuntimeRequirement)
			detectFromMCPConfigs(tt.tools, requirements)

			if len(requirements) != len(tt.expected) {
				t.Errorf("Expected %d requirements, got %d: %v", len(tt.expected), len(requirements), getRequirementIDs(requirements))
			}

			for _, expectedID := range tt.expected {
				if _, exists := requirements[expectedID]; !exists {
					t.Errorf("Expected runtime %s to be detected", expectedID)
				}
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
				{Runtime: findRuntimeByID("node"), Version: "20"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Node.js",
				"actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020",
				"node-version: '20'",
			},
		},
		{
			name: "generates python setup",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("python"), Version: "3.11"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Python",
				"actions/setup-python@a26af69be951a213d495a4c3e4e4022e16d87065",
				"python-version: '3.11'",
			},
		},
		{
			name: "generates uv setup",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("uv"), Version: ""},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup uv",
				"astral-sh/setup-uv@e58605a9b6da7c637471fab8847a5e5a6b8df081",
			},
		},
		{
			name: "generates dotnet setup",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("dotnet"), Version: "8.0"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup .NET",
				"actions/setup-dotnet@67a3573c9a986a3f9c594539f4ab511d57bb3ce9",
				"dotnet-version: '8.0'",
			},
		},
		{
			name: "generates java setup",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("java"), Version: "21"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Java",
				"actions/setup-java@c5195efecf7bdfc987ee8bae7a71cb8b11521c00",
				"java-version: '21'",
				"distribution: temurin",
			},
		},
		{
			name: "generates elixir setup",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("elixir"), Version: "1.17"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Elixir",
				"erlef/setup-beam@3559ac3b631a9560f28817e8e7fdde1638664336",
				"elixir-version: '1.17'",
			},
		},
		{
			name: "generates haskell setup",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("haskell"), Version: "9.10"},
			},
			expectSteps: 1,
			checkContent: []string{
				"Setup Haskell",
				"haskell-actions/setup@d5d0f498b388e1a0eab1cd150202f664c5738e35",
				"ghc-version: '9.10'",
			},
		},
		{
			name: "generates multiple setups",
			requirements: []RuntimeRequirement{
				{Runtime: findRuntimeByID("node"), Version: "24"},
				{Runtime: findRuntimeByID("python"), Version: "3.12"},
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
				{Runtime: findRuntimeByID("node"), Version: ""},
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

			stepsStr := stepsToString(steps)
			for _, content := range tt.checkContent {
				if !strings.Contains(stepsStr, content) {
					t.Errorf("Expected steps to contain '%s', got: %s", content, stepsStr)
				}
			}
		})
	}
}

func TestShouldSkipRuntimeSetup(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected bool
	}{
		{
			name: "never skip - runtime filtering handles existing setup actions",
			data: &WorkflowData{
				CustomSteps: `steps:
  - uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020
  - run: npm install`,
			},
			expected: false, // Changed: we no longer skip, we filter instead
		},
		{
			name: "never skip when no setup actions",
			data: &WorkflowData{
				CustomSteps: `steps:
  - run: npm install`,
			},
			expected: false,
		},
		{
			name:     "never skip when no custom steps",
			data:     &WorkflowData{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipRuntimeSetup(tt.data)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helper functions

func getRequirementIDs(requirements map[string]*RuntimeRequirement) []string {
	var ids []string
	for id := range requirements {
		ids = append(ids, id)
	}
	return ids
}

func stepsToString(steps []GitHubActionStep) string {
	var result string
	for _, step := range steps {
		for _, line := range step {
			result += line + "\n"
		}
	}
	return result
}

func TestUVDetectionAddsPython(t *testing.T) {
	// Test that when uv is detected, python is also added
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"serena": map[string]any{
				"command": "uvx",
				"args":    []any{"--from", "git+https://github.com/oraios/serena", "serena", "start-mcp-server"},
			},
		},
	}

	requirements := DetectRuntimeRequirements(workflowData)

	// Check that both uv and python are detected
	foundUV := false
	foundPython := false
	for _, req := range requirements {
		if req.Runtime.ID == "uv" {
			foundUV = true
		}
		if req.Runtime.ID == "python" {
			foundPython = true
		}
	}

	if !foundUV {
		t.Error("Expected uv to be detected from uvx command")
	}

	if !foundPython {
		t.Error("Expected python to be auto-added when uv is detected")
	}
}

func TestRuntimeFilteringWithExistingSetupActions(t *testing.T) {
	// Test that runtimes with existing setup actions are filtered out
	workflowData := &WorkflowData{
		CustomSteps: `steps:
  - uses: actions/setup-go@d35c59abb061a4a6fb18e82ac0862c26744d6ab5
    with:
      go-version-file: go.mod
  - run: go build
  - run: uv pip install package`,
		Tools: map[string]any{
			"serena": map[string]any{
				"command": "uvx",
			},
		},
	}

	requirements := DetectRuntimeRequirements(workflowData)

	// Check that uv and python are detected, but go is filtered out
	foundUV := false
	foundPython := false
	foundGo := false
	for _, req := range requirements {
		if req.Runtime.ID == "uv" {
			foundUV = true
		}
		if req.Runtime.ID == "python" {
			foundPython = true
		}
		if req.Runtime.ID == "go" {
			foundGo = true
		}
	}

	if !foundUV {
		t.Error("Expected uv to be detected from uvx command and uv pip")
	}

	if !foundPython {
		t.Error("Expected python to be auto-added when uv is detected")
	}

	if foundGo {
		t.Error("Expected go to be filtered out since it has existing setup action")
	}
}
