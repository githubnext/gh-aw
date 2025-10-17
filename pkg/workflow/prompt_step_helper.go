package workflow

import (
	"strings"
)

// generateStaticPromptStep is a helper function that generates a workflow step
// for appending static prompt text to the prompt file. It encapsulates the common
// pattern used across multiple prompt generators (XPIA, temp folder, playwright, edit tool, etc.)
// to reduce code duplication and ensure consistency.
//
// Parameters:
//   - yaml: The string builder to write the YAML to
//   - description: The name of the workflow step (e.g., "Append XPIA security instructions to prompt")
//   - promptText: The static text content to append to the prompt
//   - shouldInclude: Whether to generate the step (false means skip generation entirely)
//
// Example usage:
//
//	generateStaticPromptStep(yaml,
//	    "Append XPIA security instructions to prompt",
//	    xpiaPromptText,
//	    data.SafetyPrompt)
func generateStaticPromptStep(yaml *strings.Builder, description string, promptText string, shouldInclude bool) {
	// Skip generation if guard condition is false
	if !shouldInclude {
		return
	}

	// Use the existing appendPromptStep helper with a renderer that writes the prompt text
	appendPromptStep(yaml,
		description,
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, promptText, indent)
		},
		"", // no condition
		"          ")
}
