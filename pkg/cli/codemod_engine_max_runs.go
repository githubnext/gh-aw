package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineMaxRunsCodemodLog = logger.New("cli:codemod_engine_max_runs")

// getEngineMaxRunsToTopLevelCodemod migrates deprecated engine.max-runs to
// top-level max-runs.
func getEngineMaxRunsToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-max-runs-to-top-level",
		Name:         "Move engine.max-runs to top-level max-runs",
		Description:  "Moves deprecated 'engine.max-runs' to top-level 'max-runs' so AWF enforces invocation caps consistently across all engines.",
		IntroducedIn: "0.17.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			engineValue, hasEngine := frontmatter["engine"]
			if !hasEngine {
				return content, false, nil
			}
			engineMap, ok := engineValue.(map[string]any)
			if !ok {
				return content, false, nil
			}
			if _, hasMaxRuns := engineMap["max-runs"]; !hasMaxRuns {
				return content, false, nil
			}

			_, hasTopLevelMaxRuns := frontmatter["max-runs"]

			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if !isTopLevelKey(line) || !strings.HasPrefix(trimmed, "engine:") {
						continue
					}
					inlineValue := strings.TrimSpace(strings.TrimPrefix(trimmed, "engine:"))
					if strings.HasPrefix(inlineValue, "{") && strings.Contains(inlineValue, "max-runs:") {
						engineMaxRunsCodemodLog.Print("Skipping engine.max-runs migration for inline-map engine syntax; migrate to top-level max-runs manually")
						return lines, false
					}
				}

				maxRunsSuffix := ""
				inEngineBlock := false
				engineIndent := ""
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
						inEngineBlock = true
						engineIndent = getIndentation(line)
						continue
					}
					if inEngineBlock && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && len(getIndentation(line)) <= len(engineIndent) {
						inEngineBlock = false
					}
					if inEngineBlock && strings.HasPrefix(trimmed, "max-runs:") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							maxRunsSuffix = parts[1]
						}
						break
					}
				}

				result, removed := removeFieldFromBlock(lines, "max-runs", "engine")
				if !removed {
					return lines, false
				}

				if hasTopLevelMaxRuns {
					engineMaxRunsCodemodLog.Print("Removed deprecated engine.max-runs (top-level max-runs already present)")
					return result, true
				}

				insertAt := 0
				for i, line := range result {
					if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "engine:") {
						insertAt = i
						break
					}
				}

				maxRunsLine := "max-runs:" + maxRunsSuffix
				withTopLevel := make([]string, 0, len(result)+1)
				withTopLevel = append(withTopLevel, result[:insertAt]...)
				withTopLevel = append(withTopLevel, maxRunsLine)
				withTopLevel = append(withTopLevel, result[insertAt:]...)

				engineMaxRunsCodemodLog.Print("Migrated engine.max-runs to top-level max-runs")
				return withTopLevel, true
			})
		},
	}
}
