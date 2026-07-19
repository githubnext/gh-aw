package workflow

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/goccy/go-yaml"
)

var stepConversionLog = logger.New("workflow:compiler_yaml_step_conversion")

// ConvertStepToYAML converts a step map to YAML string with proper indentation.
// This is a shared utility function used by all engines and the compiler.
func ConvertStepToYAML(stepMap map[string]any) (string, error) {
	stepConversionLog.Printf("Converting step to YAML: fields=%d", len(stepMap))
	// Use OrderMapFields to get ordered MapSlice
	orderedStep := OrderMapFields(stepMap, constants.PriorityStepFields)

	// Wrap in array for step list format and marshal with proper options
	yamlBytes, err := yaml.MarshalWithOptions([]yaml.MapSlice{orderedStep}, DefaultMarshalOptions...)
	if err != nil {
		stepConversionLog.Printf("Step YAML marshal failed: %v", err)
		return "", fmt.Errorf("failed to marshal step to YAML: %w", err)
	}

	// Convert to string and adjust base indentation to match GitHub Actions format
	yamlStr := string(yamlBytes)

	// Post-process to move version comments outside of quoted uses values
	// This handles cases like: uses: "slug@sha # v1"  ->  uses: slug@sha # v1
	yamlStr = unquoteUsesWithComments(yamlStr)
	yamlStr = quoteEnvValuesContainingColonSpace(yamlStr)

	// Add 6 spaces to the beginning of each line to match GitHub Actions step indentation
	lines := strings.Split(strings.TrimSpace(yamlStr), "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result.WriteString("\n")
		} else {
			result.WriteString("      " + line + "\n")
		}
	}

	stepConversionLog.Printf("Step conversion complete: %d lines generated", len(lines))
	return result.String(), nil
}

// unquoteUsesWithComments removes quotes from uses values that contain version comments.
// Transforms: uses: "slug@sha # v1"  ->  uses: slug@sha # v1
// This is needed because the YAML marshaller quotes strings containing #, but GitHub Actions
// expects unquoted uses values with inline comments.
func unquoteUsesWithComments(yamlStr string) string {
	stepConversionLog.Printf("Post-processing YAML to unquote uses-with-comments: %d chars", len(yamlStr))
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		// Look for uses: followed by a quoted string containing a # comment
		// This handles various indentation levels and formats
		trimmed := strings.TrimSpace(line)

		// Check if line contains uses: with a quoted value
		if !strings.Contains(trimmed, "uses: \"") {
			continue
		}

		// Check if the quoted value contains a version comment
		if !strings.Contains(trimmed, " # ") {
			continue
		}

		// Find the position of uses: " in the original line
		usesIdx := strings.Index(line, "uses: \"")
		if usesIdx == -1 {
			continue
		}

		// Extract the part before uses: (indentation)
		prefix := line[:usesIdx]

		// Find the opening and closing quotes
		quoteStart := usesIdx + 7 // len("uses: \"")
		quoteEnd := strings.Index(line[quoteStart:], "\"")
		if quoteEnd == -1 {
			continue
		}
		quoteEnd += quoteStart

		// Extract the quoted content
		quotedContent := line[quoteStart:quoteEnd]

		// Extract any content after the closing quote
		suffix := line[quoteEnd+1:]

		// Reconstruct the line without quotes
		lines[i] = prefix + "uses: " + quotedContent + suffix
	}
	return strings.Join(lines, "\n")
}

// renderStepFromMap renders a GitHub Actions step from a map to YAML
func (c *Compiler) renderStepFromMap(out *strings.Builder, step map[string]any, data *WorkflowData, indent string) {
	if sanitized, warnings, changed := sanitizeRunStepExpressions(step); changed {
		stepConversionLog.Printf("Sanitized run-step expressions: %d warning(s) emitted", len(warnings))
		for _, w := range warnings {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(w))
			c.IncrementWarningCount()
		}
		step = sanitized
	}

	stepName, _ := step["name"].(string)
	stepConversionLog.Printf("Rendering step from map: name=%q, fields=%d", stepName, len(step))
	// Start the step with a dash
	out.WriteString(indent + "- ")

	// Track if we've written the first line
	firstField := true

	// Order of fields to write (matches GitHub Actions convention)
	fieldOrder := []string{"name", "id", "if", "uses", "with", "run", "env", "working-directory", "continue-on-error", "timeout-minutes", "shell"}

	for _, field := range fieldOrder {
		if value, exists := step[field]; exists {
			renderStepField(out, field, value, indent, &firstField, field == "run")
		}
	}

	for field, value := range step {
		skip := slices.Contains(fieldOrder, field)
		if skip {
			continue
		}
		renderStepField(out, field, value, indent, &firstField, true)
	}
}

func renderStepField(out *strings.Builder, field string, value any, indent string, firstField *bool, multilineStrings bool) {
	if !*firstField {
		out.WriteString(indent + "  ")
	}
	*firstField = false

	switch v := value.(type) {
	case string:
		renderStepStringField(out, field, v, indent, multilineStrings)
	case map[string]any:
		renderStepMapField(out, field, v, indent)
	default:
		fmt.Fprintf(out, "%s: %v\n", field, v)
	}
}

func renderStepStringField(out *strings.Builder, field, value, indent string, multilineStrings bool) {
	if multilineStrings && strings.Contains(value, "\n") {
		fmt.Fprintf(out, "%s: |\n", field)
		lines := strings.SplitSeq(value, "\n")
		for line := range lines {
			fmt.Fprintf(out, "%s    %s\n", indent, line)
		}
	} else {
		fmt.Fprintf(out, "%s: %s\n", field, value)
	}
}

func renderStepMapField(out *strings.Builder, field string, value map[string]any, indent string) {
	fmt.Fprintf(out, "%s:\n", field)
	for _, key := range sliceutil.SortedKeys(value) {
		if field == "env" {
			fmt.Fprintf(out, "%s    %s: %s\n", indent, key, formatStepEnvValueForYAML(value[key]))
		} else {
			fmt.Fprintf(out, "%s    %s: %v\n", indent, key, value[key])
		}
	}
}

func formatStepEnvValueForYAML(value any) string {
	strValue, ok := value.(string)
	if !ok {
		return fmt.Sprint(value)
	}
	return yamlStringValue(strValue)
}
