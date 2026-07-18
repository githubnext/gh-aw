//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDuplicateCodeDetectorWorkflowEnablesDetection(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/duplicate-code-detector.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	if !strings.Contains(string(sourceContent), "features:\n  gh-aw-detection: true") {
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
	if !strings.Contains(sourceContentStr, "should not be reported as misconfigured solely because this repository mirrors the upstream file") {
		t.Fatal("expected detection analysis report to document why the upstream-managed audit workflow is exempt from name-based misconfiguration checks")
	}
}
