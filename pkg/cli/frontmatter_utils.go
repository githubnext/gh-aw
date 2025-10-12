package cli

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// UpdateFieldInFrontmatter updates a field in the frontmatter while preserving the original formatting
// when possible. It tries to preserve whitespace, comments, and formatting by working with the raw
// frontmatter lines, similar to how addSourceToWorkflow works.
func UpdateFieldInFrontmatter(content, fieldName, fieldValue string) (string, error) {
	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Try to preserve original frontmatter formatting by manually updating the field
	if len(result.FrontmatterLines) > 0 {
		// Look for existing field in the raw lines
		fieldUpdated := false
		frontmatterLines := make([]string, len(result.FrontmatterLines))
		copy(frontmatterLines, result.FrontmatterLines)

		// Try to find and update the field in place
		for i, line := range frontmatterLines {
			trimmedLine := strings.TrimSpace(line)
			// Check if this line contains our field
			if strings.HasPrefix(trimmedLine, fieldName+":") {
				// Preserve the original indentation and comments
				leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

				// Check if there's a comment on the same line
				commentIndex := strings.Index(line, "#")
				var comment string
				if commentIndex > strings.Index(line, ":") && commentIndex != -1 {
					comment = line[commentIndex:]
				}

				// Update the field value while preserving formatting
				if comment != "" {
					frontmatterLines[i] = fmt.Sprintf("%s%s: %s %s", leadingSpace, fieldName, fieldValue, comment)
				} else {
					frontmatterLines[i] = fmt.Sprintf("%s%s: %s", leadingSpace, fieldName, fieldValue)
				}
				fieldUpdated = true
				break
			}
		}

		// If field wasn't found in the raw lines, add it at the end
		if !fieldUpdated {
			newField := fmt.Sprintf("%s: %s", fieldName, fieldValue)
			frontmatterLines = append(frontmatterLines, newField)
		}

		// Reconstruct the file with preserved formatting
		var lines []string
		lines = append(lines, "---")
		lines = append(lines, frontmatterLines...)
		lines = append(lines, "---")
		if result.Markdown != "" {
			lines = append(lines, result.Markdown)
		}

		return strings.Join(lines, "\n"), nil
	}

	// Fallback to marshal-based approach if no raw lines are available
	return updateFieldInFrontmatterFallback(result, fieldName, fieldValue)
}

// updateFieldInFrontmatterFallback implements the original behavior as a fallback
func updateFieldInFrontmatterFallback(result *parser.FrontmatterResult, fieldName, fieldValue string) (string, error) {
	// Initialize frontmatter if it doesn't exist
	if result.Frontmatter == nil {
		result.Frontmatter = make(map[string]any)
	}

	// Update the field
	result.Frontmatter[fieldName] = fieldValue

	// Convert back to YAML with proper field ordering
	updatedFrontmatter, err := workflow.MarshalWithFieldOrder(result.Frontmatter, constants.PriorityWorkflowFields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}

	// Clean up quoted keys - replace "on": with on: at the start of a line
	frontmatterStr := strings.TrimSuffix(string(updatedFrontmatter), "\n")
	frontmatterStr = workflow.UnquoteYAMLKey(frontmatterStr, "on")

	// Reconstruct the file
	var lines []string
	lines = append(lines, "---")
	if frontmatterStr != "" {
		lines = append(lines, strings.Split(frontmatterStr, "\n")...)
	}
	lines = append(lines, "---")
	if result.Markdown != "" {
		lines = append(lines, result.Markdown)
	}

	return strings.Join(lines, "\n"), nil
}

// addFieldToFrontmatter adds a new field to the frontmatter while preserving formatting.
// This is used when we know the field doesn't exist yet.
func addFieldToFrontmatter(content, fieldName, fieldValue string) (string, error) {
	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Try to preserve original frontmatter formatting by manually inserting the field
	if len(result.FrontmatterLines) > 0 {
		// Check if field already exists
		if result.Frontmatter != nil {
			if _, exists := result.Frontmatter[fieldName]; exists {
				// Field exists, update it instead
				return UpdateFieldInFrontmatter(content, fieldName, fieldValue)
			}
		}

		// Field doesn't exist, add it manually to preserve formatting
		frontmatterLines := make([]string, len(result.FrontmatterLines))
		copy(frontmatterLines, result.FrontmatterLines)

		// Add field at the end of the frontmatter, preserving original formatting
		newField := fmt.Sprintf("%s: %s", fieldName, fieldValue)
		frontmatterLines = append(frontmatterLines, newField)

		// Reconstruct the file with preserved formatting
		var lines []string
		lines = append(lines, "---")
		lines = append(lines, frontmatterLines...)
		lines = append(lines, "---")
		if result.Markdown != "" {
			lines = append(lines, result.Markdown)
		}

		return strings.Join(lines, "\n"), nil
	}

	// Fallback to original behavior if no frontmatter lines are available
	return updateFieldInFrontmatterFallback(result, fieldName, fieldValue)
}
