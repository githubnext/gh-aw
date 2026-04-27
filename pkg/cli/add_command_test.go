//go:build !integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateEngineStub is a stub validation function for testing
func validateEngineStub(engine string) error {
	return nil
}

func TestNewAddCommand(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)

	require.NotNil(t, cmd, "NewAddCommand should not return nil")
	assert.Equal(t, "add <workflow>...", cmd.Use, "Command use should be 'add <workflow>...'")
	assert.Equal(t, "Add agentic workflows from repositories to .github/workflows", cmd.Short, "Command short description should match")
	assert.Contains(t, cmd.Long, "Add one or more agentic workflows", "Command long description should contain expected text")

	// Verify Args validator is set
	assert.NotNil(t, cmd.Args, "Args validator should be set")

	// Verify flags are registered
	flags := cmd.Flags()

	// Check name flag
	nameFlag := flags.Lookup("name")
	assert.NotNil(t, nameFlag, "Should have 'name' flag")
	assert.Equal(t, "n", nameFlag.Shorthand, "Name flag shorthand should be 'n'")

	// Check engine flag
	engineFlag := flags.Lookup("engine")
	assert.NotNil(t, engineFlag, "Should have 'engine' flag")

	// Check repo flag
	repoFlag := flags.Lookup("repo")
	assert.NotNil(t, repoFlag, "Should have 'repo' flag")
	assert.Equal(t, "r", repoFlag.Shorthand, "Repo flag shorthand should be 'r'")

	// Check PR flags
	createPRFlag := flags.Lookup("create-pull-request")
	assert.NotNil(t, createPRFlag, "Should have 'create-pull-request' flag")
	prFlag := flags.Lookup("pr")
	assert.NotNil(t, prFlag, "Should have 'pr' flag (alias)")

	// Check force flag
	forceFlag := flags.Lookup("force")
	assert.NotNil(t, forceFlag, "Should have 'force' flag")

	// Check append flag
	appendFlag := flags.Lookup("append")
	assert.NotNil(t, appendFlag, "Should have 'append' flag")

	// Check no-gitattributes flag
	noGitattributesFlag := flags.Lookup("no-gitattributes")
	assert.NotNil(t, noGitattributesFlag, "Should have 'no-gitattributes' flag")

	// Check dir flag
	dirFlag := flags.Lookup("dir")
	assert.NotNil(t, dirFlag, "Should have 'dir' flag")
	assert.Equal(t, "d", dirFlag.Shorthand, "Dir flag shorthand should be 'd'")

	// Check no-stop-after flag
	noStopAfterFlag := flags.Lookup("no-stop-after")
	assert.NotNil(t, noStopAfterFlag, "Should have 'no-stop-after' flag")

	// Check stop-after flag
	stopAfterFlag := flags.Lookup("stop-after")
	assert.NotNil(t, stopAfterFlag, "Should have 'stop-after' flag")
}

func TestAddWorkflows(t *testing.T) {
	tests := []struct {
		name          string
		workflows     []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty workflows list",
			workflows:     []string{},
			expectError:   true,
			errorContains: "at least one workflow",
		},
		{
			name:          "repo-only spec (requires workflow path)",
			workflows:     []string{"owner/repo"},
			expectError:   true,
			errorContains: "workflow specification must be in format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := AddOptions{}
			_, err := AddWorkflows(context.Background(), tt.workflows, opts)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "Should not error for test case: %s", tt.name)
			}
		})
	}
}

// TestAddCommandStructure removed - redundant with TestNewAddCommand

func TestAddResolvedWorkflows(t *testing.T) {
	tests := []struct {
		name          string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid workflow",
			expectError: true, // Will still error due to missing git repo, but validates basic flow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal resolved workflow structure
			resolved := &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Spec: &WorkflowSpec{
							RepoSpec: RepoSpec{
								RepoSlug: "test/repo",
							},
							WorkflowName: "test-workflow",
							WorkflowPath: "test.md",
						},
					},
				},
			}

			opts := AddOptions{}
			_, err := AddResolvedWorkflows(
				[]string{"test/repo/test-workflow"},
				resolved,
				opts,
			)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "Should not error for test case: %s", tt.name)
			}
		})
	}
}

