// Package workflow provides JavaScript bundler validation for agentic workflows.
//
// # JavaScript Bundler Validation
//
// This file validates bundled JavaScript to ensure compatibility with the target runtime mode.
// Validation functions prevent runtime errors from missing modules or incompatible module references.
//
// # Runtime Mode Validation
//
// GitHub Script Mode:
//   - validateNoLocalRequires() - Ensures all local require() statements are inlined
//   - validateNoModuleReferences() - Ensures no module.exports or exports.* remain
//
// Node.js Mode:
//   - No strict validation - module.exports and local requires are allowed
//
// # Validation Functions
//
//   - validateNoLocalRequires() - Validates bundled JavaScript has no local require() statements
//   - validateNoModuleReferences() - Validates no module.exports or exports references remain
//   - isInsideStringLiteralAt() - Helper to detect if a position is inside a string literal
//
// # Validation Pattern: Bundling Verification
//
// Bundler validation ensures that local require() statements are inlined:
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
//   - It validates JavaScript code structure based on runtime mode
//
// For bundling functions, see bundler.go.
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var bundlerValidationLog = logger.New("workflow:bundler_validation")

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
	bundlerValidationLog.Printf("Validating bundled JavaScript: %d bytes, %d lines", len(bundledContent), strings.Count(bundledContent, "\n")+1)

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
		bundlerValidationLog.Printf("Validation failed: found %d un-inlined local require statements", len(foundRequires))
		return fmt.Errorf("bundled JavaScript contains %d local require(s) that were not inlined:\n  %s",
			len(foundRequires), strings.Join(foundRequires, "\n  "))
	}

	bundlerValidationLog.Print("Validation successful: no local require statements found")
	return nil
}

// validateNoModuleReferences checks that the bundled JavaScript contains no module.exports or exports references
// This is required for GitHub Script mode where no module system exists.
// Returns an error if any module references are found, otherwise returns nil
func validateNoModuleReferences(bundledContent string) error {
	bundlerValidationLog.Printf("Validating no module references: %d bytes", len(bundledContent))

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
		bundlerValidationLog.Printf("Validation failed: found %d module references", len(foundReferences))
		return fmt.Errorf("bundled JavaScript for GitHub Script mode contains %d module reference(s) that should have been removed:\n  %s\n\nGitHub Script mode does not support module.exports or exports; these references must be removed during bundling",
			len(foundReferences), strings.Join(foundReferences, "\n  "))
	}

	bundlerValidationLog.Print("Validation successful: no module references found")
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
	bundlerValidationLog.Printf("Validating embedded resources: checking %d files for missing local requires", len(sources))

	// Regular expression to match local require statements
	// Matches: require('./...') or require("../...")
	localRequireRegex := regexp.MustCompile(`require\(['"](\.\.?/[^'"]+)['"]\)`)

	var missingDeps []string

	// Check each file in sources
	for filePath, content := range sources {
		bundlerValidationLog.Printf("Checking file: %s (%d bytes)", filePath, len(content))

		// Find all local requires in this file
		matches := localRequireRegex.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			continue
		}

		bundlerValidationLog.Printf("Found %d require statements in %s", len(matches), filePath)

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
				bundlerValidationLog.Printf("Missing dependency: %s", missingDep)
			} else {
				bundlerValidationLog.Printf("Dependency OK: %s -> %s", filePath, resolvedPath)
			}
		}
	}

	if len(missingDeps) > 0 {
		bundlerValidationLog.Printf("Validation failed: found %d missing dependencies", len(missingDeps))
		return fmt.Errorf("embedded JavaScript files have %d missing local require(s):\n  %s\n\nThese files must be added to GetJavaScriptSources() in js.go",
			len(missingDeps), strings.Join(missingDeps, "\n  "))
	}

	bundlerValidationLog.Printf("Validation successful: all local requires are available in sources")
	return nil
}

