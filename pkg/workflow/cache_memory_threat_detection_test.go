package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCacheMemoryWithThreatDetection tests that cache-memory uses restore-only mode
// when threat detection is enabled, and that an update_cache_memory job is created
func TestCacheMemoryWithThreatDetection(t *testing.T) {
	tests := []struct {
		name              string
		frontmatter       string
		expectedInLock    []string
		notExpectedInLock []string
	}{
		{
			name: "cache-memory with threat detection enabled",
			frontmatter: `---
name: Test Cache Memory with Threat Detection
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  cache-memory: true
safe-outputs:
  create-issue:
    labels: ["test"]
---

# Test workflow

Test the cache-memory and threat detection integration.
`,
			expectedInLock: []string{
				// Cache should use restore-only mode in agent job
				"- name: Restore cache memory file share data",
				"actions/cache/restore@0057852bfaa89a56745cba8c7296529d2fc39830",
				"key: memory-${{ github.workflow }}-${{ github.run_id }}",
				"path: /tmp/gh-aw/cache-memory",
				// Artifact should still be uploaded
				"- name: Upload cache-memory data as artifact",
				"if: always()",
				"name: cache-memory",
				// Detection job should exist
				"detection:",
				"needs: agent",
				// update_cache_memory job should exist
				"update_cache_memory:",
				"needs:",
				"- agent",
				"- detection",
				"if: needs.detection.outputs.success == 'true'",
				"- name: Download cache-memory artifact",
				"- name: Save cache memory",
				"actions/cache/save@0057852bfaa89a56745cba8c7296529d2fc39830",
			},
			notExpectedInLock: []string{
				// Should NOT use regular cache action in agent job
				"- name: Cache memory file share data\n",
			},
		},
		{
			name: "cache-memory without threat detection",
			frontmatter: `---
name: Test Cache Memory without Threat Detection
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
tools:
  cache-memory: true
---

# Test workflow

Test cache-memory without threat detection.
`,
			expectedInLock: []string{
				// Cache should use normal cache action
				"- name: Cache memory file share data",
				"actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830",
				"key: memory-${{ github.workflow }}-${{ github.run_id }}",
			},
			notExpectedInLock: []string{
				// Should NOT have detection job
				"detection:",
				// Should NOT have update_cache_memory job
				"update_cache_memory:",
				// Should NOT use restore-only action
				"actions/cache/restore@",
			},
		},
		{
			name: "multiple cache-memory with threat detection",
			frontmatter: `---
name: Test Multiple Cache Memory with Threat Detection
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session
safe-outputs:
  create-issue:
    labels: ["test"]
---

# Test workflow

Test multiple cache-memory entries with threat detection.
`,
			expectedInLock: []string{
				// Both caches should use restore-only mode
				"- name: Restore cache memory file share data (default)",
				"actions/cache/restore@0057852bfaa89a56745cba8c7296529d2fc39830",
				"- name: Restore cache memory file share data (session)",
				"actions/cache/restore@0057852bfaa89a56745cba8c7296529d2fc39830",
				// Both artifacts should be uploaded
				"- name: Upload cache-memory data as artifact (default)",
				"name: cache-memory-default",
				"- name: Upload cache-memory data as artifact (session)",
				"name: cache-memory-session",
				// update_cache_memory job should handle both
				"update_cache_memory:",
				"- name: Download cache-memory-default artifact",
				"name: cache-memory-default",
				"- name: Download cache-memory-session artifact",
				"name: cache-memory-session",
				"- name: Save cache memory (default)",
				"key: memory-default-${{ github.run_id }}",
				"- name: Save cache memory (session)",
				"key: memory-session-${{ github.run_id }}",
			},
			notExpectedInLock: []string{
				// Should NOT use regular cache action
				"- name: Cache memory file share data (default)",
				"- name: Cache memory file share data (session)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()
			mdPath := filepath.Join(tempDir, "test.md")

			// Write markdown file
			if err := os.WriteFile(mdPath, []byte(tt.frontmatter), 0600); err != nil {
				t.Fatalf("Failed to write markdown file: %v", err)
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

			// Check expected strings are present
			for _, expected := range tt.expectedInLock {
				if !strings.Contains(lockStr, expected) {
					t.Errorf("Expected string not found in lock file:\n%s\n\nLock file content:\n%s", expected, lockStr)
				}
			}

			// Check unexpected strings are absent
			for _, notExpected := range tt.notExpectedInLock {
				if strings.Contains(lockStr, notExpected) {
					t.Errorf("Unexpected string found in lock file:\n%s", notExpected)
				}
			}
		})
	}
}
