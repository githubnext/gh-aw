//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests focus on --dir resolution and workflow discovery behavior for fix.
// Broader codemod behavior is covered in fix_command_test.go.

const timeoutMinutesWorkflow = `---
on:
  workflow_dispatch:

timeout_minutes: 30

permissions:
  contents: read
---

# Test Workflow
`

func writeFixWorkflow(t *testing.T, dir string, fileName string, content string) string {
	t.Helper()
	file := filepath.Join(dir, fileName)
	require.NoError(t, os.WriteFile(file, []byte(content), 0644))
	return file
}

func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })
	require.NoError(t, os.Chdir(dir))
}

func assertMigratedTimeoutField(t *testing.T, workflowFile string) {
	t.Helper()
	updatedContent, err := os.ReadFile(workflowFile)
	require.NoError(t, err)
	updatedStr := string(updatedContent)
	assert.NotContains(t, updatedStr, "timeout_minutes:")
	assert.Contains(t, updatedStr, "timeout-minutes: 30")
}

// TestFixWithDirFlag tests --dir behavior for custom and default workflow roots.
func TestFixWithDirFlag(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) string
		workflowDir func(workflowRoot string) string
		chdirToTmp  bool
	}{
		{
			name: "custom absolute dir",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()
				customDir := filepath.Join(tmpDir, "custom-workflows")
				require.NoError(t, os.MkdirAll(customDir, 0755))
				return customDir
			},
			workflowDir: func(workflowRoot string) string { return workflowRoot },
		},
		{
			name: "custom relative dir",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()
				customDir := filepath.Join(tmpDir, "custom-workflows")
				require.NoError(t, os.MkdirAll(customDir, 0755))
				return customDir
			},
			workflowDir: func(workflowRoot string) string { return filepath.Base(workflowRoot) },
			chdirToTmp:  true,
		},
		{
			name: "default dir when empty",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()
				defaultDir := filepath.Join(tmpDir, ".github", "workflows")
				require.NoError(t, os.MkdirAll(defaultDir, 0755))
				return defaultDir
			},
			workflowDir: func(workflowRoot string) string { return "" },
			chdirToTmp:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowRoot := tc.setup(t, tmpDir)

			if tc.chdirToTmp {
				chdirTemp(t, tmpDir)
			}

			workflowFile := writeFixWorkflow(t, workflowRoot, "test.md", timeoutMinutesWorkflow)

			config := FixConfig{
				WorkflowIDs: []string{},
				Write:       true,
				Verbose:     false,
				WorkflowDir: tc.workflowDir(workflowRoot),
			}

			require.NoError(t, RunFix(config))
			assertMigratedTimeoutField(t, workflowFile)
		})
	}
}

// TestFixWithDirFlagAndSpecificWorkflow tests --dir with specific workflow
func TestFixWithDirFlagAndSpecificWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	customDir := filepath.Join(tmpDir, "custom-workflows")
	require.NoError(t, os.MkdirAll(customDir, 0755))

	workflow1Content := `---
on: workflow_dispatch
timeout_minutes: 30
---
# Workflow 1
`
	workflow1File := writeFixWorkflow(t, customDir, "workflow1.md", workflow1Content)

	workflow2Content := `---
on: workflow_dispatch
timeout_minutes: 60
---
# Workflow 2
`
	workflow2File := writeFixWorkflow(t, customDir, "workflow2.md", workflow2Content)

	config := FixConfig{
		WorkflowIDs: []string{workflow1File},
		Write:       true,
		Verbose:     false,
		WorkflowDir: customDir,
	}
	require.NoError(t, RunFix(config))

	updated1Content, err := os.ReadFile(workflow1File)
	require.NoError(t, err)
	assert.Contains(t, string(updated1Content), "timeout-minutes: 30")
	assert.NotContains(t, string(updated1Content), "timeout_minutes:")

	updated2Content, err := os.ReadFile(workflow2File)
	require.NoError(t, err)
	assert.NotContains(t, string(updated2Content), "timeout-minutes:")
	assert.Contains(t, string(updated2Content), "timeout_minutes: 60")
}

func TestFixWithDirFlagEdgeCases(t *testing.T) {
	t.Run("nonexistent dir returns an error", func(t *testing.T) {
		tmpDir := t.TempDir()
		missingDir := filepath.Join(tmpDir, "does-not-exist")
		err := RunFix(FixConfig{
			Write:       true,
			WorkflowDir: missingDir,
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "no "+missingDir+" directory found")
	})

	t.Run("dir without markdown files is a no-op", func(t *testing.T) {
		tmpDir := t.TempDir()
		customDir := filepath.Join(tmpDir, "custom-workflows")
		require.NoError(t, os.MkdirAll(customDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(customDir, "notes.txt"), []byte("hello"), 0644))

		err := RunFix(FixConfig{
			Write:       true,
			WorkflowDir: customDir,
		})
		require.NoError(t, err)
	})
}

func TestResolveWorkflowRoot(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "nested under github workflows",
			filePath: filepath.Join("/tmp", "repo", ".github", "workflows", "nested", "foo.md"),
			expected: filepath.Join("/tmp", "repo", ".github", "workflows"),
		},
		{
			name:     "no github workflows segment falls back to dir",
			filePath: filepath.Join("/tmp", "repo", "workflows", "foo.md"),
			expected: filepath.Join("/tmp", "repo", "workflows"),
		},
		{
			name:     "messy path under github workflows is normalized",
			filePath: filepath.Join(".", "a", "..", "b", ".github", "workflows", "nested", "foo.md"),
			expected: filepath.Join("b", ".github", "workflows"),
		},
		{
			name:     "messy path without github workflows uses cleaned dir",
			filePath: filepath.Join(".", "a", "..", "b", "foo.md"),
			expected: filepath.Join("b"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolveWorkflowRoot(tc.filePath))
		})
	}
}

func TestWasCodemodApplied(t *testing.T) {
	testCases := []struct {
		name           string
		appliedCodemod []string
		codemodName    string
		want           bool
	}{
		{
			name:           "present codemod returns true",
			appliedCodemod: []string{"timeout-minutes-migration", "safe-output"},
			codemodName:    "safe-output",
			want:           true,
		},
		{
			name:           "missing codemod returns false",
			appliedCodemod: []string{"timeout-minutes-migration"},
			codemodName:    "safe-output",
			want:           false,
		},
		{
			name:           "empty list returns false",
			appliedCodemod: []string{},
			codemodName:    "safe-output",
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, wasCodemodApplied(tc.appliedCodemod, tc.codemodName))
		})
	}
}

func TestWasAnyCodemodApplied(t *testing.T) {
	testCases := []struct {
		name           string
		appliedCodemod []string
		codemodNames   []string
		want           bool
	}{
		{
			name:           "one of many codemods is present",
			appliedCodemod: []string{"timeout-minutes-migration", "safe-output"},
			codemodNames:   []string{"missing", "safe-output"},
			want:           true,
		},
		{
			name:           "none of the codemods are present",
			appliedCodemod: []string{"timeout-minutes-migration"},
			codemodNames:   []string{"missing", "safe-output"},
			want:           false,
		},
		{
			name:           "no codemod names provided",
			appliedCodemod: []string{"timeout-minutes-migration"},
			codemodNames:   nil,
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, wasAnyCodemodApplied(tc.appliedCodemod, tc.codemodNames...))
		})
	}
}