// validateNoExecSync checks that GitHub Script mode scripts do not use execSync
// GitHub Script mode should use exec instead for better async/await handling
// Returns an error if execSync is found, otherwise returns nil
func validateNoExecSync(scriptName string, content string, mode RuntimeMode) error {
	// Only validate GitHub Script mode
	if mode != RuntimeModeGitHubScript {
		return nil
	}

	bundlerValidationLog.Printf("Validating no execSync in GitHub Script: %s (%d bytes)", scriptName, len(content))

	// Regular expression to match execSync usage
	// Matches: execSync(...) with various patterns
	execSyncRegex := regexp.MustCompile(`\bexecSync\s*\(`)

	lines := strings.Split(content, "\n")
	var foundUsages []string

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment lines
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Check for execSync usage
		if execSyncRegex.MatchString(line) {
			foundUsages = append(foundUsages, fmt.Sprintf("line %d: %s", lineNum+1, strings.TrimSpace(line)))
		}
	}

	if len(foundUsages) > 0 {
		bundlerValidationLog.Printf("Validation failed: found %d execSync usage(s) in %s", len(foundUsages), scriptName)
		return fmt.Errorf("GitHub Script mode script '%s' contains %d execSync usage(s):\n  %s\n\nGitHub Script mode should use exec instead of execSync for better async/await handling",
			scriptName, len(foundUsages), strings.Join(foundUsages, "\n  "))
	}

	bundlerValidationLog.Printf("Validation successful: no execSync usage found in %s", scriptName)
	return nil
}

// validateNoGitHubScriptGlobals checks that Node.js mode scripts do not use GitHub Actions globals
// Node.js scripts should not rely on actions/github-script globals like core.*, exec.*, or github.*
// Returns an error if GitHub Actions globals are found, otherwise returns nil
func validateNoGitHubScriptGlobals(scriptName string, content string, mode RuntimeMode) error {
	// Only validate Node.js mode
	if mode != RuntimeModeNodeJS {
		return nil
	}

	bundlerValidationLog.Printf("Validating no GitHub Actions globals in Node.js script: %s (%d bytes)", scriptName, len(content))

	// Regular expressions to match GitHub Actions globals
	// Matches: core.method, exec.method, github.property
	coreGlobalRegex := regexp.MustCompile(`\bcore\.\w+`)
	execGlobalRegex := regexp.MustCompile(`\bexec\.\w+`)
	githubGlobalRegex := regexp.MustCompile(`\bgithub\.\w+`)

	lines := strings.Split(content, "\n")
	var foundUsages []string

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment lines and type references
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		if strings.Contains(trimmed, "/// <reference") {
			continue
		}

		// Check for core.* usage
		if coreGlobalRegex.MatchString(line) {
			foundUsages = append(foundUsages, fmt.Sprintf("line %d: core.* usage: %s", lineNum+1, strings.TrimSpace(line)))
		}

		// Check for exec.* usage
		if execGlobalRegex.MatchString(line) {
			foundUsages = append(foundUsages, fmt.Sprintf("line %d: exec.* usage: %s", lineNum+1, strings.TrimSpace(line)))
		}

		// Check for github.* usage
		if githubGlobalRegex.MatchString(line) {
			foundUsages = append(foundUsages, fmt.Sprintf("line %d: github.* usage: %s", lineNum+1, strings.TrimSpace(line)))
		}
	}

	if len(foundUsages) > 0 {
		bundlerValidationLog.Printf("Validation failed: found %d GitHub Actions global usage(s) in %s", len(foundUsages), scriptName)
		return fmt.Errorf("node.js mode script '%s' contains %d GitHub Actions global usage(s):\n  %s\n\nNode.js scripts should not use GitHub Actions globals (core.*, exec.*, github.*)",
			scriptName, len(foundUsages), strings.Join(foundUsages, "\n  "))
	}

	bundlerValidationLog.Printf("Validation successful: no GitHub Actions globals found in %s", scriptName)
	return nil
}

// validateNoRuntimeMixing checks that all files being bundled are compatible with the target runtime mode
// This prevents mixing nodejs-only scripts (that use child_process) with github-script scripts
// Returns an error if incompatible runtime modes are detected
func validateNoRuntimeMixing(mainScript string, sources map[string]string, targetMode RuntimeMode) error {
	bundlerValidationLog.Printf("Validating runtime mode compatibility: target_mode=%s", targetMode)

	// Track which files have been checked to avoid redundant checks
	checked := make(map[string]bool)

	// Recursively validate the main script and its dependencies
	return validateRuntimeModeRecursive(mainScript, "", sources, targetMode, checked)
}

