//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
)

func TestContributionCheckWorkflowSafeOutputContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read contribution-check workflow: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "emit exactly") || !strings.Contains(text, "one consolidated noop") {
		t.Fatalf("workflow must instruct single consolidated noop emission")
	}

	if !strings.Contains(text, "set a `temporary_id` (for example `aw_summary`)") {
		t.Fatalf("workflow must instruct create_issue temporary_id for summary issue")
	}

	if !strings.Contains(text, "set `add_labels.item_number` to `#<temporary_id>`") {
		t.Fatalf("workflow must instruct add_labels item_number linkage to temporary_id")
	}
}
