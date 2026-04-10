//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCopilotEngineWithAgentFromEngineConfig tests that copilot engine includes --agent flag when specified in engine.agent
func TestCopilotEngineWithAgentFromEngineConfig(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:    "copilot",
			Agent: "my-custom-agent",
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Copilot CLI expects agent identifier
	if !strings.Contains(stepContent, `--agent my-custom-agent`) {
		t.Errorf("Expected '--agent my-custom-agent' in copilot command, got:\n%s", stepContent)
	}
}

// TestCopilotEngineWithAgentFromImports tests that agent imports do NOT set --agent flag
// Agent imports only import markdown content, not agent configuration
func TestCopilotEngineWithAgentFromImports(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
		AgentFile: ".github/agents/test-agent.md",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Agent imports should NOT set --agent flag (only engine.agent does)
	if strings.Contains(stepContent, `--agent`) {
		t.Errorf("Did not expect '--agent' flag when only AgentFile is set (without engine.agent), got:\n%s", stepContent)
	}
}

// TestCopilotEngineAgentOnlyFromEngineConfig tests that --agent flag is only set by engine.agent
func TestCopilotEngineAgentOnlyFromEngineConfig(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:    "copilot",
			Agent: "explicit-agent",
		},
		AgentFile: ".github/agents/import-agent.md",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Should only use explicit agent from engine.agent
	if !strings.Contains(stepContent, `--agent explicit-agent`) {
		t.Errorf("Expected '--agent explicit-agent' in copilot command, got:\n%s", stepContent)
	}
	// Should not use agent from imports
	if strings.Contains(stepContent, `--agent import-agent`) {
		t.Errorf("Did not expect '--agent import-agent' when engine.agent is set, got:\n%s", stepContent)
	}
}

// TestCopilotEngineWithoutAgentFlag tests that copilot engine works without agent file
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
		t.Errorf("Did not expect '--agent' flag when agent file is not specified, got:\n%s", stepContent)
	}
}

// TestClaudeEngineWithAgentFromImports tests that claude engine does NOT handle agent files
// natively — agent file content is prepended to prompt.txt by the compiler in the activation
// job, so the engine step always reads the standard prompt.txt path.
func TestClaudeEngineWithAgentFromImports(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "claude",
		},
		AgentFile: ".github/agents/test-agent.md",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Claude does not handle the agent file natively — no awk or AGENT_CONTENT/PROMPT_TEXT
	// variable juggling should appear in the step.
	if strings.Contains(stepContent, "AGENT_CONTENT") {
		t.Errorf("Claude must NOT handle agent file natively (AGENT_CONTENT found in step); the compiler handles it:\n%s", stepContent)
	}
	if strings.Contains(stepContent, "awk") {
		t.Errorf("Claude must NOT invoke awk for agent file reading (found in step); the compiler handles it:\n%s", stepContent)
	}
	if strings.Contains(stepContent, "PROMPT_TEXT") {
		t.Errorf("Claude must NOT use a PROMPT_TEXT shell variable (found in step); the compiler handles it:\n%s", stepContent)
	}

	// The engine still reads the standard prompt.txt (which has agent content prepended by the compiler).
	if !strings.Contains(stepContent, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`) {
		t.Errorf("Expected standard prompt.txt reading in claude command, got:\n%s", stepContent)
	}

	// The engine reports that it does not support native agent file handling.
	if engine.SupportsNativeAgentFile() {
		t.Errorf("Claude engine should return false for SupportsNativeAgentFile()")
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
		t.Errorf("Did not expect AGENT_CONTENT when agent file is not specified, got:\n%s", stepContent)
	}

	// Should still have the standard prompt
	if !strings.Contains(stepContent, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`) {
		t.Errorf("Expected standard prompt reading in claude command, got:\n%s", stepContent)
	}
}

// TestCodexEngineWithAgentFromImports tests that codex engine prepends agent file content to prompt
func TestCodexEngineWithAgentFromImports(t *testing.T) {
	engine := NewCodexEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "codex",
		},
		AgentFile: ".github/agents/test-agent.md",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Check that agent content extraction is present
	if !strings.Contains(stepContent, `AGENT_CONTENT="$(awk`) {
		t.Errorf("Expected agent content extraction in codex command, got:\n%s", stepContent)
	}

	// Check that agent file path is referenced with quoted GITHUB_WORKSPACE prefix
	if !strings.Contains(stepContent, `"${GITHUB_WORKSPACE}/.github/agents/test-agent.md"`) {
		t.Errorf("Expected agent file path with quoted GITHUB_WORKSPACE prefix in codex command, got:\n%s", stepContent)
	}

	// Check that agent content is prepended to prompt using printf
	if !strings.Contains(stepContent, `INSTRUCTION="$(printf`) {
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
		t.Errorf("Did not expect AGENT_CONTENT when agent file is not specified, got:\n%s", stepContent)
	}

	// Should have the standard instruction reading
	if !strings.Contains(stepContent, `INSTRUCTION="$(cat "$GH_AW_PROMPT")"`) {
		t.Errorf("Expected standard INSTRUCTION reading in codex command, got:\n%s", stepContent)
	}
}

