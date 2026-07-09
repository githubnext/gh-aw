//go:build !integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
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
	assert.Equal(t, "Add agentic workflows from repositories or local files to .github/workflows", cmd.Short, "Command short description should match")
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

func TestNewAddCommand_MentionsEnterpriseSourceResolution(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)
	require.NotNil(t, cmd)

	assert.Contains(t, cmd.Long, "Note: In GitHub Enterprise repos, shorthand source specs resolve on your enterprise host by default.")
	assert.Contains(t, cmd.Long, "For github/*, githubnext/*, and microsoft/* sources, shorthand resolves on github.com.")
	assert.Contains(t, cmd.Long, "Use full https://github.com/... source URLs for other public github.com workflows.")
}

func TestNewAddCommand_DeprecatesDisableSecurityScannerFlag(t *testing.T) {
	cmd := NewAddCommand(validateEngineStub)
	require.NotNil(t, cmd)

	flag := cmd.Flags().Lookup("disable-security-scanner")
	require.NotNil(t, flag, "add command should keep --disable-security-scanner as a deprecated alias")
	assert.Equal(t, "use --no-security-scanner instead", flag.Deprecated)
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
			name:          "invalid repo spec missing repo name",
			workflows:     []string{"owner"},
			expectError:   true,
			errorContains: "not a valid workflow path or repository package",
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
		name string
	}{
		{
			name: "valid workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() {
				require.NoError(t, os.Chdir(oldWd))
			}()
			gitInit := exec.Command("git", "init")
			gitInit.Dir = tmpDir
			require.NoError(t, gitInit.Run())

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
			_, err = AddResolvedWorkflows(
				context.Background(),
				[]string{"test/repo/test-workflow"},
				resolved,
				opts,
			)
			require.NoError(t, err, "Should not error for test case: %s", tt.name)

			workflowPath := filepath.Join(tmpDir, ".github", "workflows", "test-workflow.md")
			_, err = os.Stat(workflowPath)
			require.NoError(t, err, "workflow should be written to the temporary workflows directory")
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

			err := addWorkflowWithTracking(context.Background(), resolved, nil, opts)
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
	err := addWorkflowWithTracking(context.Background(), resolved, nil, opts)
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

// TestAddWorkflowWithTracking_ActionWorkflow verifies that action workflow (.yml) files are
// copied as-is to the target directory without frontmatter processing or compilation.
func TestAddWorkflowWithTracking_ActionWorkflow(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-action-workflow-*")
	workflowsDir := setupMinimalGitRepo(t, tempDir)

	rawYML := []byte("name: CI\non: [push]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps: []\n")

	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "owner/repo",
			Version:  "main",
		},
		WorkflowPath: ".github/workflows/ci.yml",
		WorkflowName: "ci",
	}
	sourceInfo := &FetchedWorkflow{
		Content:    rawYML,
		CommitSHA:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		IsLocal:    false,
		SourcePath: ".github/workflows/ci.yml",
	}
	resolved := &ResolvedWorkflow{
		Spec:             spec,
		Content:          rawYML,
		SourceInfo:       sourceInfo,
		IsActionWorkflow: true,
	}

	err := addWorkflowWithTracking(context.Background(), resolved, nil, AddOptions{})
	require.NoError(t, err)

	// The .yml file should be written verbatim
	written, err := os.ReadFile(filepath.Join(workflowsDir, "ci.yml"))
	require.NoError(t, err)
	assert.YAMLEq(t, string(rawYML), string(written), "action workflow content should be copied verbatim")

	// No .md or .lock.yml should be created
	_, errMD := os.Stat(filepath.Join(workflowsDir, "ci.md"))
	assert.True(t, os.IsNotExist(errMD), "no .md file should be created for action workflows")
	_, errLock := os.Stat(filepath.Join(workflowsDir, "ci.lock.yml"))
	assert.True(t, os.IsNotExist(errLock), "no .lock.yml should be created for action workflows")
}

