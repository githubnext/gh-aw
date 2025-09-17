package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

// UpdateWorkflowFrontmatter updates the frontmatter of a workflow file using a callback function
func UpdateWorkflowFrontmatter(workflowPath string, updateFunc func(frontmatter map[string]any) error, verbose bool) error {
	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse frontmatter using existing helper
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Ensure frontmatter map exists
	if result.Frontmatter == nil {
		result.Frontmatter = make(map[string]any)
	}

	// Apply the update function
	if err := updateFunc(result.Frontmatter); err != nil {
		return err
	}

	// Convert back to YAML
	updatedFrontmatter, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}

	// Reconstruct the file content
	updatedContent, err := reconstructWorkflowFile(string(updatedFrontmatter), result.Markdown)
	if err != nil {
		return fmt.Errorf("failed to reconstruct workflow file: %w", err)
	}

	// Write the updated content back to the file
	if err := os.WriteFile(workflowPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow file: %w", err)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Updated workflow file: %s", console.ToRelativePath(workflowPath))))
	}

	return nil
}

// ensureToolsSection ensures the tools section exists in frontmatter and returns it
func ensureToolsSection(frontmatter map[string]any) map[string]any {
	if frontmatter["tools"] == nil {
		frontmatter["tools"] = make(map[string]any)
	}

	tools, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		// If tools exists but is not a map, replace it
		tools = make(map[string]any)
		frontmatter["tools"] = tools
	}

	return tools
}

// reconstructWorkflowFile reconstructs a complete workflow file from frontmatter YAML and markdown content
func reconstructWorkflowFile(frontmatterYAML, markdownContent string) (string, error) {
	var lines []string

	// Add opening frontmatter delimiter
	lines = append(lines, "---")

	// Add frontmatter content (trim trailing newline from YAML marshal)
	frontmatterStr := strings.TrimSuffix(frontmatterYAML, "\n")
	if frontmatterStr != "" {
		lines = append(lines, strings.Split(frontmatterStr, "\n")...)
	}

	// Add closing frontmatter delimiter
	lines = append(lines, "---")

	// Add markdown content if present
	if markdownContent != "" {
		lines = append(lines, markdownContent)
	}

	return strings.Join(lines, "\n"), nil
}
