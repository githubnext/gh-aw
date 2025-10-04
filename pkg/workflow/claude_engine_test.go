package workflow

import (
	"strings"
	"testing"
)

func TestClaudeEngine(t *testing.T) {
	engine := NewClaudeEngine()

	// Test basic properties
	if engine.GetID() != "claude" {
		t.Errorf("Expected ID 'claude', got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "Claude Code" {
		t.Errorf("Expected display name 'Claude Code', got '%s'", engine.GetDisplayName())
	}

	if engine.GetDescription() != "Uses Claude Code with full MCP tool support and allow-listing" {
		t.Errorf("Expected description 'Uses Claude Code with full MCP tool support and allow-listing', got '%s'", engine.GetDescription())
	}

	if engine.IsExperimental() {
		t.Error("Claude engine should not be experimental")
	}

	if !engine.SupportsToolsAllowlist() {
		t.Error("Claude engine should support MCP tools")
	}

	// Test installation steps (should have 3 steps: Node.js setup, cache, install)
	installSteps := engine.GetInstallationSteps(&WorkflowData{})
	if len(installSteps) != 3 {
		t.Errorf("Expected 3 installation steps for Claude (Node.js setup, cache, install), got %d", len(installSteps))
	}

	// Check for Node.js setup step
	nodeSetupStep := strings.Join([]string(installSteps[0]), "\n")
	if !strings.Contains(nodeSetupStep, "Setup Node.js") {
		t.Errorf("Expected 'Setup Node.js' in first installation step, got: %s", nodeSetupStep)
	}
	if !strings.Contains(nodeSetupStep, "actions/setup-node@v4") {
		t.Errorf("Expected 'actions/setup-node@v4' in first installation step, got: %s", nodeSetupStep)
	}

	// Check for cache step
	cacheStep := strings.Join([]string(installSteps[1]), "\n")
	if !strings.Contains(cacheStep, "Cache npm global packages") {
		t.Errorf("Expected 'Cache npm global packages' in second installation step, got: %s", cacheStep)
	}
	if !strings.Contains(cacheStep, "actions/cache@v4") {
		t.Errorf("Expected 'actions/cache@v4' in second installation step, got: %s", cacheStep)
	}
	if !strings.Contains(cacheStep, "/usr/local/lib/node_modules") {
		t.Errorf("Expected npm global modules path in cache step, got: %s", cacheStep)
	}

	// Check for install step
	installStep := strings.Join([]string(installSteps[2]), "\n")
	if !strings.Contains(installStep, "Install Claude Code CLI") {
		t.Errorf("Expected 'Install Claude Code CLI' in third installation step, got: %s", installStep)
	}
	if !strings.Contains(installStep, "npm install -g @anthropic-ai/claude-code@2.0.1") {
		t.Errorf("Expected 'npm install -g @anthropic-ai/claude-code@2.0.1' in install step, got: %s", installStep)
	}

	// Test execution steps
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepLines := []string(executionStep)

	// Check step name
	found := false
	for _, line := range stepLines {
		if strings.Contains(line, "name: Execute Claude Code CLI") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected step name 'Execute Claude Code CLI' in step lines: %v", stepLines)
	}

	// Check claude usage with direct command instead of npx
	found = false
	for _, line := range stepLines {
		if strings.Contains(line, "claude --print") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected claude command in step lines: %v", stepLines)
	}

	// Check that required CLI arguments are present
	stepContent := strings.Join(stepLines, "\n")
	if !strings.Contains(stepContent, "--print") {
		t.Errorf("Expected --print flag in step: %s", stepContent)
	}

	if !strings.Contains(stepContent, "--permission-mode bypassPermissions") {
		t.Errorf("Expected --permission-mode bypassPermissions in CLI args: %s", stepContent)
	}

	if !strings.Contains(stepContent, "--output-format stream-json") {
		t.Errorf("Expected --output-format stream-json in CLI args: %s", stepContent)
	}

	if !strings.Contains(stepContent, "ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}") {
		t.Errorf("Expected ANTHROPIC_API_KEY environment variable in step: %s", stepContent)
	}

	if !strings.Contains(stepContent, "GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt") {
		t.Errorf("Expected GITHUB_AW_PROMPT environment variable in step: %s", stepContent)
	}

	// When no tools/MCP servers are configured, GITHUB_AW_MCP_CONFIG should NOT be present
	if strings.Contains(stepContent, "GITHUB_AW_MCP_CONFIG: /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Did not expect GITHUB_AW_MCP_CONFIG environment variable in step (no MCP servers): %s", stepContent)
	}

	if !strings.Contains(stepContent, "MCP_TIMEOUT: \"60000\"") {
		t.Errorf("Expected MCP_TIMEOUT environment variable in step: %s", stepContent)
	}

	// When no tools/MCP servers are configured, --mcp-config flag should NOT be present
	if strings.Contains(stepContent, "--mcp-config /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Did not expect MCP config in CLI args (no MCP servers): %s", stepContent)
	}

	if !strings.Contains(stepContent, "--allowed-tools") {
		t.Errorf("Expected allowed-tools in CLI args: %s", stepContent)
	}

	// timeout should now be at step level, not input level
	if !strings.Contains(stepContent, "timeout-minutes:") {
		t.Errorf("Expected timeout-minutes at step level: %s", stepContent)
	}
}

