package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var rateLimitFieldsCodemodLog = logger.New("cli:codemod_rate_limit_fields")

// getRateLimitFieldsCodemod creates a codemod for migrating rate-limit field names:
//   - max -> max-runs-per-user
//   - window -> max-runs-per-user-window
func getRateLimitFieldsCodemod() Codemod {
	return Codemod{
		ID:           "rate-limit-fields-migration",
		Name:         "Rename rate-limit fields for clarity",
		Description:  "Renames 'rate-limit.max' to 'rate-limit.max-runs-per-user' and 'rate-limit.window' to 'rate-limit.max-runs-per-user-window'",
		IntroducedIn: "0.20.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if rate-limit block exists with old field names
			rateLimitValue, hasRateLimit := frontmatter["rate-limit"]
			if !hasRateLimit {
				return content, false, nil
			}
			rateLimitMap, ok := rateLimitValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			_, hasMax := rateLimitMap["max"]
			_, hasWindow := rateLimitMap["window"]
			if !hasMax && !hasWindow {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				var modified bool
				var inRateLimitBlock bool
				var rateLimitIndent string

				result := make([]string, len(lines))
				for i, line := range lines {
					trimmedLine := strings.TrimSpace(line)

					// Detect entering the rate-limit block
					if strings.HasPrefix(trimmedLine, "rate-limit:") {
						inRateLimitBlock = true
						rateLimitIndent = getIndentation(line)
						result[i] = line
						continue
					}

					// Detect leaving the rate-limit block
					if inRateLimitBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
						if hasExitedBlock(line, rateLimitIndent) {
							inRateLimitBlock = false
						}
					}

					// Apply renames inside the rate-limit block
					if inRateLimitBlock {
						if strings.HasPrefix(trimmedLine, "max:") {
							replacedLine, didReplace := findAndReplaceInLine(line, "max", "max-runs-per-user")
							if didReplace {
								result[i] = replacedLine
								modified = true
								rateLimitFieldsCodemodLog.Printf("Renamed rate-limit.max to rate-limit.max-runs-per-user on line %d", i+1)
								continue
							}
						} else if strings.HasPrefix(trimmedLine, "window:") {
							replacedLine, didReplace := findAndReplaceInLine(line, "window", "max-runs-per-user-window")
							if didReplace {
								result[i] = replacedLine
								modified = true
								rateLimitFieldsCodemodLog.Printf("Renamed rate-limit.window to rate-limit.max-runs-per-user-window on line %d", i+1)
								continue
							}
						}
					}

					result[i] = line
				}
				return result, modified
			})
			if applied {
				rateLimitFieldsCodemodLog.Print("Applied rate-limit fields migration")
			}
			return newContent, applied, err
		},
	}
}
