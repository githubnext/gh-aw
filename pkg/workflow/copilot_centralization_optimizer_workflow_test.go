//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestCopilotCentralizationOptimizerWorkflowHandlesTaskFetchFailures(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/copilot-centralization-optimizer.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	sourceText := string(sourceContent)
	for _, fragment := range []string{
		`_task_ids=$(mktemp)`,
		`if gh api --paginate \`,
		`done < "$_task_ids" \`,
		`echo "Agent tasks API request failed; proceeding with empty dataset" >&2`,
		`jq -s '.' /tmp/gh-aw/data/task-summaries.jsonl > /tmp/gh-aw/data/task-summaries.json`,
	} {
		if !strings.Contains(sourceText, fragment) {
			t.Fatalf("expected workflow source to contain %q", fragment)
		}
	}
}

func TestCopilotCentralizationOptimizerCompiledWorkflowHandlesTaskFetchFailures(t *testing.T) {
	lockContent, err := os.ReadFile("../../.github/workflows/copilot-centralization-optimizer.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}

	lockText := string(lockContent)
	for _, fragment := range []string{
		`_task_ids=$(mktemp)`,
		`if gh api --paginate \\\n`,
		`done < \"$_task_ids\" \\\n`,
		`echo \"Agent tasks API request failed; proceeding with empty dataset\" >&2`,
		`jq -s '.' /tmp/gh-aw/data/task-summaries.jsonl > /tmp/gh-aw/data/task-summaries.json`,
	} {
		if !strings.Contains(lockText, fragment) {
			t.Fatalf("expected compiled workflow to contain %q", fragment)
		}
	}
}
