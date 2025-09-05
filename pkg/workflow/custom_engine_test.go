package workflow

import (
	"strings"
	"testing"
)

func TestCustomEngine(t *testing.T) {
	engine := NewCustomEngine()

	// Test basic engine properties
	if engine.GetID() != "custom" {
		t.Errorf("Expected ID 'custom', got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "Custom Steps" {
		t.Errorf("Expected display name 'Custom Steps', got '%s'", engine.GetDisplayName())
	}

	if engine.GetDescription() != "Executes user-defined GitHub Actions steps" {
		t.Errorf("Expected description 'Executes user-defined GitHub Actions steps', got '%s'", engine.GetDescription())
	}

	if engine.IsExperimental() {
		t.Error("Expected custom engine to not be experimental")
	}

	if engine.SupportsToolsWhitelist() {
		t.Error("Expected custom engine to not support tools whitelist")
	}

	if engine.SupportsHTTPTransport() {
		t.Error("Expected custom engine to not support HTTP transport")
	}

	if !engine.SupportsMaxTurns() {
		t.Error("Expected custom engine to support max turns for consistency with other engines")
	}
}

func TestCustomEngineGetInstallationSteps(t *testing.T) {
	engine := NewCustomEngine()

	steps := engine.GetInstallationSteps(&WorkflowData{})
	if len(steps) != 0 {
		t.Errorf("Expected 0 installation steps for custom engine, got %d", len(steps))
	}
}

func TestCustomEngineGetExecutionSteps(t *testing.T) {
	engine := NewCustomEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Custom engine without steps should return just the log step
	if len(steps) != 1 {
		t.Errorf("Expected 1 step (log step) when no engine config provided, got %d", len(steps))
	}
}

func TestCustomEngineGetExecutionStepsWithSteps(t *testing.T) {
	engine := NewCustomEngine()

	// Create engine config with steps
	engineConfig := &EngineConfig{
		ID: "custom",
		Steps: []map[string]any{
			{
				"name": "Setup Node.js",
				"uses": "actions/setup-node@v4",
				"with": map[string]any{
					"node-version": "18",
				},
			},
			{
				"name": "Run tests",
				"run":  "npm test",
			},
		},
	}

	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: engineConfig,
	}

	config := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Test with engine config - steps should be populated (2 custom steps + 1 log step)
	if len(config) != 3 {
		t.Errorf("Expected 3 steps when engine config has 2 steps (2 custom + 1 log), got %d", len(config))
	}

	// Check the first step content
	if len(config) > 0 {
		firstStepContent := strings.Join([]string(config[0]), "\n")
		if !strings.Contains(firstStepContent, "name: Setup Node.js") {
			t.Errorf("Expected first step to contain 'name: Setup Node.js', got:\n%s", firstStepContent)
		}
		if !strings.Contains(firstStepContent, "uses: actions/setup-node@v4") {
			t.Errorf("Expected first step to contain 'uses: actions/setup-node@v4', got:\n%s", firstStepContent)
		}
	}

	// Check the second step content
	if len(config) > 1 {
		secondStepContent := strings.Join([]string(config[1]), "\n")
		if !strings.Contains(secondStepContent, "name: Run tests") {
			t.Errorf("Expected second step to contain 'name: Run tests', got:\n%s", secondStepContent)
		}
		if !strings.Contains(secondStepContent, "run:") && !strings.Contains(secondStepContent, "npm test") {
			t.Errorf("Expected second step to contain run command 'npm test', got:\n%s", secondStepContent)
		}
	}
}

func TestCustomEngineRenderMCPConfig(t *testing.T) {
	engine := NewCustomEngine()
	var yaml strings.Builder

	// This should generate MCP configuration structure like Claude
	engine.RenderMCPConfig(&yaml, map[string]any{}, []string{})

	output := yaml.String()
	expectedPrefix := "          cat > /tmp/mcp-config/mcp-servers.json << 'EOF'"
	if !strings.Contains(output, expectedPrefix) {
		t.Errorf("Expected MCP config to contain setup prefix, got '%s'", output)
	}

	if !strings.Contains(output, "\"mcpServers\"") {
		t.Errorf("Expected MCP config to contain mcpServers section, got '%s'", output)
	}
}

func TestCustomEngineParseLogMetrics(t *testing.T) {
	engine := NewCustomEngine()

	logContent := `This is a test log
Error: Something went wrong
Warning: This is a warning
Another line
ERROR: Another error`

	metrics := engine.ParseLogMetrics(logContent, false)

	if metrics.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", metrics.ErrorCount)
	}

	if metrics.WarningCount != 1 {
		t.Errorf("Expected 1 warning, got %d", metrics.WarningCount)
	}

	if metrics.TokenUsage != 0 {
		t.Errorf("Expected 0 token usage, got %d", metrics.TokenUsage)
	}

	if metrics.EstimatedCost != 0 {
		t.Errorf("Expected 0 estimated cost, got %f", metrics.EstimatedCost)
	}
}

func TestCustomEngineGetLogParserScript(t *testing.T) {
	engine := NewCustomEngine()

	script := engine.GetLogParserScript()
	if script != "parse_custom_log" {
		t.Errorf("Expected log parser script 'parse_custom_log', got '%s'", script)
	}
}
