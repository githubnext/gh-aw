package workflow

import (
	"fmt"
	"os"
	"strings"
)

var checkoutPathValidationLog = newValidationLogger("checkout_path")

// validateCrossRepoCheckoutPaths emits deprecation warnings for cross-repository
// checkout entries that have no explicit path: field, and auto-derives a path from
// the repository name.
//
// When a checkout entry targets a different repository (repository: != "") without
// an explicit path:, the compiler auto-derives one using the repository-name portion
// of the "owner/repo" slug (e.g. "githubnext/gh-aw-side-repo" → "gh-aw-side-repo").
// This mirrors what actions/checkout itself recommends when checking out multiple
// repositories simultaneously.
//
// A deprecation warning is emitted for each affected entry so that workflow authors
// can add the explicit path: field and opt out of the auto-derivation.
//
// Dynamic repository expressions (those containing "${{") are skipped: the compiler
// cannot determine at compile time whether they target a different repository, and
// common patterns such as `repository: ${{ github.repository }}` are valid
// same-repository checkouts that should not receive a warning.
func (c *Compiler) validateCrossRepoCheckoutPaths(checkoutConfigs []*CheckoutConfig, markdownPath string) {
	for _, cfg := range checkoutConfigs {
		if cfg == nil || cfg.Repository == "" || cfg.Path != "" {
			continue
		}

		// Dynamic expression: skip — cannot determine at compile time whether it
		// targets a different repository (e.g. ${{ github.repository }} is the
		// same repo checked out at a specific ref, which is a valid pattern for
		// pull_request_target and should not trigger a warning).
		if strings.Contains(cfg.Repository, "${{") {
			continue
		}

		checkoutPathValidationLog.Printf("cross-repo checkout %q has no explicit path", cfg.Repository)

		// Static repository slug: auto-derive path from the repository-name portion.
		repoName := cfg.Repository
		if slash := strings.LastIndex(cfg.Repository, "/"); slash >= 0 {
			repoName = cfg.Repository[slash+1:]
		}

		checkoutPathValidationLog.Printf("auto-deriving path %q for cross-repo checkout %q", repoName, cfg.Repository)
		cfg.Path = repoName

		msg := strings.Join([]string{
			fmt.Sprintf("checkout: repository %q has no explicit path: field.", cfg.Repository),
			fmt.Sprintf("The compiler has auto-derived path: %q from the repository name.", repoName),
			"This auto-derivation is deprecated. Add an explicit path: to silence this warning:",
			"",
			"  checkout:",
			"    - repository: " + cfg.Repository,
			"      path: " + repoName,
		}, "\n")
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", msg))
		c.IncrementWarningCount()
	}
}
