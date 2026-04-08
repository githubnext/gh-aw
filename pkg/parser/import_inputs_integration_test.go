//go:build integration

package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestImportInputsForwardedToNestedImports_Integration verifies via the parser package
// API that ${{ github.aw.import-inputs.* }} expressions inside an imported workflow's
// imports: frontmatter section are resolved before nested import discovery, enabling
// multi-level shared-workflow composition.
func TestImportInputsForwardedToNestedImports_Integration(t *testing.T) {
	tempDir := testutil.TempDir(t, "import-inputs-forwarding-*")

	sharedDir := filepath.Join(tempDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Leaf shared workflow — accepts branch-name via import-schema
	leafPath := filepath.Join(sharedDir, "repo-memory.md")
	leafContent := `---
import-schema:
  branch-name:
    type: string
    required: true
    description: "Branch name for storage"
tools:
  bash:
    - "git *"
---

Store data in branch ${{ github.aw.import-inputs.branch-name }}.
`
	if err := os.WriteFile(leafPath, []byte(leafContent), 0644); err != nil {
		t.Fatalf("Failed to write leaf workflow: %v", err)
	}

	// Intermediate shared workflow — accepts branch-name and forwards it to the leaf
	// via an expression in its own imports: section
	intermediateContent := `---
import-schema:
  branch-name:
    type: string
    required: true
    description: "Branch name for repo-memory storage"

imports:
  - uses: shared/repo-memory.md
    with:
      branch-name: ${{ github.aw.import-inputs.branch-name }}
---

Daily report workflow.
`
	intermediatePath := filepath.Join(sharedDir, "daily-report.md")
	if err := os.WriteFile(intermediatePath, []byte(intermediateContent), 0644); err != nil {
		t.Fatalf("Failed to write intermediate workflow: %v", err)
	}

	// Consumer workflow — imports the intermediate with a concrete value
	consumerContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/daily-report.md
    with:
      branch-name: "memory/my-workflow"
---

Consumer workflow.
`
	consumerPath := filepath.Join(tempDir, "consumer.md")
	if err := os.WriteFile(consumerPath, []byte(consumerContent), 0644); err != nil {
		t.Fatalf("Failed to write consumer workflow: %v", err)
	}

	// Parse the consumer workflow's frontmatter and process its imports
	result, err := parser.ExtractFrontmatterFromContent(consumerContent)
	if err != nil {
		t.Fatalf("Failed to extract frontmatter: %v", err)
	}

	importsResult, err := parser.ProcessImportsFromFrontmatterWithSource(
		result.Frontmatter,
		tempDir,
		nil,
		consumerPath,
		consumerContent,
	)
	if err != nil {
		t.Fatalf("ProcessImportsFromFrontmatterWithSource failed: %v", err)
	}

	// The leaf workflow's bash tool (git *) should be present in merged tools
	if !strings.Contains(importsResult.MergedTools, "git *") {
		t.Errorf("MergedTools should contain 'git *' from leaf workflow; got:\n%s", importsResult.MergedTools)
	}

	// No unresolved import-inputs expressions should remain anywhere
	mergedContent := importsResult.MergedTools + importsResult.MergedMarkdown
	if strings.Contains(mergedContent, "github.aw.import-inputs") {
		t.Errorf("Merged content should not contain unsubstituted github.aw.import-inputs expressions;\ngot:\n%s", mergedContent)
	}
}

// TestImportInputsMultipleForwardedToNestedImports_Integration verifies that multiple
// ${{ github.aw.import-inputs.* }} expressions in an intermediate workflow's imports:
// section are all resolved before nested import discovery.
func TestImportInputsMultipleForwardedToNestedImports_Integration(t *testing.T) {
	tempDir := testutil.TempDir(t, "import-inputs-multi-forwarding-*")

	sharedDir := filepath.Join(tempDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Leaf shared workflow accepting two inputs
	leafPath := filepath.Join(sharedDir, "publisher.md")
	leafContent := `---
import-schema:
  target-repo:
    type: string
    required: true
    description: "Target repository"
  title-prefix:
    type: string
    required: true
    description: "Title prefix"
tools:
  bash:
    - "curl *"
---

Publish to ${{ github.aw.import-inputs.target-repo }} with prefix ${{ github.aw.import-inputs.title-prefix }}.
`
	if err := os.WriteFile(leafPath, []byte(leafContent), 0644); err != nil {
		t.Fatalf("Failed to write leaf workflow: %v", err)
	}

	// Intermediate workflow — accepts both inputs and forwards them to the leaf
	intermediateContent := `---
import-schema:
  target-repo:
    type: string
    required: true
    description: "Target repository for publishing"
  title-prefix:
    type: string
    required: true
    description: "Title prefix for created items"

imports:
  - uses: shared/publisher.md
    with:
      target-repo: ${{ github.aw.import-inputs.target-repo }}
      title-prefix: ${{ github.aw.import-inputs.title-prefix }}
---

Intermediate reporter.
`
	intermediatePath := filepath.Join(sharedDir, "reporter.md")
	if err := os.WriteFile(intermediatePath, []byte(intermediateContent), 0644); err != nil {
		t.Fatalf("Failed to write intermediate workflow: %v", err)
	}

	// Consumer that provides concrete values for both inputs
	consumerContent := `---
on: issues
permissions:
  contents: read
engine: copilot
imports:
  - uses: shared/reporter.md
    with:
      target-repo: "myorg/myrepo"
      title-prefix: "daily-"
---

Consumer.
`
	consumerPath := filepath.Join(tempDir, "consumer.md")
	if err := os.WriteFile(consumerPath, []byte(consumerContent), 0644); err != nil {
		t.Fatalf("Failed to write consumer workflow: %v", err)
	}

	result, err := parser.ExtractFrontmatterFromContent(consumerContent)
	if err != nil {
		t.Fatalf("Failed to extract frontmatter: %v", err)
	}

	importsResult, err := parser.ProcessImportsFromFrontmatterWithSource(
		result.Frontmatter,
		tempDir,
		nil,
		consumerPath,
		consumerContent,
	)
	if err != nil {
		t.Fatalf("ProcessImportsFromFrontmatterWithSource failed: %v", err)
	}

	// The leaf workflow's bash tool (curl *) must be present — proving the leaf
	// was discovered and merged after both inputs were forwarded correctly.
	if !strings.Contains(importsResult.MergedTools, "curl *") {
		t.Errorf("MergedTools should contain 'curl *' from leaf workflow; got:\n%s", importsResult.MergedTools)
	}

	// No unresolved import-inputs expressions should remain anywhere
	mergedContent := importsResult.MergedTools + importsResult.MergedMarkdown
	if strings.Contains(mergedContent, "github.aw.import-inputs") {
		t.Errorf("Merged content should not contain unsubstituted github.aw.import-inputs expressions;\ngot:\n%s", mergedContent)
	}
}
