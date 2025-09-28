package main

import (
	"fmt"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

func main() {
	compiler := workflow.NewCompiler(false, "", "test")
	
	// Test with safe outputs that require git commands
	safeOutputs := &workflow.SafeOutputsConfig{
		CreatePullRequests: &workflow.CreatePullRequestsConfig{},
	}
	
	// Start with empty tools
	emptyTools := map[string]any{}
	
	// Apply default tools (this should add git commands when safe outputs require them)
	toolsWithDefaults := compiler.ApplyDefaultTools(emptyTools, safeOutputs)
	
	fmt.Printf("Tools after applying defaults: %+v\n", toolsWithDefaults)
}