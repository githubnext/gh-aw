package workflow

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var bundlerLog = logger.New("workflow:bundler")

// BundleJavaScriptFromSources bundles JavaScript from in-memory sources
// sources is a map where keys are file paths (e.g., "sanitize.cjs") and values are the content
// mainContent is the main JavaScript content that may contain require() calls
// basePath is the base directory path for resolving relative imports (e.g., "js")
func BundleJavaScriptFromSources(mainContent string, sources map[string]string, basePath string) (string, error) {
	bundlerLog.Printf("Bundling JavaScript: source_count=%d, base_path=%s, main_content_size=%d bytes", len(sources), basePath, len(mainContent))

	// Track already processed files to avoid circular dependencies
	processed := make(map[string]bool)

	// Bundle the main content recursively
	bundled, err := bundleFromSources(mainContent, basePath, sources, processed)
	if err != nil {
		bundlerLog.Printf("Bundling failed: %v", err)
		return "", err
	}

	// Deduplicate require statements (keep only the first occurrence)
	bundled = deduplicateRequires(bundled)

	// Validate that all local requires have been inlined
	if err := validateNoLocalRequires(bundled); err != nil {
		bundlerLog.Printf("Validation failed: %v", err)
		return "", err
	}

	// Log size information about the bundled output
	lines := strings.Split(bundled, "\n")
	var maxLineLength int
	for _, line := range lines {
		if len(line) > maxLineLength {
			maxLineLength = len(line)
		}
	}

	bundlerLog.Printf("Bundling completed: processed_files=%d, output_size=%d bytes, output_lines=%d, max_line_length=%d chars",
		len(processed), len(bundled), len(lines), maxLineLength)
	return bundled, nil
}

// bundleFromSources processes content and recursively bundles its dependencies from the sources map
func bundleFromSources(content string, currentPath string, sources map[string]string, processed map[string]bool) (string, error) {
	bundlerLog.Printf("Processing file for bundling: current_path=%s, content_size=%d bytes", currentPath, len(content))

	// Regular expression to match require('./...') or require("./...")
	// This matches both single-line and multi-line destructuring:
	// const { x } = require("./file.cjs");
	// const {
	//   x,
	//   y
	// } = require("./file.cjs");
	// Captures the require path where it starts with ./ or ../
	requireRegex := regexp.MustCompile(`(?s)(?:const|let|var)\s+(?:\{[^}]*\}|\w+)\s*=\s*require\(['"](\.\.?/[^'"]+)['"]\);?`)

	// Find all requires and their positions
	matches := requireRegex.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		bundlerLog.Print("No requires found in content")
		// No requires found, return content as-is
		return content, nil
	}

	bundlerLog.Printf("Found %d require statements to process", len(matches))

	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// match[0], match[1] are the start and end of the full match
		// match[2], match[3] are the start and end of the captured group (the path)
		matchStart := match[0]
		matchEnd := match[1]
		pathStart := match[2]
		pathEnd := match[3]

		// Write content before this require
		result.WriteString(content[lastEnd:matchStart])

		// Extract the require path
		requirePath := content[pathStart:pathEnd]

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
			bundlerLog.Printf("Skipping already processed file: %s", fullPath)
			// Skip - already inlined
			result.WriteString("// Already inlined: " + requirePath + "\n")
		} else {
			// Mark as processed
			processed[fullPath] = true

			// Look up the required file in sources
			requiredContent, ok := sources[fullPath]
			if !ok {
				bundlerLog.Printf("Required file not found in sources: %s", fullPath)
				return "", fmt.Errorf("required file not found in sources: %s", fullPath)
			}

			bundlerLog.Printf("Inlining file: %s (size: %d bytes)", fullPath, len(requiredContent))

			// Recursively bundle the required file
			requiredDir := filepath.Dir(fullPath)
			bundledRequired, err := bundleFromSources(requiredContent, requiredDir, sources, processed)
			if err != nil {
				return "", err
			}

			// Remove exports from the bundled content
			cleanedRequired := removeExports(bundledRequired)
			bundlerLog.Printf("Processed %s: original_size=%d, after_export_removal=%d", fullPath, len(bundledRequired), len(cleanedRequired))

			// Add a comment indicating the inlined file
			result.WriteString(fmt.Sprintf("// === Inlined from %s ===\n", requirePath))
			result.WriteString(cleanedRequired)
			result.WriteString(fmt.Sprintf("// === End of %s ===\n", requirePath))
		}

		lastEnd = matchEnd
	}

	// Write any remaining content after the last require
	result.WriteString(content[lastEnd:])

	return result.String(), nil
}

