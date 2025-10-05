package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// processIncludesWithWorkflowSpec processes @include directives in content and replaces local file references
// with workflowspec format (owner/repo/path@sha) for all includes found in the package
func processIncludesWithWorkflowSpec(content string, workflow *WorkflowSpec, commitSHA, packagePath string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Processing @include directives to replace with workflowspec"))
	}

	// Track visited includes to prevent cycles
	visited := make(map[string]bool)

	// Use a queue to process files iteratively instead of recursion
	type fileToProcess struct {
		path string
	}
	queue := []fileToProcess{}

	// Process the main content first
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line is an @include or @import directive
		if matches := parser.IncludeDirectivePattern.FindStringSubmatch(line); matches != nil {
			isOptional := matches[1] == "?"
			includePath := strings.TrimSpace(matches[2])

			// Handle section references (file.md#Section)
			var filePath, sectionName string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
				sectionName = parts[1]
			} else {
				filePath = includePath
			}

			// Check for cycle detection
			if visited[filePath] {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Cycle detected for include: %s, skipping", filePath)))
				}
				continue
			}

			// Mark as visited
			visited[filePath] = true

			// Build workflowspec for this include
			// Format: owner/repo/path@sha
			workflowSpec := workflow.Repo + "/" + filePath
			if commitSHA != "" {
				workflowSpec += "@" + commitSHA
			} else if workflow.Version != "" {
				workflowSpec += "@" + workflow.Version
			}

			// Add section if present
			if sectionName != "" {
				workflowSpec += "#" + sectionName
			}

			// Write the updated @include directive
			if isOptional {
				result.WriteString("@include? " + workflowSpec + "\n")
			} else {
				result.WriteString("@include " + workflowSpec + "\n")
			}

			// Add file to queue for processing nested includes
			queue = append(queue, fileToProcess{path: filePath})
		} else {
			// Regular line, pass through
			result.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Process queue of files to check for nested includes
	for len(queue) > 0 {
		// Dequeue the first file
		fileItem := queue[0]
		queue = queue[1:]

		fullSourcePath := filepath.Join(packagePath, fileItem.path)
		if _, err := os.Stat(fullSourcePath); err != nil {
			continue // File doesn't exist, skip
		}

		includedContent, err := os.ReadFile(fullSourcePath)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not read include file %s: %v", fullSourcePath, err)))
			}
			continue
		}

		// Extract markdown content from the included file
		markdownContent, err := parser.ExtractMarkdownContent(string(includedContent))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not extract markdown from %s: %v", fullSourcePath, err)))
			}
			continue
		}

		// Scan for nested includes
		nestedScanner := bufio.NewScanner(strings.NewReader(markdownContent))
		for nestedScanner.Scan() {
			line := nestedScanner.Text()

			if matches := parser.IncludeDirectivePattern.FindStringSubmatch(line); matches != nil {
				includePath := strings.TrimSpace(matches[2])

				// Handle section references
				var nestedFilePath string
				if strings.Contains(includePath, "#") {
					parts := strings.SplitN(includePath, "#", 2)
					nestedFilePath = parts[0]
				} else {
					nestedFilePath = includePath
				}

				// Check for cycle detection
				if visited[nestedFilePath] {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Cycle detected for include: %s, skipping", nestedFilePath)))
					}
					continue
				}

				// Mark as visited and add to queue
				visited[nestedFilePath] = true
				queue = append(queue, fileToProcess{path: nestedFilePath})
			}
		}
	}

	return result.String(), nil
}

// processIncludesInContent processes @include directives in workflow content for update command
func processIncludesInContent(content string, workflow *WorkflowSpec, commitSHA string, verbose bool) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line is an @include or @import directive
		if matches := parser.IncludeDirectivePattern.FindStringSubmatch(line); matches != nil {
			isOptional := matches[1] == "?"
			includePath := strings.TrimSpace(matches[2])

			// Skip if it's already a workflowspec (contains repo/path format)
			if isWorkflowSpecFormat(includePath) {
				result.WriteString(line + "\n")
				continue
			}

			// Handle section references (file.md#Section)
			var filePath, sectionName string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
				sectionName = parts[1]
			} else {
				filePath = includePath
			}

			// Build workflowspec for this include
			// Format: owner/repo/path@sha
			workflowSpec := workflow.Repo + "/" + filePath
			if commitSHA != "" {
				workflowSpec += "@" + commitSHA
			} else if workflow.Version != "" {
				workflowSpec += "@" + workflow.Version
			}

			// Add section if present
			if sectionName != "" {
				workflowSpec += "#" + sectionName
			}

			// Write the updated @include directive
			if isOptional {
				result.WriteString("@include? " + workflowSpec + "\n")
			} else {
				result.WriteString("@include " + workflowSpec + "\n")
			}
		} else {
			// Regular line, pass through
			result.WriteString(line + "\n")
		}
	}

	return result.String(), scanner.Err()
}

// isWorkflowSpecFormat checks if a path already looks like a workflowspec
func isWorkflowSpecFormat(path string) bool {
	// Check if it contains @ (ref separator) or looks like owner/repo/path
	if strings.Contains(path, "@") {
		return true
	}

	// Remove section reference if present
	cleanPath := path
	if idx := strings.Index(path, "#"); idx != -1 {
		cleanPath = path[:idx]
	}

	// Check if it has at least 3 parts and doesn't start with . or /
	parts := strings.Split(cleanPath, "/")
	if len(parts) >= 3 && !strings.HasPrefix(cleanPath, ".") && !strings.HasPrefix(cleanPath, "/") {
		return true
	}

	return false
}
