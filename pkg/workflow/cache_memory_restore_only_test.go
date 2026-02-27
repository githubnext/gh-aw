//go:build integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestCacheMemoryRestoreOnly(t *testing.T) {
	tests := []struct {
		name              string
		frontmatter       string
		expectedInLock    []string
		notExpectedInLock []string
	}{
		{
			name: "cache-memory with restore-only flag (object notation)",
			frontmatter: `---
name: Test Cache Memory Restore Only Object
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  cache-memory:
    restore-only: true
---`,
			expectedInLock: []string{
				"# Cache memory file share configuration from frontmatter processed below",
				"- name: Restore cache-memory file share data",
				"uses: actions/cache/restore@", // SHA varies, just check action name
				"key: memory-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory",
			},
			notExpectedInLock: []string{
				"- name: Upload cache-memory data as artifact",
				// Note: We can't use "uses: actions/cache@" here because cache/restore also matches
			},
		},
		{
			name: "cache-memory with restore-only in array notation",
			frontmatter: `---
name: Test Cache Memory Restore Only Array
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: readonly
      key: memory-readonly
      restore-only: true
---`,
			expectedInLock: []string{
				"# Cache memory file share configuration from frontmatter processed below",
				"- name: Cache cache-memory file share data (default)",
				"uses: actions/cache@", // SHA varies
				"key: memory-default-${{ github.run_id }}",
				"- name: Restore cache-memory file share data (readonly)",
				"uses: actions/cache/restore@", // SHA varies
				"key: memory-readonly-${{ github.run_id }}",
			},
			notExpectedInLock: []string{
				// Should NOT upload artifacts when detection is disabled
				"- name: Upload cache-memory data as artifact (default)",
				"name: cache-memory-default",
				"- name: Upload cache-memory data as artifact (readonly)",
				"name: cache-memory-readonly",
			},
		},
		{
			name: "cache-memory mixed restore-only and normal caches",
			frontmatter: `---
name: Test Cache Memory Mixed
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  cache-memory:
    - id: writeable
      key: memory-write
      restore-only: false
    - id: readonly1
      key: memory-read1
      restore-only: true
    - id: readonly2
      key: memory-read2
      restore-only: true
---`,
			expectedInLock: []string{
				"- name: Cache cache-memory file share data (writeable)",
				"uses: actions/cache@", // SHA varies
				"- name: Restore cache-memory file share data (readonly1)",
				"uses: actions/cache/restore@", // SHA varies
				"- name: Restore cache-memory file share data (readonly2)",
			},
			notExpectedInLock: []string{
				// Should NOT upload artifacts when detection is disabled
				"- name: Upload cache-memory data as artifact (writeable)",
				"name: cache-memory-writeable",
				"- name: Upload cache-memory data as artifact (readonly1)",
				"- name: Upload cache-memory data as artifact (readonly2)",
				"name: cache-memory-readonly1",
				"name: cache-memory-readonly2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir := t.TempDir()
			mdPath := filepath.Join(tmpDir, "test-workflow.md")

			// Write frontmatter + minimal prompt
			content := tt.frontmatter + "\n\nTest workflow for cache-memory restore-only flag.\n"
			if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write test markdown file: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(mdPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockPath := stringutil.MarkdownToLockFile(mdPath)
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockStr := string(lockContent)

			// Check expected strings are present
			for _, expected := range tt.expectedInLock {
				if !strings.Contains(lockStr, expected) {
					// Show a snippet of the lock file for context (first 100 lines)
					lines := strings.Split(lockStr, "\n")
					snippet := strings.Join(lines[:min(100, len(lines))], "\n")
					t.Errorf("Expected to find '%s' in lock file but it was missing.\nFirst 100 lines of lock file:\n%s\n...(truncated)", expected, snippet)
				}
			}

			// Check unexpected strings are NOT present
			for _, notExpected := range tt.notExpectedInLock {
				if strings.Contains(lockStr, notExpected) {
					// Find the line containing the unexpected string for context
					lines := strings.Split(lockStr, "\n")
					var contextLines []string
					for i, line := range lines {
						if strings.Contains(line, notExpected) {
							start := max(0, i-3)
							end := min(len(lines), i+4)
							contextLines = append(contextLines, fmt.Sprintf("Lines %d-%d:", start+1, end))
							contextLines = append(contextLines, lines[start:end]...)
							break
						}
					}
					t.Errorf("Did not expect to find '%s' in lock file but it was present.\nContext:\n%s", notExpected, strings.Join(contextLines, "\n"))
				}
			}
		})
	}
}
