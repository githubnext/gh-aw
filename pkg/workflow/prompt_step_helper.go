// Package workflow provides helper functions for generating prompt workflow steps.
//
// This file contains utilities for building GitHub Actions workflow steps that
// append prompt text to prompt files used by AI engines. These helpers extract
// common patterns used across multiple prompt generators (XPIA, temp folder,
// playwright, edit tool, etc.) to reduce code duplication and ensure security.
//
// # Organization Rationale
//
// These prompt step helpers are grouped here because they:
//   - Provide common patterns for prompt text generation used by 5+ generators
//   - Handle GitHub Actions expression extraction for security
//   - Ensure consistent prompt step formatting across engines
//   - Centralize template injection prevention logic
//
// This follows the helper file conventions documented in the developer instructions.
// See skills/developer/SKILL.md#helper-file-conventions for details.
//
// # Key Functions
//
// Static Prompt Generation:
//   - generateStaticPromptStep() - Generate steps for static prompt text
//   - generateStaticPromptStepWithExpressions() - Generate steps with secure expression handling
//
// # Usage Patterns
//
// These helpers are used when generating workflow steps that append text to
// prompt files. They follow two patterns:
//
//  1. **Static Text** (no GitHub Actions expressions):
//     ```go
//     generateStaticPromptStep(yaml,
//     "Append XPIA security instructions to prompt",
//     xpiaPromptText,
//     data.SafetyPrompt)
//     ```
//
//  2. **Text with Expressions** (contains ${{ ... }}):
//     ```go
//     generateStaticPromptStepWithExpressions(yaml,
//     "Append dynamic context to prompt",
//     promptWithExpressions,
//     shouldInclude)
//     ```
//
// The expression-aware helper extracts GitHub Actions expressions into
// environment variables to prevent template injection vulnerabilities.
//
// # Security Considerations
//
// Always use generateStaticPromptStepWithExpressions() when prompt text
// contains GitHub Actions expressions (${{ ... }}). This ensures:
//   - Expressions are evaluated in controlled env: context
//   - No inline shell script interpolation (prevents injection)
//   - Safe placeholder substitution via JavaScript
//
// See specs/template-injection-prevention.md for security details.
//
// # When to Use vs Alternatives
//
// Use these helpers when:
//   - Generating workflow steps that append text to prompt files
//   - Working with static or expression-containing prompt text
//   - Need consistent prompt step formatting across engines
//
// For other prompt-related functionality, see:
//   - *_engine.go files for engine-specific prompt generation
//   - engine_helpers.go for shared engine utilities
package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var promptStepHelperLog = logger.New("workflow:prompt_step_helper")

// generateStaticPromptStep is a helper function that generates a workflow step
// for appending static prompt text to the prompt file. It encapsulates the common
// pattern used across multiple prompt generators (XPIA, temp folder, playwright, edit tool, etc.)
// to reduce code duplication and ensure consistency.
//
// Parameters:
//   - yaml: The string builder to write the YAML to
//   - description: The name of the workflow step (e.g., "Append XPIA security instructions to prompt")
//   - promptText: The static text content to append to the prompt (used for backward compatibility)
//   - shouldInclude: Whether to generate the step (false means skip generation entirely)
//
// Example usage:
//
//	generateStaticPromptStep(yaml,
//	    "Append XPIA security instructions to prompt",
//	    xpiaPromptText,
//	    data.SafetyPrompt)
//
// Deprecated: This function is kept for backward compatibility with inline prompts.
// Use generateStaticPromptStepFromFile for new code.
func generateStaticPromptStep(yaml *strings.Builder, description string, promptText string, shouldInclude bool) {
	promptStepHelperLog.Printf("Generating static prompt step: description=%s, shouldInclude=%t", description, shouldInclude)
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

// generateStaticPromptStepFromFile generates a workflow step for appending a prompt file
// from /opt/gh-aw/prompts/ to the prompt file. This is the preferred approach as it
// keeps prompt content in markdown files instead of embedding in the binary.
//
// Parameters:
//   - yaml: The string builder to write the YAML to
//   - description: The name of the workflow step (e.g., "Append XPIA security instructions to prompt")
//   - promptFilename: The filename of the prompt in /opt/gh-aw/prompts/ (e.g., "xpia_prompt.md")
//   - shouldInclude: Whether to generate the step (false means skip generation entirely)
func generateStaticPromptStepFromFile(yaml *strings.Builder, description string, promptFilename string, shouldInclude bool) {
	promptStepHelperLog.Printf("Generating static prompt step from file: description=%s, file=%s, shouldInclude=%t", description, promptFilename, shouldInclude)
	// Skip generation if guard condition is false
	if !shouldInclude {
		return
	}

	// Use the existing appendPromptStep helper with a renderer that cats the file
	appendPromptStep(yaml,
		description,
		func(y *strings.Builder, indent string) {
			WritePromptFileToYAML(y, promptFilename, indent)
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
	promptStepHelperLog.Printf("Generating static prompt step with expressions: description=%s, shouldInclude=%t", description, shouldInclude)
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
	// Write prompt text with placeholders
	WritePromptTextToYAMLWithPlaceholders(yaml, modifiedPromptText, "          ")

	// Generate JavaScript-based placeholder substitution step
	generatePlaceholderSubstitutionStep(yaml, expressionMappings, "      ")
}

// TODO: generateStaticPromptStepFromFileWithExpressions could be implemented in the future
// to generate workflow steps for appending prompt files that contain GitHub Actions expressions.
// For now, we use the text-based approach with generateStaticPromptStepWithExpressions instead.
// See commit history if this needs to be restored.
