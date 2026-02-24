package cli

import (
	"github.com/github/gh-aw/pkg/logger"
)

var timeoutMinutesCodemodLog = logger.New("cli:codemod_timeout_minutes")

// getTimeoutMinutesCodemod creates a codemod for migrating timeout_minutes to timeout-minutes
func getTimeoutMinutesCodemod() Codemod {
	return Codemod{
		ID:           "timeout-minutes-migration",
		Name:         "Migrate timeout_minutes to timeout-minutes",
		Description:  "Replaces deprecated 'timeout_minutes' field with 'timeout-minutes'",
		IntroducedIn: "0.1.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if the deprecated field exists
			value, exists := frontmatter["timeout_minutes"]
			if !exists {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result := make([]string, len(lines))
				var modified bool
				for i, line := range lines {
					replacedLine, didReplace := findAndReplaceInLine(line, "timeout_minutes", "timeout-minutes")
					if didReplace {
						result[i] = replacedLine
						modified = true
						timeoutMinutesCodemodLog.Printf("Replaced timeout_minutes with timeout-minutes on line %d", i+1)
					} else {
						result[i] = line
					}
				}
				return result, modified
			})
			if applied {
				timeoutMinutesCodemodLog.Printf("Applied timeout_minutes migration (value: %v)", value)
			}
			return newContent, applied, err
		},
	}
}
