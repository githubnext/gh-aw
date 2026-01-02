// Package workflow provides JavaScript bundler safety validation for agentic workflows.
//
// # Bundle Safety Validation
//
// This file validates bundled JavaScript to ensure safe module dependencies and prevent
// runtime errors from missing modules. Validation ensures compatibility with target runtime mode.
//
// # Validation Functions
//
//   - validateNoLocalRequires() - Validates bundled JavaScript has no local require() statements
//   - validateNoModuleReferences() - Validates no module.exports or exports references remain
//   - ValidateEmbeddedResourceRequires() - Validates embedded JavaScript dependencies exist
//
// # Validation Pattern: Bundling Verification
//
// Bundle safety validation ensures that local require() statements are inlined and
// module references are removed when required:
//   - Scans bundled JavaScript for require('./...') or require('../...') patterns
//   - Ignores require statements inside string literals
//   - Returns hard errors if local requires are found (indicates bundling failure)
//   - Helps prevent runtime module-not-found errors
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates JavaScript bundling correctness
//   - It checks for missing module dependencies
//   - It validates CommonJS require() statement resolution
//
// For bundling functions, see bundler.go.
// For runtime mode validation, see bundler_runtime_validation.go.
// For script content validation, see bundler_script_validation.go.
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var bundlerSafetyLog = logger.New("workflow:bundler_safety_validation")

// Pre-compiled regular expressions for validation (compiled once at package initialization for performance)
var (
	// moduleExportsRegex matches module.exports references
	moduleExportsRegex = regexp.MustCompile(`\bmodule\.exports\b`)
	// exportsRegex matches exports.property references
	exportsRegex = regexp.MustCompile(`\bexports\.\w+`)
)

// validateNoLocalRequires checks that the bundled JavaScript contains no local require() statements
// that weren't inlined during bundling. This prevents runtime errors from missing local modules.
// Returns an error if any local requires are found, otherwise returns nil
func validateNoLocalRequires(bundledContent string) error {
	bundlerSafetyLog.Printf("Validating bundled JavaScript: %d bytes, %d lines", len(bundledContent), strings.Count(bundledContent, "\n")+1)

	// Regular expression to match local require statements
	// Matches: require('./...') or require("../...")
	localRequireRegex := regexp.MustCompile(`require\(['"](\.\.?/[^'"]+)['"]\)`)

	lines := strings.Split(bundledContent, "\n")
	var foundRequires []string

	for lineNum, line := range lines {
		// Check for local requires
		matches := localRequireRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				requirePath := match[1]
				foundRequires = append(foundRequires, fmt.Sprintf("line %d: require('%s')", lineNum+1, requirePath))
			}
		}
	}

	if len(foundRequires) > 0 {
		bundlerSafetyLog.Printf("Validation failed: found %d un-inlined local require statements", len(foundRequires))
		return fmt.Errorf("bundled JavaScript contains %d local require(s) that were not inlined:\n  %s",
			len(foundRequires), strings.Join(foundRequires, "\n  "))
	}

	bundlerSafetyLog.Print("Validation successful: no local require statements found")
	return nil
}

// validateNoModuleReferences checks that the bundled JavaScript contains no module.exports or exports references
// This is required for GitHub Script mode where no module system exists.
// Returns an error if any module references are found, otherwise returns nil
func validateNoModuleReferences(bundledContent string) error {
	bundlerSafetyLog.Printf("Validating no module references: %d bytes", len(bundledContent))

	lines := strings.Split(bundledContent, "\n")
	var foundReferences []string

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment lines
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Check for module.exports
		if moduleExportsRegex.MatchString(line) {
			foundReferences = append(foundReferences, fmt.Sprintf("line %d: module.exports reference", lineNum+1))
		}

		// Check for exports.
		if exportsRegex.MatchString(line) {
			foundReferences = append(foundReferences, fmt.Sprintf("line %d: exports reference", lineNum+1))
		}
	}

	if len(foundReferences) > 0 {
		bundlerSafetyLog.Printf("Validation failed: found %d module references", len(foundReferences))
		return fmt.Errorf("bundled JavaScript for GitHub Script mode contains %d module reference(s) that should have been removed:\n  %s\n\nGitHub Script mode does not support module.exports or exports; these references must be removed during bundling",
			len(foundReferences), strings.Join(foundReferences, "\n  "))
	}

	bundlerSafetyLog.Print("Validation successful: no module references found")
	return nil
}

