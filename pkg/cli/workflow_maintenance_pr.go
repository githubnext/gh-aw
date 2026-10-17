package cli

import (
	"context"
	"fmt"
)

var createMaintenancePRWithChanges = createPRWithChanges

// createWorkflowMaintenancePR marks update and upgrade PRs consistently in local
// and remote modes, and only retires older PRs after successful replacement.
func createWorkflowMaintenancePR(ctx context.Context, operation, body string, verbose, autoMerge, closeOlderPRs bool) (string, error) {
	var branchPrefix, commitMessage, title, markerPrefix string
	switch operation {
	case "update":
		branchPrefix = "update-workflows"
		commitMessage = "chore: update workflows"
		title = "Update workflows from source"
		markerPrefix = ghawUpdateMarkerPrefix
	case "upgrade":
		branchPrefix = "upgrade-agentic-workflows"
		commitMessage = "chore: upgrade agentic workflows"
		title = "Upgrade agentic workflows"
		markerPrefix = ghawUpgradeMarkerPrefix
	default:
		return "", fmt.Errorf("unsupported maintenance operation: %s", operation)
	}
	body += "\n" + buildOrgXMLMarker(markerPrefix, normalizeDisplayVersion(GetVersion()))
	prURL, err := createMaintenancePRWithChanges(ctx, branchPrefix, commitMessage, title, body, verbose, autoMerge)
	if err == nil && closeOlderPRs && prURL != "" {
		closeExistingOrgPRsByMarker(ctx, prURL, markerPrefix, verbose)
	}
	return prURL, err
}
