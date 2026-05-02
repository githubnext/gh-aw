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
		awContextLog.Print("No workflow_dispatch trigger found, skipping aw_context injection")
		return onSection
	}
	// Idempotency: skip if already injected
	if strings.Contains(onSection, AwContextInputName+":") {
		awContextLog.Print("aw_context already injected, skipping")
		return onSection
	}
	awContextLog.Print("Injecting aw_context input into workflow_dispatch trigger")

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
		awContextLog.Print("No bare workflow_dispatch: line found, skipping aw_context injection")
		return onSection
	}
	awContextLog.Printf("Found workflow_dispatch at line %d (indent=%d), injecting aw_context", wdLineIdx, wdIndent)

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

// injectAwContextIntoWorkflowCallYAML adds the aw_context input to the workflow_call trigger
// in the given on-section YAML string, if workflow_call is present.
//
// This mirrors injectAwContextIntoOnYAML but targets workflow_call instead of workflow_dispatch.
// When a parent workflow uses call-workflow to invoke this workflow, it forwards the
// aw_context (including experiment assignments) via a `with: aw_context:` input.
// Declaring the input here makes it accessible as `${{ inputs.aw_context }}` in the called
// workflow so the pick-experiment step can inherit the parent's variant selections.
//
// The injection is string-based to preserve existing YAML comments and formatting.
// It handles two cases:
//   - Bare "workflow_call:" (no sub-keys): adds an inputs: block with aw_context
//   - "workflow_call:" with an "inputs:" sub-key: adds aw_context inside inputs
//
// The function is idempotent and is a no-op when workflow_call is not present, or
// when aw_context is already declared in the workflow_call.inputs section.
func injectAwContextIntoWorkflowCallYAML(onSection string) string {
	if !strings.Contains(onSection, "workflow_call") {
		awContextLog.Print("No workflow_call trigger found, skipping aw_context injection for workflow_call")
		return onSection
	}

	// Quick idempotency guard: if aw_context: already appears anywhere, skip.
	// The workflow_dispatch injection runs first and inserts the key into workflow_dispatch.
	// We need to be more precise here: only skip if aw_context appears after a workflow_call: line.
	// For simplicity we use a full-parse approach via line scan below instead of this guard.

	awContextLog.Print("Injecting aw_context input into workflow_call trigger")

	lines := strings.Split(onSection, "\n")

	// Find the workflow_call: line (bare — no sub-value on same line).
	wcLineIdx := -1
	wcIndent := 0
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		rest, found := strings.CutPrefix(stripped, "workflow_call:")
		if found {
			rest = strings.TrimSpace(rest)
			if rest == "" || rest == "null" || rest == "~" {
				wcLineIdx = i
				wcIndent = len(line) - len(stripped)
				break
			}
		}
	}

	if wcLineIdx == -1 {
		awContextLog.Print("No bare workflow_call: line found, skipping aw_context injection")
		return onSection
	}

	// Check if aw_context is already declared inside this workflow_call block.
	for i := wcLineIdx + 1; i < len(lines); i++ {
		stripped := strings.TrimLeft(lines[i], " \t")
		lineIndent := len(lines[i]) - len(stripped)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		if lineIndent <= wcIndent {
			break // left the workflow_call block
		}
		if strings.HasPrefix(stripped, AwContextInputName+":") {
			awContextLog.Print("aw_context already declared in workflow_call inputs, skipping")
			return onSection
		}
	}

	awContextLog.Printf("Found workflow_call at line %d (indent=%d), injecting aw_context", wcLineIdx, wcIndent)

	// Look for an "inputs:" key directly inside workflow_call (at wcIndent+2 depth).
	// Only the first non-empty, non-comment line after wcLineIdx matters.
	inputsLineIdx := -1
	for i := wcLineIdx + 1; i < len(lines); i++ {
		stripped := strings.TrimLeft(lines[i], " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		lineIndent := len(lines[i]) - len(stripped)
		if lineIndent <= wcIndent {
			break // left workflow_call block entirely
		}
		if strings.HasPrefix(stripped, "inputs:") {
			inputsLineIdx = i
		}
		break // only inspect the first substantive child key
	}

	awContextLines := buildAwContextInputLines(wcIndent)

	result := make([]string, 0, len(lines)+len(awContextLines)+1)
	for i, line := range lines {
		// When the workflow_call: line contains an explicit null/~ value,
		// replace it with a bare workflow_call: so sub-keys can follow.
		if i == wcLineIdx && (strings.HasSuffix(strings.TrimSpace(line), " null") ||
			strings.HasSuffix(strings.TrimSpace(line), " ~")) {
			stripped := strings.TrimLeft(line, " \t")
			line = strings.Repeat(" ", wcIndent) + strings.SplitN(stripped, ":", 2)[0] + ":"
		}
		result = append(result, line)

		if inputsLineIdx != -1 && i == inputsLineIdx {
			// Insert aw_context as the first entry under existing inputs:
			result = append(result, awContextLines...)
		} else if inputsLineIdx == -1 && i == wcLineIdx {
			// workflow_call is bare — add inputs: + aw_context
			result = append(result, strings.Repeat(" ", wcIndent+2)+"inputs:")
			result = append(result, awContextLines...)
		}
	}

	return strings.Join(result, "\n")
}

// buildAwContextInputLines returns the indented YAML lines for the aw_context input
// definition, sized relative to the trigger line's indentation.
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
