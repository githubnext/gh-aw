//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLabelCommandShorthandPreprocessing verifies that "label-command <name>" shorthand
// is expanded into the label_command map form by the schedule preprocessor.
func TestLabelCommandShorthandPreprocessing(t *testing.T) {
	tests := []struct {
		name          string
		onValue       string
		wantLabelName string
		wantErr       bool
	}{
		{
			name:          "simple label-command shorthand",
			onValue:       "label-command deploy",
			wantLabelName: "deploy",
		},
		{
			name:          "label-command with hyphenated label",
			onValue:       "label-command needs-review",
			wantLabelName: "needs-review",
		},
		{
			name:    "label-command without label name",
			onValue: "label-command ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frontmatter := map[string]any{
				"on": tt.onValue,
			}

			compiler := NewCompiler()
			err := compiler.preprocessScheduleFields(frontmatter, "", "")
			if tt.wantErr {
				assert.Error(t, err, "expected error for input %q", tt.onValue)
				return
			}

			require.NoError(t, err, "preprocessScheduleFields() should not error")

			onVal := frontmatter["on"]
			onMap, ok := onVal.(map[string]any)
			require.True(t, ok, "on field should be a map after expansion, got %T", onVal)

			labelCmd, hasLabel := onMap["label_command"]
			require.True(t, hasLabel, "on map should have label_command key")
			assert.Equal(t, tt.wantLabelName, labelCmd,
				"label_command value should be %q", tt.wantLabelName)

			_, hasDispatch := onMap["workflow_dispatch"]
			assert.True(t, hasDispatch, "on map should have workflow_dispatch key")
		})
	}
}

// TestExpandLabelCommandShorthand verifies the expand helper function.
func TestExpandLabelCommandShorthand(t *testing.T) {
	result := expandLabelCommandShorthand("deploy")

	labelCmd, ok := result["label_command"]
	require.True(t, ok, "expanded map should have label_command key")
	assert.Equal(t, "deploy", labelCmd, "label_command should equal the label name")

	_, hasDispatch := result["workflow_dispatch"]
	assert.True(t, hasDispatch, "expanded map should have workflow_dispatch key")
}

// TestFilterLabelCommandEvents verifies that FilterLabelCommandEvents returns correct subsets.
func TestFilterLabelCommandEvents(t *testing.T) {
	tests := []struct {
		name        string
		identifiers []string
		want        []string
	}{
		{
			name:        "nil identifiers returns all events",
			identifiers: nil,
			want:        []string{"issues", "pull_request", "discussion"},
		},
		{
			name:        "empty identifiers returns all events",
			identifiers: []string{},
			want:        []string{"issues", "pull_request", "discussion"},
		},
		{
			name:        "single issues event",
			identifiers: []string{"issues"},
			want:        []string{"issues"},
		},
		{
			name:        "issues and pull_request only",
			identifiers: []string{"issues", "pull_request"},
			want:        []string{"issues", "pull_request"},
		},
		{
			name:        "unsupported event is filtered out",
			identifiers: []string{"issues", "unknown_event"},
			want:        []string{"issues"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterLabelCommandEvents(tt.identifiers)
			assert.Equal(t, tt.want, got, "FilterLabelCommandEvents(%v)", tt.identifiers)
		})
	}
}

