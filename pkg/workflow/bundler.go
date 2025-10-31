package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BundleJavaScript takes a JavaScript file path and bundles all local requires into a single file
// It detects require() calls to local files, inlines the content, removes export statements,
// and removes the original require calls
func BundleJavaScript(filePath string) (string, error) {
	// Track already processed files to avoid circular dependencies
	processed := make(map[string]bool)

	// Read the main file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Get the directory of the main file for resolving relative paths
	baseDir := filepath.Dir(filePath)

	// Bundle the file recursively
	bundled, err := bundleFile(string(content), baseDir, processed)
	if err != nil {
		return "", err
	}

	return bundled, nil
}

// bundleFile processes a single file and recursively bundles its dependencies
func bundleFile(content string, baseDir string, processed map[string]bool) (string, error) {
	// Regular expression to match require('./...') or require("./...")
	// Captures: require('path') or require("path") where path starts with ./ or ../
	requireRegex := regexp.MustCompile(`(?m)^.*?(?:const|let|var)\s+(?:\{[^}]*\}|\w+)\s*=\s*require\(['"](\.\.?/[^'"]+)['"]\);?\s*$`)

	var result strings.Builder
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check if this line is a local require
		matches := requireRegex.FindStringSubmatch(line)

		if len(matches) > 1 {
			// This is a local require - inline it
			requirePath := matches[1]

			// Resolve the full path
			fullPath := filepath.Join(baseDir, requirePath)

			// Ensure .cjs extension
			if !strings.HasSuffix(fullPath, ".cjs") && !strings.HasSuffix(fullPath, ".js") {
				fullPath += ".cjs"
			}

			// Normalize the path
			fullPath = filepath.Clean(fullPath)

			// Check if we've already processed this file
			if processed[fullPath] {
				// Skip - already inlined
				result.WriteString("// Already inlined: " + requirePath + "\n")
				continue
			}

			// Mark as processed
			processed[fullPath] = true

			// Read the required file
			requiredContent, err := os.ReadFile(fullPath)
			if err != nil {
				return "", fmt.Errorf("failed to read required file %s: %w", fullPath, err)
			}

			// Recursively bundle the required file
			requiredDir := filepath.Dir(fullPath)
			bundledRequired, err := bundleFile(string(requiredContent), requiredDir, processed)
			if err != nil {
				return "", err
			}

			// Remove exports from the bundled content
			cleanedRequired := removeExports(bundledRequired)

			// Add a comment indicating the inlined file
			result.WriteString(fmt.Sprintf("// === Inlined from %s ===\n", requirePath))
			result.WriteString(cleanedRequired)
			result.WriteString(fmt.Sprintf("// === End of %s ===\n", requirePath))

		} else {
			// Not a local require - keep the line as is
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		}
	}

	return result.String(), nil
}

// removeExports removes module.exports and exports statements from JavaScript code
func removeExports(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder

	// Regular expressions for export patterns
	moduleExportsRegex := regexp.MustCompile(`^\s*module\.exports\s*=`)
	exportsRegex := regexp.MustCompile(`^\s*exports\.\w+\s*=`)

	for i, line := range lines {
		// Skip lines that are module.exports or exports.* assignments
		if moduleExportsRegex.MatchString(line) || exportsRegex.MatchString(line) {
			// Skip this line - it's an export
			continue
		}

		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
