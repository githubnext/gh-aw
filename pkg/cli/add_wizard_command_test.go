//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddWizardCommandMentionsEngines(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd, "Add wizard command should be created")
	assert.Contains(t, cmd.Long, "Copilot, Claude, Codex, Gemini, Antigravity, OpenCode, or Pi", "Add wizard help should mention all interactive engine options")
}

func TestAddWizardCommand_UsesStandardThreePartWorkflowSpecWording(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	assert.Contains(t, cmd.Long, `Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)`)
	assert.Contains(t, cmd.Long, "shorthand source specs resolve on your enterprise host by default.")
	assert.Contains(t, cmd.Long, "For github/*, githubnext/*, and microsoft/* sources, shorthand resolves on github.com.")
	assert.Contains(t, cmd.Long, "Use full https://github.com/... source URLs for other public github.com workflows.")
}

func TestAddWizardCommand_FlagUsageMatchesAddCommand(t *testing.T) {
	addCmd := NewAddCommand(validateEngineStub)
	wizardCmd := NewAddWizardCommand(validateEngineStub)

	for _, flagName := range []string{"append", "no-security-scanner"} {
		addFlag := addCmd.Flags().Lookup(flagName)
		wizardFlag := wizardCmd.Flags().Lookup(flagName)

		require.NotNil(t, addFlag, "add should define %q", flagName)
		require.NotNil(t, wizardFlag, "add-wizard should define %q", flagName)
		assert.Equal(t, addFlag.Usage, wizardFlag.Usage, "%q usage should stay in sync between add and add-wizard", flagName)
	}
}

func TestAddWizardCommand_DeprecatesDisableSecurityScannerFlag(t *testing.T) {
	cmd := NewAddWizardCommand(validateEngineStub)
	require.NotNil(t, cmd)

	flag := cmd.Flags().Lookup("disable-security-scanner")
	require.NotNil(t, flag, "add-wizard command should keep --disable-security-scanner as a deprecated alias")
	assert.Equal(t, "use --no-security-scanner instead", flag.Deprecated)
}

func TestAddWizardCommand_ExamplesMentionNewFlags(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	assert.Contains(t, cmd.Example, "--append \"custom footer\"", "add-wizard examples should show append usage")
	assert.Contains(t, cmd.Example, "--no-security-scanner", "add-wizard examples should show no-security-scanner usage")
}
