package workflow

import (
	"strings"
	"testing"
)

// TestEngineConfigAgentFieldExtraction tests that the agent field is correctly extracted from frontmatter
func TestEngineConfigAgentFieldExtraction(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id":    "copilot",
			"agent": "/path/to/agent.md",
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "copilot" {
		t.Errorf("Expected engine ID 'copilot', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected non-nil engine config")
	}

	if config.Agent != "/path/to/agent.md" {
		t.Errorf("Expected agent path '/path/to/agent.md', got '%s'", config.Agent)
	}
}

// TestEngineConfigAgentFieldEmpty tests that empty agent field is handled correctly
func TestEngineConfigAgentFieldEmpty(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "claude",
		},
	}

	_, config := compiler.ExtractEngineConfig(frontmatter)

	if config == nil {
		t.Fatal("Expected non-nil engine config")
	}

	if config.Agent != "" {
		t.Errorf("Expected empty agent path, got '%s'", config.Agent)
	}
}

// TestCopilotEngineWithAgentFlag tests that copilot engine includes --agent flag when agent is specified
func TestCopilotEngineWithAgentFlag(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:    "copilot",
			Agent: "/path/to/agent.md",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	if !strings.Contains(stepContent, "--agent /path/to/agent.md") {
		t.Errorf("Expected '--agent /path/to/agent.md' in copilot command, got:\n%s", stepContent)
	}
}

// TestCopilotEngineWithoutAgentFlag tests that copilot engine works without agent flag
func TestCopilotEngineWithoutAgentFlag(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	if strings.Contains(stepContent, "--agent") {
		t.Errorf("Did not expect '--agent' flag when agent is not specified, got:\n%s", stepContent)
	}
}

// TestClaudeEngineWithAgentFile tests that claude engine prepends agent file content to prompt
func TestClaudeEngineWithAgentFile(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:    "claude",
			Agent: "/path/to/agent.md",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Check that agent content extraction is present
	if !strings.Contains(stepContent, "AGENT_CONTENT=$(awk") {
		t.Errorf("Expected agent content extraction in claude command, got:\n%s", stepContent)
	}

	// Check that agent file path is referenced
	if !strings.Contains(stepContent, "/path/to/agent.md") {
		t.Errorf("Expected agent file path in claude command, got:\n%s", stepContent)
	}

	// Check that agent content is prepended to prompt
	if !strings.Contains(stepContent, "$AGENT_CONTENT") {
		t.Errorf("Expected $AGENT_CONTENT variable in claude command, got:\n%s", stepContent)
	}
}

// TestClaudeEngineWithoutAgentFile tests that claude engine works without agent file
func TestClaudeEngineWithoutAgentFile(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "claude",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Should not have agent content extraction
	if strings.Contains(stepContent, "AGENT_CONTENT") {
		t.Errorf("Did not expect AGENT_CONTENT when agent is not specified, got:\n%s", stepContent)
	}

	// Should still have the standard prompt
	if !strings.Contains(stepContent, "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)") {
		t.Errorf("Expected standard prompt reading in claude command, got:\n%s", stepContent)
	}
}

// TestCodexEngineWithAgentFile tests that codex engine prepends agent file content to prompt
func TestCodexEngineWithAgentFile(t *testing.T) {
	engine := NewCodexEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:    "codex",
			Agent: "/path/to/agent.md",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Check that agent content extraction is present
	if !strings.Contains(stepContent, "AGENT_CONTENT=$(awk") {
		t.Errorf("Expected agent content extraction in codex command, got:\n%s", stepContent)
	}

	// Check that agent file path is referenced
	if !strings.Contains(stepContent, "/path/to/agent.md") {
		t.Errorf("Expected agent file path in codex command, got:\n%s", stepContent)
	}

	// Check that agent content is prepended to prompt using printf
	if !strings.Contains(stepContent, "INSTRUCTION=$(printf") {
		t.Errorf("Expected printf with INSTRUCTION in codex command, got:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "$AGENT_CONTENT") {
		t.Errorf("Expected $AGENT_CONTENT variable in codex command, got:\n%s", stepContent)
	}
}

// TestCodexEngineWithoutAgentFile tests that codex engine works without agent file
func TestCodexEngineWithoutAgentFile(t *testing.T) {
	engine := NewCodexEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "codex",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Should not have agent content extraction
	if strings.Contains(stepContent, "AGENT_CONTENT") {
		t.Errorf("Did not expect AGENT_CONTENT when agent is not specified, got:\n%s", stepContent)
	}

	// Should have the standard instruction reading
	if !strings.Contains(stepContent, "INSTRUCTION=$(cat $GH_AW_PROMPT)") {
		t.Errorf("Expected standard INSTRUCTION reading in codex command, got:\n%s", stepContent)
	}
}
