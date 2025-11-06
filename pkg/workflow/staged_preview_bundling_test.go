package workflow

import (
	"strings"
	"testing"
)

func TestStagedPreviewInlined(t *testing.T) {
	scripts := map[string]string{
		"create_issue":                getCreateIssueScript(),
		"add_labels":                  getAddLabelsScript(),
		"update_issue":                getUpdateIssueScript(),
		"create_pr_review_comment":    getCreatePRReviewCommentScript(),
		"push_to_pull_request_branch": getPushToPullRequestBranchScript(),
	}

	for name, script := range scripts {
		t.Run(name, func(t *testing.T) {
			// Check that staged_preview.cjs is not being required
			if strings.Contains(script, `require("./staged_preview.cjs")`) {
				t.Errorf("%s contains require(\"./staged_preview.cjs\") - should be bundled", name)
			}
			if strings.Contains(script, `require('./staged_preview.cjs')`) {
				t.Errorf("%s contains require('./staged_preview.cjs') - should be bundled", name)
			}

			// Check that generateStagedPreview function exists (was inlined)
			if !strings.Contains(script, "generateStagedPreview") {
				t.Errorf("%s does not contain generateStagedPreview function - bundling may have failed", name)
			}
		})
	}
}
