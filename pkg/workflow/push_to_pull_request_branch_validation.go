package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

var pushToPullRequestBranchValidationLog = newValidationLogger("push_to_pull_request_branch_validation")

// validatePushToPullRequestBranchWarnings emits warnings for common misconfiguration
// patterns when push-to-pull-request-branch is used with target: "*".
//
// Two warnings are emitted when applicable:
//
//  1. No wildcard fetch in checkout — target: "*" allows pushing to any PR branch, but
//     without a wildcard fetch pattern (e.g., fetch: ["*"]) the agent cannot access
//     those branches at runtime.
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
	if !hasWildcardFetch(checkoutConfigs) {
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
