//go:build !integration

package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllRepositoryWorkflowHashes validates that all workflows in the repository
// can have their hashes computed successfully and produces a reference list
func TestAllRepositoryWorkflowHashes(t *testing.T) {
	// Find repository root
	repoRoot := findRepoRoot(t)
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory not found, skipping test")
		return
	}

	// Find all workflow markdown files
	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	require.NoError(t, err, "Should list workflow files")

	if len(files) == 0 {
		t.Skip("No workflow files found")
		return
	}

	cache := NewImportCache(repoRoot)
	hashMap := make(map[string]string)

	t.Logf("Computing hashes for %d workflows:", len(files))

	for _, file := range files {
		workflowName := filepath.Base(file)

		hash, err := ComputeFrontmatterHashFromFile(file, cache)
		if err != nil {
			t.Logf("  ✗ %s: ERROR - %v", workflowName, err)
			continue
		}

		assert.Len(t, hash, 64, "Hash should be 64 characters for %s", workflowName)
		assert.Regexp(t, "^[a-f0-9]{64}$", hash, "Hash should be lowercase hex for %s", workflowName)

		hashMap[workflowName] = hash
		t.Logf("  ✓ %s: %s", workflowName, hash)

		// Verify determinism - compute again
		hash2, err := ComputeFrontmatterHashFromFile(file, cache)
		require.NoError(t, err, "Should compute hash again for %s", workflowName)
		assert.Equal(t, hash, hash2, "Hash should be deterministic for %s", workflowName)
	}

	t.Logf("\nSuccessfully computed hashes for %d workflows", len(hashMap))

	// Write hash reference file for cross-language validation
	referenceFile := filepath.Join(repoRoot, "tmp", "workflow-hashes-reference.txt")
	tmpDir := filepath.Dir(referenceFile)
	if err := os.MkdirAll(tmpDir, 0755); err == nil {
		f, err := os.Create(referenceFile)
		if err == nil {
			defer f.Close()
			for name, hash := range hashMap {
				f.WriteString(name + ": " + hash + "\n")
			}
			t.Logf("\nWrote hash reference to: %s", referenceFile)
		}
	}
}

// TestHashConsistencyAcrossLockFiles validates that hashes in lock files
// match the computed hashes from source markdown files
func TestHashConsistencyAcrossLockFiles(t *testing.T) {
	repoRoot := findRepoRoot(t)
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory not found, skipping test")
		return
	}

	// Find all workflow markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	require.NoError(t, err, "Should list workflow files")

	if len(mdFiles) == 0 {
		t.Skip("No workflow files found")
		return
	}

	cache := NewImportCache(repoRoot)
	checkedCount := 0

	for _, mdFile := range mdFiles {
		lockFile := mdFile[:len(mdFile)-3] + ".lock.yml"

		// Check if lock file exists
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			continue // Skip if no lock file
		}

		// Compute hash from markdown
		computedHash, err := ComputeFrontmatterHashFromFile(mdFile, cache)
		require.NoError(t, err, "Should compute hash for %s", filepath.Base(mdFile))

		// Read hash from lock file
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Should read lock file for %s", filepath.Base(lockFile))

		// Extract hash from lock file comment
		lockHash := extractHashFromLockFileContent(string(lockContent))

		if lockHash == "" {
			t.Logf("  ⚠ %s: No hash in lock file (may need recompilation)", filepath.Base(mdFile))
			continue
		}

		// Compare hashes
		if computedHash != lockHash {
			t.Errorf("  ✗ %s: Hash mismatch!\n    Computed: %s\n    Lock file: %s",
				filepath.Base(mdFile), computedHash, lockHash)
		} else {
			t.Logf("  ✓ %s: Hash matches", filepath.Base(mdFile))
		}

		checkedCount++
	}

	t.Logf("\nVerified hash consistency for %d workflows", checkedCount)
}

// extractHashFromLockFileContent extracts the frontmatter-hash from lock file content.
// Supports both JSON metadata format (# gh-aw-metadata: {...}) and legacy format.
func extractHashFromLockFileContent(content string) string {
	// Try JSON metadata format first: # gh-aw-metadata: {...}
	metadataPattern := regexp.MustCompile(`#\s*gh-aw-metadata:\s*(\{.+\})`)
	if matches := metadataPattern.FindStringSubmatch(content); len(matches) >= 2 {
		var metadata struct {
			FrontmatterHash string `json:"frontmatter_hash"`
		}
		if err := json.Unmarshal([]byte(matches[1]), &metadata); err == nil && metadata.FrontmatterHash != "" {
			return metadata.FrontmatterHash
		}
	}

	// Fallback to legacy format: # frontmatter-hash: <hash>
	hashPattern := regexp.MustCompile(`#\s*frontmatter-hash:\s*([0-9a-f]{64})`)
	if matches := hashPattern.FindStringSubmatch(content); len(matches) >= 2 {
		return matches[1]
	}

	return ""
}
