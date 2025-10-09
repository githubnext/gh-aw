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
			name: "skips when setup-node exists",
			customSteps: `steps:
  - uses: actions/setup-node@v4
  - run: npm install`,
			expected:       []string{},
			skipIfHasSetup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := make(map[string]*RuntimeRequirement)
			detectFromCustomSteps(tt.customSteps, requirements)

			if tt.skipIfHasSetup && len(requirements) != 0 {
				t.Errorf("Expected no requirements when setup action exists, got %v", getRequirementIDs(requirements))
				return
			}

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
				"actions/setup-node@v4",
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
				"actions/setup-python@v5",
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
				"astral-sh/setup-uv@v5",
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
				"actions/setup-dotnet@v4",
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
				"actions/setup-java@v4",
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
				"erlef/setup-beam@v1",
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
				"haskell-actions/setup@v2",
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
			name: "skip when setup action exists in custom steps",
			data: &WorkflowData{
				CustomSteps: `steps:
  - uses: actions/setup-node@v4
  - run: npm install`,
			},
			expected: true,
		},
		{
			name: "don't skip when no setup actions",
			data: &WorkflowData{
				CustomSteps: `steps:
  - run: npm install`,
			},
			expected: false,
		},
		{
			name:     "don't skip when no custom steps",
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

func findRuntimeByID(id string) *Runtime {
	for _, runtime := range knownRuntimes {
		if runtime.ID == id {
			return runtime
		}
	}
	return nil
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
