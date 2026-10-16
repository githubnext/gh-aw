//go:build !integration

package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddResolvedWorkflowsAutoMerge(t *testing.T) {
	for _, tc := range []struct {
		name        string
		autoMerge   bool
		failCreate  bool
		failEnable  bool
		failRestore bool
	}{
		{name: "default"},
		{name: "enabled", autoMerge: true},
		{name: "enable failure preserves PR", autoMerge: true, failEnable: true},
		{name: "enable and restore failure preserves both errors", autoMerge: true, failEnable: true, failRestore: true},
		{name: "creation failure skips auto-merge", autoMerge: true, failCreate: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoDir := initPRTransferRepo(t)
			chdirForTest(t, repoDir)
			prependFakeGH(t, "exit 0\n")
			originalCreatePR := createPRRunGHContextWithHost
			originalAutoMerge := autoMergeRunGH
			originalSwitch := switchAddWorkflowPRBranch
			t.Cleanup(func() {
				createPRRunGHContextWithHost = originalCreatePR
				autoMergeRunGH = originalAutoMerge
				switchAddWorkflowPRBranch = originalSwitch
			})
			const prURL = "https://github.com/octo/repo/pull/42"
			created := false
			createdBranch := ""
			createErr := errors.New("PR creation failed")
			enableErr := errors.New("auto-merge unavailable")
			restoreErr := errors.New("branch restore failed")
			if tc.failRestore {
				switchAddWorkflowPRBranch = func(branch string, verbose bool) error {
					_ = originalSwitch(branch, verbose)
					return restoreErr
				}
			}
			createPRRunGHContextWithHost = func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
				require.Equal(t, []string{"pr", "create"}, args[:2])
				if tc.failCreate {
					return nil, createErr
				}
				created = true
				createdBranch = strings.TrimSpace(gitOutput(t, repoDir, "branch", "--show-current"))
				return []byte(prURL), nil
			}
			autoMergeCalls := 0
			autoMergeRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				assert.True(t, created, "auto-merge must happen after PR creation")
				autoMergeCalls++
				if tc.failEnable {
					return nil, enableErr
				}
				if strings.Contains(strings.Join(args, " "), "enablePullRequestAutoMerge") {
					return []byte(`{"data":{"enablePullRequestAutoMerge":{"pullRequest":{"id":"PR_42"}}}}`), nil
				}
				assert.Contains(t, args, "owner=octo")
				assert.Contains(t, args, "name=repo")
				assert.Contains(t, args, "number=42")
				return []byte(`{"data":{"repository":{"autoMergeAllowed":true,"viewerDefaultMergeMethod":"SQUASH","squashMergeAllowed":true,"pullRequest":{"id":"PR_42"}}}}`), nil
			}
			resolved := &ResolvedWorkflows{Workflows: []*ResolvedWorkflow{{
				Spec:             &WorkflowSpec{WorkflowName: "test-workflow", WorkflowPath: "test-workflow.yml"},
				Content:          []byte("name: Test\non: workflow_dispatch\njobs: {}\n"),
				IsActionWorkflow: true,
			}}}
			result, err := AddResolvedWorkflows(context.Background(), []string{"test-workflow.yml"}, resolved, AddOptions{
				CreatePR:        true,
				AutoMerge:       tc.autoMerge,
				RepoSlug:        "octo/repo",
				NoGitattributes: true,
				Quiet:           true,
			})
			switch {
			case tc.failCreate:
				require.ErrorIs(t, err, createErr)
				assert.Nil(t, result)
				assert.Zero(t, autoMergeCalls)
			case tc.failEnable:
				require.ErrorIs(t, err, enableErr)
				require.ErrorContains(t, err, prURL)
				if tc.failRestore {
					require.ErrorIs(t, err, restoreErr)
				}
				require.NotNil(t, result, "a created PR must remain available after auto-merge failure")
				assert.Equal(t, 42, result.PRNumber)
				assert.Equal(t, prURL, result.PRURL)
				assert.Equal(t, 1, autoMergeCalls)
			default:
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 42, result.PRNumber)
				assert.Equal(t, prURL, result.PRURL)
				if tc.autoMerge {
					assert.Equal(t, 2, autoMergeCalls)
				} else {
					assert.Zero(t, autoMergeCalls)
				}
			}
			assert.Equal(t, "main", strings.TrimSpace(gitOutput(t, repoDir, "branch", "--show-current")), "original branch must be restored")
			assert.Empty(t, strings.TrimSpace(gitOutput(t, repoDir, "status", "--porcelain")))
			if created {
				assert.Contains(t, gitOutput(t, repoDir, "show", createdBranch+":.github/workflows/test-workflow.yml"), "name: Test")
				assert.Contains(t, gitOutput(t, repoDir, "show", "origin/"+createdBranch+":.github/workflows/test-workflow.yml"), "name: Test")
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple workflow name",
			input:    "my-workflow",
			expected: "my-workflow",
		},
		{
			name:     "workflow with .md extension",
			input:    "my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "full path",
			input:    ".github/workflows/my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with spaces",
			input:    "my workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with special chars",
			input:    "my:workflow?.md",
			expected: "my-workflow",
		},
		{
			name:     "path with dots",
			input:    "my..workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with backslashes",
			input:    "path\\to\\workflow.md",
			expected: "path-to-workflow", // On Linux, backslashes are not path separators
		},
		{
			name:     "path with tilde",
			input:    "~my~workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with caret",
			input:    "my^workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with asterisk",
			input:    "my*workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with brackets",
			input:    "my[workflow].md",
			expected: "my-workflow",
		},
		{
			name:     "path with at-brace",
			input:    "my@{workflow}.md",
			expected: "my-workflow",
		},
		{
			name:     "consecutive special chars",
			input:    "my---workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "leading special chars",
			input:    "---my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "trailing special chars",
			input:    "my-workflow---.md",
			expected: "my-workflow",
		},
		{
			name:     "empty after sanitization",
			input:    "....md",
			expected: "workflow",
		},
		{
			name:     "underscores preserved",
			input:    "my_workflow.md",
			expected: "my_workflow",
		},
		{
			name:     "numbers preserved",
			input:    "workflow123.md",
			expected: "workflow123",
		},
		{
			name:     "mixed case preserved",
			input:    "MyWorkflow.md",
			expected: "MyWorkflow",
		},
		{
			name:     "unicode characters replaced",
			input:    "workflow-日本語.md",
			expected: "workflow",
		},
		{
			name:     "emoji replaced",
			input:    "workflow-🚀-test.md",
			expected: "workflow-test",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()+=",
			expected: "workflow",
		},
		{
			name:     "only dots",
			input:    "...",
			expected: "workflow",
		},
		{
			name:     "only hyphens",
			input:    "---",
			expected: "workflow",
		},
		{
			name:     "very long string truncation behavior",
			input:    "this-is-a-very-long-workflow-name-that-exceeds-typical-branch-name-lengths.md",
			expected: "this-is-a-very-long-workflow-name-that-exceeds-typical-branch-name-lengths",
		},
		{
			name:     "spaces only",
			input:    "     ",
			expected: "workflow",
		},
		{
			name:     "control characters",
			input:    "work\tflow\nname",
			expected: "work-flow-name",
		},
		{
			name:     "null bytes",
			input:    "work\x00flow",
			expected: "work-flow",
		},
		{
			name:     "mixed unicode and ascii",
			input:    "test-αβγ-workflow.md",
			expected: "test-workflow",
		},
		{
			name:     "accented characters",
			input:    "café-workflow.md",
			expected: "caf-workflow",
		},
		{
			name:     "cyrillic characters",
			input:    "workflow-работа.md",
			expected: "workflow",
		},
		{
			name:     "chinese characters only",
			input:    "工作流程.md",
			expected: "workflow",
		},
		{
			name:     "path separators extracts basename",
			input:    "a/b\\c/d.md",
			expected: "d", // normalizeWorkflowID extracts base name
		},
		{
			name:     "question mark and asterisk",
			input:    "test?file*.md",
			expected: "test-file",
		},
		{
			name:     "colon for windows paths",
			input:    "C:\\Users\\test.md",
			expected: "C-Users-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeBranchName(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizeBranchName(%q) should return %q", tt.input, tt.expected)
		})
	}
}

