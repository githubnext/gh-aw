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

// runtimeImportMacroRe matches {{#runtime-import filepath}}, {{#runtime-import? filepath}},
// and deprecated body-level {{#import}} forms normalized by the runtime.
var runtimeImportMacroRe = regexp.MustCompile(`\{\{#(?:runtime-import|import)\??(?:[ \t]+|[ \t]*:[ \t]*)([^{}]+?)\}\}`)

// lineRangeRe matches a line range suffix of the form "digits-digits" (e.g., "10-20").
var lineRangeRe = regexp.MustCompile(`^\d+-\d+$`)

// extractRuntimeImportPaths extracts all runtime-import file paths from markdown content.
// Returns a list of file paths (not URLs) referenced in {{#runtime-import}} macros.
// URLs (http:// or https://) are excluded since they are validated separately.
func extractRuntimeImportPaths(markdownContent string) []string {
	refs := extractRuntimeImportReferences(markdownContent)
	if len(refs) == 0 {
		return nil
	}
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.importPath)
	}
	return paths
}

type runtimeImportReference struct {
	importPath string
	startLine  int
	endLine    int
}

func extractRuntimeImportReferences(markdownContent string) []runtimeImportReference {
	if markdownContent == "" {
		return nil
	}

	var refs []runtimeImportReference
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
			startLine := 0
			endLine := 0
			if colonIdx := strings.Index(pathWithRange, ":"); colonIdx > 0 {
				// Check if what follows colon looks like a line range (digits-digits)
				afterColon := pathWithRange[colonIdx+1:]
				if lineRangeRe.MatchString(afterColon) {
					importPath = pathWithRange[:colonIdx]
					rangeParts := strings.SplitN(afterColon, "-", 2)
					fmt.Sscanf(rangeParts[0], "%d", &startLine)
					fmt.Sscanf(rangeParts[1], "%d", &endLine)
				}
			}

			// Skip URLs - they don't need file validation
			if strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://") {
				continue
			}

			ref := runtimeImportReference{importPath: importPath, startLine: startLine, endLine: endLine}

			key := fmt.Sprintf("%s:%d-%d", ref.importPath, ref.startLine, ref.endLine)
			if !setutil.Contains(seen, key) {
				refs = append(refs, ref)
				seen[key] = struct {
				}{}
			}
		}
	}

	return refs
}

func resolveRuntimeImportValidationPath(filePath, workspaceDir string) (string, string, bool) {
	normalizedPath := filePath
	if strings.HasPrefix(normalizedPath, constants.GithubDir) {
		normalizedPath = normalizedPath[len(constants.GithubDir):] // Remove ".github/"
	} else if strings.HasPrefix(normalizedPath, ".github\\") {
		normalizedPath = normalizedPath[8:] // Remove ".github\" (Windows)
	}
	if strings.HasPrefix(normalizedPath, "./") {
		normalizedPath = normalizedPath[2:] // Remove "./"
	} else if strings.HasPrefix(normalizedPath, ".\\") {
		normalizedPath = normalizedPath[2:] // Remove ".\" (Windows)
	}

	githubFolder := filepath.Join(workspaceDir, ".github")
	absolutePath := filepath.Join(githubFolder, normalizedPath)
	normalizedGithubFolder := filepath.Clean(githubFolder)
	normalizedAbsolutePath := filepath.Clean(absolutePath)
	relativePath, err := filepath.Rel(normalizedGithubFolder, normalizedAbsolutePath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return absolutePath, relativePath, false
	}
	return absolutePath, relativePath, true
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
		absolutePath, relativePath, ok := resolveRuntimeImportValidationPath(filePath, workspaceDir)
		if !ok {
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

		// Best-effort: detect and validate inline sub-agent frontmatter in the
		// runtime-imported file. Unknown fields are collected and returned to the
		// caller so it can emit them through the normal warning counter.
		for _, w := range parser.ValidateInlineSubAgentsFrontmatter(string(content)) {
			subAgentWarnings = append(subAgentWarnings, fmt.Sprintf("runtime-import %q: %s", filePath, w))
		}
		for _, w := range parser.ValidateInlineSkillsFrontmatter(string(content)) {
			subAgentWarnings = append(subAgentWarnings, fmt.Sprintf("runtime-import %q: %s", filePath, w))
		}
	}

	if len(validationErrors) > 0 {
		expressionValidationLog.Printf("Runtime-import validation failed: %d file(s) with errors", len(validationErrors))
		return subAgentWarnings, newRuntimeImportValidationError(validationErrors)
	}

	expressionValidationLog.Print("All runtime-import files validated successfully")
	return subAgentWarnings, nil
}

func newRuntimeImportValidationError(validationErrors []string) error {
	return NewValidationError(
		"runtime-import",
		fmt.Sprintf("%d files with errors", len(validationErrors)),
		"runtime-import files contain expression errors:\n\n"+strings.Join(validationErrors, "\n\n"),
		"Fix the expression errors in the imported files listed above. Each file must only use allowed GitHub Actions expressions. See expression security documentation for details.",
	)
}
