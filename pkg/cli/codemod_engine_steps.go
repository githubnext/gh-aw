package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineStepsCodemodLog = logger.New("cli:codemod_engine_steps")

// getEngineStepsToTopLevelCodemod creates a codemod for moving engine.steps to the top-level steps field
func getEngineStepsToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-steps-to-top-level",
		Name:         "Move engine.steps to top-level steps",
		Description:  "Moves the 'steps' field from under 'engine' to the top-level 'steps' field, as 'engine.steps' is no longer supported",
		IntroducedIn: "0.11.0",
		Apply:        getEngineStepsToTopLevelCodemodApply,
	}
}

func getEngineStepsToTopLevelCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	// Check if engine.steps exists in frontmatter
	engineValue, hasEngine := frontmatter["engine"]
	if !hasEngine {
		return content, false, nil
	}

	engineMap, isMap := engineValue.(map[string]any)
	if !isMap {
		// engine is a string, no steps to move
		return content, false, nil
	}

	if _, hasSteps := engineMap["steps"]; !hasSteps {
		return content, false, nil
	}

	hasTopLevelSteps := getEngineStepsToTopLevelCodemodHasTopLevelSteps(frontmatter)
	return applyFrontmatterLineTransform(content, func(frontmatterLines []string) ([]string, bool) {
		return getEngineStepsToTopLevelCodemodTransform(frontmatterLines, hasTopLevelSteps)
	})
}

func getEngineStepsToTopLevelCodemodHasTopLevelSteps(frontmatter map[string]any) bool {
	if stepsVal, exists := frontmatter["steps"]; exists {
		if _, isSlice := stepsVal.([]any); isSlice {
			engineStepsCodemodLog.Print("Found existing top-level 'steps'")
			return true
		}
		engineStepsCodemodLog.Print("Top-level 'steps' exists but is not a sequence; treating as absent")
	}
	return false
}

func getEngineStepsToTopLevelCodemodTransform(frontmatterLines []string, hasTopLevelSteps bool) ([]string, bool) {
	stepsStartIdx := getEngineStepsToTopLevelCodemodFindStepsStart(frontmatterLines)
	if stepsStartIdx == -1 {
		return frontmatterLines, false
	}

	stepsEndIdx := getEngineStepsToTopLevelCodemodFindBlockEnd(frontmatterLines, stepsStartIdx)
	engineStepsCodemodLog.Printf("'engine.steps' spans lines %d to %d", stepsStartIdx+1, stepsEndIdx+1)

	topLevelStepsLines := getEngineStepsToTopLevelCodemodUnindentSteps(frontmatterLines, stepsStartIdx, stepsEndIdx)
	topLevelStepsEndIdx := -1
	if hasTopLevelSteps {
		topLevelStepsEndIdx = getEngineStepsToTopLevelCodemodFindTopLevelStepsEnd(frontmatterLines)
	}

	withoutEngineSteps := getEngineStepsToTopLevelCodemodRemoveRange(frontmatterLines, stepsStartIdx, stepsEndIdx)
	if getEngineStepsToTopLevelCodemodEngineBlockIsEmpty(withoutEngineSteps) {
		engineStepsCodemodLog.Print("Engine block is empty after removing 'steps', removing it")
		withoutEngineSteps = getEngineStepsToTopLevelCodemodRemoveEmptyEngineBlock(withoutEngineSteps)
	}

	result := getEngineStepsToTopLevelCodemodInsertSteps(
		withoutEngineSteps,
		topLevelStepsLines,
		hasTopLevelSteps,
		topLevelStepsEndIdx,
		stepsStartIdx,
		stepsEndIdx,
	)
	engineStepsCodemodLog.Print("Successfully migrated 'engine.steps' to top-level 'steps'")
	return result, true
}

func getEngineStepsToTopLevelCodemodFindStepsStart(frontmatterLines []string) int {
	engineIndent := ""
	stepsStartIdx := -1
	inEngineBlock := false
	for i, line := range frontmatterLines {
		trimmed := strings.TrimSpace(line)

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
			engineIndent = getIndentation(line)
			inEngineBlock = true
			engineStepsCodemodLog.Printf("Found 'engine:' block at line %d", i+1)
			continue
		}

		// Check if we've exited the engine block
		if inEngineBlock && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			lineIndent := getIndentation(line)
			if len(lineIndent) <= len(engineIndent) {
				inEngineBlock = false
			}
		}

		// Look for steps: within engine block
		if inEngineBlock && stepsStartIdx == -1 && strings.HasPrefix(trimmed, "steps:") {
			stepsStartIdx = i
			engineStepsCodemodLog.Printf("Found 'engine.steps' at line %d", i+1)
		}
	}
	return stepsStartIdx
}

func getEngineStepsToTopLevelCodemodFindBlockEnd(frontmatterLines []string, startIdx int) int {
	stepsIndent := getIndentation(frontmatterLines[startIdx])
	stepsEndIdx := startIdx
	for j := startIdx + 1; j < len(frontmatterLines); j++ {
		line := frontmatterLines[j]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lineIndent := getIndentation(line)
		if len(lineIndent) > len(stepsIndent) {
			stepsEndIdx = j
		} else {
			break
		}
	}
	return stepsEndIdx
}

