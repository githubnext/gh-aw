//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertPolicyCodemodValue(t *testing.T, content string, expected any, path ...string) {
	t.Helper()
	document, err := aw.Parse(content)
	require.NoError(t, err)
	var value any = document.Frontmatter
	for _, key := range path {
		mapping, ok := value.(map[string]any)
		require.True(t, ok, "path %v is not a mapping", path)
		var exists bool
		value, exists = mapping[key]
		require.True(t, exists, "path %v is absent", path)
	}
	assert.Equal(t, expected, value)
}

func TestPolicyFrontmatterASTRenames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		codemod  Codemod
		source   string
		oldToken string
		newToken string
	}{
		{getSandboxFieldsCodemod(), `'sandbox': {agent: {targets: {'a.b: target': {'authHeader': token, nested: {authHeader: keep}}}}}`, "'authHeader'", "'auth-header'"},
		{getSafeOutputRequireTitlePrefixCodemod(), `"safe-outputs": {"close-issue": {'title-prefix': "[bot]", nested: {title-prefix: keep}}}`, "'title-prefix'", "'required-title-prefix'"},
		{getSafeOutputRequireTitlePrefixCodemod(), `safe-outputs: {push-to-pull-request-branch: {'labels': [bot], nested: {labels: keep}}}`, "'labels'", "'required-labels'"},
		{getSafeOutputMergePRConstraintsCodemod(), `safe-outputs: {merge-pull-request: {'allowed-labels': [bot], nested: {allowed-labels: keep}}}`, "'allowed-labels'", "'required-labels'"},
		{getSafeOutputAddReviewerAllowlistsCodemod(), `safe-outputs: {add-reviewer: {'reviewers': [bot], nested: {reviewers: keep}}}`, "'reviewers'", "'allowed-reviewers'"},
		{getSafeOutputAddReviewerAllowlistsCodemod(), `safe-outputs: {add-reviewer: {'team-reviewers': [bot], nested: {team-reviewers: keep}}}`, "'team-reviewers'", "'allowed-team-reviewers'"},
		{getSafeOutputDispatchRepositoryKeyCodemod(), `"safe-outputs": {'dispatch_repository': {nested: {dispatch_repository: keep}}}`, "'dispatch_repository'", "'dispatch-repository'"},
		{getSafeJobRunnerCodemod(), `safe-outputs: {jobs: {'build: a.b': {'runner': linux, steps: [{runner: keep}]}}}`, "'runner'", "'runs-on'"},
		{getMentionsAllowTeamMembersCodemod(), `safe-outputs: {mentions: {'allow-team-members': true, nested: {allow-team-members: keep}}}`, "'allow-team-members'", "'allowed-collaborators'"},
	}
	for _, tt := range tests {
		t.Run(tt.codemod.ID+"/"+tt.oldToken, func(t *testing.T) {
			content := "---\n" + tt.source + " # keep comment\n---\n\n# Body\nrunner: keep\n"
			result, applied, err := tt.codemod.Apply(content, map[string]any{"stale": true})
			require.NoError(t, err)
			require.True(t, applied)
			assert.Equal(t, strings.Replace(content, tt.oldToken, tt.newToken, 1), result)
			second, applied, err := tt.codemod.Apply(result, nil)
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, result, second)
		})
	}
}

func TestPolicyFrontmatterASTConflicts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		codemod Codemod
		source  string
	}{
		{getSafeOutputRequireTitlePrefixCodemod(), `safe-outputs: {close-issue: {title-prefix: old, required-title-prefix: null}}`},
		{getSafeOutputMergePRConstraintsCodemod(), `safe-outputs: {merge-pull-request: {allowed-labels: [old], required-labels: null}}`},
		{getSafeOutputAddReviewerAllowlistsCodemod(), `safe-outputs: {add-reviewer: {reviewers: [old], allowed-reviewers: null}}`},
		{getSafeOutputDispatchRepositoryKeyCodemod(), `safe-outputs: {dispatch_repository: {}, dispatch-repository: null}`},
		{getSafeJobRunnerCodemod(), `safe-outputs: {jobs: {build: {runner: old, runs-on: null}}}`},
		{getMentionsAllowTeamMembersCodemod(), `safe-outputs: {mentions: {allow-team-members: true, allowed-collaborators: null}}`},
	}
	for _, tt := range tests {
		t.Run(tt.codemod.ID, func(t *testing.T) {
			content := "---\n" + tt.source + "\n---\n# Body"
			result, applied, err := tt.codemod.Apply(content, nil)
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, content, result)
		})
	}
	content := "---\nsandbox: {agent: {images: {apiProxy: old, cliProxy: old, cli-proxy: null}}}\n---\n"
	result, applied, err := getSandboxFieldsCodemod().Apply(content, nil)
	require.ErrorContains(t, err, "both spellings are present")
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxFieldsASTLegacySequenceTargets(t *testing.T) {
	t.Parallel()
	content := "---\nsandbox:\n  agent:\n    targets:\n      - authHeader: token # keep target comment\n        extraHeaders: {x-Test: value}\n        nested: {authHeader: keep}\n---\n# Body\n"
	result, applied, err := getSandboxFieldsCodemod().Apply(content, nil)
	require.NoError(t, err)
	require.True(t, applied)
	assertPolicyCodemodValue(t, result, []any{map[string]any{
		"auth-header":   "token",
		"extra-headers": map[string]any{"x-Test": "value"},
		"nested":        map[string]any{"authHeader": "keep"},
	}}, "sandbox", "agent", "targets")
	assert.Contains(t, result, "# keep target comment")
	assert.True(t, strings.HasSuffix(result, "---\n# Body\n"))

	collision := strings.Replace(content, "extraHeaders: {x-Test: value}", "auth-header: null", 1)
	result, applied, err = getSandboxFieldsCodemod().Apply(collision, nil)
	require.ErrorContains(t, err, "both spellings are present")
	assert.False(t, applied)
	assert.Equal(t, collision, result)
}

