//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestActivationGitHubToken tests that on.github-token is extracted and used in the activation job
func TestActivationGitHubToken(t *testing.T) {
	compiler := NewCompiler()

	t.Run("custom_token_used_in_reaction_step", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:                  "Test Workflow",
			AIReaction:            "eyes",
			ActivationGitHubToken: "${{ secrets.MY_GITHUB_TOKEN }}",
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		assert.Contains(t, stepsStr, "github-token: ${{ secrets.MY_GITHUB_TOKEN }}", "Reaction step should use custom token")
		assert.NotContains(t, stepsStr, "github-token: ${{ secrets.GITHUB_TOKEN }}", "Reaction step should not use default GITHUB_TOKEN")
	})

	t.Run("default_token_used_when_no_custom_token", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:       "Test Workflow",
			AIReaction: "eyes",
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		assert.Contains(t, stepsStr, "github-token: ${{ secrets.GITHUB_TOKEN }}", "Reaction step should use default GITHUB_TOKEN")
	})

	t.Run("custom_token_used_in_status_comment", func(t *testing.T) {
		statusComment := true
		workflowData := &WorkflowData{
			Name:                  "Test Workflow",
			StatusComment:         &statusComment,
			ActivationGitHubToken: "${{ secrets.MY_GITHUB_TOKEN }}",
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		assert.Contains(t, stepsStr, "github-token: ${{ secrets.MY_GITHUB_TOKEN }}", "Add-comment step should use custom token")
	})

	t.Run("no_github_token_in_status_comment_when_using_default", func(t *testing.T) {
		statusComment := true
		workflowData := &WorkflowData{
			Name:          "Test Workflow",
			StatusComment: &statusComment,
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		// When using default token, no explicit github-token should be added to the add-comment step
		commentIdx := strings.Index(stepsStr, "id: add-comment")
		require.Greater(t, commentIdx, -1, "add-comment step should exist")
		// Check the section around add-comment
		addCommentSection := stepsStr[commentIdx:]
		nextStepIdx := strings.Index(addCommentSection[10:], "      - name:")
		if nextStepIdx > -1 {
			addCommentSection = addCommentSection[:nextStepIdx+10]
		}
		assert.NotContains(t, addCommentSection, "github-token:", "Add-comment step should not have explicit github-token when using default")
	})
}

// TestActivationGitHubApp tests that on.github-app is extracted and used in the activation job
func TestActivationGitHubApp(t *testing.T) {
	compiler := NewCompiler()

	t.Run("app_token_minted_before_reaction", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:       "Test Workflow",
			AIReaction: "eyes",
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		// Token mint step should appear before the reaction step
		mintIdx := strings.Index(stepsStr, "id: activation-app-token")
		reactIdx := strings.Index(stepsStr, "id: react")
		assert.Greater(t, mintIdx, -1, "Token mint step should be present")
		assert.Greater(t, reactIdx, -1, "Reaction step should be present")
		assert.Less(t, mintIdx, reactIdx, "Token mint step should appear before reaction step")

		// Reaction step should use the app token
		assert.Contains(t, stepsStr, "github-token: ${{ steps.activation-app-token.outputs.token }}", "Reaction step should use app token")
		// App-id and private-key should be in the mint step
		assert.Contains(t, stepsStr, "client-id: ${{ vars.APP_ID }}", "Mint step should contain client-id")
		assert.Contains(t, stepsStr, "private-key: ${{ secrets.APP_PRIVATE_KEY }}", "Mint step should contain private-key")
	})

	t.Run("app_token_minted_before_status_comment", func(t *testing.T) {
		statusComment := true
		workflowData := &WorkflowData{
			Name:          "Test Workflow",
			StatusComment: &statusComment,
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		// Token mint step should appear before the add-comment step
		mintIdx := strings.Index(stepsStr, "id: activation-app-token")
		commentIdx := strings.Index(stepsStr, "id: add-comment")
		assert.Greater(t, mintIdx, -1, "Token mint step should be present")
		assert.Greater(t, commentIdx, -1, "Add-comment step should be present")
		assert.Less(t, mintIdx, commentIdx, "Token mint step should appear before add-comment step")

		// Add-comment step should use the app token
		assert.Contains(t, stepsStr, "github-token: ${{ steps.activation-app-token.outputs.token }}", "Add-comment step should use app token")
	})

	t.Run("repositories_wildcard_omits_repositories_input_in_activation_mint_step", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:       "Test Workflow",
			AIReaction: "eyes",
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:        "${{ vars.APP_ID }}",
				PrivateKey:   "${{ secrets.APP_PRIVATE_KEY }}",
				Repositories: []string{"*"},
			},
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		mintIdx := strings.Index(stepsStr, "id: activation-app-token")
		require.Greater(t, mintIdx, -1, "Token mint step should be present")

		mintSection := stepsStr[mintIdx:]
		nextStepIdx := strings.Index(mintSection[len("id: activation-app-token"):], "      - name:")
		if nextStepIdx > -1 {
			mintSection = mintSection[:nextStepIdx+len("id: activation-app-token")]
		}

		assert.NotContains(t, mintSection, "repositories:", "Activation mint step should omit repositories when repositories is [\"*\"]")
	})

	t.Run("missing_key_ignore_adds_guard_and_fallback_token", func(t *testing.T) {
		statusComment := true
		workflowData := &WorkflowData{
			Name:          "Test Workflow",
			AIReaction:    "eyes",
			StatusComment: &statusComment,
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:           "${{ secrets.GH_AW_APP_ID }}",
				PrivateKey:      "${{ secrets.GH_AW_APP_PRIVATE_KEY }}",
				IgnoreIfMissing: true,
			},
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		// Both credentials use secrets.* so ignore-if-missing should guard on
		// step-local env aliases instead of secrets.* directly.
		assert.NotContains(t, stepsStr, "if: ${{ secrets.")
		assert.Contains(t, stepsStr, "GH_AW_IGNORE_IF_MISSING_APP_ID: ${{ secrets.GH_AW_APP_ID }}")
		assert.Contains(t, stepsStr, "GH_AW_IGNORE_IF_MISSING_PRIVATE_KEY: ${{ secrets.GH_AW_APP_PRIVATE_KEY }}")
		assert.Contains(t, stepsStr, "if: ${{ env.GH_AW_IGNORE_IF_MISSING_APP_ID != '' && env.GH_AW_IGNORE_IF_MISSING_PRIVATE_KEY != '' }}")
		assert.Contains(t, stepsStr, "github-token: ${{ steps.activation-app-token.outputs.token || secrets.GITHUB_TOKEN }}")
	})

	t.Run("app_token_minted_once_for_both_reaction_and_comment", func(t *testing.T) {
		statusComment := true
		workflowData := &WorkflowData{
			Name:          "Test Workflow",
			AIReaction:    "eyes",
			StatusComment: &statusComment,
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		// The token should be minted exactly once
		mintCount := strings.Count(stepsStr, "id: activation-app-token")
		assert.Equal(t, 1, mintCount, "Token mint step should appear exactly once")

		// Both steps should use the same app token
		assert.Contains(t, stepsStr, "id: react", "Reaction step should be present")
		assert.Contains(t, stepsStr, "id: add-comment", "Add-comment step should be present")
		// Reaction, comment, hash check, and daily guardrail steps should all use the same app token
		assert.Equal(t, 4, strings.Count(stepsStr, "github-token: ${{ steps.activation-app-token.outputs.token }}"), "Reaction, comment, hash check, and daily guardrail steps should all use app token")
	})
	t.Run("app_token_minted_for_hash_check_even_without_reaction_or_comment", func(t *testing.T) {
		// Regression test: when ActivationGitHubApp is set but no reaction/comment/label step
		// is configured, the mint step must still be generated because the hash check step
		// references ${{ steps.activation-app-token.outputs.token }}.
		workflowData := &WorkflowData{
			Name: "Test Workflow",
			// No AIReaction, no StatusComment, no LabelCommand
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
		require.NoError(t, err, "buildActivationJob should succeed")
		require.NotNil(t, job)

		stepsStr := strings.Join(job.Steps, "")
		// The token must be minted so the hash check step can reference it
		mintCount := strings.Count(stepsStr, "id: activation-app-token")
		assert.Equal(t, 1, mintCount, "Token mint step should appear exactly once even without reaction/comment")

		// Hash check step must reference the minted token
		assert.Contains(t, stepsStr, "id: check-lock-file", "Hash check step should be present")
		assert.Contains(t, stepsStr, "github-token: ${{ steps.activation-app-token.outputs.token }}",
			"Hash check step should use the minted app token")
	})
}

func TestActivationJobNeedsAppToken(t *testing.T) {
	newCtx := func(app *GitHubAppConfig) *activationJobBuildContext {
		return &activationJobBuildContext{data: &WorkflowData{
			ActivationGitHubApp: app,
			RawFrontmatter:      map[string]any{maxDailyAICreditsField: -1},
		}}
	}
	app := &GitHubAppConfig{AppID: "1", PrivateKey: "key"}

	tests := []struct {
		name   string
		ctx    *activationJobBuildContext
		config func(*activationJobBuildContext)
		want   bool
	}{
		{
			name: "no app configured",
			ctx:  newCtx(nil),
			config: func(ctx *activationJobBuildContext) {
				ctx.hasReaction = true
			},
			want: false,
		},
		{
			name: "reaction triggers app token",
			ctx:  newCtx(app),
			config: func(ctx *activationJobBuildContext) {
				ctx.hasReaction = true
			},
			want: true,
		},
		{
			name: "status comment triggers app token",
			ctx:  newCtx(app),
			config: func(ctx *activationJobBuildContext) {
				ctx.hasStatusComment = true
			},
			want: true,
		},
		{
			name: "remove label triggers app token",
			ctx:  newCtx(app),
			config: func(ctx *activationJobBuildContext) {
				ctx.shouldRemoveLabel = true
			},
			want: true,
		},
		{
			name: "access requirement triggers app token",
			ctx:  newCtx(app),
			config: func(ctx *activationJobBuildContext) {
				ctx.needsAppTokenForAccess = true
			},
			want: true,
		},
		{
			name: "no triggers with app",
			ctx:  newCtx(app),
			config: func(_ *activationJobBuildContext) {
			},
			want: false,
		},
		{
			name: "daily guardrail triggers app token",
			ctx:  newCtx(app),
			config: func(ctx *activationJobBuildContext) {
				ctx.data.RawFrontmatter = nil
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config(tt.ctx)
			assert.Equal(t, tt.want, activationJobNeedsAppToken(tt.ctx))
		})
	}
}

func TestActivationGitHubTokenExtraction(t *testing.T) {
	compiler := NewCompiler()

	t.Run("extracts_github_token_from_on_section", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_dispatch": nil,
				"github-token":      "${{ secrets.MY_TOKEN }}",
			},
		}

		token := compiler.extractActivationGitHubToken(frontmatter)
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", token, "Should extract github-token from on section")
	})

	t.Run("returns_empty_when_no_github_token", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_dispatch": nil,
			},
		}

		token := compiler.extractActivationGitHubToken(frontmatter)
		assert.Empty(t, token, "Should return empty string when github-token not set")
	})

	t.Run("extracts_github_app_from_on_section", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_dispatch": nil,
				"github-app": map[string]any{
					"app-id":      "${{ vars.APP_ID }}",
					"private-key": "${{ secrets.KEY }}",
				},
			},
		}

		app := compiler.extractActivationGitHubApp(frontmatter)
		require.NotNil(t, app, "Should extract github-app from on section")
		assert.Equal(t, "${{ vars.APP_ID }}", app.AppID, "App ID should match")
		assert.Equal(t, "${{ secrets.KEY }}", app.PrivateKey, "Private key should match")
	})

	t.Run("returns_nil_when_no_github_app", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_dispatch": nil,
			},
		}

		app := compiler.extractActivationGitHubApp(frontmatter)
		assert.Nil(t, app, "Should return nil when github-app not set")
	})
}