// TestAgentFileValidation tests compile-time validation of agent file existence
func TestAgentFileValidation(t *testing.T) {
	// Create a temporary directory structure that mimics a repository
	tmpDir, err := os.MkdirTemp("", "agent-validation-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the directory structure: .github/agents/ and .github/workflows/
	agentsDir := filepath.Join(tmpDir, ".github", "agents")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents directory: %v", err)
	}
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a valid agent file
	agentContent := `---
on: push
title: Test Agent
---

# Test Agent Instructions

This is a test agent file.
`
	validAgentFilePath := filepath.Join(agentsDir, "valid-agent.md")
	if err := os.WriteFile(validAgentFilePath, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to create valid agent file: %v", err)
	}

	// Test 1: Valid agent file
	t.Run("valid_agent_file", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			AgentFile: validAgentFilePath,
		}

		workflowPath := filepath.Join(workflowsDir, "test.md")
		err := compiler.validateAgentFile(workflowData, workflowPath)
		if err != nil {
			t.Errorf("Expected no error for valid agent file, got: %v", err)
		}
	})

	// Test 2: Non-existent agent file
	t.Run("nonexistent_agent_file", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			AgentFile: filepath.Join(agentsDir, "nonexistent.md"),
		}

		workflowPath := filepath.Join(workflowsDir, "test.md")
		err := compiler.validateAgentFile(workflowData, workflowPath)
		if err == nil {
			t.Error("Expected error for non-existent agent file, got nil")
		} else if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})

	// Test 3: No agent file specified
	t.Run("no_agent_file", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
		}

		workflowPath := filepath.Join(workflowsDir, "test.md")
		err := compiler.validateAgentFile(workflowData, workflowPath)
		if err != nil {
			t.Errorf("Expected no error when agent file not specified, got: %v", err)
		}
	})

	// Test 4: Nil engine config
	t.Run("nil_engine_config", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{}

		workflowPath := filepath.Join(workflowsDir, "test.md")
		err := compiler.validateAgentFile(workflowData, workflowPath)
		if err != nil {
			t.Errorf("Expected no error when engine config is nil, got: %v", err)
		}
	})
}

// TestInvalidAgentFilePathGeneratesFailingStep tests that engines that handle agent files
// natively emit a clearly-failing step (rather than silently skipping execution) when the
// agent file path contains shell metacharacters.
// Engines that do NOT support native agent files (e.g. Claude) rely on the compiler's
// validateAgentFile to reject malicious paths at compile time instead.
func TestInvalidAgentFilePathGeneratesFailingStep(t *testing.T) {
	maliciousPath := `.github/agents/a";id;"b.md`

	t.Run("codex_emits_failing_step_for_invalid_path", func(t *testing.T) {
		engine := NewCodexEngine()
		workflowData := &WorkflowData{
			Name:      "test-workflow",
			AgentFile: maliciousPath,
		}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		if len(steps) != 1 {
			t.Fatalf("Expected exactly 1 failing step, got %d", len(steps))
		}
		content := strings.Join([]string(steps[0]), "\n")
		if !strings.Contains(content, "exit 1") {
			t.Errorf("Expected failing step with 'exit 1', got:\n%s", content)
		}
		if !strings.Contains(content, "Error") {
			t.Errorf("Expected error message in failing step, got:\n%s", content)
		}
		// Must NOT invoke awk (that would mean the path was used for real execution)
		if strings.Contains(content, "awk") {
			t.Errorf("Failing step must not invoke awk with the invalid path, got:\n%s", content)
		}
	})

	// Claude does not handle agent files natively; path validation is done by the compiler
	// at compile time (validateAgentFile). The engine step should proceed normally and never
	// reference the agent file path directly.
	t.Run("claude_ignores_agent_path_in_step_for_invalid_path", func(t *testing.T) {
		engine := NewClaudeEngine()
		workflowData := &WorkflowData{
			Name:      "test-workflow",
			AgentFile: maliciousPath,
		}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		if len(steps) != 1 {
			t.Fatalf("Expected exactly 1 step, got %d", len(steps))
		}
		content := strings.Join([]string(steps[0]), "\n")
		// Must NOT reference the malicious path at all in the generated step
		if strings.Contains(content, maliciousPath) {
			t.Errorf("Claude step must not reference the agent file path directly, got:\n%s", content)
		}
		if strings.Contains(content, "awk") {
			t.Errorf("Claude step must not invoke awk for agent file reading, got:\n%s", content)
		}
	})
}

