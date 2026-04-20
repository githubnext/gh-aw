//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCommandHelpTextConsistency(t *testing.T) {
	cmd := NewUpgradeCommand()
	require.NotNil(t, cmd, "upgrade command should be created")

	assert.Contains(t, cmd.Long, "Upgrade the repository to the latest version of agentic workflows.", "long description should use correct grammar")

	approveFlag := cmd.Flags().Lookup("approve")
	require.NotNil(t, approveFlag, "--approve flag should exist")
	assert.Contains(t, approveFlag.Usage, "When strict mode is active", "--approve description should match compile semantics")

	preReleasesFlag := cmd.Flags().Lookup("pre-releases")
	require.NotNil(t, preReleasesFlag, "--pre-releases flag should exist")
	assert.Contains(t, preReleasesFlag.Usage, "Include pre-release versions", "--pre-releases description should mention pre-release upgrades")
}
