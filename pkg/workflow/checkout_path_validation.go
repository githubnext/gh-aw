package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var checkoutPathValidationLog = logger.New("workflow:checkout_path_validation")

// deriveAndWarnCrossRepoCheckoutPaths emits warnings for cross-repository
// checkout entries that have no explicit path: field, and auto-derives a path from
// the repository name.
//
// When a checkout entry targets a different repository (repository: != "") without
// an explicit path:, the compiler auto-derives one using the repository-name portion
// of the "owner/repo" slug (e.g. "githubnext/gh-aw-side-repo" → "gh-aw-side-repo").
// This mirrors what actions/checkout itself recommends when checking out multiple
// repositories simultaneously.
//
// A warning is emitted for each affected entry so that workflow authors can add
// the explicit path: field and opt out of the auto-derivation.
//
// Dynamic repository expressions (those containing "${{") are skipped: the compiler
// cannot determine at compile time whether they target a different repository, and
// common patterns such as `repository: ${{ github.repository }}` are valid
// same-repository checkouts that should not receive a warning.
//
// NOTE: This pass intentionally mutates the provided CheckoutConfig entries before
// YAML generation. It also intentionally remains warning-only, even in strict mode,
// to preserve backward compatibility with existing workflows.
func (c *Compiler) deriveAndWarnCrossRepoCheckoutPaths(checkoutConfigs []*CheckoutConfig, markdownPath string) {
	for _, cfg := range checkoutConfigs {
		if cfg == nil || cfg.Repository == "" || cfg.Path != "" || cfg.PathExplicit {
			continue
		}

		// Dynamic expression: skip — cannot determine at compile time whether it
		// targets a different repository (e.g. ${{ github.repository }} is the
		// same repo checked out at a specific ref, which is a valid pattern for
		// pull_request_target and should not trigger a warning).
		repository := strings.TrimSpace(cfg.Repository)
		if repository == "" || strings.Contains(repository, "${{") {
			continue
		}

		repository = strings.TrimRight(repository, "/")
		if repository == "" {
			continue
		}

		checkoutPathValidationLog.Printf("cross-repo checkout %q has no explicit path", cfg.Repository)
		// Static repository slug: auto-derive path from the repository-name portion.
		repoName := repository
		if slash := strings.LastIndex(repository, "/"); slash >= 0 {
			repoName = repository[slash+1:]
		}
		if repoName == "" {
			checkoutPathValidationLog.Printf("skipping auto-derivation for cross-repo checkout %q: empty repository name", cfg.Repository)
			continue
		}

		checkoutPathValidationLog.Printf("auto-deriving path %q for cross-repo checkout %q", repoName, cfg.Repository)
		cfg.Path = repoName

		msg := strings.Join([]string{
			fmt.Sprintf("checkout: repository %q has no explicit path: field.", cfg.Repository),
			fmt.Sprintf("The compiler has auto-derived path: %q from the repository name.", repoName),
			"Add an explicit path: field to silence this warning. Auto-derivation may become an error in a future release:",
			"",
			"  checkout:",
			"    - repository: " + cfg.Repository,
			"      path: " + repoName,
		}, "\n")
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", msg))
		c.IncrementWarningCount()
	}
}
