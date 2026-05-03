package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var awContextLog = logger.New("workflow:compiler_aw_context")

// AwContextInputName is the name of the internal aw_context workflow_dispatch input.
// It is managed internally by the agentic workflow system and should not be surfaced to users.
const AwContextInputName = "aw_context"

// awContextInputDescription is the description for the aw_context workflow_dispatch input.
// It signals to users that this input is managed internally by the agentic workflow system.
const awContextInputDescription = "Agent caller context (used internally by Agentic Workflows)."

// injectAwContextIntoOnYAML adds the aw_context input to internal workflow triggers
// in the given on-section YAML string.
//
// The injection is string-based to preserve existing YAML comments and formatting.
// It handles these triggers independently:
//   - workflow_dispatch
//   - workflow_call
//
// For each trigger it supports two cases:
//   - Bare trigger line (no sub-keys): adds an inputs: block with aw_context
//   - Trigger with an existing inputs: sub-key: adds aw_context inside inputs
//
// The function is idempotent: calling it twice produces the same result.
func injectAwContextIntoOnYAML(onSection string) string {
	updated := injectAwContextIntoTrigger(onSection, "workflow_dispatch")
	updated = injectAwContextIntoTrigger(updated, "workflow_call")
	return updated
}

func injectAwContextIntoTrigger(onSection string, triggerName string) string {
	if !strings.Contains(onSection, triggerName) {
		awContextLog.Printf("No %s trigger found, skipping aw_context injection", triggerName)
		return onSection
	}
	awContextLog.Printf("Injecting aw_context input into %s trigger", triggerName)

	lines := strings.Split(onSection, "\n")

	// Find the trigger line (bare — no sub-value on same line)
	triggerLineIdx := -1
	triggerIndent := 0
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		rest, found := strings.CutPrefix(stripped, triggerName+":")
		if found {
			rest = strings.TrimSpace(rest)
			if rest == "" || rest == "null" || rest == "~" {
				triggerLineIdx = i
				triggerIndent = len(line) - len(stripped)
				break
			}
		}
	}

	if triggerLineIdx == -1 {
		awContextLog.Printf("No bare %s: line found, skipping aw_context injection", triggerName)
		return onSection
	}
	awContextLog.Printf("Found %s at line %d (indent=%d), injecting aw_context", triggerName, triggerLineIdx, triggerIndent)

	// Look for an "inputs:" key directly inside the trigger block.
	// Only the first non-empty, non-comment line after the trigger matters.
	inputsLineIdx := -1
	for i := triggerLineIdx + 1; i < len(lines); i++ {
		stripped := strings.TrimLeft(lines[i], " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		lineIndent := len(lines[i]) - len(stripped)
		if lineIndent <= triggerIndent {
			break // left workflow_dispatch block entirely
		}
		if strings.HasPrefix(stripped, "inputs:") {
			inputsLineIdx = i
		}
		break // only inspect the first substantive child key
	}

	if inputsLineIdx != -1 {
		inputsIndent := len(lines[inputsLineIdx]) - len(strings.TrimLeft(lines[inputsLineIdx], " \t"))
		for i := inputsLineIdx + 1; i < len(lines); i++ {
			stripped := strings.TrimLeft(lines[i], " \t")
			if stripped == "" || strings.HasPrefix(stripped, "#") {
				continue
			}
			lineIndent := len(lines[i]) - len(stripped)
			if lineIndent <= inputsIndent {
				break
			}
			if strings.HasPrefix(stripped, AwContextInputName+":") {
				awContextLog.Printf("aw_context already injected into %s, skipping", triggerName)
				return onSection
			}
		}
	}

	awContextLines := buildAwContextInputLines(triggerIndent)

	result := make([]string, 0, len(lines)+len(awContextLines)+1)
	for i, line := range lines {
		// When the trigger line contains an explicit null/~ value,
		// replace it with a bare trigger so sub-keys can follow.
		if i == triggerLineIdx && (strings.HasSuffix(strings.TrimSpace(line), " null") ||
			strings.HasSuffix(strings.TrimSpace(line), " ~")) {
			stripped := strings.TrimLeft(line, " \t")
			line = strings.Repeat(" ", triggerIndent) + strings.SplitN(stripped, ":", 2)[0] + ":"
		}
		result = append(result, line)

		if inputsLineIdx != -1 && i == inputsLineIdx {
			// Insert aw_context as the first entry under existing inputs:
			result = append(result, awContextLines...)
		} else if inputsLineIdx == -1 && i == triggerLineIdx {
			// Trigger is bare — add inputs: + aw_context
			result = append(result, strings.Repeat(" ", triggerIndent+2)+"inputs:")
			result = append(result, awContextLines...)
		}
	}

	return strings.Join(result, "\n")
}

// buildAwContextInputLines returns the indented YAML lines for the aw_context input
// definition, sized relative to the workflow_dispatch: line's indentation.
func buildAwContextInputLines(wdIndent int) []string {
	awIndent := strings.Repeat(" ", wdIndent+4)   // under inputs:
	propIndent := strings.Repeat(" ", wdIndent+6) // properties of aw_context
	return []string{
		awIndent + AwContextInputName + ":",
		propIndent + "default: \"\"",
		propIndent + "description: " + awContextInputDescription,
		propIndent + "required: false",
		propIndent + "type: string",
	}
}
