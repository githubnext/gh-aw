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
	assert.Contains(t, cmd.Long, "Copilot, Claude, Codex, Gemini, or Crush", "Add wizard help should mention all interactive engine options")
}

func TestAddWizardCommand_UsesStandardThreePartWorkflowSpecWording(t *testing.T) {
	cmd := NewAddWizardCommand(func(string) error { return nil })
	require.NotNil(t, cmd)

	assert.Contains(t, cmd.Long, `Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)`)
	assert.Contains(t, cmd.Long, "shorthand specs resolve on your enterprise host by default.")
	assert.Contains(t, cmd.Long, "For github/*, githubnext/*, and microsoft/*, shorthand resolves on github.com.")
	assert.Contains(t, cmd.Long, "Use full https://github.com/... URLs when sourcing other public github.com workflows.")
}
