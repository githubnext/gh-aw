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
			newLine, wasModified, newChildIndent := renameRateLimitFields(line, trimmed, userRateLimitIndent, userRateLimitChildIndent, i)
			userRateLimitChildIndent = newChildIndent
			result = append(result, newLine)
			if wasModified {
				modified = true
			}
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

func renameRateLimitFields(line, trimmed, rateLimitIndent, childIndent string, lineIdx int) (string, bool, string) {
	lineIndent := getIndentation(line)
	if !isDescendant(lineIndent, rateLimitIndent) {
		return line, false, childIndent
	}
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && childIndent == "" {
		childIndent = lineIndent
	}
	if childIndent != "" && lineIndent != childIndent {
		return line, false, childIndent
	}
	newLine, replaced := findAndReplaceInLine(line, "max-runs", "max-runs-per-window")
	if !replaced {
		newLine, replaced = findAndReplaceInLine(line, "max", "max-runs-per-window")
	}
	if replaced {
		userRateLimitCodemodLog.Printf("Renamed max field to 'max-runs-per-window' on line %d", lineIdx+1)
		return newLine, true, childIndent
	}
	return line, false, childIndent
}
