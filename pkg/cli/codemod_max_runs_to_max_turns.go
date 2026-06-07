package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var maxRunsToMaxTurnsCodemodLog = logger.New("cli:codemod_max_runs_to_max_turns")

// getMaxRunsToMaxTurnsCodemod migrates deprecated top-level max-runs to max-turns.
func getMaxRunsToMaxTurnsCodemod() Codemod {
	return Codemod{
		ID:           "max-runs-to-max-turns",
		Name:         "Rename top-level max-runs to max-turns",
		Description:  "Renames deprecated top-level 'max-runs' to top-level 'max-turns'. If both exist, preserves 'max-turns' and removes 'max-runs'.",
		IntroducedIn: "1.0.76",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if frontmatter == nil {
				return content, false, nil
			}
			if _, hasMaxRuns := frontmatter["max-runs"]; !hasMaxRuns {
				return content, false, nil
			}

			_, hasMaxTurns := frontmatter["max-turns"]

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				modified := false
				result := make([]string, 0, len(lines))
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if isTopLevelKey(line) && strings.HasPrefix(trimmed, "max-runs:") {
						if hasMaxTurns {
							modified = true
							continue
						}
						updatedLine, replaced := findAndReplaceInLine(line, "max-runs", "max-turns")
						if !replaced {
							result = append(result, line)
							continue
						}
						result = append(result, updatedLine)
						modified = true
						continue
					}
					result = append(result, line)
				}
				return result, modified
			})
			if applied {
				if hasMaxTurns {
					maxRunsToMaxTurnsCodemodLog.Print("Removed deprecated top-level max-runs (top-level max-turns already present)")
				} else {
					maxRunsToMaxTurnsCodemodLog.Print("Migrated top-level max-runs to top-level max-turns")
				}
			}
			return newContent, applied, err
		},
	}
}
