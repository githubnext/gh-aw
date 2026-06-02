package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineMaxTurnsCodemodLog = logger.New("cli:codemod_engine_max_turns")

// getEngineMaxTurnsToTopLevelCodemod migrates deprecated engine.max-turns to
// top-level max-turns.
func getEngineMaxTurnsToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-max-turns-to-top-level",
		Name:         "Move engine.max-turns to top-level max-turns",
		Description:  "Moves deprecated 'engine.max-turns' to top-level 'max-turns' so AWF enforces turn caps consistently across all agentic engines.",
		IntroducedIn: "0.68.4",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			engineValue, hasEngine := frontmatter["engine"]
			if !hasEngine {
				return content, false, nil
			}
			engineMap, ok := engineValue.(map[string]any)
			if !ok {
				return content, false, nil
			}
			if _, hasMaxTurns := engineMap["max-turns"]; !hasMaxTurns {
				return content, false, nil
			}

			_, hasTopLevelMaxTurns := frontmatter["max-turns"]

			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if !isTopLevelKey(line) || !strings.HasPrefix(trimmed, "engine:") {
						continue
					}
					inlineValue := strings.TrimSpace(strings.TrimPrefix(trimmed, "engine:"))
					if strings.HasPrefix(inlineValue, "{") && strings.Contains(inlineValue, "max-turns:") {
						engineMaxTurnsCodemodLog.Print("Skipping engine.max-turns migration for inline-map engine syntax; migrate to top-level max-turns manually")
						return lines, false
					}
				}

				maxTurnsSuffix := ""
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
					if inEngineBlock && strings.HasPrefix(trimmed, "max-turns:") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							maxTurnsSuffix = parts[1]
						}
						break
					}
				}

				result, removed := removeFieldFromBlock(lines, "max-turns", "engine")
				if !removed {
					return lines, false
				}

				if hasTopLevelMaxTurns {
					engineMaxTurnsCodemodLog.Print("Removed deprecated engine.max-turns (top-level max-turns already present)")
					return result, true
				}

				insertAt := 0
				for i, line := range result {
					if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "engine:") {
						insertAt = i
						break
					}
				}

				maxTurnsLine := "max-turns:" + maxTurnsSuffix
				withTopLevel := make([]string, 0, len(result)+1)
				withTopLevel = append(withTopLevel, result[:insertAt]...)
				withTopLevel = append(withTopLevel, maxTurnsLine)
				withTopLevel = append(withTopLevel, result[insertAt:]...)

				engineMaxTurnsCodemodLog.Print("Migrated engine.max-turns to top-level max-turns")
				return withTopLevel, true
			})
		},
	}
}