func TestClaudeEngineWithOutput(t *testing.T) {
	engine := NewClaudeEngine()

	// Test execution steps with hasOutput=true
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: &SafeOutputsConfig{}, // non-nil means hasOutput=true
	}
	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Should include GITHUB_AW_SAFE_OUTPUTS when hasOutput=true in environment section
	if !strings.Contains(stepContent, "GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}") {
		t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS in env section when hasOutput=true in step content:\n%s", stepContent)
	}
}

func TestClaudeEngineConfiguration(t *testing.T) {
	engine := NewClaudeEngine()

	// Test different workflow names and log files
	testCases := []struct {
		workflowName string
		logFile      string
	}{
		{"simple-workflow", "simple-log"},
		{"complex workflow with spaces", "complex-log"},
		{"workflow-with-hyphens", "workflow-log"},
	}

	for _, tc := range testCases {
		t.Run(tc.workflowName, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name: tc.workflowName,
			}
			steps := engine.GetExecutionSteps(workflowData, tc.logFile)
			if len(steps) != 2 {
				t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
			}

			// Check the main execution step
			executionStep := steps[0]
			stepContent := strings.Join([]string(executionStep), "\n")

			// Verify the step contains expected content regardless of input
			if !strings.Contains(stepContent, "name: Execute Claude Code CLI") {
				t.Errorf("Expected step name 'Execute Claude Code CLI' in step content")
			}

			if !strings.Contains(stepContent, "claude --print") {
				t.Errorf("Expected claude command in step content")
			}

			// Verify all required CLI elements are present
			requiredElements := []string{"--print", "ANTHROPIC_API_KEY", "--permission-mode", "--output-format"}
			for _, element := range requiredElements {
				if !strings.Contains(stepContent, element) {
					t.Errorf("Expected element '%s' to be present in step content", element)
				}
			}

			// When no tools/MCP servers are configured, --mcp-config should NOT be present
			if strings.Contains(stepContent, "--mcp-config") {
				t.Errorf("Did not expect --mcp-config in step content (no MCP servers)")
			}

			// timeout should be at step level, not input level
			if !strings.Contains(stepContent, "timeout-minutes:") {
				t.Errorf("Expected timeout-minutes at step level")
			}
		})
	}
}

func TestClaudeEngineWithVersion(t *testing.T) {
	engine := NewClaudeEngine()

	// Test with custom version
	engineConfig := &EngineConfig{
		ID:      "claude",
		Version: "v1.2.3",
		Model:   "claude-3-5-sonnet-20241022",
	}

	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: engineConfig,
	}

	// Check installation steps for custom version
	installSteps := engine.GetInstallationSteps(workflowData)
	if len(installSteps) != 3 {
		t.Fatalf("Expected 3 installation steps, got %d", len(installSteps))
	}

	// Check that install step uses the custom version
	installStep := strings.Join([]string(installSteps[2]), "\n")
	if !strings.Contains(installStep, "npm install -g @anthropic-ai/claude-code@v1.2.3") {
		t.Errorf("Expected npm install with custom version v1.2.3 in install step:\n%s", installStep)
	}

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Check that claude command is used directly (not npx)
	if !strings.Contains(stepContent, "claude --print") {
		t.Errorf("Expected claude command in step content:\n%s", stepContent)
	}

	// Check that model is set in CLI args
	if !strings.Contains(stepContent, "--model claude-3-5-sonnet-20241022") {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022' in CLI args:\n%s", stepContent)
	}
}

