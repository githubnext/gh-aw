package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineDriverCodemodLog = logger.New("cli:codemod_engine_driver")

// getEngineCopilotSDKDriverToDriverCodemod creates a codemod that renames the deprecated
// 'copilot-sdk-driver:' field to 'driver:' within the engine configuration block.
func getEngineCopilotSDKDriverToDriverCodemod() Codemod {
	return Codemod{
		ID:           "engine-copilot-sdk-driver-to-driver",
		Name:         "Rename 'engine.copilot-sdk-driver' to 'engine.driver'",
		Description:  "Renames the deprecated 'copilot-sdk-driver:' field to 'driver:' inside the engine configuration block.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasDeprecatedCopilotSDKDriverField(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, renameCopilotSDKDriverToDriver)
			if applied {
				engineDriverCodemodLog.Print("Renamed 'engine.copilot-sdk-driver' to 'engine.driver'")
			}
			return newContent, applied, err
		},
	}
}

// hasDeprecatedCopilotSDKDriverField returns true if the engine block has the deprecated
// 'copilot-sdk-driver' field and does not already have a 'driver' field.
func hasDeprecatedCopilotSDKDriverField(frontmatter map[string]any) bool {
	engineAny, hasEngine := frontmatter["engine"]
	if !hasEngine {
		return false
	}
	engineMap, ok := engineAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasOldField := engineMap["copilot-sdk-driver"]
	_, hasNewField := engineMap["driver"]
	return hasOldField && !hasNewField
}

// renameCopilotSDKDriverToDriver renames 'copilot-sdk-driver:' to 'driver:' within
// the engine configuration block.
func renameCopilotSDKDriverToDriver(lines []string) ([]string, bool) {
	var result []string
	modified := false

	inEngine := false
	var engineIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines without resetting state
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Exit engine block when indentation signals we've left it
		if !strings.HasPrefix(trimmed, "#") && inEngine && hasExitedBlock(line, engineIndent) {
			inEngine = false
		}

		// Detect top-level 'engine:' block
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
			inEngine = true
			engineIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Rename 'copilot-sdk-driver:' to 'driver:' when inside the engine block
		if inEngine && strings.HasPrefix(trimmed, "copilot-sdk-driver:") {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, engineIndent) {
				newLine, replaced := findAndReplaceInLine(line, "copilot-sdk-driver", "driver")
				if replaced {
					result = append(result, newLine)
					modified = true
					engineDriverCodemodLog.Printf("Renamed 'copilot-sdk-driver' to 'driver' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
