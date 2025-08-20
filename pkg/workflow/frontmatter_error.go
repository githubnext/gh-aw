package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// createFrontmatterError creates a detailed error for frontmatter parsing issues
// frontmatterLineOffset is the line number where the frontmatter content begins (1-based)
func (c *Compiler) createFrontmatterError(filePath, content string, err error, frontmatterLineOffset int) error {
	lines := strings.Split(content, "\n")

	// Check if this is a YAML parsing error that we can enhance
	if strings.Contains(err.Error(), "failed to parse frontmatter:") {
		// Extract the inner YAML error
		parts := strings.SplitN(err.Error(), "failed to parse frontmatter: ", 2)
		if len(parts) > 1 {
			yamlErr := errors.New(parts[1])
			line, column, message := parser.ExtractYAMLError(yamlErr, frontmatterLineOffset)

			if line > 0 || column > 0 {
				// Create context lines around the error
				var context []string
				startLine := max(1, line-2)
				endLine := min(len(lines), line+2)

				for i := startLine; i <= endLine; i++ {
					if i-1 < len(lines) {
						context = append(context, lines[i-1])
					}
				}

				compilerErr := console.CompilerError{
					Position: console.ErrorPosition{
						File:   filePath,
						Line:   line,
						Column: column,
					},
					Type:    "error",
					Message: fmt.Sprintf("frontmatter parsing failed: %s", message),
					Context: context,
					Hint:    "check YAML syntax in frontmatter section",
				}

				// Format and return the error
				formattedErr := console.FormatError(compilerErr)
				return errors.New(formattedErr)
			}
		}
	} else {
		// Try to extract YAML error directly from the original error
		line, column, message := parser.ExtractYAMLError(err, frontmatterLineOffset)

		if line > 0 || column > 0 {
			// Create context lines around the error
			var context []string
			startLine := max(1, line-2)
			endLine := min(len(lines), line+2)

			for i := startLine; i <= endLine; i++ {
				if i-1 < len(lines) {
					context = append(context, lines[i-1])
				}
			}

			compilerErr := console.CompilerError{
				Position: console.ErrorPosition{
					File:   filePath,
					Line:   line,
					Column: column,
				},
				Type:    "error",
				Message: fmt.Sprintf("frontmatter parsing failed: %s", message),
				Context: context,
				Hint:    "check YAML syntax in frontmatter section",
			}

			// Format and return the error
			formattedErr := console.FormatError(compilerErr)
			return errors.New(formattedErr)
		}
	}

	// Fallback to original error
	return fmt.Errorf("failed to extract frontmatter: %w", err)
}