func TestClaudeEngineWithoutVersion(t *testing.T) {
	engine := NewClaudeEngine()

	// Test without version (should use default)
	engineConfig := &EngineConfig{
		ID:    "claude",
		Model: "claude-3-5-sonnet-20241022",
	}

	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: engineConfig,
	}

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Check that claude command is used directly (not npx) with default version
	if !strings.Contains(stepContent, "claude --print") {
		t.Errorf("Expected claude command in step content:\n%s", stepContent)
	}
}

func TestClaudeEngineWithNilConfig(t *testing.T) {
	engine := NewClaudeEngine()

	// Test with nil engine config (should use default latest)
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: nil,
	}

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Check that claude command is used directly (not npx) when no engine config
	if !strings.Contains(stepContent, "claude --print") {
		t.Errorf("Expected claude command when no engine config in step content:\n%s", stepContent)
	}
}

func TestClaudeEngineConvertStepToYAMLWithSection(t *testing.T) {
	engine := NewClaudeEngine()

	// Test step with 'with' section to ensure keys are sorted
	stepMap := map[string]any{
		"name": "Test step with sorted with section",
		"uses": "actions/checkout@v4",
		"with": map[string]any{
			"zebra": "value-z",
			"alpha": "value-a",
			"beta":  "value-b",
			"gamma": "value-g",
		},
	}

	yaml, err := engine.convertStepToYAML(stepMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that the with keys are in alphabetical order
	lines := strings.Split(yaml, "\n")
	withSection := false
	withKeyOrder := []string{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "with:" {
			withSection = true
			continue
		}
		if withSection && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			// End of with section if we hit another top-level key
			break
		}
		if withSection && strings.Contains(line, ":") {
			// Extract the key (before the colon)
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) > 0 {
				withKeyOrder = append(withKeyOrder, strings.TrimSpace(parts[0]))
			}
		}
	}

	expectedOrder := []string{"alpha", "beta", "gamma", "zebra"}
	if len(withKeyOrder) != len(expectedOrder) {
		t.Errorf("Expected %d with keys, got %d", len(expectedOrder), len(withKeyOrder))
	}

	for i, key := range expectedOrder {
		if i >= len(withKeyOrder) || withKeyOrder[i] != key {
			t.Errorf("Expected with key at position %d to be '%s', got '%s'. Full order: %v", i, key, withKeyOrder[i], withKeyOrder)
		}
	}
}

func TestClaudeEngineWithMCPServers(t *testing.T) {
	engine := NewClaudeEngine()

	// Test with GitHub MCP tool configured
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{},
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// When MCP servers are configured, --mcp-config flag SHOULD be present
	if !strings.Contains(stepContent, "--mcp-config /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Expected --mcp-config in CLI args when MCP servers are configured: %s", stepContent)
	}

	// When MCP servers are configured, GITHUB_AW_MCP_CONFIG SHOULD be present
	if !strings.Contains(stepContent, "GITHUB_AW_MCP_CONFIG: /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Expected GITHUB_AW_MCP_CONFIG environment variable when MCP servers are configured: %s", stepContent)
	}
}

func TestClaudeEngineWithSafeOutputs(t *testing.T) {
	engine := NewClaudeEngine()

	// Test with safe-outputs configured (which adds safe-outputs MCP server)
	workflowData := &WorkflowData{
		Name:  "test-workflow",
		Tools: map[string]any{},
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
			},
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// When safe-outputs is configured, --mcp-config flag SHOULD be present
	if !strings.Contains(stepContent, "--mcp-config /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Expected --mcp-config in CLI args when safe-outputs are configured: %s", stepContent)
	}

	// When safe-outputs is configured, GITHUB_AW_MCP_CONFIG SHOULD be present
	if !strings.Contains(stepContent, "GITHUB_AW_MCP_CONFIG: /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Expected GITHUB_AW_MCP_CONFIG environment variable when safe-outputs are configured: %s", stepContent)
	}
}
