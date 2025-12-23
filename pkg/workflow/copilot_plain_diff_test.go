package workflow

import (
	"strings"
	"testing"
)

// TestCopilotEnginePlainDiffFlag tests that the --plain-diff flag can be passed through engine args
func TestCopilotEnginePlainDiffFlag(t *testing.T) {
	engine := NewCopilotEngine()

	t.Run("plain-diff flag in engine args", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-plain-diff",
			EngineConfig: &EngineConfig{
				Args: []string{"--plain-diff"},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}

		stepContent := strings.Join([]string(steps[0]), "\n")

		// Verify --plain-diff flag is present in the command
		if !strings.Contains(stepContent, "--plain-diff") {
			t.Errorf("Expected --plain-diff flag in copilot command, got:\n%s", stepContent)
		}

		// Verify it's placed before the --prompt argument (custom args come before prompt)
		promptIndex := strings.Index(stepContent, "--prompt")
		plainDiffIndex := strings.Index(stepContent, "--plain-diff")

		if promptIndex == -1 {
			t.Error("Expected --prompt flag to be present")
		}

		if plainDiffIndex == -1 {
			t.Error("Expected --plain-diff flag to be present")
		}

		if plainDiffIndex > promptIndex {
			t.Error("Expected --plain-diff flag to come before --prompt flag")
		}
	})

	t.Run("plain-diff with other args", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-plain-diff-with-args",
			EngineConfig: &EngineConfig{
				Args: []string{"--plain-diff", "--verbose"},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}

		stepContent := strings.Join([]string(steps[0]), "\n")

		// Verify both flags are present
		if !strings.Contains(stepContent, "--plain-diff") {
			t.Errorf("Expected --plain-diff flag in copilot command, got:\n%s", stepContent)
		}

		if !strings.Contains(stepContent, "--verbose") {
			t.Errorf("Expected --verbose flag in copilot command, got:\n%s", stepContent)
		}
	})

	t.Run("without plain-diff flag", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-without-plain-diff",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}

		stepContent := strings.Join([]string(steps[0]), "\n")

		// Verify --plain-diff flag is not present when not configured
		if strings.Contains(stepContent, "--plain-diff") {
			t.Errorf("Did not expect --plain-diff flag when not configured, got:\n%s", stepContent)
		}
	})
}

// TestCopilotPlainDiffWithFirewall tests --plain-diff flag works with firewall enabled
func TestCopilotPlainDiffWithFirewall(t *testing.T) {
	engine := NewCopilotEngine()

	workflowData := &WorkflowData{
		Name: "test-plain-diff-firewall",
		EngineConfig: &EngineConfig{
			Args: []string{"--plain-diff"},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Verify --plain-diff flag is present even with firewall enabled
	if !strings.Contains(stepContent, "--plain-diff") {
		t.Errorf("Expected --plain-diff flag with firewall enabled, got:\n%s", stepContent)
	}

	// Verify AWF wrapper is present (firewall enabled)
	if !strings.Contains(stepContent, "awf") {
		t.Errorf("Expected AWF wrapper to be present with firewall enabled, got:\n%s", stepContent)
	}
}

// TestCopilotPlainDiffExtractEngineConfig tests that --plain-diff is properly extracted from frontmatter
func TestCopilotPlainDiffExtractEngineConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("extract plain-diff from engine args", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":   "copilot",
				"args": []any{"--plain-diff"},
			},
		}

		_, config := compiler.ExtractEngineConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected EngineConfig to be populated")
		}

		if len(config.Args) != 1 {
			t.Fatalf("Expected 1 arg, got %d: %v", len(config.Args), config.Args)
		}

		if config.Args[0] != "--plain-diff" {
			t.Errorf("Expected --plain-diff arg, got: %s", config.Args[0])
		}
	})

	t.Run("extract plain-diff with other args", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":   "copilot",
				"args": []string{"--plain-diff", "--verbose"},
			},
		}

		_, config := compiler.ExtractEngineConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected EngineConfig to be populated")
		}

		if len(config.Args) != 2 {
			t.Fatalf("Expected 2 args, got %d: %v", len(config.Args), config.Args)
		}

		if config.Args[0] != "--plain-diff" {
			t.Errorf("Expected first arg to be --plain-diff, got: %s", config.Args[0])
		}

		if config.Args[1] != "--verbose" {
			t.Errorf("Expected second arg to be --verbose, got: %s", config.Args[1])
		}
	})
}
