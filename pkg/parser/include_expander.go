package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var includeExpanderLog = logger.New("parser:include_expander")

// hasIncludeDirectives reports whether content contains any include/import directive that
// ParseImportDirective could match. Used as a fast pre-check to avoid scanner allocations.
// Matches @include, @import (legacy), and {{#import (legacy) forms.
func hasIncludeDirectives(content string) bool {
	return strings.Contains(content, "@include") ||
		strings.Contains(content, "@import") ||
		strings.Contains(content, "{{#import")
}

// ExpandIncludes recursively expands @include and @import directives until no more remain
// This matches the bash expand_includes function behavior

// ExpandIncludesWithManifest recursively expands @include and @import directives and returns list of included files
func ExpandIncludesWithManifest(content, baseDir string, extractTools bool) (string, []string, error) {
	includeExpanderLog.Printf("Expanding includes: baseDir=%s, extractTools=%t, content_size=%d", baseDir, extractTools, len(content))

	// Fast path: skip expansion entirely when no include/import directives are present.
	// This avoids scanner and buffer allocations in the common case where there are no includes.
	// For content mode, preserve the scanner's trailing-newline normalization behavior.
	if !hasIncludeDirectives(content) {
		includeExpanderLog.Print("Fast path: no include directives found")
		if extractTools {
			return "{}", nil, nil
		}
		if !strings.HasSuffix(content, "\n") {
			return content + "\n", nil, nil
		}
		return content, nil, nil
	}

	const maxDepth = 10
	currentContent := content
	visited := make(map[string]bool)

	for depth := range maxDepth {
		includeExpanderLog.Printf("Include expansion depth: %d", depth)
		// Process includes in current content
		processedContent, err := processIncludesWithVisited(currentContent, baseDir, extractTools, visited)
		if err != nil {
			return "", nil, err
		}

		// For tools mode, check if we still have @include or @import directives
		if extractTools {
			if !strings.Contains(processedContent, "@include") && !strings.Contains(processedContent, "@import") {
				// No more includes to process for tools mode
				currentContent = processedContent
				break
			}
		} else {
			// For content mode, check if content changed
			if processedContent == currentContent {
				// No more includes to process
				break
			}
		}

		currentContent = processedContent
	}

	// Find the repo root by walking up from baseDir to the parent of the .github folder.
	// This allows files outside baseDir (e.g. .github/shared/ when baseDir is .github/workflows/)
	// to be recorded with a clean repo-root-relative path instead of an absolute path.
	repoRoot := findGitHubRepoRoot(baseDir)

	// Convert visited map to slice of file paths (make them relative to baseDir if possible,
	// falling back to repo-root-relative, and only as a last resort using the absolute path)
	var includedFiles []string
	for filePath := range visited {
		// First: try to make path relative to baseDir for cleaner output
		relPath, err := filepath.Rel(baseDir, filePath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			// Normalize to Unix paths (forward slashes) for cross-platform compatibility
			relPath = filepath.ToSlash(relPath)
			includedFiles = append(includedFiles, relPath)
			continue
		}

		// Second: try repo-root-relative path to avoid absolute paths for files in sibling
		// directories (e.g. .github/shared/ relative to .github/workflows/)
		if repoRoot != "" {
			repoRelPath, repoRelErr := filepath.Rel(repoRoot, filePath)
			if repoRelErr == nil && !strings.HasPrefix(repoRelPath, "..") {
				repoRelPath = filepath.ToSlash(repoRelPath)
				includedFiles = append(includedFiles, repoRelPath)
				continue
			}
		}

		// Fallback: use the absolute path (should be rare)
		includedFiles = append(includedFiles, filepath.ToSlash(filePath))
	}

	includeExpanderLog.Printf("Include expansion complete: visited_files=%d", len(includedFiles))
	if extractTools {
		// For tools mode, merge all extracted JSON objects
		mergedTools, err := mergeToolsFromJSON(currentContent)
		return mergedTools, includedFiles, err
	}

	return currentContent, includedFiles, nil
}

