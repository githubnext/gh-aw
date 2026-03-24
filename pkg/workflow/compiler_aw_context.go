package workflow

import (
	"strings"
)

// awContextInputDescription is the description for the aw_context workflow_dispatch input.
// It signals to users that this input is managed internally by the agentic workflow system.
const awContextInputDescription = "Agent caller context (used internally by Agentic Workflows)."

// injectAwContextIntoOnYAML adds the aw_context input to the workflow_dispatch trigger
// in the given on-section YAML string, if workflow_dispatch is present.
//
// The injection is string-based to preserve existing YAML comments and formatting.
// It handles two cases:
//   - Bare "workflow_dispatch:" (no sub-keys): adds an inputs: block with aw_context
//   - "workflow_dispatch:" with an "inputs:" sub-key: adds aw_context inside inputs
//
// The function is idempotent: calling it twice produces the same result.
func injectAwContextIntoOnYAML(onSection string) string {
	if !strings.Contains(onSection, "workflow_dispatch") {
		return onSection
	}
	// Idempotency: skip if already injected
	if strings.Contains(onSection, "aw_context:") {
		return onSection
	}

	lines := strings.Split(onSection, "\n")

	// Find the workflow_dispatch: line (bare — no sub-value on same line)
	wdLineIdx := -1
	wdIndent := 0
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		rest, found := strings.CutPrefix(stripped, "workflow_dispatch:")
		if found {
			rest = strings.TrimSpace(rest)
			if rest == "" || rest == "null" || rest == "~" {
				wdLineIdx = i
				wdIndent = len(line) - len(stripped)
				break
			}
		}
	}

	if wdLineIdx == -1 {
		return onSection
	}

	// Look for an "inputs:" key directly inside workflow_dispatch (at wdIndent+2 depth).
	// Only the first non-empty, non-comment line after wdLineIdx matters.
	inputsLineIdx := -1
	for i := wdLineIdx + 1; i < len(lines); i++ {
		stripped := strings.TrimLeft(lines[i], " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		lineIndent := len(lines[i]) - len(stripped)
		if lineIndent <= wdIndent {
			break // left workflow_dispatch block entirely
		}
		if strings.HasPrefix(stripped, "inputs:") {
			inputsLineIdx = i
		}
		break // only inspect the first substantive child key
	}

	awContextLines := buildAwContextInputLines(wdIndent)

	result := make([]string, 0, len(lines)+len(awContextLines)+1)
	for i, line := range lines {
		// When the workflow_dispatch: line contains an explicit null/~ value,
		// replace it with a bare workflow_dispatch: so sub-keys can follow.
		if i == wdLineIdx && (strings.HasSuffix(strings.TrimSpace(line), " null") ||
			strings.HasSuffix(strings.TrimSpace(line), " ~")) {
			stripped := strings.TrimLeft(line, " \t")
			line = strings.Repeat(" ", wdIndent) + strings.SplitN(stripped, ":", 2)[0] + ":"
		}
		result = append(result, line)

		if inputsLineIdx != -1 && i == inputsLineIdx {
			// Insert aw_context as the first entry under existing inputs:
			result = append(result, awContextLines...)
		} else if inputsLineIdx == -1 && i == wdLineIdx {
			// workflow_dispatch is bare — add inputs: + aw_context
			result = append(result, strings.Repeat(" ", wdIndent+2)+"inputs:")
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
		awIndent + "aw_context:",
		propIndent + "default: \"\"",
		propIndent + "description: " + awContextInputDescription,
		propIndent + "required: false",
		propIndent + "type: string",
	}
}
