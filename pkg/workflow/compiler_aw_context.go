package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var awContextLog = logger.New("workflow:compiler_aw_context")

// awContextInputName is the name of the internal aw_context input appended to
// every workflow_dispatch trigger. It carries caller metadata at dispatch time
// and is ignored when the workflow is invoked manually.
const awContextInputName = "aw_context"

// awContextInputDescription is the user-visible description for the aw_context
// input. It signals that the field is managed by gh-aw and should not be set
// manually.
const awContextInputDescription = "Internal: contextual dispatch information (JSON). Do not provide manually."

// injectAwContextInput appends an "aw_context" optional string input to the
// on.workflow_dispatch.inputs section of the serialised on: YAML block.
//
// The input is inserted only when the on section contains a workflow_dispatch
// trigger. Existing user-defined inputs and YAML comments (such as commented-out
// gh-aw extension fields) are fully preserved because only the workflow_dispatch
// block is re-serialised; the rest of the string is left unchanged.
//
// If aw_context is already present, the function is a no-op.
func injectAwContextInput(onSection string) string {
	if !strings.Contains(onSection, "workflow_dispatch") {
		return onSection
	}

	awContextLog.Print("Injecting aw_context input into workflow_dispatch.inputs")

	// Parse the on: YAML block (read-only - only to inspect the current structure)
	var onData map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &onData); err != nil {
		awContextLog.Printf("Warning: failed to parse on section for aw_context injection: %v", err)
		return onSection
	}

	onMap, ok := onData["on"].(map[string]any)
	if !ok {
		return onSection
	}

	wdVal, hasWD := onMap["workflow_dispatch"]
	if !hasWD {
		return onSection
	}

	// Normalise workflow_dispatch value: may be nil when declared without options.
	var wdMap map[string]any
	if wdVal == nil {
		wdMap = make(map[string]any)
	} else if m, ok := wdVal.(map[string]any); ok {
		wdMap = m
	} else {
		// workflow_dispatch is a non-map value (e.g. a boolean shorthand) — leave as-is.
		return onSection
	}

	// Ensure inputs map exists and check for existing aw_context.
	var inputsMap map[string]any
	if existing, ok := wdMap["inputs"].(map[string]any); ok {
		inputsMap = existing
		if _, exists := inputsMap[awContextInputName]; exists {
			awContextLog.Print("aw_context input already present, skipping injection")
			return onSection
		}
	} else {
		inputsMap = make(map[string]any)
	}

	// Append the aw_context input definition.
	inputsMap[awContextInputName] = map[string]any{
		"description": awContextInputDescription,
		"type":        "string",
		"required":    false,
	}
	wdMap["inputs"] = inputsMap

	// Serialise only the workflow_dispatch block (not the entire on: section).
	// This avoids re-marshaling commented-out fields (e.g. "# permissions:") that
	// were written by commentOutProcessedFieldsInOnSection.
	orderedWD := OrderMapFields(wdMap, []string{})
	newWDYAML, err := yaml.MarshalWithOptions(
		yaml.MapSlice{{Key: "workflow_dispatch", Value: orderedWD}},
		DefaultMarshalOptions...,
	)
	if err != nil {
		awContextLog.Printf("Warning: failed to marshal workflow_dispatch section: %v", err)
		return onSection
	}

	// Re-quote any cron expressions that the marshaller may have un-quoted.
	newWDStr := parser.QuoteCronExpressions(strings.TrimSuffix(string(newWDYAML), "\n"))

	// Splice the new workflow_dispatch block into the original string, replacing
	// only the old workflow_dispatch block.  The rest of the string (including
	// comments) is preserved unchanged.
	result, ok := replaceWorkflowDispatchBlock(onSection, newWDStr)
	if !ok {
		awContextLog.Print("Warning: could not locate workflow_dispatch block for splicing; returning original")
		return onSection
	}

	awContextLog.Print("Successfully injected aw_context input")
	return result
}

// replaceWorkflowDispatchBlock replaces the workflow_dispatch: block in
// onSection with newWDStr (the newly-serialised block).
//
// It locates the line that starts with whitespace + "workflow_dispatch:" and
// determines the extent of the block by finding all subsequent lines that are
// indented more deeply than the workflow_dispatch: line.  The block is then
// replaced by the indented lines from newWDStr.
//
// Returns the updated string and true on success, or the original string and
// false if the workflow_dispatch: line could not be found.
func replaceWorkflowDispatchBlock(onSection, newWDStr string) (string, bool) {
	lines := strings.Split(onSection, "\n")

	// Find the workflow_dispatch: line.
	wdLineIdx := -1
	wdIndent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
		keyPart := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(keyPart, "workflow_dispatch:") {
			wdIndent = line[:leadingSpaces]
			wdLineIdx = i
			break
		}
	}

	if wdLineIdx == -1 {
		return onSection, false
	}

	// Determine the extent of the workflow_dispatch block: collect all following
	// lines that are more deeply indented than workflow_dispatch:.
	blockEnd := wdLineIdx
	for i := wdLineIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			// blank or comment lines — include them only if they belong to the block
			// (we peek ahead; for simplicity we skip them)
			continue
		}
		lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
		if lineIndent <= len(wdIndent) {
			break
		}
		blockEnd = i
	}

	// Build the replacement lines by taking the new workflow_dispatch YAML and
	// prepending each line with the same base indentation as the original.
	newWDLines := strings.Split(newWDStr, "\n")
	replacement := make([]string, 0, len(newWDLines))
	for i, l := range newWDLines {
		if i == 0 {
			// First line already has the correct indentation from the original.
			replacement = append(replacement, wdIndent+l)
		} else if strings.TrimSpace(l) == "" {
			replacement = append(replacement, l)
		} else {
			replacement = append(replacement, wdIndent+l)
		}
	}

	// Assemble: lines before the block + replacement + lines after the block.
	result := make([]string, 0, len(lines))
	result = append(result, lines[:wdLineIdx]...)
	result = append(result, replacement...)
	result = append(result, lines[blockEnd+1:]...)

	return strings.Join(result, "\n"), true
}

// awContextInputForYAML returns the InputDefinition for the aw_context input.
// It is used by tests to verify the compiled YAML structure.
func awContextInputDefinition() map[string]any {
	return map[string]any{
		"description": awContextInputDescription,
		"type":        "string",
		"required":    false,
	}
}

// hasAwContextInput reports whether the serialised workflow YAML already
// declares aw_context as a workflow_dispatch input.
func hasAwContextInput(onSection string) bool {
	return strings.Contains(onSection, fmt.Sprintf("%s:", awContextInputName))
}
