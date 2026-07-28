//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// deprecatedActivationOutputsPattern matches the deprecated needs.activation.outputs.{text,title,body}
// expressions that should no longer appear in workflow markdown files.
// The pattern uses \b (word boundary) to avoid matching longer identifiers like
// needs.activation.outputs.text_custom, which are not deprecated.
var deprecatedActivationOutputsPattern = regexp.MustCompile(
	`needs\.activation\.outputs\.(text|title|body)\b`,
)

// TestNoDeprecatedActivationOutputsInWorkflowMarkdown verifies that no workflow markdown
// file in .github/workflows/ uses the deprecated needs.activation.outputs.{text,title,body}
// expressions. These should use steps.sanitized.outputs.{text,title,body} instead.
//
// The compiler provides backward-compatibility auto-rewrite for external consumers that have
// not yet migrated (via transformActivationOutputs and detectTextOutputUsage in
// expression_extraction.go and compiler_orchestrator_tools.go respectively), but this
// repository's own workflow sources must always use the modern form to keep the compatibility
// code path cold and reduce confusion in workflow prompt authoring.
//
// To fix a failure: replace ${{ needs.activation.outputs.text }} with
// ${{ steps.sanitized.outputs.text }} (and similarly for title/body).
func TestNoDeprecatedActivationOutputsInWorkflowMarkdown(t *testing.T) {
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		t.Fatalf("Failed to glob workflow markdown files: %v", err)
	}
	if len(mdFiles) == 0 {
		t.Fatal("Expected to find at least one workflow markdown file in " + workflowsDir)
	}

	for _, mdFile := range mdFiles {
		content, err := os.ReadFile(mdFile)
		if err != nil {
			t.Errorf("Failed to read %s: %v", filepath.Base(mdFile), err)
			continue
		}

		if match := deprecatedActivationOutputsPattern.FindString(string(content)); match != "" {
			t.Errorf(
				"%s contains deprecated expression %q: use steps.sanitized.outputs.* instead",
				filepath.Base(mdFile),
				match,
			)
		}
	}
}