// TestActivationGitHubTokenCompiledWorkflow tests that github-token and github-app are
// properly handled in the generated workflow YAML
func TestActivationGitHubTokenCompiledWorkflow(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-github-token-test")
	compiler := NewCompiler()

	t.Run("github_token_used_in_reaction_step", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
  github-token: ${{ secrets.MY_TOKEN }}
  reaction: eyes
engine: copilot
---
Do something useful.
`
		mdPath := filepath.Join(tmpDir, "token-workflow.md")
		err := os.WriteFile(mdPath, []byte(workflowContent), 0600)
		require.NoError(t, err)

		lockPath := filepath.Join(tmpDir, "token-workflow.lock.yml")
		err = compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Compilation should succeed")

		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err)

		lockStr := string(lockContent)
		// github-token should NOT appear as an on: section key (it's either filtered or commented)
		assert.NotContains(t, lockStr, "\n    github-token: ${{ secrets.MY_TOKEN }}", "github-token should not appear as on: event key")

		// The token should be used in the reaction step
		assert.Contains(t, lockStr, "github-token: ${{ secrets.MY_TOKEN }}", "Token should be used in the reaction step")
	})

	t.Run("github_token_commented_when_no_reaction", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
  github-token: ${{ secrets.MY_TOKEN }}
engine: copilot
---
Do something useful.
`
		mdPath := filepath.Join(tmpDir, "token-only-workflow.md")
		err := os.WriteFile(mdPath, []byte(workflowContent), 0600)
		require.NoError(t, err)

		lockPath := filepath.Join(tmpDir, "token-only-workflow.lock.yml")
		err = compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Compilation should succeed")

		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err)

		lockStr := string(lockContent)
		// github-token should be commented out in the on: section
		assert.Contains(t, lockStr, "# github-token:", "github-token should be commented out in on section")
		assert.NotContains(t, lockStr, "\n  github-token: ${{ secrets.MY_TOKEN }}", "github-token should not appear uncommented in on section")
	})

	t.Run("github_app_token_minted_and_used_in_reaction", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_KEY }}
  reaction: eyes
engine: copilot
---
Do something useful.
`
		mdPath := filepath.Join(tmpDir, "app-workflow.md")
		err := os.WriteFile(mdPath, []byte(workflowContent), 0600)
		require.NoError(t, err)

		lockPath := filepath.Join(tmpDir, "app-workflow.lock.yml")
		err = compiler.CompileWorkflow(mdPath)
		require.NoError(t, err, "Compilation should succeed")

		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err)

		lockStr := string(lockContent)
		// The token mint step should be generated
		assert.Contains(t, lockStr, "id: activation-app-token", "Token mint step should be generated")
		assert.Contains(t, lockStr, "client-id: ${{ vars.APP_ID }}", "Token mint step should use client-id")
		assert.Contains(t, lockStr, "github-token: ${{ steps.activation-app-token.outputs.token }}", "Reaction step should use app token")
	})
}
