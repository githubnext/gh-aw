package workflow

import (
	"strings"
	"testing"
)

// TestBashDefaultsConsistency tests that Claude and Copilot engines handle default bash tools consistently
func TestBashDefaultsConsistency(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	claudeEngine := NewClaudeEngine()
	copilotEngine := NewCopilotEngine()

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
	}{
		{
			name:        "empty tools, no safe outputs",
			tools:       map[string]any{},
			safeOutputs: nil,
		},
		{
			name:  "empty tools with create-pull-request safe output",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
		},
		{
			name: "bash nil with create-pull-request safe output",
			tools: map[string]any{
				"bash": nil,
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
		},
		{
			name: "bash with star wildcard",
			tools: map[string]any{
				"bash": []any{"*"},
			},
			safeOutputs: nil,
		},
		{
			name: "bash with colon-star wildcard",
			tools: map[string]any{
				"bash": []any{":*"},
			},
			safeOutputs: nil,
		},
		{
			name: "bash with empty array (no tools)",
			tools: map[string]any{
				"bash": []any{},
			},
			safeOutputs: nil,
		},
		{
			name: "bash enabled with true",
			tools: map[string]any{
				"bash": true,
			},
			safeOutputs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create copies of input tools to avoid modifying test data
			claudeTools := make(map[string]any)
			copilotTools := make(map[string]any)
			for k, v := range tt.tools {
				claudeTools[k] = v
				copilotTools[k] = v
			}

			// Apply default tools (this should add git commands when safe outputs require them)
			claudeTools = compiler.applyDefaultTools(claudeTools, tt.safeOutputs)
			copilotTools = compiler.applyDefaultTools(copilotTools, tt.safeOutputs)

			// Get results from both engines
			claudeResult := claudeEngine.computeAllowedClaudeToolsString(claudeTools, tt.safeOutputs)
			copilotResult := copilotEngine.computeCopilotToolArguments(copilotTools, tt.safeOutputs)

			t.Logf("Claude tools after defaults: %+v", claudeTools)
			t.Logf("Copilot tools after defaults: %+v", copilotTools)
			t.Logf("Claude result: %s", claudeResult)
			t.Logf("Copilot result: %v", copilotResult)

			// Parse Claude result
			claudeResultParts := []string{}
			if claudeResult != "" {
				claudeResultParts = strings.Split(claudeResult, ",")
			}

			// Check for bash-related tools in both results
			claudeHasBash := false
			claudeHasGit := false
			for _, tool := range claudeResultParts {
				tool = strings.TrimSpace(tool)
				if tool == "Bash" || strings.HasPrefix(tool, "Bash(") {
					claudeHasBash = true
					if strings.Contains(tool, "git") {
						claudeHasGit = true
					}
				}
			}

			copilotHasShell := false
			copilotHasGit := false
			for i := 0; i < len(copilotResult); i += 2 {
				if i+1 < len(copilotResult) && copilotResult[i] == "--allow-tool" {
					tool := copilotResult[i+1]
					if tool == "shell" || strings.HasPrefix(tool, "shell(") {
						copilotHasShell = true
						if strings.Contains(tool, "git") {
							copilotHasGit = true
						}
					}
				}
			}

			// Log detailed analysis for debugging
			t.Logf("Analysis - Claude has bash: %v (git: %v), Copilot has shell: %v (git: %v)",
				claudeHasBash, claudeHasGit, copilotHasShell, copilotHasGit)

			// Both engines should agree on whether bash/shell tools are present
			if claudeHasBash != copilotHasShell {
				t.Errorf("Inconsistency: Claude has bash tools: %v, Copilot has shell tools: %v", claudeHasBash, copilotHasShell)
			}

			// Both engines should agree on whether git commands are present
			if claudeHasGit != copilotHasGit {
				t.Errorf("Inconsistency: Claude has git commands: %v, Copilot has git commands: %v", claudeHasGit, copilotHasGit)
			}
		})
	}
}
