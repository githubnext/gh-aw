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

// TestSlashCommandMembershipCheckConditional verifies that for slash_command workflows the
// check_command_position step runs before check_membership and that check_membership carries
// an if: condition so it is skipped when the slash command was not present in the trigger text.
// This prevents a confusing "access denied" warning on every issue-opened event.
func TestSlashCommandMembershipCheckConditional(t *testing.T) {
	tempDir := t.TempDir()

	workflowContent := `---
name: Test Slash Command Roles
on:
  slash_command:
    name: fix
permissions:
  contents: read
engine: copilot
---

Test workflow content
`

	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err)

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Failed to compile workflow")

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err)
	compiled := string(lockContent)

	// check_command_position must appear before check_membership in the pre_activation steps.
	cmdIdx := strings.Index(compiled, "id: check_command_position")
	membershipIdx := strings.Index(compiled, "id: check_membership")
	require.NotEqual(t, -1, cmdIdx, "Expected check_command_position step in compiled workflow")
	require.NotEqual(t, -1, membershipIdx, "Expected check_membership step in compiled workflow")
	assert.Less(t, cmdIdx, membershipIdx, "check_command_position must appear before check_membership")

	// check_membership must carry an if: condition referencing check_command_position.
	membershipSection := compiled[membershipIdx:]
	nextStepIdx := strings.Index(membershipSection[1:], "\n      - ")
	if nextStepIdx != -1 {
		membershipSection = membershipSection[:nextStepIdx+1]
	}
	assert.Contains(t, membershipSection, "check_command_position.outputs.command_position_ok",
		"check_membership step must be conditional on command_position_ok for slash_command workflows")
}

// TestSlashCommandOutputReferencesPreActivation ensures that the slash_command output
// in the activation job references needs.pre_activation.outputs.matched_command
// instead of steps.check_command_position.outputs.matched_command
func TestSlashCommandOutputReferencesPreActivation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test workflow with slash_command trigger
	workflowContent := `---
name: Test Slash Command
on:
  slash_command:
    name: test
permissions:
  contents: read
engine: copilot
---

Test workflow content
`

	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err)

	// Compile the workflow
	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Failed to compile workflow")

	// Get the lock file path
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)

	// Read the compiled workflow
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err)

	// Parse the YAML
	var workflow map[string]any
	err = yaml.Unmarshal(lockContent, &workflow)
	require.NoError(t, err)

	// Get the jobs
	jobs, ok := workflow["jobs"].(map[string]any)
	require.True(t, ok, "Expected jobs to be a map")

	// Check pre_activation job exists and has matched_command output
	preActivation, ok := jobs["pre_activation"].(map[string]any)
	require.True(t, ok, "Expected pre_activation job to exist")

	preActivationOutputs, ok := preActivation["outputs"].(map[string]any)
	require.True(t, ok, "Expected pre_activation job to have outputs")

	matchedCommand, ok := preActivationOutputs["matched_command"]
	require.True(t, ok, "Expected pre_activation job to have matched_command output")
	require.Contains(t, matchedCommand, "steps.check_command_position.outputs.matched_command",
		"Expected matched_command to reference check_command_position step output")

	// Check activation job exists and has slash_command output
	activation, ok := jobs["activation"].(map[string]any)
	require.True(t, ok, "Expected activation job to exist")

	activationOutputs, ok := activation["outputs"].(map[string]any)
	require.True(t, ok, "Expected activation job to have outputs")

	slashCommand, ok := activationOutputs["slash_command"]
	require.True(t, ok, "Expected activation job to have slash_command output")

	// Verify it references needs.pre_activation.outputs.matched_command
	assert.Contains(t, slashCommand, "needs.pre_activation.outputs.matched_command",
		"Expected slash_command to reference needs.pre_activation.outputs.matched_command")

	// Verify it does NOT reference steps.check_command_position
	assert.NotContains(t, slashCommand, "steps.check_command_position",
		"Expected slash_command to NOT reference steps.check_command_position directly")
}
