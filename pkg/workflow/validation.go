package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
)

// validateExpressionSizes validates that no expression values in the generated YAML exceed GitHub Actions limits
func (c *Compiler) validateExpressionSizes(yamlContent string) error {
	lines := strings.Split(yamlContent, "\n")
	maxSize := MaxExpressionSize

	for lineNum, line := range lines {
		// Check the line length (actual content that will be in the YAML)
		if len(line) > maxSize {
			// Extract the key/value for better error message
			trimmed := strings.TrimSpace(line)
			key := ""
			if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 {
				key = strings.TrimSpace(trimmed[:colonIdx])
			}

			// Format sizes for display
			actualSize := pretty.FormatFileSize(int64(len(line)))
			maxSizeFormatted := pretty.FormatFileSize(int64(maxSize))

			var errorMsg string
			if key != "" {
				errorMsg = fmt.Sprintf("expression value for '%s' (%s) exceeds maximum allowed size (%s) at line %d. GitHub Actions has a 21KB limit for expression values including environment variables. Consider chunking the content or using artifacts instead.",
					key, actualSize, maxSizeFormatted, lineNum+1)
			} else {
				errorMsg = fmt.Sprintf("line %d (%s) exceeds maximum allowed expression size (%s). GitHub Actions has a 21KB limit for expression values.",
					lineNum+1, actualSize, maxSizeFormatted)
			}

			return errors.New(errorMsg)
		}
	}

	return nil
}
