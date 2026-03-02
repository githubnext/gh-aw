//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalSorted_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "test", `"test"`},
		{"number", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marshalSorted(tt.input)
			assert.Equal(t, tt.expected, result, "Should marshal primitive correctly")
		})
	}
}

func TestMarshalSorted_EmptyContainers(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"empty object", map[string]any{}, "{}"},
		{"empty array", []any{}, "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marshalSorted(tt.input)
			assert.Equal(t, tt.expected, result, "Should marshal empty container correctly")
		})
	}
}

func TestMarshalSorted_SortedKeys(t *testing.T) {
	input := map[string]any{
		"zebra":   1,
		"apple":   2,
		"banana":  3,
		"charlie": 4,
	}

	result := marshalSorted(input)
	expected := `{"apple":2,"banana":3,"charlie":4,"zebra":1}`
	assert.Equal(t, expected, result, "Keys should be sorted alphabetically")
}

func TestMarshalSorted_NestedSorting(t *testing.T) {
	input := map[string]any{
		"outer": map[string]any{
			"z": 1,
			"a": 2,
		},
		"another": map[string]any{
			"nested": map[string]any{
				"y": 3,
				"b": 4,
			},
		},
	}

	result := marshalSorted(input)
	// Keys at all levels should be sorted
	assert.Contains(t, result, `"another":`, "Should contain outer key")
	assert.Contains(t, result, `"outer":`, "Should contain outer key")
	assert.Contains(t, result, `"a":2`, "Should contain sorted nested keys")
	assert.Contains(t, result, `"z":1`, "Should contain sorted nested keys")
}

func TestComputeFrontmatterHashFromFile_NonExistent(t *testing.T) {
	cache := NewImportCache("")

	hash, err := ComputeFrontmatterHashFromFile("/nonexistent/file.md", cache)
	require.Error(t, err, "Should error for nonexistent file")
	assert.Empty(t, hash, "Hash should be empty on error")
}

func TestComputeFrontmatterHashFromFile_ValidFile(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowFile := filepath.Join(tempDir, "test-workflow.md")

	content := `---
engine: copilot
description: Test workflow
on:
  schedule: daily
---

# Test Workflow

This is a test workflow.
`

	err := os.WriteFile(workflowFile, []byte(content), 0644)
	require.NoError(t, err, "Should write test file")

	cache := NewImportCache("")

	hash, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.NoError(t, err, "Should compute hash from file")
	assert.Len(t, hash, 64, "Hash should be 64 characters")

	// Compute again to verify determinism
	hash2, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.NoError(t, err, "Should compute hash again")
	assert.Equal(t, hash, hash2, "Hash should be deterministic")
}

func TestComputeFrontmatterHash_WithImports(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create a shared workflow
	sharedDir := filepath.Join(tempDir, "shared")
	err := os.MkdirAll(sharedDir, 0755)
	require.NoError(t, err, "Should create shared directory")

	sharedFile := filepath.Join(sharedDir, "common.md")
	sharedContent := `---
tools:
  playwright:
    version: v1.41.0
labels:
  - shared
  - common
---

# Shared Content

This is shared.
`
	err = os.WriteFile(sharedFile, []byte(sharedContent), 0644)
	require.NoError(t, err, "Should write shared file")

	// Create a main workflow that imports the shared workflow
	mainFile := filepath.Join(tempDir, "main.md")
	mainContent := `---
engine: copilot
description: Main workflow
imports:
  - shared/common.md
labels:
  - main
---

# Main Workflow

This is the main workflow.
`
	err = os.WriteFile(mainFile, []byte(mainContent), 0644)
	require.NoError(t, err, "Should write main file")

	cache := NewImportCache("")

	hash, err := ComputeFrontmatterHashFromFile(mainFile, cache)
	require.NoError(t, err, "Should compute hash with imports")
	assert.Len(t, hash, 64, "Hash should be 64 characters")

	// The hash should include contributions from the imported file
	// We can't easily verify the exact hash, but we can verify it's deterministic
	hash2, err := ComputeFrontmatterHashFromFile(mainFile, cache)
	require.NoError(t, err, "Should compute hash again with imports")
	assert.Equal(t, hash, hash2, "Hash with imports should be deterministic")
}

func TestComputeFrontmatterHashFromFileWithReader_CustomReader(t *testing.T) {
	// Create in-memory file system mock
	mockFS := map[string]string{
		"/test/workflow.md": `---
engine: copilot
description: Test workflow
---

# Workflow Body`,
		"/test/shared/imported.md": `---
tools:
  bash: true
---

# Imported Content`,
	}

	// Create custom file reader
	customReader := func(filePath string) ([]byte, error) {
		content, exists := mockFS[filePath]
		if !exists {
			return nil, os.ErrNotExist
		}
		return []byte(content), nil
	}

	// Test basic hash computation
	hash, err := ComputeFrontmatterHashFromFileWithReader("/test/workflow.md", nil, customReader)
	require.NoError(t, err, "Should compute hash with custom reader")
	assert.Len(t, hash, 64, "Hash should be 64 characters")
	assert.Regexp(t, "^[a-f0-9]{64}$", hash, "Hash should be lowercase hex")

	// Verify determinism
	hash2, err := ComputeFrontmatterHashFromFileWithReader("/test/workflow.md", nil, customReader)
	require.NoError(t, err, "Should compute hash again")
	assert.Equal(t, hash, hash2, "Hash should be deterministic")
}

func TestComputeFrontmatterHashFromFileWithReader_WithImports(t *testing.T) {
	// Create in-memory file system mock with imports
	mockFS := map[string]string{
		"/test/workflow.md": `---
engine: copilot
imports:
  - shared/imported.md
---

# Main Workflow`,
		"/test/shared/imported.md": `---
tools:
  bash: true
---

# Imported Content`,
	}

	// Create custom file reader
	customReader := func(filePath string) ([]byte, error) {
		content, exists := mockFS[filePath]
		if !exists {
			return nil, os.ErrNotExist
		}
		return []byte(content), nil
	}

	// Test hash computation with imports
	hash, err := ComputeFrontmatterHashFromFileWithReader("/test/workflow.md", nil, customReader)
	require.NoError(t, err, "Should compute hash with imports using custom reader")
	assert.Len(t, hash, 64, "Hash should be 64 characters")
	assert.Regexp(t, "^[a-f0-9]{64}$", hash, "Hash should be lowercase hex")
}
