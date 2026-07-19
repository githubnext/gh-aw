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
	state := renameRateLimitToUserRateLimitState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			result = append(result, line)
			continue
		}

		renameRateLimitToUserRateLimitUpdateBlockState(line, trimmed, &state)

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "rate-limit:") {
			if newLine, replaced := renameRateLimitToUserRateLimitTopLevel(line, i, &state); replaced {
				result = append(result, newLine)
				modified = true
				continue
			}
		}

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "user-rate-limit:") {
			renameRateLimitToUserRateLimitEnterBlock(getIndentation(line), &state)
			result = append(result, line)
			continue
		}

		if newLine, replaced := renameRateLimitToUserRateLimitNestedLine(line, trimmed, i, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameRateLimitToUserRateLimitState struct {
	inUserRateLimit          bool
	userRateLimitIndent      string
	userRateLimitChildIndent string
}

func renameRateLimitToUserRateLimitUpdateBlockState(line, trimmed string, state *renameRateLimitToUserRateLimitState) {
	if !strings.HasPrefix(trimmed, "#") && state.inUserRateLimit && hasExitedBlock(line, state.userRateLimitIndent) {
		state.inUserRateLimit = false
		state.userRateLimitChildIndent = ""
	}
}

func renameRateLimitToUserRateLimitEnterBlock(indent string, state *renameRateLimitToUserRateLimitState) {
	state.inUserRateLimit = true
	state.userRateLimitIndent = indent
	state.userRateLimitChildIndent = ""
}

func renameRateLimitToUserRateLimitTopLevel(line string, lineIndex int, state *renameRateLimitToUserRateLimitState) (string, bool) {
	newLine, replaced := findAndReplaceInLine(line, "rate-limit", "user-rate-limit")
	if replaced {
		renameRateLimitToUserRateLimitEnterBlock(getIndentation(line), state)
		userRateLimitCodemodLog.Printf("Renamed 'rate-limit' to 'user-rate-limit' on line %d", lineIndex+1)
	}
	return newLine, replaced
}

func renameRateLimitToUserRateLimitNestedLine(line, trimmed string, lineIndex int, state *renameRateLimitToUserRateLimitState) (string, bool) {
	if !state.inUserRateLimit {
		return "", false
	}
	lineIndent := getIndentation(line)
	if !isDescendant(lineIndent, state.userRateLimitIndent) {
		return "", false
	}
	if trimmed != "" && !strings.HasPrefix(trimmed, "#") && state.userRateLimitChildIndent == "" {
		state.userRateLimitChildIndent = lineIndent
	}
	if state.userRateLimitChildIndent != "" && lineIndent != state.userRateLimitChildIndent {
		return "", false
	}
	newLine, replaced := findAndReplaceInLine(line, "max-runs", "max-runs-per-window")
	if !replaced {
		newLine, replaced = findAndReplaceInLine(line, "max", "max-runs-per-window")
	}
	if replaced {
		userRateLimitCodemodLog.Printf("Renamed max field to 'max-runs-per-window' on line %d", lineIndex+1)
	}
	return newLine, replaced
}
