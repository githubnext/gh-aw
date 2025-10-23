package workflow

import (
	"strings"
	"testing"
)

// Test NewCompilerWithCustomOutput function
func TestNewCompilerWithCustomOutput(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		engineOverride string
		customOutput   string
		version        string
		expectedFields map[string]any
	}{
		{
			name:           "create compiler with basic custom output",
			verbose:        false,
			engineOverride: "",
			customOutput:   "/custom/output/path.yml",
			version:        "1.0.0",
			expectedFields: map[string]any{
				"verbose":        false,
				"engineOverride": "",
				"customOutput":   "/custom/output/path.yml",
				"version":        "1.0.0",
				"skipValidation": true,
			},
		},
		{
			name:           "create compiler with verbose and engine override",
			verbose:        true,
			engineOverride: "codex",
			customOutput:   "/tmp/gh-aw/th-aw/test.yml",
			version:        "v2.1.3",
			expectedFields: map[string]any{
				"verbose":        true,
				"engineOverride": "codex",
				"customOutput":   "/tmp/gh-aw/th-aw/test.yml",
				"version":        "v2.1.3",
				"skipValidation": true,
			},
		},
		{
			name:           "create compiler with empty values",
			verbose:        false,
			engineOverride: "",
			customOutput:   "",
			version:        "",
			expectedFields: map[string]any{
				"verbose":        false,
				"engineOverride": "",
				"customOutput":   "",
				"version":        "",
				"skipValidation": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompilerWithCustomOutput(tt.verbose, tt.engineOverride, tt.customOutput, tt.version)

			// Check that compiler is not nil
			if compiler == nil {
				t.Errorf("NewCompilerWithCustomOutput() returned nil")
				return
			}

			// Check verbose setting
			if compiler.verbose != tt.expectedFields["verbose"] {
				t.Errorf("NewCompilerWithCustomOutput() verbose = %v, expected %v",
					compiler.verbose, tt.expectedFields["verbose"])
			}

			// Check engineOverride setting
			if compiler.engineOverride != tt.expectedFields["engineOverride"] {
				t.Errorf("NewCompilerWithCustomOutput() engineOverride = %v, expected %v",
					compiler.engineOverride, tt.expectedFields["engineOverride"])
			}

			// Check customOutput setting
			if compiler.customOutput != tt.expectedFields["customOutput"] {
				t.Errorf("NewCompilerWithCustomOutput() customOutput = %v, expected %v",
					compiler.customOutput, tt.expectedFields["customOutput"])
			}

			// Check version setting
			if compiler.version != tt.expectedFields["version"] {
				t.Errorf("NewCompilerWithCustomOutput() version = %v, expected %v",
					compiler.version, tt.expectedFields["version"])
			}

			// Check skipValidation is properly set to true
			if compiler.skipValidation != tt.expectedFields["skipValidation"] {
				t.Errorf("NewCompilerWithCustomOutput() skipValidation = %v, expected %v",
					compiler.skipValidation, tt.expectedFields["skipValidation"])
			}

			// Check that jobManager is initialized
			if compiler.jobManager == nil {
				t.Errorf("NewCompilerWithCustomOutput() jobManager should not be nil")
			}
		})
	}
}

