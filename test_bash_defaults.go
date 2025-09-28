package main

import (
	"fmt"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

func main() {
	// Test Claude engine with empty tools
	claudeEngine := workflow.NewClaudeEngine()
	claudeResult := claudeEngine.ComputeAllowedClaudeToolsString(map[string]any{}, nil)
	fmt.Printf("Claude with empty tools: %s\n", claudeResult)

	// Test Copilot engine with empty tools
	copilotEngine := workflow.NewCopilotEngine()
	copilotResult := copilotEngine.ComputeCopilotToolArguments(map[string]any{}, nil)
	fmt.Printf("Copilot with empty tools: %v\n", copilotResult)

	// Test with minimal bash config
	claudeResultBash := claudeEngine.ComputeAllowedClaudeToolsString(map[string]any{"bash": nil}, nil)
	fmt.Printf("Claude with bash=nil: %s\n", claudeResultBash)

	copilotResultBash := copilotEngine.ComputeCopilotToolArguments(map[string]any{"bash": nil}, nil)
	fmt.Printf("Copilot with bash=nil: %v\n", copilotResultBash)
}