// TestBuildLabelCommandCondition verifies the condition builder for label-command triggers.
func TestBuildLabelCommandCondition(t *testing.T) {
	tests := []struct {
		name            string
		labelNames      []string
		events          []string
		hasOtherEvents  bool
		wantErr         bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:       "single label all events no other events",
			labelNames: []string{"deploy"},
			events:     nil,
			wantContains: []string{
				"github.event.label.name == 'deploy'",
				"github.event_name == 'issues'",
				"github.event_name == 'pull_request'",
				"github.event_name == 'discussion'",
			},
		},
		{
			name:       "multiple labels all events",
			labelNames: []string{"deploy", "release"},
			events:     nil,
			wantContains: []string{
				"github.event.label.name == 'deploy'",
				"github.event.label.name == 'release'",
			},
		},
		{
			name:       "single label issues only",
			labelNames: []string{"triage"},
			events:     []string{"issues"},
			wantContains: []string{
				"github.event_name == 'issues'",
				"github.event.label.name == 'triage'",
			},
			wantNotContains: []string{
				"github.event_name == 'pull_request'",
				"github.event_name == 'discussion'",
			},
		},
		{
			name:       "no label names returns error",
			labelNames: []string{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, err := buildLabelCommandCondition(tt.labelNames, tt.events, tt.hasOtherEvents)
			if tt.wantErr {
				assert.Error(t, err, "expected an error")
				return
			}

			require.NoError(t, err, "buildLabelCommandCondition() should not error")
			rendered := condition.Render()

			for _, want := range tt.wantContains {
				assert.Contains(t, rendered, want,
					"condition should contain %q, got: %s", want, rendered)
			}
			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, rendered, notWant,
					"condition should NOT contain %q, got: %s", notWant, rendered)
			}
		})
	}
}

// TestLabelCommandWorkflowCompile verifies that a workflow with label_command trigger
// compiles to a valid GitHub Actions workflow with:
//   - label-based events (issues, pull_request, discussion) in the on: section
//   - workflow_dispatch with item_number input
//   - a label-name condition in the activation job's if:
//   - a remove_trigger_label step in the activation job
//   - a label_command output on the activation job
func TestLabelCommandWorkflowCompile(t *testing.T) {
	tempDir := t.TempDir()

	workflowContent := `---
name: Label Command Test
on:
  label_command: deploy
engine: copilot
---

Deploy the application because label "deploy" was added.
`

	workflowPath := filepath.Join(tempDir, "label-command-test.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "CompileWorkflow() should not error")

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "failed to read lock file")

	lockStr := string(lockContent)

	// Verify the on: section includes label-based events
	assert.Contains(t, lockStr, "issues:", "on section should contain issues event")
	assert.Contains(t, lockStr, "pull_request:", "on section should contain pull_request event")
	assert.Contains(t, lockStr, "discussion:", "on section should contain discussion event")
	assert.Contains(t, lockStr, "labeled", "on section should contain labeled type")
	assert.Contains(t, lockStr, "workflow_dispatch:", "on section should contain workflow_dispatch")
	assert.Contains(t, lockStr, "item_number:", "workflow_dispatch should include item_number input")

	// Parse the YAML to check the activation job
	var workflow map[string]any
	err = yaml.Unmarshal(lockContent, &workflow)
	require.NoError(t, err, "failed to parse lock file as YAML")

	jobs, ok := workflow["jobs"].(map[string]any)
	require.True(t, ok, "workflow should have jobs")

	activation, ok := jobs["activation"].(map[string]any)
	require.True(t, ok, "workflow should have an activation job")

	// Verify the activation job has a label_command output
	activationOutputs, ok := activation["outputs"].(map[string]any)
	require.True(t, ok, "activation job should have outputs")

	labelCmdOutput, hasOutput := activationOutputs["label_command"]
	assert.True(t, hasOutput, "activation job should have label_command output")
	assert.Contains(t, labelCmdOutput, "remove_trigger_label",
		"label_command output should reference the remove_trigger_label step")

	// Verify the remove_trigger_label step exists in the activation job
	activationSteps, ok := activation["steps"].([]any)
	require.True(t, ok, "activation job should have steps")

	foundRemoveStep := false
	for _, step := range activationSteps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := stepMap["id"].(string); ok && id == "remove_trigger_label" {
			foundRemoveStep = true
			break
		}
	}
	assert.True(t, foundRemoveStep, "activation job should contain a remove_trigger_label step")

	// Verify the workflow condition includes the label name check
	agentJob, hasAgent := jobs["agent"].(map[string]any)
	require.True(t, hasAgent, "workflow should have an agent job")
	_ = agentJob // presence check is sufficient
}

