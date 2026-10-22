package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var userRateLimitCodemodLog = logger.New("cli:codemod_user_rate_limit")

// getRateLimitToUserRateLimitCodemod creates a codemod that renames:
//   - top-level "rate-limit" to "user-rate-limit"
//   - nested "max-runs" (and legacy "max") to "max-runs-per-window"
func getRateLimitToUserRateLimitCodemod() Codemod {
	return Codemod{
		ID:           "rate-limit-to-user-rate-limit",
		Name:         "Rename 'rate-limit' to 'user-rate-limit'",
		Description:  "Renames top-level 'rate-limit' to 'user-rate-limit' and migrates nested run-limit key to 'max-runs-per-window'.",
		IntroducedIn: "1.0.44",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			_, hasRateLimit := frontmatter["rate-limit"]
			_, hasUserRateLimit := frontmatter["user-rate-limit"]

			// Skip ambiguous documents where both old and new keys are present.
			if !hasRateLimit || hasUserRateLimit {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if err := f.Rename([]string{"rate-limit"}, "user-rate-limit"); err != nil {
					return err
				}
				for _, old := range []string{"max-runs", "max"} {
					if err := f.Rename([]string{"user-rate-limit", old}, "max-runs-per-window"); err != nil {
						return err
					}
				}
				return nil
			})
			if applied {
				userRateLimitCodemodLog.Print("Renamed 'rate-limit' to 'user-rate-limit' and migrated max field")
			}
			return newContent, applied, err
		},
	}
}