// ValidateEmbeddedResourceRequires checks that all embedded JavaScript files in the sources map
// have their local require() dependencies available in the sources map. This prevents bundling failures
// when a file requires a local module that isn't embedded.
//
// This validation helps catch missing files in GetJavaScriptSources() at build/test time rather than
// at runtime when bundling fails.
//
// Parameters:
//   - sources: map of file paths to their content (from GetJavaScriptSources())
//
// Returns an error if any embedded file has local requires that reference files not in sources
func ValidateEmbeddedResourceRequires(sources map[string]string) error {
	bundlerSafetyLog.Printf("Validating embedded resources: checking %d files for missing local requires", len(sources))

	// Regular expression to match local require statements
	// Matches: require('./...') or require("../...")
	localRequireRegex := regexp.MustCompile(`require\(['"](\.\.?/[^'"]+)['"]\)`)

	var missingDeps []string

	// Check each file in sources
	for filePath, content := range sources {
		bundlerSafetyLog.Printf("Checking file: %s (%d bytes)", filePath, len(content))

		// Find all local requires in this file
		matches := localRequireRegex.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			continue
		}

		bundlerSafetyLog.Printf("Found %d require statements in %s", len(matches), filePath)

		// Check each require
		for _, match := range matches {
			if len(match) <= 1 {
				continue
			}

			requirePath := match[1]

			// Resolve the required file path relative to the current file
			currentDir := ""
			if strings.Contains(filePath, "/") {
				parts := strings.Split(filePath, "/")
				currentDir = strings.Join(parts[:len(parts)-1], "/")
			}

			var resolvedPath string
			if currentDir == "" {
				resolvedPath = requirePath
			} else {
				resolvedPath = currentDir + "/" + requirePath
			}

			// Ensure .cjs extension
			if !strings.HasSuffix(resolvedPath, ".cjs") && !strings.HasSuffix(resolvedPath, ".js") {
				resolvedPath += ".cjs"
			}

			// Normalize the path (remove ./ and ../)
			resolvedPath = normalizePath(resolvedPath)

			// Check if the required file exists in sources
			if _, ok := sources[resolvedPath]; !ok {
				missingDep := fmt.Sprintf("%s requires '%s' (resolved to '%s') but it's not in sources map",
					filePath, requirePath, resolvedPath)
				missingDeps = append(missingDeps, missingDep)
				bundlerSafetyLog.Printf("Missing dependency: %s", missingDep)
			} else {
				bundlerSafetyLog.Printf("Dependency OK: %s -> %s", filePath, resolvedPath)
			}
		}
	}

	if len(missingDeps) > 0 {
		bundlerSafetyLog.Printf("Validation failed: found %d missing dependencies", len(missingDeps))
		return fmt.Errorf("embedded JavaScript files have %d missing local require(s):\n  %s\n\nThese files must be added to GetJavaScriptSources() in js.go",
			len(missingDeps), strings.Join(missingDeps, "\n  "))
	}

	bundlerSafetyLog.Printf("Validation successful: all local requires are available in sources")
	return nil
}

// normalizePath normalizes a file path by resolving . and .. components
func normalizePath(path string) string {
	// Split path into parts
	parts := strings.Split(path, "/")
	var result []string

	for _, part := range parts {
		if part == "" || part == "." {
			// Skip empty parts and current directory references
			continue
		}
		if part == ".." {
			// Go up one directory
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
		} else {
			result = append(result, part)
		}
	}

	return strings.Join(result, "/")
}
