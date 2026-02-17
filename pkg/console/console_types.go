package console

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrorPosition represents a position in a source file
type ErrorPosition struct {
	File   string
	Line   int
	Column int
}

// CompilerError represents a structured compiler error with position information
type CompilerError struct {
	Position ErrorPosition
	Type     string // "error", "warning", "info"
	Message  string
	Context  []string // Source code lines for context
	Hint     string   // Optional hint for fixing the error
}

// TableConfig represents configuration for table rendering
type TableConfig struct {
	Headers   []string
	Rows      [][]string
	Title     string
	ShowTotal bool
	TotalRow  []string
}

// TreeNode represents a node in a hierarchical tree structure
type TreeNode struct {
	Value    string
	Children []TreeNode
}

// SelectOption represents a selectable option with a label and value
type SelectOption struct {
	Label string
	Value string
}

// FormField represents a generic form field configuration
type FormField struct {
	Type        string // "input", "password", "confirm", "select"
	Title       string
	Description string
	Placeholder string
	Value       any                // Pointer to the value to store the result
	Options     []SelectOption     // For select fields
	Validate    func(string) error // For input/password fields
}

// ListItem represents an item in an interactive list
type ListItem struct {
	title       string
	description string
	value       string
}

// NewListItem creates a new list item with title, description, and value
func NewListItem(title, description, value string) ListItem {
	return ListItem{
		title:       title,
		description: description,
		value:       value,
	}
}

// Title returns the item's title
func (i ListItem) Title() string { return i.title }

// Description returns the item's description
func (i ListItem) Description() string { return i.description }

// FilterValue returns the value used for filtering
func (i ListItem) FilterValue() string { return i.title }

// ToRelativePath converts an absolute path to a relative path from the current working directory
// If the relative path contains "..", returns the absolute path instead for clarity
func ToRelativePath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}

	wd, err := os.Getwd()
	if err != nil {
		return path
	}

	relPath, err := filepath.Rel(wd, path)
	if err != nil {
		return path
	}

	if strings.Contains(relPath, "..") {
		return path
	}

	return relPath
}

// RenderTableAsJSON renders a table configuration as JSON
func RenderTableAsJSON(config TableConfig) (string, error) {
	if len(config.Headers) == 0 {
		return "[]", nil
	}

	var result []map[string]string
	for _, row := range config.Rows {
		obj := make(map[string]string)
		for i, cell := range row {
			if i < len(config.Headers) {
				key := strings.ToLower(strings.ReplaceAll(config.Headers[i], " ", "_"))
				obj[key] = cell
			}
		}
		result = append(result, obj)
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal table to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// FormatErrorWithSuggestions formats an error message with actionable suggestions
func FormatErrorWithSuggestions(message string, suggestions []string) string {
	var output strings.Builder
	output.WriteString(FormatErrorMessage(message))

	if len(suggestions) > 0 {
		output.WriteString("\n\nSuggestions:\n")
		for _, suggestion := range suggestions {
			output.WriteString("  • " + suggestion + "\n")
		}
	}

	return output.String()
}

// renderTreeSimple renders a simple text-based tree without styling
func renderTreeSimple(node TreeNode, prefix string, isLast bool) string {
	var output strings.Builder

	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		output.WriteString(node.Value + "\n")
	} else {
		output.WriteString(prefix + connector + node.Value + "\n")
	}

	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		var childPrefix string
		if prefix == "" {
			childPrefix = ""
		} else {
			if isLast {
				childPrefix = prefix + "    "
			} else {
				childPrefix = prefix + "│   "
			}
		}
		output.WriteString(renderTreeSimple(child, childPrefix, childIsLast))
	}

	return output.String()
}

// findWordEnd finds the end of a word starting at the given position
func findWordEnd(line string, start int) int {
	if start >= len(line) {
		return len(line)
	}

	end := start
	for end < len(line) {
		char := line[end]
		if char == ' ' || char == '\t' || char == ':' || char == '\n' || char == '\r' {
			break
		}
		end++
	}

	return end
}
