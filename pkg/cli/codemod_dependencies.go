package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var dependenciesCodemodLog = logger.New("cli:codemod_dependencies")

// getDependenciesToImportsAPMPackagesCodemod creates a codemod that migrates the top-level
// `dependencies:` field to `imports.apm-packages:`. The `dependencies:` field is deprecated
// in favour of `imports.apm-packages:`, which co-locates APM package configuration alongside
// shared agentic workflow imports under the unified `imports` key.
//
// Migration rules:
//   - The entire `dependencies:` block (array or object form) is moved to `imports.apm-packages:`
//   - If `imports` is absent, a new `imports:` block is created with `apm-packages:` inside it
//   - If `imports` is an array, it is converted to the object form with the existing items
//     placed under `aw:` and the dependencies placed under `apm-packages:`
//   - If `imports` is already an object, `apm-packages:` is added to it
//   - If `imports.apm-packages` already exists the codemod is skipped to avoid clobbering
func getDependenciesToImportsAPMPackagesCodemod() Codemod {
	return Codemod{
		ID:           "dependencies-to-imports-apm-packages",
		Name:         "Migrate dependencies to imports.apm-packages",
		Description:  "Moves the top-level 'dependencies' field to 'imports.apm-packages'. The 'dependencies' field is deprecated in favour of 'imports.apm-packages'.",
		IntroducedIn: "1.18.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			_, hasDeps := frontmatter["dependencies"]
			if !hasDeps {
				return content, false, nil
			}

			// Skip if imports.apm-packages already exists to avoid clobbering user config.
			if importsAny, hasImports := frontmatter["imports"]; hasImports {
				if importsMap, ok := importsAny.(map[string]any); ok {
					if _, hasAPM := importsMap["apm-packages"]; hasAPM {
						dependenciesCodemodLog.Print("'imports.apm-packages' already exists – skipping migration to avoid overwriting")
						return content, false, nil
					}
				}
			}

			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return migrateDependenciesToImportsAPMPackages(lines, frontmatter)
			})
		},
	}
}

// migrateDependenciesToImportsAPMPackages rewrites the frontmatter lines to move the
// `dependencies:` block to `imports.apm-packages:`, handling three cases:
//  1. No `imports` field: create `imports:\n  apm-packages:\n    ...`
//  2. `imports` is an array: convert to object form with `aw:` and `apm-packages:`
//  3. `imports` is already an object: append `apm-packages:` to it
func migrateDependenciesToImportsAPMPackages(lines []string, frontmatter map[string]any) ([]string, bool) {
	// Locate the dependencies: block.
	depsIdx, depsEnd := findTopLevelBlock(lines, "dependencies")
	if depsIdx == -1 {
		return lines, false
	}
	depsIndent := getIndentation(lines[depsIdx])
	// depsBodyRaw are the raw lines of the dependencies block body (everything after the key line).
	depsBodyRaw := lines[depsIdx+1 : depsEnd]
	// The body items are indented by depsIndent+"  " in the original file.
	depsBodyItemIndent := depsIndent + "  "

	// Locate the imports: block (if any).
	importsIdx, importsEnd := findTopLevelBlock(lines, "imports")

	// Determine the current form of imports (absent / array / object).
	_, importsIsObject := frontmatter["imports"].(map[string]any)

	var result []string

	switch {
	case importsIdx == -1:
		// Case 1: No imports field — replace dependencies block with imports block.
		result = make([]string, 0, len(lines)+len(depsBodyRaw)+2)
		for i, line := range lines {
			if i == depsIdx {
				result = append(result, "imports:")
				result = append(result, "  apm-packages:")
				result = append(result, reindentBlock(depsBodyRaw, depsBodyItemIndent, "    ")...)
				continue
			}
			if i > depsIdx && i < depsEnd {
				continue
			}
			result = append(result, line)
		}

	case !importsIsObject:
		// Case 2: imports is an array — convert to object form with aw and apm-packages.
		importsBodyRaw := lines[importsIdx+1 : importsEnd]
		// Imports body items are indented by 2 spaces (top-level imports).
		importsBodyItemIndent := "  "

		result = make([]string, 0, len(lines)+len(importsBodyRaw)+len(depsBodyRaw)+3)

		insertedImports := false
		for i, line := range lines {
			if i >= importsIdx && i < importsEnd {
				if i == importsIdx && !insertedImports {
					result = append(result, "imports:")
					result = append(result, "  aw:")
					result = append(result, reindentBlock(importsBodyRaw, importsBodyItemIndent, "    ")...)
					result = append(result, "  apm-packages:")
					result = append(result, reindentBlock(depsBodyRaw, depsBodyItemIndent, "    ")...)
					insertedImports = true
				}
				continue
			}
			if i >= depsIdx && i < depsEnd {
				continue
			}
			result = append(result, line)
		}

	default:
		// Case 3: imports is already an object — append apm-packages to it.
		result = make([]string, 0, len(lines)+len(depsBodyRaw)+2)

		for i, line := range lines {
			if i == importsEnd {
				result = append(result, "  apm-packages:")
				result = append(result, reindentBlock(depsBodyRaw, depsBodyItemIndent, "    ")...)
			}
			if i >= depsIdx && i < depsEnd {
				continue
			}
			result = append(result, line)
		}
	}

	dependenciesCodemodLog.Print("Migrated 'dependencies' to 'imports.apm-packages'")
	return result, true
}

// findTopLevelBlock returns the start index (inclusive) and end index (exclusive)
// of the top-level YAML block with the given key name. Returns (-1, -1) if not found.
func findTopLevelBlock(lines []string, key string) (startIdx, endIdx int) {
	startIdx = -1
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), key+":") {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		return -1, -1
	}
	blockIndent := getIndentation(lines[startIdx])
	endIdx = startIdx + 1
	for endIdx < len(lines) {
		line := lines[endIdx]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			endIdx++
			continue
		}
		if isNestedUnder(line, blockIndent) {
			endIdx++
			continue
		}
		break
	}
	return startIdx, endIdx
}

// reindentBlock changes the indentation prefix of a set of lines from oldPrefix to newPrefix.
// Lines whose indentation does not start with oldPrefix are left unchanged (safe fallback).
func reindentBlock(lines []string, oldPrefix, newPrefix string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}
		currentIndent := getIndentation(line)
		// Only reindent lines whose current indent starts with oldPrefix.
		if !strings.HasPrefix(currentIndent, oldPrefix) {
			result = append(result, line)
			continue
		}
		// Compute how many extra spaces this line has beyond the old prefix length.
		extra := currentIndent[len(oldPrefix):]
		result = append(result, newPrefix+extra+trimmed)
	}
	return result
}