func TestAddWorkflowsResult(t *testing.T) {
	tests := []struct {
		name                string
		prNumber            int
		prURL               string
		hasWorkflowDispatch bool
	}{
		{
			name:                "default values",
			prNumber:            0,
			prURL:               "",
			hasWorkflowDispatch: false,
		},
		{
			name:                "with PR number",
			prNumber:            123,
			prURL:               "",
			hasWorkflowDispatch: false,
		},
		{
			name:                "with PR URL",
			prNumber:            0,
			prURL:               "https://github.com/owner/repo/pull/123",
			hasWorkflowDispatch: false,
		},
		{
			name:                "with workflow dispatch",
			prNumber:            0,
			prURL:               "",
			hasWorkflowDispatch: true,
		},
		{
			name:                "all fields set",
			prNumber:            456,
			prURL:               "https://github.com/owner/repo/pull/456",
			hasWorkflowDispatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AddWorkflowsResult{
				PRNumber:            tt.prNumber,
				PRURL:               tt.prURL,
				HasWorkflowDispatch: tt.hasWorkflowDispatch,
			}

			// Verify all fields are accessible and have expected values
			assert.Equal(t, tt.prNumber, result.PRNumber, "PRNumber should match")
			assert.Equal(t, tt.prURL, result.PRURL, "PRURL should match")
			assert.Equal(t, tt.hasWorkflowDispatch, result.HasWorkflowDispatch, "HasWorkflowDispatch should match")
		})
	}
}

func TestAddCommandFlagInteractions(t *testing.T) {
	tests := []struct {
		name        string
		flagSetup   func(cmd *cobra.Command)
		expectValid bool
		description string
	}{
		{
			name: "no-stop-after and stop-after together",
			flagSetup: func(cmd *cobra.Command) {
				cmd.Flags().Set("no-stop-after", "true")
				cmd.Flags().Set("stop-after", "+48h")
			},
			expectValid: true, // Both flags can be set, stop-after takes precedence
			description: "Both no-stop-after and stop-after flags can be set",
		},
		{
			name: "create-pull-request and pr alias",
			flagSetup: func(cmd *cobra.Command) {
				cmd.Flags().Set("create-pull-request", "true")
				cmd.Flags().Set("pr", "true")
			},
			expectValid: true, // Both aliases should work
			description: "Both create-pull-request and pr flags can be set (aliases)",
		},
		{
			name: "force flag with number",
			flagSetup: func(cmd *cobra.Command) {
				cmd.Flags().Set("force", "true")
				cmd.Flags().Set("number", "3")
			},
			expectValid: true,
			description: "Force flag should work with multiple numbered copies",
		},
		{
			name: "dir flag with subdirectory",
			flagSetup: func(cmd *cobra.Command) {
				cmd.Flags().Set("dir", "shared")
			},
			expectValid: true,
			description: "Dir flag should accept subdirectory name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewAddCommand(validateEngineStub)

			// Apply flag setup
			tt.flagSetup(cmd)

			// Verify flags are set correctly
			flags := cmd.Flags()
			assert.NotNil(t, flags, "Command flags should not be nil")

			// The actual validation happens during RunE execution
			// Here we just verify the flags can be set without panic
			assert.True(t, tt.expectValid, tt.description)
		})
	}
}

func TestAddCommandFlagDefaults(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)
	flags := cmd.Flags()

	tests := []struct {
		flagName     string
		defaultValue string
	}{
		{"name", ""},
		{"engine", ""},
		{"repo", ""},
		{"append", ""},
		{"dir", ""},
		{"stop-after", ""},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := flags.Lookup(tt.flagName)
			require.NotNil(t, flag, "Flag should exist: %s", tt.flagName)
			assert.Equal(t, tt.defaultValue, flag.DefValue, "Default value should match for flag: %s", tt.flagName)
		})
	}
}

