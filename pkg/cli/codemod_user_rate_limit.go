package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
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

			newContent, applied, err := applyFrontmatterLineTransform(content, renameRateLimitToUserRateLimit)
			if applied {
				userRateLimitCodemodLog.Print("Renamed 'rate-limit' to 'user-rate-limit' and migrated max field")
			}
			return newContent, applied, err
		},
	}
}

func renameRateLimitToUserRateLimit(lines []string) ([]string, bool) {
	var result []string
	modified := false

	inUserRateLimit := false
	userRateLimitIndent := ""
	userRateLimitChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if len(trimmed) == 0 {
			result = append(result, line)
			continue
		}

		if !strings.HasPrefix(trimmed, "#") && inUserRateLimit && hasExitedBlock(line, userRateLimitIndent) {
			inUserRateLimit = false
			userRateLimitChildIndent = ""
		}

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "rate-limit:") {
			lineIndent := getIndentation(line)
			newLine, replaced := findAndReplaceInLine(line, "rate-limit", "user-rate-limit")
			if replaced {
				result = append(result, newLine)
				modified = true
				inUserRateLimit = true
				userRateLimitIndent = lineIndent
				userRateLimitChildIndent = ""
				userRateLimitCodemodLog.Printf("Renamed 'rate-limit' to 'user-rate-limit' on line %d", i+1)
				continue
			}
		}

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "user-rate-limit:") {
			inUserRateLimit = true
			userRateLimitIndent = getIndentation(line)
			userRateLimitChildIndent = ""
			result = append(result, line)
			continue
		}

		if inUserRateLimit {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, userRateLimitIndent) {
				if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && userRateLimitChildIndent == "" {
					userRateLimitChildIndent = lineIndent
				}
				if userRateLimitChildIndent != "" && lineIndent != userRateLimitChildIndent {
					result = append(result, line)
					continue
				}
				newLine, replaced := findAndReplaceInLine(line, "max-runs", "max-runs-per-window")
				if !replaced {
					newLine, replaced = findAndReplaceInLine(line, "max", "max-runs-per-window")
				}
				if replaced {
					result = append(result, newLine)
					modified = true
					userRateLimitCodemodLog.Printf("Renamed max field to 'max-runs-per-window' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
