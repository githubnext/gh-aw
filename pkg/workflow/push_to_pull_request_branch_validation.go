package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

var pushToPullRequestBranchValidationLog = newValidationLogger("push_to_pull_request_branch_validation")

// computeIsPublicRepo queries the GitHub API to determine whether the repository
// associated with this compiler instance is publicly visible.
//
// Returns true only when the API confirms visibility == "public". Returns false
// for private/internal repos, when no repository slug is set, or when the API
// call fails (e.g. no authentication, network error). This fail-safe default
// ensures safety warnings are always shown when visibility cannot be determined.
func (c *Compiler) computeIsPublicRepo() bool {
	slug := c.repositorySlug
	if slug == "" || strings.Count(slug, "/") != 1 {
		pushToPullRequestBranchValidationLog.Printf("Skipping public-repo check: slug %q is not in owner/repo format", slug)
		return false
	}
	owner, repo, _ := strings.Cut(slug, "/")
	if owner == "" || repo == "" {
		pushToPullRequestBranchValidationLog.Printf("Skipping public-repo check: slug %q has empty owner or repo", slug)
		return false
	}

	pushToPullRequestBranchValidationLog.Printf("Checking repository visibility for: %s", slug)
	output, err := RunGH("Checking repository visibility...", "api", "/repos/"+slug, "--jq", ".visibility")
	if err != nil {
		pushToPullRequestBranchValidationLog.Printf("Could not determine repository visibility: %v", err)
		return false
	}

	visibility := strings.TrimSpace(string(output))
	isPublic := visibility == "public"
	pushToPullRequestBranchValidationLog.Printf("Repository visibility: %s (isPublic=%v)", visibility, isPublic)
	return isPublic
}

// validatePushToPullRequestBranchWarnings emits warnings for common misconfiguration
// patterns when push-to-pull-request-branch is used with target: "*".
//
// Two warnings are emitted when applicable:
//
//  1. No wildcard fetch in checkout — target: "*" allows pushing to any PR branch, but
//     without a wildcard fetch pattern (e.g., fetch: ["*"]) the agent cannot access
//     those branches at runtime. This warning is suppressed for public repositories
//     because all PR branches are always accessible in public repos.
//
//  2. No constraints — target: "*" without title-prefix or labels means the agent may
//     push to any PR in the repository with no additional gating.
func (c *Compiler) validatePushToPullRequestBranchWarnings(safeOutputs *SafeOutputsConfig, checkoutConfigs []*CheckoutConfig) {
	if safeOutputs == nil || safeOutputs.PushToPullRequestBranch == nil {
		return
	}

	cfg := safeOutputs.PushToPullRequestBranch
	if cfg.Target != "*" {
		return
	}

	pushToPullRequestBranchValidationLog.Printf("Validating push-to-pull-request-branch with target: \"*\"")

	// Warning 1: no wildcard fetch pattern in any checkout configuration.
	// Suppressed for public repositories — PR branches are always accessible
	// in public repos regardless of the fetch configuration.
	if !hasWildcardFetch(checkoutConfigs) {
		if !c.computeIsPublicRepo() {
			msg := strings.Join([]string{
				"push-to-pull-request-branch: target: \"*\" requires that all PR branches are fetched at checkout.",
				"Your checkout configuration does not include a wildcard fetch pattern (e.g., fetch: [\"*\"]).",
				"Without this the agent may fail to access PR branches when pushing.",
				"",
				"Add a wildcard fetch to your checkout configuration:",
				"",
				"  checkout:",
				"    fetch: [\"*\"]      # fetch all remote branches",
				"    fetch-depth: 0   # fetch full history",
			}, "\n")
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
			c.IncrementWarningCount()
		}
	}

	// Warning 2: no constraints restricting which PRs can be targeted.
	if cfg.TitlePrefix == "" && len(cfg.Labels) == 0 {
		msg := strings.Join([]string{
			"push-to-pull-request-branch: target: \"*\" allows pushing to any PR branch with no additional constraints.",
			"Consider adding title-prefix: or labels: to restrict which PRs can receive pushes.",
			"",
			"Example:",
			"",
			"  push-to-pull-request-branch:",
			"    target: \"*\"",
			"    title-prefix: \"[bot] \"  # only PRs whose title starts with this prefix",
			"    labels: [automated]      # only PRs that carry all of these labels",
		}, "\n")
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
		c.IncrementWarningCount()
	}
}

// hasWildcardFetch reports whether any checkout configuration includes a fetch pattern
// that contains a wildcard ("*"), such as fetch: ["*"] or fetch: ["feature/*"].
func hasWildcardFetch(checkoutConfigs []*CheckoutConfig) bool {
	for _, cfg := range checkoutConfigs {
		for _, ref := range cfg.Fetch {
			if strings.Contains(ref, "*") {
				return true
			}
		}
	}
	return false
}