// removeExports removes module.exports and exports statements from JavaScript code
// but preserves conditional exports (wrapped in if statements) as they may be needed for testing
func removeExports(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder

	// Regular expressions for export patterns
	moduleExportsRegex := regexp.MustCompile(`^\s*module\.exports\s*=`)
	exportsRegex := regexp.MustCompile(`^\s*exports\.\w+\s*=`)

	// Track if we're inside a conditional export block
	inConditionalExport := false
	conditionalDepth := 0

	// Track if we're inside an unconditional module.exports block
	inModuleExports := false
	moduleExportsDepth := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this starts a conditional export block
		// Pattern: if (typeof module !== "undefined" && module.exports) {
		if strings.Contains(trimmed, "if") &&
			strings.Contains(trimmed, "module") &&
			strings.Contains(trimmed, "exports") &&
			strings.Contains(trimmed, "{") {
			inConditionalExport = true
			conditionalDepth = 1
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		// Track braces if we're in a conditional export
		if inConditionalExport {
			for _, ch := range trimmed {
				if ch == '{' {
					conditionalDepth++
				} else if ch == '}' {
					conditionalDepth--
					if conditionalDepth == 0 {
						inConditionalExport = false
					}
				}
			}
			// Keep all lines inside conditional export blocks
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		// Check if this line starts an unconditional module.exports assignment
		if moduleExportsRegex.MatchString(line) {
			// Check if it's a multi-line object export (ends with {)
			if strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
				// This is a multi-line module.exports = { ... }
				inModuleExports = true
				moduleExportsDepth = 1
				// Skip this line and start tracking the export block
				continue
			} else {
				// Single-line export, skip just this line
				continue
			}
		}

		// Track braces if we're in an unconditional module.exports block
		if inModuleExports {
			// Count braces to track when the export block ends
			for _, ch := range trimmed {
				if ch == '{' {
					moduleExportsDepth++
				} else if ch == '}' {
					moduleExportsDepth--
					if moduleExportsDepth == 0 {
						inModuleExports = false
						// Skip this closing line and continue
						continue
					}
				}
			}
			// Skip all lines inside the export block
			continue
		}

		// Skip lines that are unconditional exports.* assignments
		if exportsRegex.MatchString(line) {
			// Skip this line - it's an unconditional export
			continue
		}

		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// deduplicateRequires removes duplicate require() statements from bundled JavaScript
// For destructured imports from the same module, it merges them into a single require statement
// keeping only the first occurrence of each unique require for non-destructured imports
func deduplicateRequires(content string) string {
	lines := strings.Split(content, "\n")
	
	// Track module imports: map[moduleName] -> list of destructured names or "" for default import
	moduleImports := make(map[string][]string)
	// Track which lines are require statements to skip during first pass
	requireLines := make(map[int]bool)
	// Track order of first appearance of each module
	moduleOrder := []string{}

	// Regular expression to match destructured require statements
	// Matches: const/let/var { name1, name2 } = require('module');
	destructuredRequireRegex := regexp.MustCompile(`^\s*(?:const|let|var)\s+\{\s*([^}]+)\s*\}\s*=\s*require\(['"']([^'"']+)['"']\);?\s*$`)
	// Regular expression to match non-destructured require statements
	// Matches: const/let/var name = require('module');
	simpleRequireRegex := regexp.MustCompile(`^\s*(?:const|let|var)\s+(\w+)\s*=\s*require\(['"']([^'"']+)['"']\);?\s*$`)

	// First pass: collect all require statements and their destructured imports
	for i, line := range lines {
		// Try destructured require first
		destructuredMatches := destructuredRequireRegex.FindStringSubmatch(line)
		if len(destructuredMatches) > 2 {
			moduleName := destructuredMatches[2]
			destructuredNames := destructuredMatches[1]
			
			requireLines[i] = true
			
			// Parse the destructured names (split by comma and trim whitespace)
			names := strings.Split(destructuredNames, ",")
			for _, name := range names {
				name = strings.TrimSpace(name)
				if name != "" {
					moduleImports[moduleName] = append(moduleImports[moduleName], name)
				}
			}
			
			// Track order of first appearance
			if len(moduleImports[moduleName]) == len(names) {
				moduleOrder = append(moduleOrder, moduleName)
			}
			continue
		}
		
		// Try simple require
		simpleMatches := simpleRequireRegex.FindStringSubmatch(line)
		if len(simpleMatches) > 2 {
			moduleName := simpleMatches[2]
			varName := simpleMatches[1]
			
			requireLines[i] = true
			
			// For simple requires, store the variable name with a marker
			if _, exists := moduleImports[moduleName]; !exists {
				moduleOrder = append(moduleOrder, moduleName)
			}
			moduleImports[moduleName] = append(moduleImports[moduleName], "VAR:"+varName)
		}
	}

	// Second pass: write output
	var result strings.Builder
	wroteRequires := false
	
	for i, line := range lines {
		// Skip original require lines, we'll write merged ones at the first require position
		if requireLines[i] {
			if !wroteRequires {
				// Write all merged require statements here
				for _, moduleName := range moduleOrder {
					imports := moduleImports[moduleName]
					if len(imports) == 0 {
						continue
					}
					
					// Check if this is a simple require (has VAR: prefix) or destructured
					if len(imports) == 1 && strings.HasPrefix(imports[0], "VAR:") {
						// Simple require
						varName := strings.TrimPrefix(imports[0], "VAR:")
						result.WriteString(fmt.Sprintf("const %s = require(\"%s\");\n", varName, moduleName))
						bundlerLog.Printf("Keeping simple require: %s", moduleName)
					} else {
						// Destructured require - merge all names
						var cleanedImports []string
						for _, imp := range imports {
							if !strings.HasPrefix(imp, "VAR:") {
								cleanedImports = append(cleanedImports, imp)
							}
						}
						
						if len(cleanedImports) > 0 {
							// Remove duplicates while preserving order
							seen := make(map[string]bool)
							var uniqueImports []string
							for _, imp := range cleanedImports {
								if !seen[imp] {
									seen[imp] = true
									uniqueImports = append(uniqueImports, imp)
								}
							}
							
							result.WriteString(fmt.Sprintf("const { %s } = require(\"%s\");\n", 
								strings.Join(uniqueImports, ", "), moduleName))
							bundlerLog.Printf("Merged destructured require for %s: %v", moduleName, uniqueImports)
						}
					}
				}
				wroteRequires = true
			}
			continue
		}

		// Keep non-require lines
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
