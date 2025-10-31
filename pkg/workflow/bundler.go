package workflow

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// BundleJavaScriptFromSources bundles JavaScript from in-memory sources
// sources is a map where keys are file paths (e.g., "lib/sanitize.cjs") and values are the content
// mainContent is the main JavaScript content that may contain require() calls
// basePath is the base directory path for resolving relative imports (e.g., "js")
func BundleJavaScriptFromSources(mainContent string, sources map[string]string, basePath string) (string, error) {
	// Track already processed files to avoid circular dependencies
	processed := make(map[string]bool)

	// Bundle the main content recursively
	bundled, err := bundleFromSources(mainContent, basePath, sources, processed)
	if err != nil {
		return "", err
	}

	return bundled, nil
}

// bundleFromSources processes content and recursively bundles its dependencies from the sources map
func bundleFromSources(content string, currentPath string, sources map[string]string, processed map[string]bool) (string, error) {
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

			// Resolve the full path relative to current path
			var fullPath string
			if currentPath == "" {
				fullPath = requirePath
			} else {
				fullPath = filepath.Join(currentPath, requirePath)
			}

			// Ensure .cjs extension
			if !strings.HasSuffix(fullPath, ".cjs") && !strings.HasSuffix(fullPath, ".js") {
				fullPath += ".cjs"
			}

			// Normalize the path (clean up ./ and ../)
			fullPath = filepath.Clean(fullPath)

			// Convert Windows path separators to forward slashes for consistency
			fullPath = filepath.ToSlash(fullPath)

			// Check if we've already processed this file
			if processed[fullPath] {
				// Skip - already inlined
				result.WriteString("// Already inlined: " + requirePath + "\n")
				continue
			}

			// Mark as processed
			processed[fullPath] = true

			// Look up the required file in sources
			requiredContent, ok := sources[fullPath]
			if !ok {
				return "", fmt.Errorf("required file not found in sources: %s", fullPath)
			}

			// Recursively bundle the required file
			requiredDir := filepath.Dir(fullPath)
			bundledRequired, err := bundleFromSources(requiredContent, requiredDir, sources, processed)
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