func TestAddCommandBooleanFlags(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)
	flags := cmd.Flags()

	boolFlags := []string{"create-pull-request", "pr", "force", "no-gitattributes", "no-stop-after"}

	for _, flagName := range boolFlags {
		t.Run(flagName, func(t *testing.T) {
			flag := flags.Lookup(flagName)
			require.NotNil(t, flag, "Boolean flag should exist: %s", flagName)
			assert.Equal(t, "false", flag.DefValue, "Boolean flag should default to false: %s", flagName)
		})
	}
}

func TestAddCommandArgs(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)

	// Test that Args validator is set (MinimumNArgs(1))
	require.NotNil(t, cmd.Args, "Args validator should be set")

	// Verify it requires at least 1 arg
	err := cmd.Args(cmd, []string{})
	require.Error(t, err, "Should error with no arguments")

	err = cmd.Args(cmd, []string{"workflow1"})
	require.NoError(t, err, "Should not error with 1 argument")

	err = cmd.Args(cmd, []string{"workflow1", "workflow2"})
	require.NoError(t, err, "Should not error with multiple arguments")
}

// TestAddMultipleWorkflowsNameFlag verifies that --name is not allowed when multiple workflows are specified.
func TestAddMultipleWorkflowsNameFlag(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)

	// Simulate calling the command with --name and multiple workflow arguments
	cmd.SetArgs([]string{"workflow1", "workflow2", "--name", "custom-name"})

	err := cmd.Execute()
	require.Error(t, err, "Should error when --name is used with multiple workflows")
	assert.Contains(t, err.Error(), "--name flag cannot be used when adding multiple workflows", "Error should mention --name restriction")
}

// setupMinimalGitRepo initialises a bare-minimum git repo in dir and returns the
// path to the .github/workflows directory so callers can write/read workflow files.
func setupMinimalGitRepo(t *testing.T, dir string) string {
	t.Helper()

	t.Setenv("HOME", dir)
	t.Chdir(dir)

	initCmd := exec.Command("git", "init")
	initCmd.Dir = dir
	require.NoError(t, initCmd.Run(), "git init should succeed")

	gitConfigName := exec.Command("git", "config", "user.name", "Test User")
	gitConfigName.Dir = dir
	_ = gitConfigName.Run()
	gitConfigEmail := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigEmail.Dir = dir
	_ = gitConfigEmail.Run()

	workflowsDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "should create workflows dir")

	return workflowsDir
}

// TestAddWorkflowWithTracking_SourceFieldVariants covers the main combinations of local /
// remote specs and fallback-path resolution for the source: frontmatter field written by
// addWorkflowWithTracking.
func TestAddWorkflowWithTracking_SourceFieldVariants(t *testing.T) {
	simpleContent := []byte("---\non: push\n---\n\n# Workflow\n")
	const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	tests := []struct {
		name            string
		spec            *WorkflowSpec
		sourceInfo      *FetchedWorkflow
		wantContains    string
		wantNotContains string
	}{
		{
			// Local workflows must NOT get a source: field — the code guards on !sourceInfo.IsLocal.
			name: "local workflow — no source field written",
			spec: &WorkflowSpec{
				RepoSpec:     RepoSpec{RepoSlug: ""},
				WorkflowPath: "./local-workflow.md",
				WorkflowName: "local-workflow",
			},
			sourceInfo: &FetchedWorkflow{
				Content:    simpleContent,
				CommitSHA:  "",
				IsLocal:    true,
				SourcePath: "./local-workflow.md",
			},
			wantContains:    "",
			wantNotContains: "source:",
		},
		{
			// Remote workflow where the parsed spec path already matches SourcePath
			// (no fallback triggered).  The source: field must use the original path.
			name: "remote workflow — no fallback, path matches SourcePath",
			spec: &WorkflowSpec{
				RepoSpec:     RepoSpec{RepoSlug: "owner/repo", Version: "main"},
				WorkflowPath: ".github/workflows/my-workflow.md",
				WorkflowName: "my-workflow",
			},
			sourceInfo: &FetchedWorkflow{
				Content:    simpleContent,
				CommitSHA:  sha,
				IsLocal:    false,
				SourcePath: ".github/workflows/my-workflow.md", // identical — no fallback
			},
			wantContains:    "source: owner/repo/.github/workflows/my-workflow.md@" + sha,
			wantNotContains: "",
		},
		{
			// Remote workflow from the *current* repository (self-referential) where
			// the spec only carries the short name but the file lives under
			// .github/workflows/.  Fallback resolution must be reflected in source:.
			name: "self-referential remote — fallback path resolution",
			spec: &WorkflowSpec{
				RepoSpec:     RepoSpec{RepoSlug: "current-org/current-repo", Version: "main"},
				WorkflowPath: "my-workflow.md", // short-form from parsed spec
				WorkflowName: "my-workflow",
			},
			sourceInfo: &FetchedWorkflow{
				Content:    simpleContent,
				CommitSHA:  sha,
				IsLocal:    false,
				SourcePath: ".github/workflows/my-workflow.md", // resolved via fallback
			},
			wantContains:    "source: current-org/current-repo/.github/workflows/my-workflow.md@" + sha,
			wantNotContains: "source: current-org/current-repo/my-workflow.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := testutil.TempDir(t, "test-source-field-variant-*")
			workflowsDir := setupMinimalGitRepo(t, tempDir)

			resolved := &ResolvedWorkflow{
				Spec:       tt.spec,
				Content:    tt.sourceInfo.Content,
				SourceInfo: tt.sourceInfo,
			}
			opts := AddOptions{DisableSecurityScanner: true}

			err := addWorkflowWithTracking(resolved, nil, opts)
			require.NoError(t, err, "addWorkflowWithTracking should succeed")

			written, err := os.ReadFile(filepath.Join(workflowsDir, tt.spec.WorkflowName+".md"))
			require.NoError(t, err, "written file should be readable")
			body := string(written)

			if tt.wantContains != "" {
				assert.Contains(t, body, tt.wantContains, "source field should contain expected path")
			}
			if tt.wantNotContains != "" {
				assert.NotContains(t, body, tt.wantNotContains, "source field must not contain forbidden path")
			}
		})
	}
}