func TestPolicyFrontmatterASTNoOpNestedNames(t *testing.T) {
	t.Parallel()
	content := `---
wrapper:
  sandbox: {agent: {images: {apiProxy: keep}}}
  safe-outputs:
    close-issue: {title-prefix: keep}
    merge-pull-request: {allowed-labels: [keep]}
    add-reviewer: {reviewers: [keep]}
    dispatch_repository: {}
    jobs: {build: {runner: keep}}
    mentions: {allow-team-members: true}
    create-issue: {expires: 7}
  tools:
    github: {allowed-repos: current}
    bash: ["echo 'keep'"]
  permissions: write
  on:
    discussion: {types: [General]}
    schedule: [{cron: daily at 09:00}]
---
# Body
`
	for _, migration := range []Codemod{
		getSandboxFieldsCodemod(), getSafeOutputRequireTitlePrefixCodemod(),
		getSafeOutputMergePRConstraintsCodemod(), getSafeOutputAddReviewerAllowlistsCodemod(),
		getSafeOutputDispatchRepositoryKeyCodemod(), getSafeJobRunnerCodemod(),
		getMentionsAllowTeamMembersCodemod(), getAllowedReposCurrentToGitHubRepositoryCodemod(),
		getExpandPermissionsShorthandCodemod(), getMigrateWritePermissionsToReadCodemod(),
		getScheduleAtToAroundCodemod(), getDiscussionTriggerCategoriesLowercaseCodemod(),
		getExpiresIntegerToDayStringCodemod(), getBashSingleQuotedArgsCodemod(),
	} {
		t.Run(migration.ID, func(t *testing.T) {
			result, applied, err := migration.Apply(content, nil)
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, content, result)
		})
	}
}

func TestPolicyFrontmatterASTValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		codemod Codemod
		source  string
		path    []string
		value   any
	}{
		{getAllowedReposCurrentToGitHubRepositoryCodemod(), `'tools': {'github': {'allowed-repos': 'current', nested: {allowed-repos: current}}}`, []string{"tools", "github", "allowed-repos"}, "${{ github.repository }}"},
		{getExpandPermissionsShorthandCodemod(), `'permissions': 'read'`, []string{"permissions"}, "read-all"},
		{getMigrateWritePermissionsToReadCodemod(), `'permissions': {contents: 'write', id-token: write, copilot-requests: write}`, []string{"permissions"}, map[string]any{"contents": "read", "id-token": "write", "copilot-requests": "write"}},
		{getScheduleAtToAroundCodemod(), `'on': {schedule: [{cron: 'daily at 09:00'}], workflow_dispatch: {inputs: {schedule: {default: 'daily at 09:00'}}}}`, []string{"on", "schedule"}, []any{map[string]any{"cron": "daily around 09:00"}}},
		{getScheduleAtToAroundCodemod(), `'on': {schedule: [{cron: 'monthly on 15 at 14:00'}], workflow_dispatch: {inputs: {schedule: {default: 'monthly on 15 at 14:00'}}}}`, []string{"on", "schedule"}, []any{map[string]any{"cron": "0 9 15 * *"}}},
		{getDiscussionTriggerCategoriesLowercaseCodemod(), `'on': {discussion: {types: ['General # Chat', 'Mixed,Category', '${{ inputs.Category }}', 5], nested: {types: [Keep]}}}`, []string{"on", "discussion", "types"}, []any{"general # chat", "mixed,category", "${{ inputs.Category }}", uint64(5)}},
		{getExpiresIntegerToDayStringCodemod(), `'safe-outputs': {'create-issue': {'expires': 7, nested: {expires: 8}}, 'create-discussion': {expires: '9'}}`, []string{"safe-outputs", "create-issue", "expires"}, "7d"},
		{getBashSingleQuotedArgsCodemod(), `'tools': {bash: ["grep 'word'", 5, "echo 'unclosed"], nested: {bash: ["grep 'keep'"]}}`, []string{"tools", "bash"}, []any{`grep "word"`, uint64(5), "echo 'unclosed"}},
	}
	for _, tt := range tests {
		t.Run(tt.codemod.ID, func(t *testing.T) {
			content := "---\n" + tt.source + " # keep comment\nunrelated: 'exact'\n---\n\n# Body\n"
			result, applied, err := tt.codemod.Apply(content, map[string]any{"stale": true})
			require.NoError(t, err)
			require.True(t, applied)
			assertPolicyCodemodValue(t, result, tt.value, tt.path...)
			assert.Contains(t, result, "# keep comment")
			assert.Contains(t, result, "unrelated: 'exact'")
			assert.True(t, strings.HasSuffix(result, "---\n\n# Body\n"))
			original, err := aw.Parse(content)
			require.NoError(t, err)
			after, err := aw.Parse(result)
			require.NoError(t, err)
			assert.Equal(t, original.Markdown, after.Markdown)
			expected := original.Frontmatter
			for _, key := range tt.path[:len(tt.path)-1] {
				expected = expected[key].(map[string]any)
			}
			expected[tt.path[len(tt.path)-1]] = tt.value
			assert.Equal(t, original.Frontmatter, after.Frontmatter)
			second, changed, err := tt.codemod.Apply(result, nil)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, result, second)
		})
	}
}
