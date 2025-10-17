package workflow

import (
	"fmt"
	"strings"
	"testing"
)

// TestInjectCustomEngineSteps verifies that custom steps are properly injected
func TestInjectCustomEngineSteps(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		expectedSteps  int
		expectedErr    bool
		convertErrStep int // Which step should fail conversion (0 = none)
	}{
		{
			name: "No custom steps",
			workflowData: &WorkflowData{
				EngineConfig: nil,
			},
			expectedSteps: 0,
		},
		{
			name: "Empty custom steps",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{},
				},
			},
			expectedSteps: 0,
		},
		{
			name: "Single custom step",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{
						{
							"name": "Test Step",
							"run":  "echo 'test'",
						},
					},
				},
			},
			expectedSteps: 1,
		},
		{
			name: "Multiple custom steps",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{
						{
							"name": "Step 1",
							"run":  "echo 'step1'",
						},
						{
							"name": "Step 2",
							"run":  "echo 'step2'",
						},
						{
							"name": "Step 3",
							"run":  "echo 'step3'",
						},
					},
				},
			},
			expectedSteps: 3,
		},
		{
			name: "Step conversion error - should continue",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Steps: []map[string]any{
						{
							"name": "Step 1",
							"run":  "echo 'step1'",
						},
						{
							"name": "Step 2 - will fail",
							"run":  "echo 'step2'",
						},
						{
							"name": "Step 3",
							"run":  "echo 'step3'",
						},
					},
				},
			},
			expectedSteps:  2, // Only 2 steps should succeed
			convertErrStep: 2, // Second step fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock convert function
			stepCounter := 0
			convertStepFunc := func(stepMap map[string]any) (string, error) {
				stepCounter++
				// Simulate conversion error for specific step
				if tt.convertErrStep > 0 && stepCounter == tt.convertErrStep {
					return "", fmt.Errorf("conversion error for step %d", stepCounter)
				}
				// Return a simple YAML representation
				name := stepMap["name"]
				return fmt.Sprintf("      - name: %v\n        run: test\n", name), nil
			}

			steps := InjectCustomEngineSteps(tt.workflowData, convertStepFunc)

			if len(steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(steps))
			}

			// Verify each step contains valid YAML
			for i, step := range steps {
				if len(step) == 0 {
					t.Errorf("Step %d is empty", i)
				}
			}
		})
	}
}

// TestHandleCustomMCPToolInSwitch verifies custom MCP tool handling in switch statements
func TestHandleCustomMCPToolInSwitch(t *testing.T) {
	tests := []struct {
		name          string
		toolName      string
		tools         map[string]any
		isLast        bool
		shouldHandle  bool
		renderCalled  bool
		simulateError bool
	}{
		{
			name:     "Valid custom MCP tool",
			toolName: "custom-tool",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"type":    "stdio",
					"command": "node",
					"args":    []string{"server.js"},
				},
			},
			isLast:       false,
			shouldHandle: true,
			renderCalled: true,
		},
		{
			name:     "Valid custom MCP tool - last in list",
			toolName: "custom-tool",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"type":    "http",
					"url":     "https://example.com",
					"headers": map[string]string{"key": "value"},
				},
			},
			isLast:       true,
			shouldHandle: true,
			renderCalled: true,
		},
		{
			name:     "Tool config is not a map",
			toolName: "invalid-tool",
			tools: map[string]any{
				"invalid-tool": "just a string",
			},
			isLast:       false,
			shouldHandle: false,
			renderCalled: false,
		},
		{
			name:     "Tool has no MCP config",
			toolName: "non-mcp-tool",
			tools: map[string]any{
				"non-mcp-tool": map[string]any{
					"some-key": "some-value",
				},
			},
			isLast:       false,
			shouldHandle: false,
			renderCalled: false,
		},
		{
			name:     "Render function returns error",
			toolName: "error-tool",
			tools: map[string]any{
				"error-tool": map[string]any{
					"type":    "stdio",
					"command": "node",
				},
			},
			isLast:        false,
			shouldHandle:  true,
			renderCalled:  true,
			simulateError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			renderCalled := false

			// Create a mock render function
			renderFunc := func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
				renderCalled = true
				if tt.simulateError {
					return fmt.Errorf("simulated render error")
				}
				// Write some output to verify it was called
				yaml.WriteString(fmt.Sprintf("rendered: %s, isLast: %v\n", toolName, isLast))
				return nil
			}

			handled := HandleCustomMCPToolInSwitch(&yaml, tt.toolName, tt.tools, tt.isLast, renderFunc)

			if handled != tt.shouldHandle {
				t.Errorf("Expected handled=%v, got %v", tt.shouldHandle, handled)
			}

			if renderCalled != tt.renderCalled {
				t.Errorf("Expected renderCalled=%v, got %v", tt.renderCalled, renderCalled)
			}

			// If render was called and no error, verify output
			if tt.renderCalled && !tt.simulateError {
				output := yaml.String()
				if !strings.Contains(output, tt.toolName) {
					t.Errorf("Expected output to contain tool name %q, got: %q", tt.toolName, output)
				}
				if !strings.Contains(output, fmt.Sprintf("isLast: %v", tt.isLast)) {
					t.Errorf("Expected output to contain isLast=%v, got: %q", tt.isLast, output)
				}
			}
		})
	}
}

// TestInjectCustomEngineStepsWithRealConversion tests with actual ConvertStepToYAML function
func TestInjectCustomEngineStepsWithRealConversion(t *testing.T) {
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			Steps: []map[string]any{
				{
					"name": "Install dependencies",
					"run":  "npm install",
				},
				{
					"name": "Run tests",
					"run":  "npm test",
				},
			},
		},
	}

	steps := InjectCustomEngineSteps(workflowData, ConvertStepToYAML)

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Verify the YAML content of the first step
	firstStepYAML := steps[0][0]
	if !strings.Contains(firstStepYAML, "Install dependencies") {
		t.Errorf("First step should contain 'Install dependencies', got: %s", firstStepYAML)
	}
	if !strings.Contains(firstStepYAML, "npm install") {
		t.Errorf("First step should contain 'npm install', got: %s", firstStepYAML)
	}

	// Verify the YAML content of the second step
	secondStepYAML := steps[1][0]
	if !strings.Contains(secondStepYAML, "Run tests") {
		t.Errorf("Second step should contain 'Run tests', got: %s", secondStepYAML)
	}
	if !strings.Contains(secondStepYAML, "npm test") {
		t.Errorf("Second step should contain 'npm test', got: %s", secondStepYAML)
	}
}
