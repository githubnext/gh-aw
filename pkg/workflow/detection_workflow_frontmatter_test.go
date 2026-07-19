//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
)

func TestDuplicateCodeDetectorWorkflowEnablesDetection(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/duplicate-code-detector.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	result, err := parser.ExtractFrontmatterFromContent(string(sourceContent))
	if err != nil {
		t.Fatalf("failed to parse workflow frontmatter: %v", err)
	}

	features, ok := result.Frontmatter["features"].(map[string]any)
	if !ok {
		t.Fatal("expected Duplicate Code Detector workflow to define features in frontmatter")
	}
	if enabled, ok := features["gh-aw-detection"].(bool); !ok || !enabled {
		t.Fatal("expected Duplicate Code Detector workflow to enable gh-aw-detection in frontmatter")
	}
}

func TestDetectionAnalysisReportDocumentsAgenticTokenAuditOptOut(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/detection-analysis-report.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	sourceContentStr := string(sourceContent)
	if !strings.Contains(sourceContentStr, "Daily Agentic Workflow AIC Usage Audit") {
		t.Fatal("expected detection analysis report to mention the Daily Agentic Workflow AIC Usage Audit opt-out")
	}
	if !strings.Contains(sourceContentStr, "source-managed") || !strings.Contains(sourceContentStr, "should not be reported as misconfigured") {
		t.Fatal("expected detection analysis report to document a source-managed opt-out from name-based misconfiguration checks")
	}
}