func getEngineStepsToTopLevelCodemodUnindentSteps(frontmatterLines []string, stepsStartIdx int, stepsEndIdx int) []string {
	stepsIndent := getIndentation(frontmatterLines[stepsStartIdx])
	topLevelStepsLines := make([]string, 0, stepsEndIdx-stepsStartIdx+1)
	for i := stepsStartIdx; i <= stepsEndIdx; i++ {
		line := frontmatterLines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			topLevelStepsLines = append(topLevelStepsLines, "")
			continue
		}
		// Strip the stepsIndent prefix to un-indent to top level
		if strings.HasPrefix(line, stepsIndent) {
			topLevelStepsLines = append(topLevelStepsLines, line[len(stepsIndent):])
		} else {
			topLevelStepsLines = append(topLevelStepsLines, trimmed)
		}
	}
	return topLevelStepsLines
}

func getEngineStepsToTopLevelCodemodFindTopLevelStepsEnd(frontmatterLines []string) int {
	for i, line := range frontmatterLines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "steps:") {
			topStepsIndent := getIndentation(line)
			topLevelStepsEndIdx := getEngineStepsToTopLevelCodemodFindBlockEnd(frontmatterLines, i)
			if topStepsIndent == getIndentation(line) {
				engineStepsCodemodLog.Printf("Top-level 'steps:' ends at line %d", topLevelStepsEndIdx+1)
			}
			return topLevelStepsEndIdx
		}
	}
	return -1
}

func getEngineStepsToTopLevelCodemodRemoveRange(frontmatterLines []string, stepsStartIdx int, stepsEndIdx int) []string {
	withoutEngineSteps := make([]string, 0, len(frontmatterLines))
	for i, line := range frontmatterLines {
		if i >= stepsStartIdx && i <= stepsEndIdx {
			continue
		}
		withoutEngineSteps = append(withoutEngineSteps, line)
	}
	return withoutEngineSteps
}

func getEngineStepsToTopLevelCodemodEngineBlockIsEmpty(withoutEngineSteps []string) bool {
	inEngine := false
	engineIndentLen := 0
	for _, line := range withoutEngineSteps {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
			inEngine = true
			engineIndentLen = len(getIndentation(line))
			// Check for inline value (e.g., "engine: claude")
			val := strings.TrimPrefix(trimmed, "engine:")
			if strings.TrimSpace(val) != "" {
				return false
			}
			continue
		}
		if inEngine {
			if trimmed == "" {
				continue
			}
			lineIndentLen := len(getIndentation(line))
			if lineIndentLen <= engineIndentLen {
				return true
			}
			return false
		}
	}
	return inEngine // if we're still in engine at EOF, it's empty
}

func getEngineStepsToTopLevelCodemodRemoveEmptyEngineBlock(withoutEngineSteps []string) []string {
	cleaned := make([]string, 0, len(withoutEngineSteps))
	engineIndentLen := 0
	inEngine := false
	for _, line := range withoutEngineSteps {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
			inEngine = true
			engineIndentLen = len(getIndentation(line))
			// Remove trailing blank lines already added
			for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
				cleaned = cleaned[:len(cleaned)-1]
			}
			continue
		}
		if inEngine {
			if trimmed == "" {
				continue
			}
			if len(getIndentation(line)) <= engineIndentLen {
				inEngine = false
			} else {
				continue
			}
		}
		cleaned = append(cleaned, line)
	}
	return cleaned
}

func getEngineStepsToTopLevelCodemodInsertSteps(
	withoutEngineSteps []string,
	topLevelStepsLines []string,
	hasTopLevelSteps bool,
	topLevelStepsEndIdx int,
	stepsStartIdx int,
	stepsEndIdx int,
) []string {
	if !hasTopLevelSteps {
		engineStepsCodemodLog.Print("Added engine steps as new top-level 'steps'")
		return append(withoutEngineSteps, topLevelStepsLines...)
	}

	adjustedTopLevelEnd := topLevelStepsEndIdx
	removedCount := stepsEndIdx - stepsStartIdx + 1
	if stepsEndIdx < topLevelStepsEndIdx {
		adjustedTopLevelEnd -= removedCount
	} else if stepsStartIdx <= topLevelStepsEndIdx && stepsEndIdx >= topLevelStepsEndIdx {
		adjustedTopLevelEnd -= removedCount
	}

	result := make([]string, 0, len(withoutEngineSteps)+len(topLevelStepsLines))
	insertedSteps := false
	for i, line := range withoutEngineSteps {
		result = append(result, line)
		if !insertedSteps && i == adjustedTopLevelEnd {
			result = getEngineStepsToTopLevelCodemodAppendStepItems(result, topLevelStepsLines)
			insertedSteps = true
			engineStepsCodemodLog.Print("Appended engine steps to existing top-level 'steps'")
		}
	}
	return result
}

func getEngineStepsToTopLevelCodemodAppendStepItems(result []string, topLevelStepsLines []string) []string {
	for _, stepLine := range topLevelStepsLines {
		if strings.TrimSpace(stepLine) == "steps:" {
			continue
		}
		result = append(result, stepLine)
	}
	return result
}
