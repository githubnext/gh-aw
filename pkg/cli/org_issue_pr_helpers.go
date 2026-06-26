package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var orgIPLog = logger.New("cli:org_ip")

const (
	// ghawUpgradeMarkerPrefix is the XML marker prefix embedded in upgrade org PRs/issues.
	// Full marker format: <!-- GHAW-upgrade: vX.Y.Z -->
	ghawUpgradeMarkerPrefix = "<!-- GHAW-upgrade:"

	// ghawUpdateMarkerPrefix is the XML marker prefix embedded in update org PRs/issues.
	// Full marker format: <!-- GHAW-update: vX.Y.Z -->
	ghawUpdateMarkerPrefix = "<!-- GHAW-update:"

	// ghawReleaseRepo is the GitHub repository for gh-aw releases.
	ghawReleaseRepo = "github/gh-aw"

	// agenticWorkflowsLabel is the GitHub label applied to org runner PRs and issues.
	agenticWorkflowsLabel = "agentic-workflows"
)

// buildOrgXMLMarker builds a full XML comment marker string for a given prefix and release tag.
// Example: buildOrgXMLMarker("<!-- GHAW-upgrade:", "v1.2.3") → "<!-- GHAW-upgrade: v1.2.3 -->"
// If tag is empty the marker is still written with a placeholder so it remains searchable.
func buildOrgXMLMarker(prefix, tag string) string {
	if tag == "" {
		return prefix + " latest -->"
	}
	return fmt.Sprintf("%s %s -->", prefix, tag)
}

// getGhawReleaseInfo returns the latest stable gh-aw release tag and its HTML URL.
// Both values are empty strings when the release cannot be determined; callers must
// handle this gracefully (e.g. omit the release link rather than failing).
func getGhawReleaseInfo() (tag, releaseURL string) {
	tag, err := getLatestRelease(false)
	if err != nil || tag == "" {
		orgIPLog.Printf("Could not resolve latest gh-aw release: %v", err)
		return "", ""
	}
	releaseURL = fmt.Sprintf("https://github.com/%s/releases/tag/%s", ghawReleaseRepo, tag)
	return tag, releaseURL
}

type orgListIssue struct {
	Number int    `json:"number"`
	Body   string `json:"body"`
}

type orgListPR struct {
	Number int    `json:"number"`
	Body   string `json:"body"`
}

// closeExistingOrgIssuesByMarker finds all open issues in repo whose body contains
// the given marker prefix string and closes them as not_planned. Errors are
// non-fatal: a warning is logged and the function continues.
func closeExistingOrgIssuesByMarker(ctx context.Context, repo, markerPrefix string, verbose bool) {
	output, err := workflow.RunGHContext(ctx, "Checking for existing issues...",
		"api",
		fmt.Sprintf("/repos/%s/issues?state=open&per_page=100", repo),
	)
	if err != nil {
		orgIPLog.Printf("Failed to list open issues in %s: %v", repo, err)
		return
	}

	var issues []orgListIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		orgIPLog.Printf("Failed to parse issue list for %s: %v", repo, err)
		return
	}

	for _, issue := range issues {
		if !strings.Contains(issue.Body, markerPrefix) {
			continue
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(
				fmt.Sprintf("Closing outdated issue #%d in %s", issue.Number, repo),
			))
		}
		if _, closeErr := workflow.RunGHContext(ctx, "Closing outdated issue...",
			"api", "--method", "PATCH",
			fmt.Sprintf("/repos/%s/issues/%d", repo, issue.Number),
			"-f", "state=closed",
			"-f", "state_reason=not_planned",
		); closeErr != nil {
			orgIPLog.Printf("Failed to close issue #%d in %s: %v", issue.Number, repo, closeErr)
		}
	}
}

// closeExistingOrgPRsByMarker finds all open PRs in repo whose body contains the
// given marker prefix string and closes them. Errors are non-fatal.
func closeExistingOrgPRsByMarker(ctx context.Context, repo, markerPrefix string, verbose bool) {
	output, err := workflow.RunGHContext(ctx, "Checking for existing PRs...",
		"api",
		fmt.Sprintf("/repos/%s/pulls?state=open&per_page=100", repo),
	)
	if err != nil {
		orgIPLog.Printf("Failed to list open PRs in %s: %v", repo, err)
		return
	}

	var prs []orgListPR
	if err := json.Unmarshal(output, &prs); err != nil {
		orgIPLog.Printf("Failed to parse PR list for %s: %v", repo, err)
		return
	}

	for _, pr := range prs {
		if !strings.Contains(pr.Body, markerPrefix) {
			continue
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(
				fmt.Sprintf("Closing outdated PR #%d in %s", pr.Number, repo),
			))
		}
		if _, closeErr := workflow.RunGHContext(ctx, "Closing outdated PR...",
			"api", "--method", "PATCH",
			fmt.Sprintf("/repos/%s/pulls/%d", repo, pr.Number),
			"-f", "state=closed",
		); closeErr != nil {
			orgIPLog.Printf("Failed to close PR #%d in %s: %v", pr.Number, repo, closeErr)
		}
	}
}

// addLabelToOrgPR adds a label to a PR identified by URL using gh pr edit.
// Errors are non-fatal and emitted as warnings.
func addLabelToOrgPR(prURL, label string, verbose bool) {
	remoteHost := getHostFromOriginRemote()
	if _, err := workflow.RunGHWithHost("Adding label to PR...", remoteHost, "pr", "edit", prURL, "--add-label", label); err != nil {
		orgIPLog.Printf("Failed to add label %q to PR %s: %v", label, prURL, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Failed to add label %q to PR (non-fatal): %v", label, err),
			))
		}
	}
}

// createOrgIssue creates a GitHub issue with the given title, body, and label. If
// creating with the label fails (e.g. label does not exist), it retries once without
// the label so the issue is always created.
func createOrgIssue(ctx context.Context, repo, title, body, label string) error {
	endpoint := fmt.Sprintf("/repos/%s/issues", repo)
	_, err := workflow.RunGHContext(ctx, "Creating issue...",
		"api",
		"--method", "POST",
		endpoint,
		"-f", "title="+title,
		"-f", "body="+body,
		"-f", "labels[]="+label,
	)
	if err == nil {
		return nil
	}
	// Label may not exist; retry without it so the issue is always created.
	orgIPLog.Printf("Failed to create issue with label %q, retrying without: %v", label, err)
	_, err = workflow.RunGHContext(ctx, "Creating issue...",
		"api",
		"--method", "POST",
		endpoint,
		"-f", "title="+title,
		"-f", "body="+body,
	)
	return err
}
