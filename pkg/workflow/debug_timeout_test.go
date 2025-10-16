package workflow

import (
"strings"
"testing"
)

func TestDebugTimeout(t *testing.T) {
compiler := &Compiler{}

// Test with different timeout values
tools := map[string]any{
"timeout": 90,
"github":  map[string]any{},
}

timeout := compiler.extractToolsTimeout(tools)
t.Logf("Extracted timeout: %d", timeout)

if timeout != 90 {
t.Errorf("Expected timeout 90, got %d", timeout)
}

// Test with engine
engine := NewClaudeEngine()
workflowData := &WorkflowData{
ToolsTimeout: 90,
Tools:        map[string]any{"github": map[string]any{}},
}

steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
if len(steps) == 0 {
t.Fatal("No execution steps generated")
}

stepContent := strings.Join([]string(steps[0]), "\n")
t.Logf("Step content:\n%s", stepContent)

if !strings.Contains(stepContent, `MCP_TIMEOUT: "90000"`) {
t.Errorf("Expected MCP_TIMEOUT: \"90000\" in step")
}
}
