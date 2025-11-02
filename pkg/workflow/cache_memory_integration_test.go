package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheMemoryMultipleIntegration(t *testing.T) {
	tests := []struct {
		name              string
		frontmatter       string
		expectedInLock    []string
		notExpectedInLock []string
	}{
		{
			name: "single cache-memory (backward compatible)",
			frontmatter: `---
name: Test Cache Memory Single
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  cache-memory: true
  github:
    allowed: [get_repository]
---`,
			expectedInLock: []string{
				"# Cache memory file share configuration from frontmatter processed below",
				"- name: Create cache-memory directory",
				"- name: Cache memory file share data",
				"uses: actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830",
				"key: memory-${{ github.workflow }}-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory",
				"- name: Upload cache-memory data as artifact",
				"name: cache-memory",
				"## Cache Folder Available",
				"You have access to a persistent cache folder at `/tmp/gh-aw/cache-memory/`",
			},
			notExpectedInLock: []string{
				"## Cache Folders Available",
				"cache-memory/default/",
				"cache-memory/session/",
			},
		},
		{
			name: "multiple cache-memory with array notation",
			frontmatter: `---
name: Test Cache Memory Multiple
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session
  github:
    allowed: [get_repository]
---`,
			expectedInLock: []string{
				"# Cache memory file share configuration from frontmatter processed below",
				"- name: Create cache-memory directory (default)",
				"mkdir -p /tmp/gh-aw/cache-memory",
				"- name: Cache memory file share data (default)",
				"key: memory-default-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory",
				"- name: Upload cache-memory data as artifact (default)",
				"name: cache-memory-default",
				"- name: Create cache-memory directory (session)",
				"mkdir -p /tmp/gh-aw/cache-memory-session",
				"- name: Cache memory file share data (session)",
				"key: memory-session-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory-session",
				"- name: Upload cache-memory data as artifact (session)",
				"name: cache-memory-session",
				"## Cache Folders Available",
				"- **default**: `/tmp/gh-aw/cache-memory/`",
				"- **session**: `/tmp/gh-aw/cache-memory-session/`",
			},
			notExpectedInLock: []string{
				"## Cache Folder Available",
			},
		},
		{
			name: "multiple cache-memory without explicit keys",
			frontmatter: `---
name: Test Cache Memory Multiple No Keys
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  cache-memory:
    - id: data
    - id: logs
  github:
    allowed: [get_repository]
---`,
			expectedInLock: []string{
				"- name: Create cache-memory directory (data)",
				"mkdir -p /tmp/gh-aw/cache-memory-data",
				"key: memory-data-${{ github.workflow }}-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory-data",
				"- name: Create cache-memory directory (logs)",
				"mkdir -p /tmp/gh-aw/cache-memory-logs",
				"key: memory-logs-${{ github.workflow }}-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory-logs",
				"## Cache Folders Available",
				"- **data**: `/tmp/gh-aw/cache-memory-data/`",
				"- **logs**: `/tmp/gh-aw/cache-memory-logs/`",
			},
			notExpectedInLock: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tmpDir := t.TempDir()

			// Write the markdown file
			mdPath := filepath.Join(tmpDir, "test-workflow.md")
			content := tt.frontmatter + "\n\n# Test Workflow\n\nTest cache-memory configuration.\n"
			if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write test markdown file: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			if err := compiler.CompileWorkflow(mdPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockPath := strings.TrimSuffix(mdPath, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockStr := string(lockContent)

			// Check expected strings
			for _, expected := range tt.expectedInLock {
				if !strings.Contains(lockStr, expected) {
					t.Errorf("Expected to find '%s' in lock file but it was missing.\nLock file content:\n%s", expected, lockStr)
				}
			}

			// Check that unexpected strings are NOT present
			for _, notExpected := range tt.notExpectedInLock {
				if strings.Contains(lockStr, notExpected) {
					t.Errorf("Did not expect to find '%s' in lock file but it was present.\nLock file content:\n%s", notExpected, lockStr)
				}
			}
		})
	}
}