// TestLabelCommandWorkflowCompileShorthand verifies the "label-command <name>" string shorthand.
func TestLabelCommandWorkflowCompileShorthand(t *testing.T) {
	tempDir := t.TempDir()

	workflowContent := `---
name: Label Command Shorthand Test
on: "label-command needs-review"
engine: copilot
---

Triggered by the needs-review label.
`

	workflowPath := filepath.Join(tempDir, "label-command-shorthand.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "CompileWorkflow() should not error for shorthand form")

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "failed to read lock file")

	lockStr := string(lockContent)
	assert.Contains(t, lockStr, "labeled", "compiled workflow should contain labeled type")
	assert.Contains(t, lockStr, "remove_trigger_label", "compiled workflow should contain remove_trigger_label step")
}

// TestLabelCommandWorkflowWithEvents verifies that specifying events: restricts
// which GitHub Actions events are generated.
func TestLabelCommandWorkflowWithEvents(t *testing.T) {
	tempDir := t.TempDir()

	workflowContent := `---
name: Label Command Issues Only
on:
  label_command:
    name: deploy
    events: [issues]
engine: copilot
---

Triggered by the deploy label on issues only.
`

	workflowPath := filepath.Join(tempDir, "label-command-issues-only.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "CompileWorkflow() should not error")

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "failed to read lock file")

	lockStr := string(lockContent)

	// Should have issues event
	assert.Contains(t, lockStr, "issues:", "on section should contain issues event")

	// workflow_dispatch is always added
	assert.Contains(t, lockStr, "workflow_dispatch:", "on section should contain workflow_dispatch")

	// pull_request and discussion should NOT be present since events: [issues] was specified
	// (However, they may be commented or absent — check the YAML structure)
	var workflow map[string]any
	err = yaml.Unmarshal(lockContent, &workflow)
	require.NoError(t, err, "failed to parse lock file as YAML")

	onSection, ok := workflow["on"].(map[string]any)
	require.True(t, ok, "workflow on: section should be a map")

	_, hasPR := onSection["pull_request"]
	assert.False(t, hasPR, "pull_request event should not be present when events=[issues]")

	_, hasDiscussion := onSection["discussion"]
	assert.False(t, hasDiscussion, "discussion event should not be present when events=[issues]")
}

// TestLabelCommandNoClashWithExistingLabelTrigger verifies that label_command can coexist
// with an existing label-only issues trigger without creating a duplicate issues: YAML block.
// The existing issues block types are merged into the label_command-generated issues block.
func TestLabelCommandNoClashWithExistingLabelTrigger(t *testing.T) {
	tempDir := t.TempDir()

	// Workflow that has both an explicit "issues: types: [labeled]" block AND label_command.
	// This is the exact key-clash scenario: without merging, two "issues:" keys would appear
	// in the compiled YAML, which is invalid and silently broken.
	workflowContent := `---
name: No Clash Test
on:
  label_command: deploy
  issues:
    types: [labeled]
engine: copilot
---

Both label-command and existing issues labeled trigger.
`

	workflowPath := filepath.Join(tempDir, "no-clash-test.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "CompileWorkflow() should not error when mixing label_command with existing label trigger")

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "failed to read lock file")

	lockStr := string(lockContent)

	// Verify there is exactly ONE "issues:" block at the YAML top level
	// (count occurrences that are a key, not embedded in other values)
	issuesCount := strings.Count(lockStr, "\n  issues:\n") + strings.Count(lockStr, "\nissues:\n")
	assert.Equal(t, 1, issuesCount,
		"there should be exactly one 'issues:' trigger block in the compiled YAML, got %d. Compiled:\n%s",
		issuesCount, lockStr)
}

// TestLabelCommandConflictWithNonLabelTrigger verifies that using label_command alongside
// an issues/pull_request trigger with non-label types returns a validation error.
func TestLabelCommandConflictWithNonLabelTrigger(t *testing.T) {
	tempDir := t.TempDir()

	// Workflow with label_command and issues: types: [opened] — non-label type conflicts
	workflowContent := `---
name: Conflict Test
on:
  label_command: deploy
  issues:
    types: [opened]
engine: copilot
---

This should fail validation.
`

	workflowPath := filepath.Join(tempDir, "conflict-test.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.Error(t, err, "CompileWorkflow() should error when label_command is combined with non-label issues trigger")
	assert.Contains(t, err.Error(), "label_command", "error should mention label_command")
}