// TestCheckoutWithAgentFromImports tests that checkout step is added when agent file is imported
func TestCheckoutWithAgentFromImports(t *testing.T) {
	t.Run("checkout_added_with_agent", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			AgentFile:   "/path/to/agent.md",
			Permissions: "permissions:\n  contents: read\n",
		}

		shouldCheckout := compiler.shouldAddCheckoutStep(workflowData)
		if !shouldCheckout {
			t.Error("Expected checkout to be added when agent file is specified")
		}
	})

	t.Run("checkout_added_with_agent_no_contents_permission", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			AgentFile:   "/path/to/agent.md",
			Permissions: "permissions:\n  issues: read\n",
		}

		shouldCheckout := compiler.shouldAddCheckoutStep(workflowData)
		if !shouldCheckout {
			t.Error("Expected checkout to be added when agent file is specified, even without contents permission")
		}
	})

	t.Run("checkout_added_without_agent", func(t *testing.T) {
		compiler := NewCompiler()
		// Set action mode to release
		compiler.SetActionMode(ActionModeRelease)

		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			Permissions: "permissions:\n  issues: read\n",
		}

		shouldCheckout := compiler.shouldAddCheckoutStep(workflowData)
		if !shouldCheckout {
			t.Error("Expected checkout to be added (always needed unless already in custom steps)")
		}
	})

	t.Run("checkout_with_custom_steps_containing_checkout", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			AgentFile:   "/path/to/agent.md",
			CustomSteps: "steps:\n  - uses: actions/checkout@v4\n",
		}

		shouldCheckout := compiler.shouldAddCheckoutStep(workflowData)
		if shouldCheckout {
			t.Error("Expected checkout NOT to be added when custom steps already contain checkout, even with agent file")
		}
	})
}

// TestCompilerIncludesAgentFileViaImportPaths verifies that when a non-native engine (Claude)
// is used with an agent file, the agent file path is included in the prompt via the standard
// ImportPaths/runtime-import mechanism (Step 1b in generatePrompt), so that prompt.txt
// already contains the agent file content when the engine reads it.
func TestCompilerIncludesAgentFileViaImportPaths(t *testing.T) {
	agentFilePath := ".github/agents/my-agent.md"

	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, ".github", "workflows", "test.md")
	if err := os.MkdirAll(filepath.Dir(workflowFile), 0o755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}
	if err := os.WriteFile(workflowFile, []byte("# Do the thing\n"), 0o644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Simulate what the orchestrator populates: the agent file is in ImportPaths (no inputs).
	workflowData := &WorkflowData{
		Name: "test-workflow",
		AI:   "claude",
		EngineConfig: &EngineConfig{
			ID: "claude",
		},
		AgentFile: agentFilePath,
		// ImportPaths mirrors what import_bfs.go populates for agent files without inputs.
		ImportPaths: []string{agentFilePath},
	}

	compiler := NewCompiler()
	compiler.markdownPath = workflowFile

	var buf strings.Builder
	compiler.generatePrompt(&buf, workflowData, false, nil)
	generated := buf.String()

	// The runtime-import macro for the agent file must appear in the generated YAML (exactly once).
	agentImportMacro := "{{#runtime-import " + agentFilePath + "}}"
	count := strings.Count(generated, agentImportMacro)
	if count == 0 {
		t.Errorf("Expected runtime-import macro %q in generated prompt YAML, got:\n%s", agentImportMacro, generated)
	} else if count > 1 {
		t.Errorf("Expected runtime-import macro %q exactly once, but found %d occurrences:\n%s", agentImportMacro, count, generated)
	}

	// The agent file import must appear before the main workflow markdown import.
	mainWorkflowMacro := "{{#runtime-import .github/workflows/test.md}}"
	agentIdx := strings.Index(generated, agentImportMacro)
	mainIdx := strings.Index(generated, mainWorkflowMacro)
	if mainIdx == -1 {
		t.Errorf("Expected main workflow runtime-import macro %q in generated prompt YAML, got:\n%s", mainWorkflowMacro, generated)
	} else if agentIdx > mainIdx {
		t.Errorf("Agent file runtime-import macro must appear before main workflow macro in prompt:\n%s", generated)
	}
}
