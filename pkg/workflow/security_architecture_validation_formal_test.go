//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const formalVALWorkflowFrontmatter = `---
name: formal-validation-suite
on:
  pull_request:
    types: [opened]
  roles:
    - write
engine: copilot
permissions:
  contents: read
safe-outputs:
  create-issue:
concurrency:
  group: "formal-${{ github.event.pull_request.number }}"
  cancel-in-progress: true
---`

var (
	formalVALWritePermissionLine = regexp.MustCompile(`(?m)^\s+[a-z0-9-]+:\s*write\s*$`)
	formalVALUsesLine            = regexp.MustCompile(`(?m)^\s*uses:\s*(\S+)`)
	formalVALSHARef              = regexp.MustCompile(`^[0-9a-f]{40}$`)
	formalVALConcurrencyGroup    = regexp.MustCompile(`(?m)^\s*group:\s*(.+)$`)
)

func compileFormalVALWorkflow(t *testing.T) string {
	t.Helper()
	return compileFormalVALWorkflowFromFrontmatter(t, formalVALWorkflowFrontmatter)
}

func compileFormalVALStrictWorkflow(t *testing.T) string {
	t.Helper()
	frontmatter := strings.Replace(formalVALWorkflowFrontmatter, "engine: copilot", "engine: copilot\nstrict: true", 1)
	return compileFormalVALWorkflowFromFrontmatter(t, frontmatter)
}

func compileFormalVALWorkflowFromFrontmatter(t *testing.T, frontmatter string) string {
	t.Helper()

	md := frontmatter + `

# Mission

