//go:build !integration

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeployCommand_BasicShape(t *testing.T) {
	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)
	assert.Equal(t, "deploy <workflow>...", cmd.Use)
	assert.Equal(t, "deploy", cmd.Name())
}

func TestNewDeployCommand_RequiresWorkflowArg(t *testing.T) {
	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	err := cmd.Args(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing workflow specification")
}

func TestNewDeployCommand_RegistersCoreFlags(t *testing.T) {
	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	expectedFlags := []string{
		"repo",
		"name",
		"engine",
		"force",
		"append",
		"no-gitattributes",
		"dir",
		"no-stop-after",
		"stop-after",
		"no-security-scanner",
		"cool-down",
	}

	for _, flagName := range expectedFlags {
		t.Run(flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(flagName)
			require.NotNil(t, flag, "expected flag %q to be registered", flagName)
		})
	}
}

func TestNewDeployCommand_CoolDownFlagUsageMatchesUpdate(t *testing.T) {
	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	coolDownFlag := cmd.Flags().Lookup("cool-down")
	require.NotNil(t, coolDownFlag)
	assert.Equal(t, coolDownFlagUsage, coolDownFlag.Usage)
}

func TestNewDeployCommand_RequiresRepoFlag(t *testing.T) {
	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)
	cmd.SetArgs([]string{"githubnext/agentics/ci-doctor"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repo flag is required")
}

func TestBuildDeployPRMetadata_SingleWorkflow(t *testing.T) {
	title, body := buildDeployPRMetadata([]string{"githubnext/agentics/ci-doctor"}, "owner/repo")
	assert.Equal(t, deployCommitMessage, title)
	assert.Contains(t, body, "Deploy ci-doctor to owner/repo.")
	assert.Contains(t, body, "compile --purge")
}

func TestBuildDeployPRMetadata_MultipleWorkflows(t *testing.T) {
	title, body := buildDeployPRMetadata([]string{"a", "b", "c"}, "owner/repo")
	assert.Equal(t, deployCommitMessage, title)
	assert.Contains(t, body, "Deploy 3 workflows to owner/repo.")
}

func TestExcludeExistingSourcedWorkflows_SkipsExistingSourcedWorkflow(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "ci-doctor.md"), []byte(`---
source: githubnext/agentics/ci-doctor.md@v1
---

# Existing
`), 0o644))

	toAdd, skipped, err := excludeExistingSourcedWorkflows([]string{"githubnext/agentics/ci-doctor"}, AddOptions{WorkflowDir: workflowsDir})
	require.NoError(t, err)
	assert.Empty(t, toAdd)
	assert.Equal(t, []string{"ci-doctor"}, skipped)
}

func TestExcludeExistingSourcedWorkflows_LeavesExistingNonSourcedWorkflowForAdd(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "ci-doctor.md"), []byte(`---
name: CI Doctor
---

# Existing
`), 0o644))

	toAdd, skipped, err := excludeExistingSourcedWorkflows([]string{"githubnext/agentics/ci-doctor"}, AddOptions{WorkflowDir: workflowsDir})
	require.NoError(t, err)
	assert.Equal(t, []string{"githubnext/agentics/ci-doctor"}, toAdd)
	assert.Empty(t, skipped)
}

func TestExcludeExistingSourcedWorkflows_UsesNameFlagForSingleWorkflow(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "custom-name.md"), []byte(`---
source: githubnext/agentics/ci-doctor.md@v1
---

# Existing
`), 0o644))

	toAdd, skipped, err := excludeExistingSourcedWorkflows(
		[]string{"githubnext/agentics/ci-doctor"},
		AddOptions{WorkflowDir: workflowsDir, Name: "custom-name"},
	)
	require.NoError(t, err)
	assert.Empty(t, toAdd)
	assert.Equal(t, []string{"custom-name"}, skipped)
}

func TestExcludeExistingSourcedWorkflows_MalformedFrontmatterNotSkipped(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "ci-doctor.md"), []byte(`---
source: [unterminated
---

# Existing
`), 0o644))

	toAdd, skipped, err := excludeExistingSourcedWorkflows([]string{"githubnext/agentics/ci-doctor"}, AddOptions{WorkflowDir: workflowsDir})
	require.NoError(t, err)
	assert.Equal(t, []string{"githubnext/agentics/ci-doctor"}, toAdd)
	assert.Empty(t, skipped)
}

func TestResolveDeployWorkflowSpecs_ResolvesRelativeLocalPathsAgainstOriginalDirectory(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	absoluteWorkflow := filepath.Join(baseDir, "absolute-workflow.md")
	workflows := resolveDeployWorkflowSpecs(
		[]string{"./my-workflow.md", absoluteWorkflow, "githubnext/agentics/ci-doctor"},
		baseDir,
	)
	require.Len(t, workflows, 3)

	assert.Equal(t, filepath.Join(baseDir, "my-workflow.md"), workflows[0])
	assert.Equal(t, absoluteWorkflow, workflows[1])
	assert.Equal(t, "githubnext/agentics/ci-doctor", workflows[2])
}

func TestResolveDeployWorkflowSpecs_ResolvesRelativeWildcardLocalPaths(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	workflows := resolveDeployWorkflowSpecs([]string{"./*.md"}, baseDir)
	require.Len(t, workflows, 1)

	assert.Equal(t, filepath.Join(baseDir, "*.md"), workflows[0])
}

func TestParseDeployCommandOptions_NameFlagWithMultipleWorkflows(t *testing.T) {
	t.Parallel()

	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)
	require.NoError(t, cmd.Flags().Set("name", "custom-workflow"))

	validateEngineCalled := false
	opts, coolDown, err := parseDeployCommandOptions(cmd, []string{"a", "b"}, func(string) error {
		validateEngineCalled = true
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name flag cannot be used when adding multiple workflows at once")
	assert.Equal(t, AddOptions{}, opts)
	assert.Zero(t, coolDown)
	assert.False(t, validateEngineCalled)
}

func TestParseDeployCommandOptions_InvalidCoolDown(t *testing.T) {
	t.Parallel()

	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)
	require.NoError(t, cmd.Flags().Set("cool-down", "not-a-duration"))

	opts, coolDown, err := parseDeployCommandOptions(cmd, []string{"a"}, func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --cool-down value")
	assert.Equal(t, AddOptions{}, opts)
	assert.Zero(t, coolDown)
}

func TestParseDeployCommandOptions_EngineValidationError(t *testing.T) {
	t.Parallel()

	cmd := NewDeployCommand(func(string) error { return nil })
	require.NotNil(t, cmd)
	require.NoError(t, cmd.Flags().Set("engine", "custom-engine"))

	var validatedEngine string
	expectedErr := errors.New("engine invalid")
	opts, coolDown, err := parseDeployCommandOptions(cmd, []string{"a"}, func(engine string) error {
		validatedEngine = engine
		return expectedErr
	})
	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, "custom-engine", validatedEngine)
	assert.Equal(t, AddOptions{}, opts)
	assert.Zero(t, coolDown)
}
