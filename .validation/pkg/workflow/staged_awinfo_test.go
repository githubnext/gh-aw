//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestGenerateCreateAwInfoWithStaged(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler()

	// Test with staged: true
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}},
			Staged:       true, // pointer to true
		},
	}

	// Create a test engine
	engine := NewClaudeEngine()

	var yaml strings.Builder
	c.generateCreateAwInfo(&yaml, workflowData, engine)

	result := yaml.String()

	// Check that GH_AW_INFO_STAGED: "true" is included in the step env when staged is true
	if !strings.Contains(result, `GH_AW_INFO_STAGED: "true"`) {
		t.Error("Expected 'GH_AW_INFO_STAGED: \"true\"' to be included in aw_info step env when staged is true")
	}

	// Test with staged: false
	workflowData.SafeOutputs.Staged = false

	yaml.Reset()
	c.generateCreateAwInfo(&yaml, workflowData, engine)

	result = yaml.String()

	// Check that GH_AW_INFO_STAGED: "false" is included in the step env when staged is false
	if !strings.Contains(result, `GH_AW_INFO_STAGED: "false"`) {
		t.Error("Expected 'GH_AW_INFO_STAGED: \"false\"' to be included in aw_info step env when staged is false")
	}

	// Test with no SafeOutputs config
	workflowData.SafeOutputs = nil

	yaml.Reset()
	c.generateCreateAwInfo(&yaml, workflowData, engine)

	result = yaml.String()

	// Check that GH_AW_INFO_STAGED: "false" is included in the step env when SafeOutputs is nil
	if !strings.Contains(result, `GH_AW_INFO_STAGED: "false"`) {
		t.Error("Expected 'GH_AW_INFO_STAGED: \"false\"' to be included in aw_info step env when SafeOutputs is nil")
	}
}
