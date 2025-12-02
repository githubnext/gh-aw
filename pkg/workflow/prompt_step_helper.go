package workflow

import (
	"fmt"
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

// generateStaticPromptStepWithExpressions generates a workflow step for appending prompt text
// that contains GitHub Actions expressions (${{ ... }}). It extracts the expressions into
// environment variables and uses shell variable expansion in the heredoc for security.
//
// This prevents template injection vulnerabilities by ensuring expressions are evaluated
// in the env: section (controlled context) rather than inline in shell scripts.
//
// Parameters:
//   - yaml: The string builder to write the YAML to
//   - description: The name of the workflow step
//   - promptText: The prompt text content that may contain ${{ ... }} expressions
//   - shouldInclude: Whether to generate the step (false means skip generation entirely)
func generateStaticPromptStepWithExpressions(yaml *strings.Builder, description string, promptText string, shouldInclude bool) {
	// Skip generation if guard condition is false
	if !shouldInclude {
		return
	}

	// Extract GitHub Actions expressions and create environment variable mappings
	extractor := NewExpressionExtractor()
	expressionMappings, err := extractor.ExtractExpressions(promptText)
	if err != nil {
		// If extraction fails, fall back to the standard method
		generateStaticPromptStep(yaml, description, promptText, shouldInclude)
		return
	}

	// Replace expressions with environment variable references in the prompt text
	modifiedPromptText := promptText
	if len(expressionMappings) > 0 {
		modifiedPromptText = extractor.ReplaceExpressionsWithEnvVars(promptText)
	}

	// Generate the step with env vars for the extracted expressions
	yaml.WriteString("      - name: " + description + "\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")

	// Add environment variables for each extracted expression
	// The expressions are evaluated in the env: section (controlled context)
	for _, mapping := range expressionMappings {
		fmt.Fprintf(yaml, "          %s: ${{ %s }}\n", mapping.EnvVar, mapping.Content)
	}

	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, modifiedPromptText, "          ")
}
