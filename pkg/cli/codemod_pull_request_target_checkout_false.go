package cli

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var pullRequestTargetCheckoutFalseCodemodLog = logger.New("cli:codemod_pull_request_target_checkout_false")
var gitCheckoutPattern = regexp.MustCompile(`\bgit\s+checkout(?:\s|$)`)

// getPullRequestTargetCheckoutFalseCodemod adds checkout: false for pull_request_target workflows
// when checkout is not disabled and no explicit checkout command is detected in workflow content.
func getPullRequestTargetCheckoutFalseCodemod() Codemod {
	return Codemod{
		ID:           "pull-request-target-checkout-false",
		Name:         "Add checkout: false for pull_request_target",
		Description:  "Adds checkout: false to workflows using on.pull_request_target when checkout is not disabled and no explicit checkout command is detected",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if isFrontmatterStrictFalse(frontmatter) {
				return content, false, nil
			}

			if !hasPullRequestTargetTrigger(frontmatter) || isPullRequestTargetCheckoutDisabled(frontmatter) {
				return content, false, nil
			}

			if isCheckoutMapping(frontmatter) {
				pullRequestTargetCheckoutFalseCodemodLog.Print("Skipping pull_request_target checkout codemod: checkout is already a mapping with sub-keys")
				return content, false, nil
			}

			if hasExplicitCheckoutCommands(content) {
				pullRequestTargetCheckoutFalseCodemodLog.Print("Skipping pull_request_target checkout codemod: explicit checkout command detected")
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Set([]string{"checkout"}, false)
			})
			if applied {
				pullRequestTargetCheckoutFalseCodemodLog.Print("Added checkout: false for pull_request_target workflow")
			}
			return newContent, applied, err
		},
	}
}

func hasPullRequestTargetTrigger(frontmatter map[string]any) bool {
	onAny, hasOn := frontmatter["on"]
	if !hasOn {
		return false
	}

	switch on := onAny.(type) {
	case map[string]any:
		_, hasPullRequestTarget := on["pull_request_target"]
		return hasPullRequestTarget
	case []any:
		for _, entry := range on {
			event, ok := entry.(string)
			if ok && strings.TrimSpace(event) == "pull_request_target" {
				return true
			}
		}
	case []string:
		for _, event := range on {
			if strings.TrimSpace(event) == "pull_request_target" {
				return true
			}
		}
	case string:
		return strings.TrimSpace(on) == "pull_request_target"
	}

	return false
}

func isPullRequestTargetCheckoutDisabled(frontmatter map[string]any) bool {
	checkoutAny, hasCheckout := frontmatter["checkout"]
	if !hasCheckout {
		return false
	}

	checkoutDisabled, ok := checkoutAny.(bool)
	return ok && !checkoutDisabled
}

// isCheckoutMapping returns true when the frontmatter "checkout" key is a YAML
// mapping (e.g. sparse-checkout:, fetch-depth:) rather than a scalar or absent.
// Overwriting a mapping with a scalar would orphan the child keys and produce
// invalid YAML, so callers should skip the codemod in this case.
func isCheckoutMapping(frontmatter map[string]any) bool {
	checkoutAny, hasCheckout := frontmatter["checkout"]
	if !hasCheckout {
		return false
	}
	_, isMap := checkoutAny.(map[string]any)
	return isMap
}

func hasExplicitCheckoutCommands(content string) bool {
	lowerContent := strings.ToLower(content)

	unsafeCheckoutPatterns := []string{
		"actions/checkout",
		"uses: actions/checkout",
		"gh pr checkout",
		"git checkout ",
		"git checkout\n",
		"refs/pull/",
	}

	for _, pattern := range unsafeCheckoutPatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}

	return gitCheckoutPattern.MatchString(lowerContent)
}