Formal validation workflow.
`

	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err)

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err)
	require.NotEmpty(t, yamlOut)

	return yamlOut
}

func assertFormalVALNoWritePermissions(t *testing.T, section string) {
	t.Helper()
	assert.False(t, formalVALWritePermissionLine.MatchString(section), "section must not contain write permissions:\n%s", section)
}

func TestFormalVAL_P1_TripartiteJobArchitecture(t *testing.T) {
	yamlOut := compileFormalVALWorkflow(t)
	require.NotEmpty(t, extractJobSection(yamlOut, string(constants.ActivationJobName)), "OI-01: activation job must be present")
	require.NotEmpty(t, extractJobSection(yamlOut, string(constants.AgentJobName)), "OI-01: agent job must be present")
	require.NotEmpty(t, extractJobSection(yamlOut, string(constants.SafeOutputsJobName)), "OI-01: safe_outputs job must be present")
}

func TestFormalVAL_P2_SanitizationPipelineOrder(t *testing.T) {
	step := map[string]any{"run": `echo "${{ github.event.issue.title }}" && echo done`}

	sanitized, descriptions, changed := sanitizeRunStepExpressions(step)
	require.True(t, changed, "IS-10: run expressions must be extracted into env before compilation continues")
	require.NotEmpty(t, descriptions, "IS-10: extraction must emit a substitution description")

	runVal, ok := sanitized["run"].(string)
	require.True(t, ok, "IS-10: sanitized run field must remain a string")
	assert.NotContains(t, runVal, "${{", "IS-10: sanitized run field must not retain raw expressions")

	envMap, ok := sanitized["env"].(map[string]any)
	require.True(t, ok, "IS-10: sanitized step must include an env block")
	assert.Equal(t, "${{ github.event.issue.title }}", envMap["GH_AW_GITHUB_EVENT_ISSUE_TITLE"])
}

func TestFormalVAL_P3_AgentJobHasOnlyReadPermissions(t *testing.T) {
	agentSection := extractJobSection(compileFormalVALWorkflow(t), string(constants.AgentJobName))
	require.NotEmpty(t, agentSection, "PM-01/PM-02: compiled workflow must contain an agent job")
	assert.Contains(t, agentSection, "permissions:")
	assert.Contains(t, agentSection, "contents: read")
	assertFormalVALNoWritePermissions(t, agentSection)
}

func TestFormalVAL_P4_ForkProtectionRepoIDComparison(t *testing.T) {
	yamlOut := compileFormalVALWorkflow(t)
	activationSection := extractJobSection(yamlOut, string(constants.ActivationJobName))
	preActivationSection := extractJobSection(yamlOut, string(constants.PreActivationJobName))

	assert.Contains(t, activationSection, "github.event.pull_request.head.repo.id == github.repository_id")
	assert.Contains(t, preActivationSection, "github.event.pull_request.head.repo.id == github.repository_id")
}

func TestFormalVAL_P5_PreActivationGatesActivation(t *testing.T) {
	yamlOut := compileFormalVALWorkflow(t)
	activationSection := extractJobSection(yamlOut, string(constants.ActivationJobName))
	preActivationSection := extractJobSection(yamlOut, string(constants.PreActivationJobName))

	assert.Contains(t, activationSection, "needs: pre_activation")
	assert.Contains(t, activationSection, "needs.pre_activation.outputs.activated == 'true'")
	assert.Contains(t, preActivationSection, "check_membership.cjs")
	assert.Contains(t, preActivationSection, `GH_AW_REQUIRED_ROLES: "write"`)
}

func TestFormalVAL_P6_DetectionJobBetweenAgentAndSafeOutputs(t *testing.T) {
	yamlOut := compileFormalVALWorkflow(t)
	agentIndex := indexInNonCommentLines(yamlOut, "  agent:")
	detectionIndex := indexInNonCommentLines(yamlOut, "  detection:")
	safeOutputsIndex := indexInNonCommentLines(yamlOut, "  safe_outputs:")

	require.NotEqual(t, -1, agentIndex)
	require.NotEqual(t, -1, detectionIndex)
	require.NotEqual(t, -1, safeOutputsIndex)
	assert.Less(t, agentIndex, detectionIndex, "TD-01: detection job must appear after agent")
	assert.Less(t, detectionIndex, safeOutputsIndex, "TD-01: safe_outputs job must appear after detection")
	assert.Contains(t, extractJobSection(yamlOut, string(constants.SafeOutputsJobName)), "- detection")
}

func TestFormalVAL_P7_ActionPinsAre40CharSHA(t *testing.T) {
	matches := formalVALUsesLine.FindAllStringSubmatch(compileFormalVALWorkflow(t), -1)
	require.NotEmpty(t, matches, "CS-10: compiled workflow must contain uses steps")

	remoteUsesCount := 0
	for _, match := range matches {
		ref := match[1]
		if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") {
			continue
		}
		parts := strings.SplitN(ref, "@", 2)
		require.Len(t, parts, 2, "CS-10: remote uses step must include @ref: %s", ref)
		assert.True(t, formalVALSHARef.MatchString(parts[1]), "CS-10: remote uses step must be pinned to a 40-char SHA: %s", ref)
		remoteUsesCount++
	}

	assert.Positive(t, remoteUsesCount, "CS-10: expected at least one remote uses step")
}

func TestFormalVAL_P8_ActivationContainsTimestampCheck(t *testing.T) {
	activationSection := extractJobSection(compileFormalVALWorkflow(t), string(constants.ActivationJobName))
	require.NotEmpty(t, activationSection)
	assert.Contains(t, activationSection, "check_workflow_timestamp_api.cjs")
}

func TestFormalVAL_P9_ConcurrencyGroupContainsDynamicExpr(t *testing.T) {
	match := formalVALConcurrencyGroup.FindStringSubmatch(compileFormalVALWorkflow(t))
	require.Len(t, match, 2, "RS-16/RS-17: compiled workflow must contain a concurrency group")
	assert.Contains(t, match[1], "${{", "RS-16/RS-17: concurrency.group must contain a dynamic expression")
}

func TestFormalVAL_P10_WritePermissionsIsolatedToSafeOutputs(t *testing.T) {
	yamlOut := compileFormalVALWorkflow(t)
	activationSection := extractJobSection(yamlOut, string(constants.ActivationJobName))
	agentSection := extractJobSection(yamlOut, string(constants.AgentJobName))
	safeOutputsSection := extractJobSection(yamlOut, string(constants.SafeOutputsJobName))

	assertFormalVALNoWritePermissions(t, activationSection)
	assertFormalVALNoWritePermissions(t, agentSection)
	assert.Contains(t, safeOutputsSection, "issues: write", "OI-07: safe_outputs must be the write-capable job")
}

func TestFormalVAL_P11_QueueMaxConflictsWithCancelInProgress(t *testing.T) {
	err := validateConcurrencyQueueConfiguration(`concurrency:
  group: "formal-${{ github.ref }}"
  queue: max
  cancel-in-progress: true`)
	require.Error(t, err, "RS-22: queue:max with cancel-in-progress:true must be rejected")
	assert.Contains(t, err.Error(), "queue: max cannot be combined with cancel-in-progress: true")
}

func TestFormalVAL_P12_RunStepExpressionsExtractedToEnv(t *testing.T) {
	tests := []struct {
		name        string
		step        map[string]any
		wantChanged bool
		wantEnvKey  string
	}{
		{name: "empty run", step: map[string]any{"run": ""}, wantChanged: false},
		{name: "no expression", step: map[string]any{"run": "echo ${MY_VAR}"}, wantChanged: false},
		{name: "multiple expressions", step: map[string]any{"run": `echo "${{ github.event.issue.title }}" && echo "${{ github.event.issue.body }}"`}, wantChanged: true, wantEnvKey: "GH_AW_GITHUB_EVENT_ISSUE_TITLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, changed := sanitizeRunStepExpressions(tt.step)
			assert.Equal(t, tt.wantChanged, changed)
			if !tt.wantChanged {
				assert.Equal(t, tt.step, result)
				return
			}

			envMap, ok := result["env"].(map[string]any)
			require.True(t, ok)
			assert.Contains(t, envMap, tt.wantEnvKey)
			assert.NotContains(t, result["run"].(string), "${{")
		})
	}
}

func TestFormalVAL_P13_EmptyConcurrencyGroupRejected(t *testing.T) {
	for _, group := range []string{"", "   \n\t  "} {
		err := validateConcurrencyGroupExpression(group)
		require.Error(t, err, "RS-17: empty concurrency groups must be rejected")
		assert.Contains(t, err.Error(), "empty concurrency group expression")
	}
}

func TestFormalVAL_P14_MalformedConcurrencyExprRejected(t *testing.T) {
	for _, group := range []string{"formal-${{ github.ref }", "formal-${{ (github.ref }}"} {
		assert.Error(t, validateConcurrencyGroupExpression(group), "RS-17: malformed concurrency expressions must be rejected")
	}
}

func TestFormalVAL_P15_WritePermissionsRejectedInStrictMode(t *testing.T) {
	yamlOut := compileFormalVALStrictWorkflow(t)
	assert.Contains(t, yamlOut, "permissions: {}", "PM-01: strict-mode compiled workflows must default to top-level permissions:{}")
	assertFormalVALNoWritePermissions(t, extractJobSection(yamlOut, string(constants.ActivationJobName)))
	assertFormalVALNoWritePermissions(t, extractJobSection(yamlOut, string(constants.AgentJobName)))
}
