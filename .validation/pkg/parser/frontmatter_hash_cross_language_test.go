//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossLanguageHashCompatibility validates that Go and JavaScript implementations
// produce identical hashes for the same workflows.
//
// This test creates test workflows and verifies that both implementations produce
// matching hashes. The JavaScript implementation should eventually call the Go binary
// or implement the exact same algorithm.
func TestCrossLanguageHashCompatibility(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowFile := filepath.Join(tempDir, "test-workflow.md")

	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "FH-TV-001 empty frontmatter",
			content: `---
---

# Empty Workflow
`,
			expected: "4c8309afbcf816cd80c0824dce2b50047834b29e14b34b96953e88ae81048c46",
		},
		{
			name: "FH-TV-002 simple frontmatter",
			content: `---
engine: copilot
description: Test workflow
on:
  schedule: daily
---

# Test Workflow
`,
			expected: "b9def9907e3328e2e03e8c47c315723df39788f251627313b1a984bb61b9cbce",
		},
		{
			name: "FH-TV-003 complex frontmatter",
			content: `---
engine: claude
description: Complex workflow
tracker-id: complex-test
timeout-minutes: 30
on:
  schedule: daily
  workflow_dispatch: true
permissions:
  contents: read
  actions: read
tools:
  playwright:
    version: v1.41.0
labels:
  - test
  - complex
bots:
  - copilot
---

# Complex Workflow
`,
			expected: "8c63a05ef42cbfaff9be87a06257282cb4dcb952f71481d9d65ec3037003dbe8",
		},
	}

	cache := NewImportCache("")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write test workflow
			err := os.WriteFile(workflowFile, []byte(tc.content), 0644)
			require.NoError(t, err, "Should write test file")

			// Compute hash with Go implementation
			hash, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
			require.NoError(t, err, "Should compute hash")
			assert.Len(t, hash, 64, "Hash should be 64 characters")
			assert.Regexp(t, "^[a-f0-9]{64}$", hash, "Hash should be lowercase hex")
			assert.Equal(t, tc.expected, hash, "Hash should match specification vector")

			// For now, we just verify the Go implementation works
			// The JavaScript implementation will be tested separately
			// and should produce the same hash

			// Store the computed hash for reference
			t.Logf("Hash for %s: %s", tc.name, hash)

			// Verify determinism
			hash2, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
			require.NoError(t, err, "Should compute hash again")
			assert.Equal(t, hash, hash2, "Hash should be deterministic")
		})
	}
}

// TestFrontmatterHashVectorFH_TV_NEG_001 validates the oversized input rejection vector.
func TestFrontmatterHashVectorFH_TV_NEG_001(t *testing.T) {
	tempDir := t.TempDir()
	workflowFile := filepath.Join(tempDir, "oversized-workflow.md")
	oversizedDescription := strings.Repeat("a", maxFrontmatterHashInputBytes+1)

	require.NoError(t, os.WriteFile(workflowFile, []byte(`---
description: `+oversizedDescription+`
---

# Oversized Workflow
`), 0o644), "Should write oversized workflow file")

	cache := NewImportCache(tempDir)
	_, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.Error(t, err, "Should reject oversized normalized frontmatter input")
	require.EqualError(t, err, "frontmatter hash input exceeds 1048576 bytes after normalization")
}

// TestHashWithRealWorkflow tests hash computation with an actual workflow from the repository
func TestHashWithRealWorkflow(t *testing.T) {
	// Find a real workflow file
	repoRoot := findRepoRoot(t)
	workflowFile := filepath.Join(repoRoot, ".github", "workflows", "audit-workflows.md")

	// Check if file exists
	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		t.Skip("Real workflow file not found, skipping test")
		return
	}

	cache := NewImportCache(repoRoot)

	hash, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.NoError(t, err, "Should compute hash for real workflow")
	assert.Len(t, hash, 64, "Hash should be 64 characters")
	assert.Regexp(t, "^[a-f0-9]{64}$", hash, "Hash should be lowercase hex")

	t.Logf("Hash for audit-workflows.md: %s", hash)

	// Verify determinism
	hash2, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.NoError(t, err, "Should compute hash again")
	assert.Equal(t, hash, hash2, "Hash should be deterministic")
}

