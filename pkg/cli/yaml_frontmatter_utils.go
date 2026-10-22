package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var yamlUtilsLog = logger.New("cli:yaml_frontmatter_utils")

// isFrontmatterStrictFalse returns true when the frontmatter explicitly sets strict: false.
// These codemods only need to run in strict mode; if the workflow has opted out of strict
// mode, the deprecated keys are still valid and should not be touched.
func isFrontmatterStrictFalse(frontmatter map[string]any) bool {
	strictVal, ok := frontmatter["strict"]
	if !ok {
		return false
	}
	strictBool, ok := strictVal.(bool)
	return ok && !strictBool
}

// parseFrontmatterLines extracts frontmatter lines from content
func parseFrontmatterLines(content string) ([]string, string, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}
	return result.FrontmatterLines, result.Markdown, nil
}

// getIndentation extracts the leading whitespace from a line
func getIndentation(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// isTopLevelKey checks if a line is a top-level YAML key (no indentation, contains colon, not a comment)
func isTopLevelKey(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	indent := getIndentation(line)
	return indent == "" && strings.Contains(line, ":")
}

// applyFrontmatterLineTransform parses frontmatter from content, applies a transform
// function to the frontmatter lines, and reconstructs the content if any changes were made.
// The transform function receives the frontmatter lines and returns the modified lines
// and a boolean indicating whether any changes were made.
func applyFrontmatterLineTransform(content string, transform func([]string) ([]string, bool)) (string, bool, error) {
	frontmatterLines, _, err := parseFrontmatterLines(content)
	if err != nil {
		return content, false, err
	}

	result, modified := transform(frontmatterLines)
	if !modified {
		return content, false, nil
	}

	yamlUtilsLog.Print("Frontmatter transformation applied successfully")
	originalFrontmatter := strings.Join(frontmatterLines, "\n")
	updatedFrontmatter := strings.Join(result, "\n")
	firstNewline := strings.IndexByte(content, '\n')
	if firstNewline < 0 {
		return content, false, errors.New("unable to locate frontmatter text in workflow content")
	}
	frontmatterStart := firstNewline + 1
	if !strings.HasPrefix(content[frontmatterStart:], originalFrontmatter) {
		return content, false, errors.New("unable to locate frontmatter text in workflow content")
	}
	return content[:frontmatterStart] + updatedFrontmatter + content[frontmatterStart+len(originalFrontmatter):], true, nil
}
