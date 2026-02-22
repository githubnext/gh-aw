//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportCache(t *testing.T) {
	tempDir := t.TempDir()

	// Create a new cache
	cache := NewImportCache(tempDir)

	// Test Set and Get
	testContent := []byte("# Test Workflow\n\nTest content")
	owner := "testowner"
	repo := "testrepo"
	path := "workflows/test.md"
	sha := "abc123"

	cachedPath, err := cache.Set(owner, repo, path, sha, testContent)
	require.NoError(t, err, "Set should succeed for valid inputs")

	// Verify file was created
	require.FileExists(t, cachedPath, "cache file should be created at expected path")

	// Verify content
	content, err := os.ReadFile(cachedPath)
	require.NoError(t, err, "reading cached file should succeed")
	assert.Equal(t, string(testContent), string(content), "cached content should match original")

	// Test Get
	retrievedPath, found := cache.Get(owner, repo, path, sha)
	assert.True(t, found, "cache entry should be found after Set")
	assert.Equal(t, cachedPath, retrievedPath, "retrieved path from Get should match path returned by Set")

	// Test that a new cache instance can find the file
	cache2 := NewImportCache(tempDir)
	retrievedPath2, found := cache2.Get(owner, repo, path, sha)
	assert.True(t, found, "cache entry should be found from new cache instance")
	assert.Equal(t, cachedPath, retrievedPath2, "path from new cache instance should match original cached path")
}

func TestImportCacheDirectory(t *testing.T) {
	tempDir := t.TempDir()

	cache := NewImportCache(tempDir)

	// Test cache directory path
	expectedDir := filepath.Join(tempDir, ImportCacheDir)
	assert.Equal(t, expectedDir, cache.GetCacheDir(), "GetCacheDir should return expected path")

	// Create a cache entry to trigger directory creation
	testContent := []byte("test")
	_, err := cache.Set("owner", "repo", "test.md", "sha1", testContent)
	require.NoError(t, err, "Set should succeed for valid inputs")

	// Verify directory was created
	assert.DirExists(t, expectedDir, "cache directory should be created after Set")

	// Verify .gitattributes was auto-generated
	gitAttributesPath := filepath.Join(expectedDir, ".gitattributes")
	require.FileExists(t, gitAttributesPath, ".gitattributes file should be created in cache directory")

	// Verify .gitattributes content
	content, err := os.ReadFile(gitAttributesPath)
	require.NoError(t, err, "reading .gitattributes should succeed")
	contentStr := string(content)
	assert.Contains(t, contentStr, "linguist-generated=true", ".gitattributes should mark files as linguist-generated")
	assert.Contains(t, contentStr, "merge=ours", ".gitattributes should set merge=ours strategy")
}

func TestImportCacheMissingFile(t *testing.T) {
	tempDir := t.TempDir()

	cache := NewImportCache(tempDir)

	// Add entry to cache
	testContent := []byte("test")
	cachedPath, err := cache.Set("owner", "repo", "test.md", "sha1", testContent)
	require.NoError(t, err, "Set should succeed for valid inputs")

	// Delete the cached file
	err = os.Remove(cachedPath)
	require.NoError(t, err, "removing cached file should succeed")

	// Try to get the entry - should return not found since file is missing
	_, found := cache.Get("owner", "repo", "test.md", "sha1")
	assert.False(t, found, "Get should return cache miss when backing file has been deleted")
}

func TestImportCacheEmptyCache(t *testing.T) {
	tempDir := t.TempDir()

	cache := NewImportCache(tempDir)

	// Try to get from empty cache - should return not found
	_, found := cache.Get("owner", "repo", "test.md", "nonexistent-sha")
	assert.False(t, found, "Get should return cache miss for empty cache")
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "workflows/test.md",
			expected: "workflows_test.md",
		},
		{
			name:     "nested path",
			input:    "a/b/c/file.md",
			expected: "a_b_c_file.md",
		},
		{
			name:     "already flat",
			input:    "file.md",
			expected: "file.md",
		},
		{
			name:     "path with dots cleaned",
			input:    "a/./b/file.md",
			expected: "a_b_file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizePath(%q) should return expected value", tt.input)
		})
	}
}

func TestValidatePathComponents(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		repo      string
		path      string
		sha       string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid components",
			owner:     "testowner",
			repo:      "testrepo",
			path:      "workflows/test.md",
			sha:       "abc123",
			shouldErr: false,
		},
		{
			name:      "empty owner",
			owner:     "",
			repo:      "testrepo",
			path:      "test.md",
			sha:       "abc123",
			shouldErr: true,
			errMsg:    "empty component",
		},
		{
			name:      "empty sha",
			owner:     "testowner",
			repo:      "testrepo",
			path:      "test.md",
			sha:       "",
			shouldErr: true,
			errMsg:    "empty component",
		},
		{
			name:      "path traversal in owner",
			owner:     "../etc",
			repo:      "testrepo",
			path:      "test.md",
			sha:       "abc123",
			shouldErr: true,
			errMsg:    "..",
		},
		{
			name:      "path traversal in path",
			owner:     "testowner",
			repo:      "testrepo",
			path:      "../../etc/passwd",
			sha:       "abc123",
			shouldErr: true,
			errMsg:    "..",
		},
		{
			name:      "absolute path in sha",
			owner:     "testowner",
			repo:      "testrepo",
			path:      "test.md",
			sha:       "/absolute/path",
			shouldErr: true,
			errMsg:    "absolute path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathComponents(tt.owner, tt.repo, tt.path, tt.sha)
			if tt.shouldErr {
				require.Error(t, err, "should return error for: %s", tt.name)
				assert.Contains(t, err.Error(), tt.errMsg, "error message should mention: %s", tt.errMsg)
			} else {
				assert.NoError(t, err, "should not return error for valid components")
			}
		})
	}
}

func TestImportCacheSet_Validation(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		owner     string
		repo      string
		path      string
		sha       string
		content   []byte
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "oversized content rejected",
			owner:     "owner",
			repo:      "repo",
			path:      "test.md",
			sha:       "abc123",
			content:   make([]byte, 10*1024*1024+1), // 10MB + 1 byte
			shouldErr: true,
			errMsg:    "exceeds maximum",
		},
		{
			name:      "path traversal in owner rejected",
			owner:     "../etc",
			repo:      "repo",
			path:      "test.md",
			sha:       "abc123",
			content:   []byte("content"),
			shouldErr: true,
			errMsg:    "invalid path components",
		},
		{
			name:      "path traversal in path rejected",
			owner:     "owner",
			repo:      "repo",
			path:      "../../etc/passwd",
			sha:       "abc123",
			content:   []byte("content"),
			shouldErr: true,
			errMsg:    "invalid path components",
		},
		{
			name:      "empty owner rejected",
			owner:     "",
			repo:      "repo",
			path:      "test.md",
			sha:       "abc123",
			content:   []byte("content"),
			shouldErr: true,
			errMsg:    "invalid path components",
		},
	}

	cache := NewImportCache(tempDir)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cache.Set(tt.owner, tt.repo, tt.path, tt.sha, tt.content)
			if tt.shouldErr {
				require.Error(t, err, "Set should reject: %s", tt.name)
				assert.Contains(t, err.Error(), tt.errMsg, "error message should contain %q", tt.errMsg)
			} else {
				assert.NoError(t, err, "Set should succeed for: %s", tt.name)
			}
		})
	}
}
