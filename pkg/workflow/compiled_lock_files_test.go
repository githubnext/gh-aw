//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workflowsDir is the path to the .github/workflows directory relative to this test file.
const workflowsDir = "../../.github/workflows"

// lockFileOutputMapping describes what outputs a compiled lock file must expose, derived from
// the SafeOutputsConfig parsed from the source .md file.
type lockFileOutputMapping struct {
	// configPresent returns true when the safe-output type is configured.
	configPresent func(*SafeOutputsConfig) bool
	// jobOutputKeys are the keys that must appear in the safe_outputs job's outputs section.
	jobOutputKeys []string
	// workflowCallOutputKeys are the keys that must appear in on.workflow_call.outputs when the
	// workflow uses workflow_call as a trigger.
	workflowCallOutputKeys []string
}

// lockFileOutputMappings enumerates all safe-output types that produce named outputs.
var lockFileOutputMappings = []lockFileOutputMapping{
	{
		configPresent:          func(s *SafeOutputsConfig) bool { return s.CreateIssues != nil },
		jobOutputKeys:          []string{"created_issue_number:", "created_issue_url:"},
		workflowCallOutputKeys: []string{"created_issue_number:", "created_issue_url:"},
	},
	{
		configPresent:          func(s *SafeOutputsConfig) bool { return s.CreatePullRequests != nil },
		jobOutputKeys:          []string{"created_pr_number:", "created_pr_url:"},
		workflowCallOutputKeys: []string{"created_pr_number:", "created_pr_url:"},
	},
	{
		configPresent:          func(s *SafeOutputsConfig) bool { return s.AddComments != nil },
		jobOutputKeys:          []string{"comment_id:", "comment_url:"},
		workflowCallOutputKeys: []string{"comment_id:", "comment_url:"},
	},
	{
		configPresent:          func(s *SafeOutputsConfig) bool { return s.PushToPullRequestBranch != nil },
		jobOutputKeys:          []string{"push_commit_sha:", "push_commit_url:"},
		workflowCallOutputKeys: []string{"push_commit_sha:", "push_commit_url:"},
	},
}

