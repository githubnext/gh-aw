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
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
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

			// Parse frontmatter lines
			frontmatterLines, markdown, err := parseFrontmatterLines(content)
			if err != nil {
				return content, false, err
			}

			// Find engine block and the steps field within it
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
				if inEngineBlock && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
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

			if stepsStartIdx == -1 {
				return content, false, nil
			}

			// Find end of the steps block within engine
			stepsIndent := getIndentation(frontmatterLines[stepsStartIdx])
			stepsEndIdx := stepsStartIdx
			for j := stepsStartIdx + 1; j < len(frontmatterLines); j++ {
				line := frontmatterLines[j]
				trimmed := strings.TrimSpace(line)

				if len(trimmed) == 0 {
					continue
				}

				lineIndent := getIndentation(line)
				if len(lineIndent) > len(stepsIndent) {
					stepsEndIdx = j
				} else {
					break
				}
			}

			engineStepsCodemodLog.Printf("'engine.steps' spans lines %d to %d", stepsStartIdx+1, stepsEndIdx+1)

			// Extract the steps lines and un-indent them (remove the engine-level indentation)
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

			// Find existing top-level steps block (if any)
			// Only treat as existing steps if it's actually a sequence
			topLevelStepsEndIdx := -1
			hasTopLevelSteps := false
			if stepsVal, exists := frontmatter["steps"]; exists {
				if _, isSlice := stepsVal.([]any); isSlice {
					hasTopLevelSteps = true
					engineStepsCodemodLog.Print("Found existing top-level 'steps'")
				} else {
					engineStepsCodemodLog.Print("Top-level 'steps' exists but is not a sequence; treating as absent")
				}
			}

			if hasTopLevelSteps {
				// Find the end of the top-level steps block in the lines
				for i, line := range frontmatterLines {
					trimmed := strings.TrimSpace(line)
					if isTopLevelKey(line) && strings.HasPrefix(trimmed, "steps:") {
						topStepsIndent := getIndentation(line)
						topLevelStepsEndIdx = i
						for j := i + 1; j < len(frontmatterLines); j++ {
							l := frontmatterLines[j]
							t := strings.TrimSpace(l)
							if len(t) == 0 {
								continue
							}
							if len(getIndentation(l)) > len(topStepsIndent) {
								topLevelStepsEndIdx = j
							} else {
								break
							}
						}
						engineStepsCodemodLog.Printf("Top-level 'steps:' ends at line %d", topLevelStepsEndIdx+1)
						break
					}
				}
			}

			// Build new frontmatter: remove engine.steps lines and insert at top level
			// Pass 1: build lines without engine.steps
			withoutEngineSteps := make([]string, 0, len(frontmatterLines))
			for i, line := range frontmatterLines {
				if i >= stepsStartIdx && i <= stepsEndIdx {
					continue
				}
				withoutEngineSteps = append(withoutEngineSteps, line)
			}

			// Pass 1b: if the engine block is now empty (only blank lines or id: key),
			// check whether any non-steps content remains under engine:
			engineBlockIsEmpty := func() bool {
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
						if len(trimmed) == 0 {
							continue
						}
						lineIndentLen := len(getIndentation(line))
						if lineIndentLen <= engineIndentLen {
							// Exited engine block with no content found
							return true
						}
						// There is content under engine (e.g., id:, model:, env:)
						return false
					}
				}
				return inEngine // if we're still in engine at EOF, it's empty
			}()

			if engineBlockIsEmpty {
				engineStepsCodemodLog.Print("Engine block is empty after removing 'steps', removing it")
				// Remove the engine block (the engine: line and any blank lines around it)
				cleaned := make([]string, 0, len(withoutEngineSteps))
				engineIndentLen := 0
				inEngine := false
				for i, line := range withoutEngineSteps {
					trimmed := strings.TrimSpace(line)
					if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
						inEngine = true
						engineIndentLen = len(getIndentation(line))
						// Remove trailing blank lines already added
						for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
							cleaned = cleaned[:len(cleaned)-1]
						}
						_ = i
						continue
					}
					if inEngine {
						if len(trimmed) == 0 {
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
				withoutEngineSteps = cleaned
			}

			// Pass 2: insert engine steps at top level
			var result []string
			if !hasTopLevelSteps {
				// Append engine steps at the end (as new top-level steps field)
				result = append(withoutEngineSteps, topLevelStepsLines...)
				engineStepsCodemodLog.Print("Added engine steps as new top-level 'steps'")
			} else {
				// Append engine step items after the top-level steps block
				// Since we removed engine.steps lines, re-find the end of top-level steps
				adjustedTopLevelEnd := topLevelStepsEndIdx
				removedCount := stepsEndIdx - stepsStartIdx + 1
				// Only adjust if the engine.steps came before the top-level steps end
				if stepsEndIdx < topLevelStepsEndIdx {
					adjustedTopLevelEnd -= removedCount
				} else if stepsStartIdx <= topLevelStepsEndIdx && stepsEndIdx >= topLevelStepsEndIdx {
					// engine.steps overlaps with top-level steps end (shouldn't happen but handle gracefully)
					adjustedTopLevelEnd -= removedCount
				}

				result = make([]string, 0, len(withoutEngineSteps)+len(topLevelStepsLines))
				insertedSteps := false
				for i, line := range withoutEngineSteps {
					result = append(result, line)
					if !insertedSteps && i == adjustedTopLevelEnd {
						// Append the step items (skip the "steps:" header since one already exists)
						for _, stepLine := range topLevelStepsLines {
							if strings.TrimSpace(stepLine) == "steps:" {
								continue
							}
							result = append(result, stepLine)
						}
						insertedSteps = true
						engineStepsCodemodLog.Print("Appended engine steps to existing top-level 'steps'")
					}
				}
			}

			newContent := reconstructContent(result, markdown)
			engineStepsCodemodLog.Print("Successfully migrated 'engine.steps' to top-level 'steps'")
			return newContent, true, nil
		},
	}
}
