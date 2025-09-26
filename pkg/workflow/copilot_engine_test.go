package workflow

import (
	"strings"
	"testing"
)

func TestCopilotEngine(t *testing.T) {
	engine := NewCopilotEngine()

	// Test basic properties
	if engine.GetID() != "copilot" {
		t.Errorf("Expected copilot engine ID, got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "GitHub Copilot CLI" {
		t.Errorf("Expected 'GitHub Copilot CLI' display name, got '%s'", engine.GetDisplayName())
	}

	if !engine.IsExperimental() {
		t.Error("Expected copilot engine to be experimental")
	}

	if !engine.SupportsToolsAllowlist() {
		t.Error("Expected copilot engine to support tools allowlist")
	}

	if !engine.SupportsHTTPTransport() {
		t.Error("Expected copilot engine to support HTTP transport")
	}

	if engine.SupportsMaxTurns() {
		t.Error("Expected copilot engine to not support max-turns yet")
	}
}

func TestCopilotEngineInstallationSteps(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with no version
	workflowData := &WorkflowData{}
	steps := engine.GetInstallationSteps(workflowData)
	if len(steps) != 3 {
		t.Errorf("Expected 3 installation steps, got %d", len(steps))
	}

	// Test with version
	workflowDataWithVersion := &WorkflowData{
		EngineConfig: &EngineConfig{Version: "1.0.0"},
	}
	stepsWithVersion := engine.GetInstallationSteps(workflowDataWithVersion)
	if len(stepsWithVersion) != 3 {
		t.Errorf("Expected 3 installation steps with version, got %d", len(stepsWithVersion))
	}
}

func TestCopilotEngineExecutionSteps(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step for Copilot CLI execution, got %d", len(steps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	if !strings.Contains(stepContent, "name: Execute GitHub Copilot CLI") {
		t.Errorf("Expected step name 'Execute GitHub Copilot CLI' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "copilot --add-dir /tmp/ --log-level debug --log-dir") {
		t.Errorf("Expected command to contain 'copilot --add-dir /tmp/ --log-level debug --log-dir' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "/tmp/test.log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "GITHUB_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN  }}") {
		t.Errorf("Expected GITHUB_TOKEN environment variable in step content:\n%s", stepContent)
	}

	// Test that GITHUB_AW_SAFE_OUTPUTS is not present when SafeOutputs is nil
	if strings.Contains(stepContent, "GITHUB_AW_SAFE_OUTPUTS") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS to not be present when SafeOutputs is nil")
	}
}

func TestCopilotEngineExecutionStepsWithOutput(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: &SafeOutputsConfig{}, // Non-nil to trigger output handling
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step for Copilot CLI execution with output, got %d", len(steps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Test that GITHUB_AW_SAFE_OUTPUTS is present when SafeOutputs is not nil
	if !strings.Contains(stepContent, "GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}") {
		t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS environment variable when SafeOutputs is not nil in step content:\n%s", stepContent)
	}
}

func TestCopilotEngineGetLogParserScript(t *testing.T) {
	engine := NewCopilotEngine()
	script := engine.GetLogParserScriptId()

	if script != "parse_copilot_log" {
		t.Errorf("Expected 'parse_copilot_log', got '%s'", script)
	}
}

func TestCopilotEngineMCPConfigGeneration(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with GitHub tool (should be skipped since it's built-in)
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"get_issue", "create_issue"},
		},
	}
	mcpTools := []string{"github"}
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

	output := yaml.String()

	// Check that it contains the cat command for JSON creation
	if !strings.Contains(output, "cat > /tmp/.copilot/mcp-config.json << 'EOF'") {
		t.Errorf("Expected cat command for JSON creation in output:\n%s", output)
	}

	// GitHub MCP should NOT be in the output since it's built-in to Copilot CLI
	if strings.Contains(output, "\"GitHub\"") {
		t.Errorf("Expected GitHub server to NOT be in output (it's built-in):\n%s", output)
	}

	// Should have empty mcpServers since GitHub is built-in
	if !strings.Contains(output, "\"mcpServers\": {}") {
		t.Errorf("Expected empty mcpServers object in output:\n%s", output)
	}

	// Check that it ends with EOF
	if !strings.Contains(output, "EOF") {
		t.Errorf("Expected EOF marker in output:\n%s", output)
	}
}

func TestCopilotEngineMCPConfigWithMultipleTools(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with multiple tools including custom MCP tool
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"get_issue"},
		},
		"playwright": map[string]any{
			"allowed_domains": []any{"example.com"},
		},
		"custom-server": map[string]any{
			"type":    "stdio",
			"command": "python",
			"args":    []any{"-m", "my_server"},
			"env": map[string]any{
				"API_KEY": "secret",
			},
			"allowed": []any{"*"},
		},
	}
	mcpTools := []string{"github", "playwright", "custom-server"}
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

	output := yaml.String()

	// GitHub should NOT be in the output since it's built-in
	if strings.Contains(output, "\"GitHub\"") {
		t.Errorf("Expected GitHub server to NOT be in output (it's built-in):\n%s", output)
	}

	if !strings.Contains(output, "\"playwright\"") {
		t.Errorf("Expected playwright server in output:\n%s", output)
	}

	if !strings.Contains(output, "\"custom-server\"") {
		t.Errorf("Expected custom-server in output:\n%s", output)
	}

	// Check custom server configuration - should use "local" instead of "stdio"
	if !strings.Contains(output, "\"type\": \"local\"") {
		t.Errorf("Expected 'local' type for custom server (stdio converted) in output:\n%s", output)
	}

	if !strings.Contains(output, "\"command\": \"python\"") {
		t.Errorf("Expected python command for custom server in output:\n%s", output)
	}

	if !strings.Contains(output, "\"API_KEY\": \"secret\"") {
		t.Errorf("Expected environment variable for custom server in output:\n%s", output)
	}
}

func TestCopilotEnginePlaywrightVersionHandling(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with Playwright tool with custom version
	tools := map[string]any{
		"playwright": map[string]any{
			"docker_image_version": "v1.40.0",
			"allowed_domains":      []any{"example.com"},
		},
	}
	mcpTools := []string{"playwright"}
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

	output := yaml.String()

	// Check that custom version is used
	if !strings.Contains(output, "@playwright/mcp@v1.40.0") {
		t.Errorf("Expected custom Playwright version v1.40.0 in output:\n%s", output)
	}

	// Should not contain the default "latest"
	if strings.Contains(output, "@playwright/mcp@latest") {
		t.Errorf("Expected NOT to find default 'latest' when custom version is specified:\n%s", output)
	}
}