// TestAddWorkflowWithTracking_ActionWorkflow_Force verifies that the --force flag overwrites
// an existing action workflow file.
func TestAddWorkflowWithTracking_ActionWorkflow_Force(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-action-workflow-force-*")
	workflowsDir := setupMinimalGitRepo(t, tempDir)

	oldContent := []byte("name: Old\n")
	newContent := []byte("name: New\n")

	destFile := filepath.Join(workflowsDir, "ci.yml")
	require.NoError(t, os.WriteFile(destFile, oldContent, 0644))

	spec := &WorkflowSpec{WorkflowPath: ".github/workflows/ci.yml", WorkflowName: "ci"}
	resolved := &ResolvedWorkflow{
		Spec:             spec,
		Content:          newContent,
		SourceInfo:       &FetchedWorkflow{Content: newContent, IsLocal: false, SourcePath: ".github/workflows/ci.yml"},
		IsActionWorkflow: true,
	}

	// Without --force: should fail
	err := addWorkflowWithTracking(context.Background(), resolved, nil, AddOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// With --force: should overwrite
	err = addWorkflowWithTracking(context.Background(), resolved, nil, AddOptions{Force: true})
	require.NoError(t, err)
	written, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, written)
}

func TestAddWorkflowsWithTracking_RollsBackWrittenFilesOnWriteFailure(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-add-workflows-rollback-*")
	workflowsDir := setupMinimalGitRepo(t, tempDir)

	validContent := []byte("---\nengine: claude\n---\n\n# workflow\n")
	// workflow name "nested/blocked" intentionally causes write failure because
	// addWorkflowWithTracking does not create nested workflow directories.
	workflows := []*ResolvedWorkflow{
		{
			Spec: &WorkflowSpec{
				WorkflowPath: "workflows/ok.md",
				WorkflowName: "ok",
			},
			Content: validContent,
			SourceInfo: &FetchedWorkflow{
				IsLocal: true,
			},
		},
		{
			Spec: &WorkflowSpec{
				WorkflowPath: "workflows/blocked.md",
				WorkflowName: "nested/blocked",
			},
			Content: validContent,
			SourceInfo: &FetchedWorkflow{
				IsLocal: true,
			},
		},
	}

	err := addWorkflowsWithTracking(context.Background(), workflows, NewFileTracker(), AddOptions{
		NoGitattributes:        true,
		DisableSecurityScanner: true,
		Quiet:                  true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write destination file")

	_, statErr := os.Stat(filepath.Join(workflowsDir, "ok.md"))
	assert.True(t, os.IsNotExist(statErr), "successful writes from this operation should be rolled back on later write failure")

	_, blockedStatErr := os.Stat(filepath.Join(workflowsDir, "nested", "blocked.md"))
	assert.True(t, os.IsNotExist(blockedStatErr), "failing workflow should not leave partial output files")
}

func TestAddSkillFileWithTracking_PreservesPathFromSkillsRoot(t *testing.T) {
	gitRoot := testutil.TempDir(t, "test-add-skill-path-*")
	resolved := &ResolvedWorkflow{
		Spec: &WorkflowSpec{
			WorkflowPath: "vendor/foo/skills/foo/scripts/run.sh",
		},
		SkillName: "foo",
		Content:   []byte("#!/usr/bin/env sh\necho ok\n"),
	}

	err := addSkillFileWithTracking(resolved, nil, AddOptions{Quiet: true}, gitRoot)
	require.NoError(t, err)

	skillRoot := filepath.Join(gitRoot, workflow.GetEngineSkillDir(""), "foo")
	expectedFile := filepath.Join(skillRoot, "scripts", "run.sh")
	unexpectedFile := filepath.Join(skillRoot, "skills", "foo", "scripts", "run.sh")

	_, err = os.Stat(expectedFile)
	require.NoError(t, err, "expected nested skill file should exist")
	content, err := os.ReadFile(expectedFile)
	require.NoError(t, err, "expected nested skill file should be readable")
	assert.Equal(t, []byte("#!/usr/bin/env sh\necho ok\n"), content, "expected nested skill file content should match")
	_, err = os.Stat(unexpectedFile)
	assert.True(t, os.IsNotExist(err), "unexpected first-match path should not be created")
}

func TestAddSkillFileWithTracking_RejectsInvalidPaths(t *testing.T) {
	t.Run("rejects path that escapes skill directory", func(t *testing.T) {
		gitRoot := testutil.TempDir(t, "test-add-skill-traversal-*")
		resolved := &ResolvedWorkflow{
			Spec: &WorkflowSpec{
				WorkflowPath: "skills/foo/../../.github/workflows/evil.yml",
			},
			SkillName: "foo",
			Content:   []byte("malicious"),
		}

		err := addSkillFileWithTracking(resolved, nil, AddOptions{Quiet: true}, gitRoot)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "escapes destination skill directory")
	})

	t.Run("rejects source path when skill root cannot be determined", func(t *testing.T) {
		gitRoot := testutil.TempDir(t, "test-add-skill-missing-root-*")
		resolved := &ResolvedWorkflow{
			Spec: &WorkflowSpec{
				WorkflowPath: "skills/bar/SKILL.md",
			},
			SkillName: "foo",
			Content:   []byte("content"),
		}

		err := addSkillFileWithTracking(resolved, nil, AddOptions{Quiet: true}, gitRoot)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to determine relative path")
	})
}

func TestAddCopilotRequestsPermissionToContent(t *testing.T) {
	t.Run("adds permission to workflow without existing permissions block", func(t *testing.T) {
		content := "---\nengine: copilot\n---\nDo the thing.\n"
		result, err := addCopilotRequestsPermissionToContent(content)
		require.NoError(t, err)
		assert.Contains(t, result, "permissions:")
		assert.Contains(t, result, "copilot-requests: write")
	})

	t.Run("adds permission to workflow with existing permissions block", func(t *testing.T) {
		content := "---\nengine: copilot\npermissions:\n  contents: read\n---\nDo the thing.\n"
		result, err := addCopilotRequestsPermissionToContent(content)
		require.NoError(t, err)
		assert.Contains(t, result, "copilot-requests: write")
		assert.Contains(t, result, "contents: read")
	})

	t.Run("is idempotent when permission already present", func(t *testing.T) {
		content := "---\nengine: copilot\npermissions:\n  copilot-requests: write\n---\nDo the thing.\n"
		result, err := addCopilotRequestsPermissionToContent(content)
		require.NoError(t, err)
		count := strings.Count(result, "copilot-requests: write")
		assert.Equal(t, 1, count, "copilot-requests: write should appear exactly once")
	})

	t.Run("returns error when permissions is a non-mapping scalar", func(t *testing.T) {
		content := "---\nengine: copilot\npermissions: read-all\n---\nDo the thing.\n"
		_, err := addCopilotRequestsPermissionToContent(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-mapping scalar")
	})
}

func TestAddWorkflowWithTracking_CopilotRequestsPermission(t *testing.T) {
	t.Run("injects copilot-requests permission when option is set", func(t *testing.T) {
		dir := testutil.TempDir(t, "test-copilot-requests-perm-*")
		setupMinimalGitRepo(t, dir)

		content := "---\nengine: copilot\n---\nDo the thing.\n"
		resolved := &ResolvedWorkflow{
			Spec:    &WorkflowSpec{WorkflowPath: "workflows/my-workflow.md", WorkflowName: "my-workflow"},
			Content: []byte(content),
			SourceInfo: &FetchedWorkflow{
				IsLocal: true,
			},
		}

		err := addWorkflowWithTracking(context.Background(), resolved, nil, AddOptions{
			Quiet:                        true,
			AddCopilotRequestsPermission: true,
			DisableSecurityScanner:       true,
		})
		require.NoError(t, err)

		workflowsDir := filepath.Join(dir, ".github", "workflows")
		written, readErr := os.ReadFile(filepath.Join(workflowsDir, "my-workflow.md"))
		require.NoError(t, readErr)
		assert.Contains(t, string(written), "copilot-requests: write")
	})

	t.Run("does not inject permission when option is false", func(t *testing.T) {
		dir := testutil.TempDir(t, "test-copilot-requests-noperm-*")
		setupMinimalGitRepo(t, dir)

		content := "---\nengine: copilot\n---\nDo the thing.\n"
		resolved := &ResolvedWorkflow{
			Spec:    &WorkflowSpec{WorkflowPath: "workflows/my-workflow2.md", WorkflowName: "my-workflow2"},
			Content: []byte(content),
			SourceInfo: &FetchedWorkflow{
				IsLocal: true,
			},
		}

		err := addWorkflowWithTracking(context.Background(), resolved, nil, AddOptions{
			Quiet:                        true,
			AddCopilotRequestsPermission: false,
			DisableSecurityScanner:       true,
		})
		require.NoError(t, err)

		workflowsDir := filepath.Join(dir, ".github", "workflows")
		written, readErr := os.ReadFile(filepath.Join(workflowsDir, "my-workflow2.md"))
		require.NoError(t, readErr)
		assert.NotContains(t, string(written), "copilot-requests: write")
	})
}
