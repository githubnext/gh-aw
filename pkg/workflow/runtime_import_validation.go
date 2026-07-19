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

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
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
	seen := make(map[string]struct {
	})

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
			if !setutil.Contains(seen, importPath) {
				paths = append(paths, importPath)
				seen[importPath] = struct {
				}{}
			}
		}
	}

	return paths
}

// validateRuntimeImportFiles validates expressions in all runtime-import files at compile time.
// This catches expression errors early, before the workflow runs.
// workspaceDir should be the root of the repository (containing .github folder).
// It returns any best-effort sub-agent frontmatter warnings and a non-nil error for fatal
// expression validation failures.
func validateRuntimeImportFiles(markdownContent string, workspaceDir string) ([]string, error) {
	expressionValidationLog.Print("Validating runtime-import files")

	// Extract all runtime-import file paths
	paths := extractRuntimeImportPaths(markdownContent)
	if len(paths) == 0 {
		expressionValidationLog.Print("No runtime-import files to validate")
		return nil, nil
	}

	expressionValidationLog.Printf("Found %d runtime-import file(s) to validate", len(paths))

	var validationErrors []string
	var subAgentWarnings []string

	for _, filePath := range paths {
		warnings, errText, ok := validateRuntimeImportFile(filePath, workspaceDir)
		subAgentWarnings = append(subAgentWarnings, warnings...)
		if errText != "" {
			validationErrors = append(validationErrors, errText)
		}
		if !ok {
			continue
		}
	}

	if len(validationErrors) > 0 {
		expressionValidationLog.Printf("Runtime-import validation failed: %d file(s) with errors", len(validationErrors))
		return subAgentWarnings, NewValidationError(
			"runtime-import",
			fmt.Sprintf("%d files with errors", len(validationErrors)),
			"runtime-import files contain expression errors:\n\n"+strings.Join(validationErrors, "\n\n"),
			"Fix the expression errors in the imported files listed above. Each file must only use allowed GitHub Actions expressions. See expression security documentation for details.",
		)
	}

	expressionValidationLog.Print("All runtime-import files validated successfully")
	return subAgentWarnings, nil
}

func validateRuntimeImportFile(filePath string, workspaceDir string) ([]string, string, bool) {
	absolutePath, relativePath, err := resolveRuntimeImportPath(filePath, workspaceDir)
	if err != nil {
		return nil, fmt.Sprintf("%s: Security: Path must be within .github folder (resolves to: %s)", filePath, relativePath), false
	}

	if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
		expressionValidationLog.Printf("Skipping validation for non-existent file: %s", filePath)
		return nil, "", false
	}

	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, fmt.Sprintf("%s: failed to read file: %v", filePath, err), false
	}

	if err := validateExpressionSafety(string(content)); err != nil {
		return nil, fmt.Sprintf("%s: %v", filePath, err), true
	}
	expressionValidationLog.Printf("✓ Validated expressions in %s", filePath)
	return runtimeImportSubAgentWarnings(filePath, string(content)), "", true
}

func resolveRuntimeImportPath(filePath string, workspaceDir string) (string, string, error) {
	normalizedPath := normalizeRuntimeImportPath(filePath)
	githubFolder := filepath.Join(workspaceDir, ".github")
	absolutePath := filepath.Join(githubFolder, normalizedPath)
	normalizedGithubFolder := filepath.Clean(githubFolder)
	normalizedAbsolutePath := filepath.Clean(absolutePath)
	relativePath, err := filepath.Rel(normalizedGithubFolder, normalizedAbsolutePath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return "", relativePath, fmt.Errorf("path escapes .github")
	}
	return absolutePath, relativePath, nil
}

func normalizeRuntimeImportPath(filePath string) string {
	normalizedPath := filePath
	if strings.HasPrefix(normalizedPath, constants.GithubDir) {
		normalizedPath = normalizedPath[len(constants.GithubDir):]
	} else if strings.HasPrefix(normalizedPath, ".github\\") {
		normalizedPath = normalizedPath[8:]
	}
	if strings.HasPrefix(normalizedPath, "./") || strings.HasPrefix(normalizedPath, ".\\") {
		normalizedPath = normalizedPath[2:]
	}
	return normalizedPath
}

func runtimeImportSubAgentWarnings(filePath string, content string) []string {
	var warnings []string
	for _, w := range parser.ValidateInlineSubAgentsFrontmatter(content) {
		warnings = append(warnings, fmt.Sprintf("runtime-import %q: %s", filePath, w))
	}
	for _, w := range parser.ValidateInlineSkillsFrontmatter(content) {
		warnings = append(warnings, fmt.Sprintf("runtime-import %q: %s", filePath, w))
	}
	return warnings
}
