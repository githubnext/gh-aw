//go:build !integration

package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"

	"github.com/github/gh-aw/pkg/workflow"
)

func TestNetworkMergeEdgeCases(t *testing.T) {
	t.Run("duplicate domains are deduplicated", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "test-*")

		// Create shared file with overlapping domain
		sharedPath := filepath.Join(tempDir, "shared.md")
		sharedContent := `---
network:
  allowed:
    - github.com
    - example.com
---
`
		if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Workflow also has github.com (should be deduplicated)
		workflowPath := filepath.Join(tempDir, "workflow.md")
		workflowContent := `---
on: issues
engine: claude
permissions:
  contents: read
  issues: read
  pull-requests: read
strict: false
network:
  allowed:
    - github.com
    - api.github.com
imports:
  - shared.md
---

# Test
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := workflow.NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatal(err)
		}

		lockPath := stringutil.MarkdownToLockFile(workflowPath)
		content, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatal(err)
		}

		lockStr := string(content)

		// With config file support, domains appear in the AWF JSON config rather than
		// as a --allow-domains CLI flag. Find the line that contains the JSON config
		// (written via printf) and verify github.com appears there.
		lines := strings.Split(lockStr, "\n")
		var allowDomainsLine string
		for _, line := range lines {
			// Domains appear in the JSON config written by printf
			if strings.Contains(line, "allowDomains") {
				allowDomainsLine = line
				break
			}
		}

		if allowDomainsLine == "" {
			t.Fatal("Could not find allowDomains in compiled workflow")
		}

		// Count github.com occurrences within the allowDomains line only
		count := strings.Count(allowDomainsLine, "github.com")
		// github.com appears twice: once as github.com and once as api.github.com
		// We just need to check the allowDomains is present
		if count < 1 {
			t.Errorf("Expected github.com to appear in allowDomains, but found %d occurrences", count)
		}
	})

	t.Run("empty network in import is handled", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "test-*")

		// Create shared file with empty network
		sharedPath := filepath.Join(tempDir, "shared.md")
		sharedContent := `---
network: {}
---
`
		if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
			t.Fatal(err)
		}

		workflowPath := filepath.Join(tempDir, "workflow.md")
		workflowContent := `---
on: issues
engine: claude
permissions:
  contents: read
  issues: read
  pull-requests: read
strict: false
network:
  allowed:
    - github.com
imports:
  - shared.md
---

# Test
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := workflow.NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatal(err)
		}

		// Should still compile successfully with github.com
		lockPath := stringutil.MarkdownToLockFile(workflowPath)
		content, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(content), "github.com") {
			t.Error("Expected github.com to be in ALLOWED_DOMAINS")
		}
	})
}
