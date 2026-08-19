//go:build !integration

package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectPurgeDataWithPatterns_BadLockPattern(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := collectPurgeDataWithPatterns(dir, nil, false, "[", "*.invalid.yml")
	require.Error(t, err)
	require.ErrorIs(t, err, filepath.ErrBadPattern)
}

func TestCollectPurgeDataWithPatterns_BadSecondPattern(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := collectPurgeDataWithPatterns(dir, nil, false, "*.lock.yml", "[")
	require.Error(t, err)
	require.ErrorIs(t, err, filepath.ErrBadPattern)
}

func TestCollectPurgeDataWithPatterns_ValidPatterns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	data, err := collectPurgeDataWithPatterns(dir, nil, false, "*.lock.yml", "*.invalid.yml")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Empty(t, data.existingLockFiles)
	require.Empty(t, data.existingInvalidFiles)
}