// findGitHubRepoRoot walks up the directory tree from dir to find the parent of the
// first ".github" directory encountered. It is used to compute repo-root-relative
// paths for files that live in sibling .github/ subdirectories (e.g. .github/shared/)
// so that the lock file Includes header shows ".github/shared/editorial.md" rather
// than an absolute system path.
//
// Returns the repo root directory (the parent of ".github"), or "" if no ".github"
// ancestor directory is found before reaching the filesystem root.
func findGitHubRepoRoot(dir string) string {
	current := filepath.Clean(dir)
	for {
		if filepath.Base(current) == ".github" {
			return filepath.Dir(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return ""
		}
		current = parent
	}
}

// BodyLevelImport represents a single {{#runtime-import}} or deprecated {{#import}} directive
// found in a markdown body, with the path resolved to be workspace-root-relative.
type BodyLevelImport struct {
	Path     string // workspace-root-relative path for the {{#runtime-import}} macro
	Optional bool   // true when the original directive used the ? form
}

// bodyLevelRuntimeImportRe matches {{#runtime-import}} and {{#runtime-import?}} directives
// in a single line of markdown (same pattern as runtime_import.cjs uses at runtime).
var bodyLevelRuntimeImportRe = regexp.MustCompile(`^\{\{#runtime-import(\?)?[ \t]+([^\}]+?)\}\}$`)

// ExtractBodyLevelImportPaths scans the markdown body (content is the body after frontmatter
// has been stripped) for {{#runtime-import}} directives and returns them as BodyLevelImport entries
// whose Path fields are ready to use in explicit {{#runtime-import}} macros in the compiled lock file.
//
// Relative paths (e.g. "shared/tools.md") are converted to workspace-root-relative form
// (e.g. ".github/workflows/shared/tools.md") using baseDir and the repo root.
// Paths that already start with ".github/" are kept as-is.
// Deprecated {{#import}} and legacy @include / @import directives are ignored;
// they are handled (with deprecation warnings) by include_processor.go.
func ExtractBodyLevelImportPaths(content, baseDir string) []BodyLevelImport {
	// Fast path: no {{#runtime-import}} directives present.
	if !strings.Contains(content, "{{#runtime-import") {
		return nil
	}

	repoRoot := findGitHubRepoRoot(baseDir)

	var results []BodyLevelImport
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Match {{#runtime-import}} directives only.
		m := bodyLevelRuntimeImportRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		optional := m[1] == "?"

		// Skip optional directives — they are handled with proper semantics at runtime
		// when runtime_import.cjs processes the workflow body. Promoting an optional
		// directive as a required macro would cause failures if the file is missing.
		if optional {
			continue
		}
		importPath := strings.TrimSpace(m[2])

		// Strip section reference (e.g. "file.md#Section" → "file.md")
		if idx := strings.Index(importPath, "#"); idx >= 0 {
			importPath = importPath[:idx]
		}
		importPath = strings.TrimSpace(importPath)

		// Skip URLs — these are fetched at runtime and don't need promotion.
		if strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://") {
			continue
		}

		// Convert relative paths to workspace-root-relative.
		// Paths already starting with ".github/" are workspace-root-relative.
		// Absolute paths are used as-is.
		if !strings.HasPrefix(importPath, ".github/") && !filepath.IsAbs(importPath) {
			if repoRoot != "" {
				fullPath := filepath.Join(baseDir, importPath)
				if rel, err := filepath.Rel(repoRoot, fullPath); err == nil && !strings.HasPrefix(rel, "..") {
					importPath = rel
				}
			}
		}

		results = append(results, BodyLevelImport{
			Path:     filepath.ToSlash(importPath),
			Optional: false, // optional directives are skipped above; only required imports are promoted
		})
	}
	return results
}

func ExpandIncludesForEngines(content, baseDir string) ([]string, error) {
	includeExpanderLog.Printf("Expanding includes for engines: baseDir=%s", baseDir)
	return expandIncludesForField(content, baseDir, func(c string) (string, error) {
		return extractFrontmatterField(c, "engine", "")
	}, "")
}

// ExpandIncludesForSafeOutputs recursively expands @include and @import directives to extract safe-outputs configurations
func ExpandIncludesForSafeOutputs(content, baseDir string) ([]string, error) {
	includeExpanderLog.Printf("Expanding includes for safe-outputs: baseDir=%s", baseDir)
	return expandIncludesForField(content, baseDir, func(c string) (string, error) {
		return extractFrontmatterField(c, "safe-outputs", "{}")
	}, "{}")
}

// expandIncludesForField recursively expands includes to extract a specific frontmatter field
func expandIncludesForField(content, baseDir string, extractFunc func(string) (string, error), emptyValue string) ([]string, error) {
	// Fast path: skip expansion entirely when no include/import directives are present.
	if !hasIncludeDirectives(content) {
		return nil, nil
	}

	const maxDepth = 10
	var results []string
	currentContent := content

	for range maxDepth {
		// Process includes in current content to extract the field
		processedResults, processedContent, err := processIncludesForField(currentContent, baseDir, extractFunc, emptyValue)
		if err != nil {
			return nil, err
		}

		// Add found results to the list
		results = append(results, processedResults...)

		// Check if content changed
		if processedContent == currentContent {
			// No more includes to process
			break
		}

		currentContent = processedContent
	}

	includeExpanderLog.Printf("Field expansion complete: results=%d", len(results))
	return results, nil
}

// processIncludesForField processes import directives to extract a specific frontmatter field
func processIncludesForField(content, baseDir string, extractFunc func(string) (string, error), emptyValue string) ([]string, string, error) {
	// Fast path: skip scanner allocation when no include/import directives are present.
	if !hasIncludeDirectives(content) {
		return nil, content, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var result bytes.Buffer
	var results []string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse import directive
		directive := ParseImportDirective(line)
		if directive != nil {
			isOptional := directive.IsOptional
			includePath := directive.Path

			// Handle section references (file.md#Section) - for frontmatter fields, we ignore sections
			var filePath string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
				// Note: section references are ignored for frontmatter field extraction
			} else {
				filePath = includePath
			}

			// Resolve file path
			fullPath, err := ResolveIncludePath(filePath, baseDir, nil)
			if err != nil {
				if isOptional {
					// For optional includes, skip extraction
					continue
				}
				// For required includes, fail compilation with an error
				return nil, "", fmt.Errorf("failed to resolve required include '%s': %w", filePath, err)
			}

			// Read the included file
			fileContent, err := readFileFunc(fullPath)
			if err != nil {
				// For any processing errors, fail compilation
				return nil, "", fmt.Errorf("failed to read included file '%s': %w", fullPath, err)
			}

			// Extract the field using the provided extraction function
			fieldJSON, err := extractFunc(string(fileContent))
			if err != nil {
				return nil, "", fmt.Errorf("failed to extract field from '%s': %w", fullPath, err)
			}

			if fieldJSON != "" && fieldJSON != emptyValue {
				results = append(results, fieldJSON)
			}
		} else {
			// Regular line, just pass through
			result.WriteString(line + "\n")
		}
	}

	return results, result.String(), nil
}