func TestBuildAddWorkflowPRBody(t *testing.T) {
	originalVersion := GetVersion()
	SetVersionInfo("v1.2.3")
	t.Cleanup(func() { SetVersionInfo(originalVersion) })

	workflow := &ResolvedWorkflow{
		Spec: &WorkflowSpec{
			RepoSpec:     RepoSpec{RepoSlug: "githubnext/agentics", Version: "main"},
			WorkflowPath: "workflows/repo-assist.md",
			WorkflowName: "repo-assist",
		},
		Content:     []byte("---\ndescription: Helps maintain the repository\non:\n  schedule: weekly\n  workflow_dispatch:\n---\n"),
		SourceInfo:  &FetchedWorkflow{CommitSHA: "abc123", SourcePath: "workflows/repo-assist.md"},
		Description: "Helps maintain the repository",
	}
	opts := AddOptions{
		EngineOverride: "copilot",
		addWizard: &addWizardOptions{
			secretSource: secretSourceOrganizationSelected,
			initializedFiles: []addInitializedFile{
				{path: "/home/user/repo/.gitattributes", displayPath: ".gitattributes"},
				{path: "/home/user/repo/.github/aw/actions-lock.json", displayPath: ".github/aw/actions-lock.json"},
			},
			disableGitHubAppPermissionInference: true,
		},
	}

	body := buildAddWorkflowPRBody([]*ResolvedWorkflow{workflow}, opts)

	require.Contains(t, body, "[`gh aw add-wizard`](https://github.github.com/gh-aw/)")
	assert.Contains(t, body, "[GitHub Agentic Workflows](https://github.com/github/gh-aw), version `v1.2.3`")
	assert.Contains(t, body, "[githubnext/agentics/workflows/repo-assist.md@main](https://github.com/githubnext/agentics/blob/abc123/workflows/repo-assist.md)")
	assert.Contains(t, body, "Helps maintain the repository")
	assert.Contains(t, body, "`schedule` (`weekly`), `workflow_dispatch`")
	assert.Contains(t, body, "**Delivery:** pull request")
	assert.Contains(t, body, "existing `COPILOT_GITHUB_TOKEN` organization (selected repository) secret")
	assert.Contains(t, body, "**GitHub App permission and event inference:** disabled")
	assert.Contains(t, body, "`.gitattributes`, `.github/aw/actions-lock.json`")
	assert.NotContains(t, body, "/home/user/repo")
	assert.Contains(t, body, "## Review criteria")
	assert.Contains(t, body, "## Forward progress")
	assert.NotContains(t, body, "will configure `COPILOT_GITHUB_TOKEN`")
}

