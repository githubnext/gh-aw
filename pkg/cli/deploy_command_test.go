//go:build !integration

package cli

import (
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
		"disable-security-scanner",
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
