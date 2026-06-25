package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImpactCommandIsRegisteredHiddenAnalysisCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"impact"})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "impact", cmd.Name())
	assert.Equal(t, "analysis", cmd.GroupID)
	assert.True(t, cmd.Hidden)
}
