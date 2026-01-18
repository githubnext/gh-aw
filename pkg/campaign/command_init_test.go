package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignCommand_HasInitSubcommand(t *testing.T) {
	cmd := NewCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() == "init" {
			found = true
			break
		}
	}

	require.True(t, found, "expected campaign command to include 'init' subcommand")
}

func TestInitCampaignGenerator_CopiesWorkflowFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create source directory and file
	sourceDir := filepath.Join(tmpDir, ".github", "aw")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	sourceFile := filepath.Join(sourceDir, "campaign-generator.md")
	sourceContent := []byte("test workflow content")
	require.NoError(t, os.WriteFile(sourceFile, sourceContent, 0644))

	// Change to temp directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(oldCwd) })

	// Run init command
	err = initCampaignGenerator(false)
	require.NoError(t, err)

	// Verify destination file exists
	destFile := filepath.Join(tmpDir, ".github", "workflows", "campaign-generator.md")
	assert.FileExists(t, destFile)

	// Verify content matches
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, sourceContent, destContent)
}

func TestInitCampaignGenerator_RejectsExistingFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create source and destination directories
	sourceDir := filepath.Join(tmpDir, ".github", "aw")
	destDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.MkdirAll(destDir, 0755))

	// Create source file
	sourceFile := filepath.Join(sourceDir, "campaign-generator.md")
	require.NoError(t, os.WriteFile(sourceFile, []byte("source"), 0644))

	// Create existing destination file
	destFile := filepath.Join(destDir, "campaign-generator.md")
	require.NoError(t, os.WriteFile(destFile, []byte("existing"), 0644))

	// Change to temp directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(oldCwd) })

	// Run init command without force flag
	err = initCampaignGenerator(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Verify original content unchanged
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("existing"), destContent)
}

func TestInitCampaignGenerator_ForceFlagOverwrites(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create source and destination directories
	sourceDir := filepath.Join(tmpDir, ".github", "aw")
	destDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.MkdirAll(destDir, 0755))

	// Create source file
	sourceFile := filepath.Join(sourceDir, "campaign-generator.md")
	newContent := []byte("new content")
	require.NoError(t, os.WriteFile(sourceFile, newContent, 0644))

	// Create existing destination file
	destFile := filepath.Join(destDir, "campaign-generator.md")
	require.NoError(t, os.WriteFile(destFile, []byte("old content"), 0644))

	// Change to temp directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(oldCwd) })

	// Run init command with force flag
	err = initCampaignGenerator(true)
	require.NoError(t, err)

	// Verify content was overwritten
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, destContent)
}

func TestInitCampaignGenerator_ErrorWhenSourceMissing(t *testing.T) {
	// Create a temporary directory without source file
	tmpDir := t.TempDir()

	// Change to temp directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(oldCwd) })

	// Run init command
	err = initCampaignGenerator(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