// extractSafeOutputsJobSection returns the text of the safe_outputs job block from a lock file.
// The block starts at "  safe_outputs:" and extends until the next top-level job entry or EOF.
func extractSafeOutputsJobSection(lockContent string) string {
	marker := "\n  safe_outputs:"
	idx := strings.Index(lockContent, marker)
	if idx < 0 {
		return ""
	}
	rest := lockContent[idx+1:] // skip the leading newline
	lines := strings.Split(rest, "\n")
	var sb strings.Builder
	for i, line := range lines {
		if i == 0 {
			sb.WriteString(line + "\n")
			continue
		}
		// A new top-level job starts with exactly two spaces then a word char (not more spaces).
		if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && strings.Contains(line, ":") {
			break
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

// extractWorkflowCallSection returns the text of the on section (up to but not including the
// first top-level non-on key) from a compiled lock file.
func extractWorkflowCallSection(lockContent string) string {
	for _, trigger := range []string{"\n\"on\":\n", "\non:\n"} {
		idx := strings.Index(lockContent, trigger)
		if idx < 0 {
			continue
		}
		onStart := idx + len(trigger)
		rest := lockContent[onStart:]
		lines := strings.Split(rest, "\n")
		var sb strings.Builder
		for _, line := range lines {
			// Top-level YAML key (no leading spaces) ends the "on" block.
			if len(line) > 0 && line[0] != ' ' && strings.Contains(line, ":") {
				break
			}
			sb.WriteString(line + "\n")
		}
		return sb.String()
	}
	return ""
}

// parseSafeOutputsForLockFileTest parses the source .md workflow file and returns the
// SafeOutputsConfig and the "on" section text, using the compiler's own parsing logic.
// Returns nil config when the workflow does not have safe-outputs configured.
func parseSafeOutputsForLockFileTest(mdPath string) (*SafeOutputsConfig, string, error) {
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowFile(mdPath)
	if err != nil {
		return nil, "", err
	}
	return data.SafeOutputs, data.On, nil
}

// TestCompiledLockFiles_SafeOutputsJobOutputs validates that every compiled lock file exposes
// the expected individual named outputs on its safe_outputs job, based on what safe-output
// types are configured in the corresponding source .md file.
func TestCompiledLockFiles_SafeOutputsJobOutputs(t *testing.T) {
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	require.NoError(t, err, "should glob .md workflow files")
	require.NotEmpty(t, mdFiles, "should find at least one .md workflow file")

	checkedWorkflows := 0

	for _, mdPath := range mdFiles {
		lockPath := strings.TrimSuffix(mdPath, ".md") + ".lock.yml"

		safeOutputs, _, parseErr := parseSafeOutputsForLockFileTest(mdPath)
		if parseErr != nil || safeOutputs == nil {
			continue // skip workflows without safe-outputs or that fail to parse
		}

		lockBytes, err := os.ReadFile(lockPath)
		if err != nil {
			continue // lock file may not exist yet
		}
		lockContent := string(lockBytes)

		safeOutputsJob := extractSafeOutputsJobSection(lockContent)
		if safeOutputsJob == "" {
			continue // workflow may not produce a safe_outputs job (e.g. runtime-import)
		}

		baseName := filepath.Base(mdPath)

		for _, mapping := range lockFileOutputMappings {
			if !mapping.configPresent(safeOutputs) {
				continue
			}
			for _, outputKey := range mapping.jobOutputKeys {
				assert.Contains(t, safeOutputsJob, outputKey,
					"lock file %s: safe_outputs job should expose %s", baseName, outputKey)
			}
		}

		checkedWorkflows++
	}

	assert.Positive(t, checkedWorkflows, "should have validated at least one workflow with safe-outputs")
	t.Logf("Validated safe_outputs job outputs for %d workflow(s)", checkedWorkflows)
}

// TestCompiledLockFiles_WorkflowCallOutputs validates that compiled lock files for workflows
// using workflow_call + safe-outputs automatically include on.workflow_call.outputs declarations.
func TestCompiledLockFiles_WorkflowCallOutputs(t *testing.T) {
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	require.NoError(t, err, "should glob .md workflow files")
	require.NotEmpty(t, mdFiles, "should find at least one .md workflow file")

	checkedWorkflows := 0

	for _, mdPath := range mdFiles {
		lockPath := strings.TrimSuffix(mdPath, ".md") + ".lock.yml"

		safeOutputs, onSection, parseErr := parseSafeOutputsForLockFileTest(mdPath)
		if parseErr != nil || safeOutputs == nil {
			continue
		}
		// Only check workflows that have workflow_call as a trigger.
		if !strings.Contains(onSection, "workflow_call") {
			continue
		}

		lockBytes, err := os.ReadFile(lockPath)
		if err != nil {
			t.Errorf("lock file missing for %s: %v", filepath.Base(mdPath), err)
			continue
		}
		lockContent := string(lockBytes)
		baseName := filepath.Base(mdPath)

		onLockSection := extractWorkflowCallSection(lockContent)

		// The on section must contain a workflow_call: key.
		assert.Contains(t, onLockSection, "workflow_call:", "lock file %s should have workflow_call trigger", baseName)

		workflowCallIdx := strings.Index(onLockSection, "workflow_call:")
		if workflowCallIdx < 0 {
			continue
		}
		workflowCallBlock := onLockSection[workflowCallIdx:]

		// Determine which outputs we expect based on configured safe-output types.
		expectedOutputs := buildWorkflowCallOutputsMap(safeOutputs)
		if len(expectedOutputs) > 0 {
			assert.Contains(t, workflowCallBlock, "outputs:",
				"lock file %s: on.workflow_call should contain an outputs: section", baseName)

			for _, mapping := range lockFileOutputMappings {
				if !mapping.configPresent(safeOutputs) {
					continue
				}
				for _, outputKey := range mapping.workflowCallOutputKeys {
					assert.Contains(t, workflowCallBlock, outputKey,
						"lock file %s: on.workflow_call.outputs should include %s", baseName, outputKey)
				}
			}
		}

		checkedWorkflows++
	}

	assert.Positive(t, checkedWorkflows,
		"should have validated at least one workflow with workflow_call + safe-outputs")
	t.Logf("Validated on.workflow_call.outputs for %d workflow(s)", checkedWorkflows)
}

// TestCompiledLockFiles_NoSpuriousWorkflowCallOutputs validates that workflows WITHOUT
// workflow_call do NOT have outputs injected into the on section.
func TestCompiledLockFiles_NoSpuriousWorkflowCallOutputs(t *testing.T) {
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	require.NoError(t, err, "should glob .md workflow files")
	require.NotEmpty(t, mdFiles, "should find at least one .md workflow file")

	checkedWorkflows := 0

	for _, mdPath := range mdFiles {
		lockPath := strings.TrimSuffix(mdPath, ".md") + ".lock.yml"

		safeOutputs, onSection, parseErr := parseSafeOutputsForLockFileTest(mdPath)
		if parseErr != nil || safeOutputs == nil {
			continue
		}
		// Only check workflows that have safe-outputs but do NOT use workflow_call.
		if strings.Contains(onSection, "workflow_call") {
			continue
		}

		lockBytes, err := os.ReadFile(lockPath)
		if err != nil {
			continue
		}
		lockContent := string(lockBytes)

		onLockSection := extractWorkflowCallSection(lockContent)
		baseName := filepath.Base(mdPath)

		// If the on section somehow has workflow_call, its outputs sub-key must be absent.
		if strings.Contains(onLockSection, "workflow_call:") {
			workflowCallIdx := strings.Index(onLockSection, "workflow_call:")
			workflowCallBlock := onLockSection[workflowCallIdx:]
			assert.NotContains(t, workflowCallBlock, "outputs:",
				"lock file %s: on.workflow_call should NOT have outputs (no workflow_call trigger in source)", baseName)
		}

		checkedWorkflows++
	}

	assert.Positive(t, checkedWorkflows,
		"should have validated at least one workflow with safe-outputs but without workflow_call")
	t.Logf("Validated no spurious workflow_call outputs for %d workflow(s)", checkedWorkflows)
}

// TestCompiledLockFiles_SmokeWorkflowCallHasExpectedOutputs is a focused test on the
// smoke-workflow-call workflow, the canonical workflow_call example in this repo.
func TestCompiledLockFiles_SmokeWorkflowCallHasExpectedOutputs(t *testing.T) {
	lockPath := filepath.Join(workflowsDir, "smoke-workflow-call.lock.yml")
	mdPath := filepath.Join(workflowsDir, "smoke-workflow-call.md")

	safeOutputs, onSection, err := parseSafeOutputsForLockFileTest(mdPath)
	require.NoError(t, err, "should parse smoke-workflow-call.md")
	require.NotNil(t, safeOutputs, "smoke-workflow-call.md should have safe-outputs")

	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err, "smoke-workflow-call.lock.yml should be readable")
	lockContent := string(lockBytes)

	t.Run("SourceHasWorkflowCallTrigger", func(t *testing.T) {
		assert.Contains(t, onSection, "workflow_call", "source md should have workflow_call trigger")
	})

	t.Run("LockHasWorkflowCallTrigger", func(t *testing.T) {
		assert.Contains(t, lockContent, "workflow_call:", "lock file should contain workflow_call trigger")
	})

	t.Run("LockHasSafeOutputsJob", func(t *testing.T) {
		assert.Contains(t, lockContent, "safe_outputs:", "lock file should contain safe_outputs job")
	})

	// The smoke workflow uses add-comment – verify its outputs appear in both places.
	require.NotNil(t, safeOutputs.AddComments, "smoke-workflow-call.md should have add-comment configured")

	t.Run("WorkflowCallOutputs_CommentID", func(t *testing.T) {
		onLockSection := extractWorkflowCallSection(lockContent)
		workflowCallIdx := strings.Index(onLockSection, "workflow_call:")
		require.GreaterOrEqual(t, workflowCallIdx, 0, "on section should contain workflow_call:")
		workflowCallBlock := onLockSection[workflowCallIdx:]
		assert.Contains(t, workflowCallBlock, "outputs:", "on.workflow_call should have outputs")
		assert.Contains(t, workflowCallBlock, "comment_id:", "on.workflow_call.outputs should include comment_id")
		assert.Contains(t, workflowCallBlock, "comment_url:", "on.workflow_call.outputs should include comment_url")
		assert.Contains(t, workflowCallBlock, "jobs.safe_outputs.outputs.comment_id",
			"workflow_call output value should reference safe_outputs job")
	})

	t.Run("SafeOutputsJobOutputs_CommentID", func(t *testing.T) {
		safeOutputsJob := extractSafeOutputsJobSection(lockContent)
		assert.Contains(t, safeOutputsJob, "comment_id:", "safe_outputs job should expose comment_id output")
		assert.Contains(t, safeOutputsJob, "comment_url:", "safe_outputs job should expose comment_url output")
		assert.Contains(t, safeOutputsJob, "steps.process_safe_outputs.outputs.comment_id",
			"safe_outputs job output should reference process_safe_outputs step")
	})
}