// Test convertStepToYAML function
func TestConvertStepToYAML(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")

	tests := []struct {
		name     string
		stepMap  map[string]any
		expected string
		hasError bool
	}{
		{
			name: "step with name only",
			stepMap: map[string]any{
				"name": "Test Step",
			},
			expected: "      - name: Test Step\n",
			hasError: false,
		},
		{
			name: "step with name and uses",
			stepMap: map[string]any{
				"name": "Checkout Code",
				"uses": "actions/checkout@v5",
			},
			expected: "      - name: Checkout Code\n        uses: actions/checkout@v5\n",
			hasError: false,
		},
		{
			name: "step with name and run command",
			stepMap: map[string]any{
				"name": "Run Tests",
				"run":  "go test ./...",
			},
			expected: "      - name: Run Tests\n        run: go test ./...\n",
			hasError: false,
		},
		{
			name: "step with name, run command and env variables",
			stepMap: map[string]any{
				"name": "Build Project",
				"run":  "make build",
				"env": map[string]string{
					"GO_VERSION": "1.21",
					"ENV":        "test",
				},
			},
			expected: "      - name: Build Project\n        run: make build\n        env:\n          ENV: test\n          GO_VERSION: \"1.21\"\n",
			hasError: false,
		},
		{
			name: "step with working-directory",
			stepMap: map[string]any{
				"name":              "Test in Subdirectory",
				"run":               "npm test",
				"working-directory": "./frontend",
			},
			expected: "      - name: Test in Subdirectory\n        run: npm test\n        working-directory: ./frontend\n",
			hasError: false,
		},
		{
			name: "step with complex with parameters",
			stepMap: map[string]any{
				"name": "Setup Node",
				"uses": "actions/setup-node@v4",
				"with": map[string]any{
					"node-version": "18",
					"cache":        "npm",
				},
			},
			expected: "      - name: Setup Node\n        uses: actions/setup-node@v4\n        with:\n          cache: npm\n          node-version: \"18\"\n",
			hasError: false,
		},
		{
			name:     "empty step map",
			stepMap:  map[string]any{},
			expected: "",
			hasError: false,
		},
		{
			name: "step without name",
			stepMap: map[string]any{
				"run": "echo 'no name'",
			},
			expected: "        run: echo 'no name'\n",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compiler.convertStepToYAML(tt.stepMap)

			if tt.hasError {
				if err == nil {
					t.Errorf("convertStepToYAML() expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("convertStepToYAML() unexpected error: %v", err)
				}

				if !strings.Contains(result, strings.TrimSpace(strings.Split(tt.expected, "\n")[0])) {
					t.Errorf("convertStepToYAML() result doesn't contain expected content\nGot: %q\nExpected to contain: %q",
						result, tt.expected)
				}
			}
		})
	}
}

func TestGenerateAgentVersionCapture(t *testing.T) {
	tests := []struct {
		name           string
		engineID       string
		expectedCmd    string
		expectEmptyVar bool
	}{
		{
			name:           "Claude engine",
			engineID:       "claude",
			expectedCmd:    "claude --version",
			expectEmptyVar: false,
		},
		{
			name:           "Copilot engine",
			engineID:       "copilot",
			expectedCmd:    "copilot --version",
			expectEmptyVar: false,
		},
		{
			name:           "Codex engine",
			engineID:       "codex",
			expectedCmd:    "codex --version",
			expectEmptyVar: false,
		},
		{
			name:           "Custom engine (no version command)",
			engineID:       "custom",
			expectedCmd:    "",
			expectEmptyVar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test-version")
			registry := GetGlobalEngineRegistry()
			engine, err := registry.GetEngine(tt.engineID)
			if err != nil {
				t.Fatalf("Failed to get %s engine: %v", tt.engineID, err)
			}

			// Create minimal WorkflowData for the test
			workflowData := &WorkflowData{
				NetworkPermissions: nil, // No firewall enabled
			}

			var yaml strings.Builder
			compiler.generateAgentVersionCapture(&yaml, engine, workflowData)
			output := yaml.String()

			if tt.expectEmptyVar {
				// Should set empty AGENT_VERSION
				if !strings.Contains(output, "AGENT_VERSION=") {
					t.Errorf("Expected output to set AGENT_VERSION for custom engine")
				}
				if !strings.Contains(output, "not available") {
					t.Errorf("Expected output to mention 'not available' for custom engine")
				}
			} else {
				// Should capture version from command
				if !strings.Contains(output, "Capture agent version") {
					t.Errorf("Expected output to contain 'Capture agent version'")
				}
				if !strings.Contains(output, tt.expectedCmd) {
					t.Errorf("Expected output to contain command '%s', got:\n%s", tt.expectedCmd, output)
				}
				if !strings.Contains(output, "AGENT_VERSION=") {
					t.Errorf("Expected output to set AGENT_VERSION environment variable")
				}
			}
		})
	}
}
