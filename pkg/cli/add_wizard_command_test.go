//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddWizardCommandMentionsCrush(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd, "Add wizard command should be created")
	assert.Contains(t, cmd.Long, "copilot, claude, codex, gemini, or crush", "Add wizard help should mention all interactive engine options")
}

func TestAddWizardCommand_UsesStandardThreePartWorkflowSpecWording(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	assert.Contains(t, cmd.Long, `Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)`)
	assert.Contains(t, cmd.Long, "shorthand source specs resolve on your enterprise host by default.")
	assert.Contains(t, cmd.Long, "For github/*, githubnext/*, and microsoft/* sources, shorthand resolves on github.com.")
	assert.Contains(t, cmd.Long, "Use full https://github.com/... source URLs for other public github.com workflows.")
}

func TestAddWizardCommand_NameFlagRejectsMultipleWorkflows(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd)
	cmd.SetArgs([]string{"workflow-one", "workflow-two", "--name", "custom-name"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.EqualError(t, err, "--name flag cannot be used when adding multiple workflows at once")
}

func TestAddWizardCommand_HasForceFlag(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	flag := cmd.Flags().Lookup("force")
	require.NotNil(t, flag, "add-wizard should register --force")
	assert.Equal(t, "bool", flag.Value.Type())
	shortFlag := cmd.Flags().ShorthandLookup("f")
	require.NotNil(t, shortFlag, "-f shorthand should be registered")
	assert.Equal(t, "force", shortFlag.Name)
}

func TestAddWizardCommand_HasAppendFlag(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	flag := cmd.Flags().Lookup("append")
	require.NotNil(t, flag, "add-wizard should register --append")
	assert.Equal(t, "string", flag.Value.Type())
}

func TestAddWizardCommand_HasNoSecurityScannerFlag(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	flag := cmd.Flags().Lookup("no-security-scanner")
	require.NotNil(t, flag, "add-wizard should register --no-security-scanner")
	assert.Equal(t, "bool", flag.Value.Type())
}