// validateRuntimeModeRecursive recursively validates that all required files are compatible with the target runtime mode
func validateRuntimeModeRecursive(content string, currentPath string, sources map[string]string, targetMode RuntimeMode, checked map[string]bool) error {
	// Extract all local require statements
	requireRegex := regexp.MustCompile(`require\(['"](\.\.?/[^'"]+)['"]\)`)
	matches := requireRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) <= 1 {
			continue
		}

		requirePath := match[1]

		// Resolve the full path
		var fullPath string
		if currentPath == "" {
			fullPath = requirePath
		} else {
			fullPath = currentPath + "/" + requirePath
		}

		// Ensure .cjs extension
		if !strings.HasSuffix(fullPath, ".cjs") && !strings.HasSuffix(fullPath, ".js") {
			fullPath += ".cjs"
		}

		// Normalize the path
		fullPath = normalizePath(fullPath)

		// Skip if already checked
		if checked[fullPath] {
			continue
		}
		checked[fullPath] = true

		// Get the required file content
		requiredContent, ok := sources[fullPath]
		if !ok {
			// File not found - this will be caught by other validation
			continue
		}

		// Detect the runtime mode of the required file
		detectedMode := detectRuntimeMode(requiredContent)

		// Check for incompatibility
		if detectedMode != RuntimeModeGitHubScript && targetMode != detectedMode {
			return fmt.Errorf("runtime mode conflict: script requires '%s' which is a %s script, but the main script is compiled for %s mode.\n\nNode.js scripts cannot be bundled with GitHub Script mode scripts because they use incompatible APIs (e.g., child_process, fs).\n\nTo fix this:\n- Use only GitHub Script compatible scripts (core.*, exec.*, github.*) for GitHub Script mode\n- Or change the main script to Node.js mode if it needs Node.js APIs",
				fullPath, detectedMode, targetMode)
		}

		// Recursively check the required file's dependencies
		requiredDir := ""
		if strings.Contains(fullPath, "/") {
			parts := strings.Split(fullPath, "/")
			requiredDir = strings.Join(parts[:len(parts)-1], "/")
		}

		if err := validateRuntimeModeRecursive(requiredContent, requiredDir, sources, targetMode, checked); err != nil {
			return err
		}
	}

	return nil
}

// detectRuntimeMode attempts to detect the intended runtime mode of a JavaScript file
// by analyzing its content for runtime-specific patterns.
// This is used to detect if a LOCAL file being bundled is incompatible with the target mode.
func detectRuntimeMode(content string) RuntimeMode {
	// Check for Node.js-specific APIs that are CALLED in the code
	// These indicate the script uses Node.js-only functionality
	// Note: We only check for APIs that are fundamentally incompatible with github-script,
	// specifically child_process APIs like execSync/spawnSync
	nodeOnlyPatterns := []string{
		`\bexecSync\s*\(`,  // execSync function call
		`\bspawnSync\s*\(`, // spawnSync function call
	}

	for _, pattern := range nodeOnlyPatterns {
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			bundlerValidationLog.Printf("Detected Node.js mode: pattern '%s' found", pattern)
			return RuntimeModeNodeJS
		}
	}

	// Check for github-script specific APIs
	// These indicate the script is intended for GitHub Script mode
	githubScriptPatterns := []string{
		`\bcore\.\w+`,   // @actions/core
		`\bgithub\.\w+`, // github context
	}

	for _, pattern := range githubScriptPatterns {
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			bundlerValidationLog.Printf("Detected GitHub Script mode: pattern '%s' found", pattern)
			return RuntimeModeGitHubScript
		}
	}

	// If no specific patterns found, assume it's compatible with both (utility/helper functions)
	// and return GitHub Script mode as the default/most restrictive
	bundlerValidationLog.Print("No runtime-specific patterns found, assuming GitHub Script compatible")
	return RuntimeModeGitHubScript
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