func TestBuildAddWorkflowPRBodyUsesLocalSourceAndSecretNextStep(t *testing.T) {
	workflow := &ResolvedWorkflow{
		Spec:       &WorkflowSpec{WorkflowPath: "./review.md", WorkflowName: "review"},
		Content:    []byte("---\non: issues\n---\n"),
		SourceInfo: &FetchedWorkflow{IsLocal: true, SourcePath: "./review.md"},
	}
	opts := AddOptions{EngineOverride: "copilot", addWizard: &addWizardOptions{}}

	body := buildAddWorkflowPRBody([]*ResolvedWorkflow{workflow}, opts)

	assert.Contains(t, body, "`./review.md` (local source)")
	assert.Contains(t, body, "**Triggers:** `issues`")
	assert.Contains(t, body, "After merge, the add wizard will configure `COPILOT_GITHUB_TOKEN` when needed")
	assert.Contains(t, body, "recompile it with `gh aw compile`")
}

func TestBuildAddWorkflowPRBodyOmitsEmptyEngine(t *testing.T) {
	workflow := &ResolvedWorkflow{
		Spec:       &WorkflowSpec{WorkflowPath: "./review.md", WorkflowName: "review"},
		Content:    []byte("---\non: issues\n---\n"),
		SourceInfo: &FetchedWorkflow{IsLocal: true, SourcePath: "./review.md"},
	}

	body := buildAddWorkflowPRBody([]*ResolvedWorkflow{workflow}, AddOptions{})

	assert.NotContains(t, body, "**Engine:**")
}

func TestBuildAddWorkflowPRBodyPreservesDescriptionMarkdown(t *testing.T) {
	content := `---
description: |
  A friendly repository assistant.

  - Labels and triages open issues
  - Creates draft pull requests with fixes
on: workflow_dispatch
---
`
	workflow := &ResolvedWorkflow{
		Spec:        &WorkflowSpec{WorkflowPath: "./repo-assist.md", WorkflowName: "repo-assist"},
		Content:     []byte(content),
		SourceInfo:  &FetchedWorkflow{IsLocal: true, SourcePath: "./repo-assist.md"},
		Description: ExtractWorkflowDescription(content),
	}

	body := buildAddWorkflowPRBody([]*ResolvedWorkflow{workflow}, AddOptions{EngineOverride: "copilot"})

	assert.Contains(t, body, "A friendly repository assistant.\n\n- Labels and triages open issues\n- Creates draft pull requests with fixes")
	assert.NotContains(t, body, "assistant. - Labels")
}
