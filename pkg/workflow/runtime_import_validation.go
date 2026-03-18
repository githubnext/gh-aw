// This file provides validation for runtime-import file paths and their expressions.
//
// # Runtime Import Validation
//
// This file validates runtime-import file references in workflow markdown:
// - Extracts file paths from {{#runtime-import}} macros
// - Validates that imported files contain only allowed expressions
//
// # Validation Functions
//
//   - extractRuntimeImportPaths() - Extracts file paths from {{#runtime-import}} macros
//   - validateRuntimeImportFiles() - Validates expressions in all runtime-import files
//
// For expression security and allowlist validation, see expression_safety_validation.go.
// For expression syntax validation, see expression_syntax_validation.go.

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// runtimeImportMacroRe matches {{#runtime-import filepath}} or {{#runtime-import? filepath}}.
var runtimeImportMacroRe = regexp.MustCompile(`\{\{#runtime-import\??[ \t]+([^\}]+)\}\}`)

// lineRangeRe matches a line range suffix of the form "digits-digits" (e.g., "10-20").
var lineRangeRe = regexp.MustCompile(`^\d+-\d+$`)

// extractRuntimeImportPaths extracts all runtime-import file paths from markdown content.
// Returns a list of file paths (not URLs) referenced in {{#runtime-import}} macros.
// URLs (http:// or https://) are excluded since they are validated separately.
func extractRuntimeImportPaths(markdownContent string) []string {
	if markdownContent == "" {
		return nil
	}

	var paths []string
	seen := make(map[string]bool)

	matches := runtimeImportMacroRe.FindAllStringSubmatch(markdownContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			pathWithRange := strings.TrimSpace(match[1])

			// Skip macros with empty or whitespace-only targets
			if pathWithRange == "" {
				expressionValidationLog.Print("Skipping runtime-import macro with empty target")
				continue
			}

			// Remove line range if present (e.g., "file.md:10-20" -> "file.md")
			importPath := pathWithRange
			if colonIdx := strings.Index(pathWithRange, ":"); colonIdx > 0 {
				// Check if what follows colon looks like a line range (digits-digits)
				afterColon := pathWithRange[colonIdx+1:]
				if lineRangeRe.MatchString(afterColon) {
					importPath = pathWithRange[:colonIdx]
				}
			}

			// Skip URLs - they don't need file validation
			if strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://") {
				continue
			}

			// Add to list if not already seen
			if !seen[importPath] {
				paths = append(paths, importPath)
				seen[importPath] = true
			}
		}
	}

	return paths
}

// validateRuntimeImportFiles validates expressions in all runtime-import files at compile time.
// This catches expression errors early, before the workflow runs.
// workspaceDir should be the root of the repository (containing .github folder).
func validateRuntimeImportFiles(markdownContent string, workspaceDir string) error {
	expressionValidationLog.Print("Validating runtime-import files")

	// Extract all runtime-import file paths
	paths := extractRuntimeImportPaths(markdownContent)
	if len(paths) == 0 {
		expressionValidationLog.Print("No runtime-import files to validate")
		return nil
	}

	expressionValidationLog.Printf("Found %d runtime-import file(s) to validate", len(paths))

	var validationErrors []string

	for _, filePath := range paths {
		// Normalize the path to be relative to .github folder
		normalizedPath := filePath
		if strings.HasPrefix(normalizedPath, ".github/") {
			normalizedPath = normalizedPath[8:] // Remove ".github/"
		} else if strings.HasPrefix(normalizedPath, ".github\\") {
			normalizedPath = normalizedPath[8:] // Remove ".github\" (Windows)
		}
		if strings.HasPrefix(normalizedPath, "./") {
			normalizedPath = normalizedPath[2:] // Remove "./"
		} else if strings.HasPrefix(normalizedPath, ".\\") {
			normalizedPath = normalizedPath[2:] // Remove ".\" (Windows)
		}

		// Build absolute path to the file
		githubFolder := filepath.Join(workspaceDir, ".github")
		absolutePath := filepath.Join(githubFolder, normalizedPath)

		// Security check: ensure the resolved path is within the .github folder
		// Use filepath.Rel to check if the path escapes the .github folder
		normalizedGithubFolder := filepath.Clean(githubFolder)
		normalizedAbsolutePath := filepath.Clean(absolutePath)
		relativePath, err := filepath.Rel(normalizedGithubFolder, normalizedAbsolutePath)
		if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: Security: Path must be within .github folder (resolves to: %s)", filePath, relativePath))
			continue
		}

		// Check if file exists; missing files (optional or not) are deferred to runtime
		if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
			expressionValidationLog.Printf("Skipping validation for non-existent file: %s", filePath)
			continue
		}

		// Read the file content
		content, err := os.ReadFile(absolutePath)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: failed to read file: %v", filePath, err))
			continue
		}

		// Validate expressions in the imported file
		if err := validateExpressionSafety(string(content)); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", filePath, err))
		} else {
			expressionValidationLog.Printf("✓ Validated expressions in %s", filePath)
		}
	}

	if len(validationErrors) > 0 {
		expressionValidationLog.Printf("Runtime-import validation failed: %d file(s) with errors", len(validationErrors))
		return NewValidationError(
			"runtime-import",
			fmt.Sprintf("%d files with errors", len(validationErrors)),
			"runtime-import files contain expression errors:\n\n"+strings.Join(validationErrors, "\n\n"),
			"Fix the expression errors in the imported files listed above. Each file must only use allowed GitHub Actions expressions. See expression security documentation for details.",
		)
	}

	expressionValidationLog.Print("All runtime-import files validated successfully")
	return nil
}