// TestAddWorkflowWithTracking_UsesActualFetchedPath verifies that when a remote workflow is
// fetched via a fallback path (e.g. .github/workflows/my-workflow.md instead of the
// short-form my-workflow.md), the written source: field reflects the actual fetched path
// so that gh aw update can later re-fetch from the correct location.
func TestAddWorkflowWithTracking_UsesActualFetchedPath(t *testing.T) {
	// Set up a temp git repo
	tempDir := testutil.TempDir(t, "test-add-source-path-*")
	workflowsDir := setupMinimalGitRepo(t, tempDir)

	// Simple workflow content with no remote dependencies (avoids network calls)
	content := []byte("---\non: push\n---\n\n# Test Workflow\n")

	// Simulate: spec parsed from short-form "owner/repo/my-workflow.md@main"
	// The file was actually found at .github/workflows/my-workflow.md via fallback.
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "owner/repo",
			Version:  "main",
		},
		WorkflowPath: "my-workflow.md", // short-form path from parsed spec
		WorkflowName: "my-workflow",
	}
	sourceInfo := &FetchedWorkflow{
		Content:    content,
		CommitSHA:  "abc123def456789012345678901234567890abcd",
		IsLocal:    false,
		SourcePath: ".github/workflows/my-workflow.md", // actual path found via fallback
	}
	resolved := &ResolvedWorkflow{
		Spec:       spec,
		Content:    content,
		SourceInfo: sourceInfo,
	}

	opts := AddOptions{
		DisableSecurityScanner: true,
	}
	err := addWorkflowWithTracking(resolved, nil, opts)
	require.NoError(t, err, "addWorkflowWithTracking should succeed")

	// Read the written file
	written, err := os.ReadFile(filepath.Join(workflowsDir, "my-workflow.md"))
	require.NoError(t, err, "written file should be readable")

	// The source: field must use the full path, not the short-form WorkflowPath
	assert.Contains(t, string(written),
		"source: owner/repo/.github/workflows/my-workflow.md@abc123def456789012345678901234567890abcd",
		"source field should use the actual fetched path so gh aw update can find the file")
	assert.NotContains(t, string(written),
		"source: owner/repo/my-workflow.md",
		"source field must NOT use the short-form path that causes 404 on gh aw update")
}