// TestHashWithTemplateExpressions tests hash computation including template expressions
func TestHashWithTemplateExpressions(t *testing.T) {
	tempDir := t.TempDir()
	workflowFile := filepath.Join(tempDir, "test-with-expressions.md")

	content := `---
engine: copilot
description: Test workflow with template expressions
---

# Test Workflow

Use environment variable: ${{ env.MY_VAR }}
Use config variable: ${{ vars.MY_CONFIG }}
Use github context: ${{ github.repository }}
`

	err := os.WriteFile(workflowFile, []byte(content), 0644)
	require.NoError(t, err, "Should write test file")

	cache := NewImportCache("")

	hash, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.NoError(t, err, "Should compute hash with template expressions")
	assert.Len(t, hash, 64, "Hash should be 64 characters")

	// Verify that changing template expressions changes the hash
	content2 := `---
engine: copilot
description: Test workflow with template expressions
---

# Test Workflow

Use environment variable: ${{ env.MY_VAR }}
Use config variable: ${{ vars.DIFFERENT_CONFIG }}
Use github context: ${{ github.repository }}
`

	workflowFile2 := filepath.Join(tempDir, "test-with-different-expressions.md")
	err = os.WriteFile(workflowFile2, []byte(content2), 0644)
	require.NoError(t, err, "Should write second test file")

	hash2, err := ComputeFrontmatterHashFromFile(workflowFile2, cache)
	require.NoError(t, err, "Should compute hash for second file")
	assert.NotEqual(t, hash, hash2, "Different template expressions should produce different hash")

	// Verify that non-env/vars expressions don't affect hash
	content3 := `---
engine: copilot
description: Test workflow with template expressions
---

# Test Workflow

Use environment variable: ${{ env.MY_VAR }}
Use config variable: ${{ vars.MY_CONFIG }}
Use github context: ${{ github.repository_owner }}
`

	workflowFile3 := filepath.Join(tempDir, "test-with-github-expression.md")
	err = os.WriteFile(workflowFile3, []byte(content3), 0644)
	require.NoError(t, err, "Should write third test file")

	hash3, err := ComputeFrontmatterHashFromFile(workflowFile3, cache)
	require.NoError(t, err, "Should compute hash for third file")
	assert.Equal(t, hash, hash3, "Non-env/vars github expressions should not affect hash")
}

// TestFrontmatterHashVectorFH_TV_004 validates the agent-import hash vector in the specification.
func TestFrontmatterHashVectorFH_TV_004(t *testing.T) {
	tempDir := t.TempDir()

	workflowFile := filepath.Join(tempDir, "workflow.md")
	agentsDir := filepath.Join(tempDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755), "Should create agents directory")

	require.NoError(t, os.WriteFile(workflowFile, []byte(`---
engine: copilot
imports:
  - ./agents/router.agent.md
  - ./agents/summarizer.agent.md
---

# Import-based Workflow
`), 0o644), "Should write workflow file")

	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "router.agent.md"), []byte(`---
description: Router agent
imports:
  - ./shared.agent.md
---

# Router
`), 0o644), "Should write router agent file")

	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "summarizer.agent.md"), []byte(`---
description: Summarizer agent
imports:
  - ./shared.agent.md
---

# Summarizer
`), 0o644), "Should write summarizer agent file")

	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "shared.agent.md"), []byte(`---
description: Shared helper
model: gpt-5
---

# Shared
`), 0o644), "Should write shared agent file")

	cache := NewImportCache(tempDir)
	hash, err := ComputeFrontmatterHashFromFile(workflowFile, cache)
	require.NoError(t, err, "Should compute hash for import graph")
	assert.Equal(t,
		"701dc12776a417c6ce4c82b16d1fcc9de343130efb554fda27a701386b17d134",
		hash,
		"FH-TV-004 hash should match specification vector")
}

// TestFrontmatterHashVectorFH_BFS_002 validates the FH-BFS-002 single-level import vector
// from the specification (§5.1 BFS Diamond-Import Test Vectors).
func TestFrontmatterHashVectorFH_BFS_002(t *testing.T) {
	tempDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "root.md"), []byte("---\nengine: copilot\nimports:\n  - ./helper.md\n---\n"), 0o644), "Should write root file")
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "helper.md"), []byte("---\ndescription: Helper agent\n---\n"), 0o644), "Should write helper file")

	cache := NewImportCache(tempDir)
	hash, err := ComputeFrontmatterHashFromFile(filepath.Join(tempDir, "root.md"), cache)
	require.NoError(t, err, "Should compute hash for single-level import")
	assert.Equal(t,
		"3946bb0dc0698a31e37a1efc7012071939db1be2c8365f12f8a240bc01ba2e9e",
		hash,
		"FH-BFS-002 hash should match specification vector")
}

// TestFrontmatterHashVectorFH_BFS_003 validates the FH-BFS-003 diamond-import deduplication
// vector from the specification (§5.1 BFS Diamond-Import Test Vectors).
// shared.md is reachable from both a.md and b.md but must appear only once in the hash input.
func TestFrontmatterHashVectorFH_BFS_003(t *testing.T) {
	tempDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "root.md"), []byte("---\nengine: copilot\nimports:\n  - ./a.md\n  - ./b.md\n---\n"), 0o644), "Should write root file")
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "a.md"), []byte("---\ndescription: Agent A\nimports:\n  - ./shared.md\n---\n"), 0o644), "Should write a.md")
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "b.md"), []byte("---\ndescription: Agent B\nimports:\n  - ./shared.md\n---\n"), 0o644), "Should write b.md")
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "shared.md"), []byte("---\ndescription: Shared helper\n---\n"), 0o644), "Should write shared.md")

	cache := NewImportCache(tempDir)
	hash, err := ComputeFrontmatterHashFromFile(filepath.Join(tempDir, "root.md"), cache)
	require.NoError(t, err, "Should compute hash for diamond-import graph")
	assert.Equal(t,
		"13f1c69f5761454beac63c7dc259fa212f020d3dab9e0dd04d2e1bdcc242b108",
		hash,
		"FH-BFS-003 hash should match specification vector")
}

// findRepoRoot finds the repository root directory
func findRepoRoot(t *testing.T) string {
	// Start from current directory and walk up to find .git
	dir, err := os.Getwd()
	require.NoError(t, err, "Should get current directory")

	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find repository root")
		}
		dir = parent
	}
}
