package cli

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var serenaImportCodemodLog = logger.New("cli:codemod_serena_import")

// getSerenaToSharedImportCodemod creates a codemod that migrates removed tools.serena
// configuration to an equivalent imports entry using shared/mcp/serena.md.
func getSerenaToSharedImportCodemod() Codemod {
	return Codemod{
		ID:           "serena-tools-to-shared-import",
		Name:         "Migrate tools.serena to shared Serena import",
		Description:  "Removes 'tools.serena' and adds an equivalent 'imports' entry using shared/mcp/serena.md with languages.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsAny, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsAny.(map[string]any)
			if !ok {
				return content, false, nil
			}

			serenaAny, hasSerena := toolsMap["serena"]
			if !hasSerena {
				return content, false, nil
			}

			languages, ok := extractSerenaLanguages(serenaAny)
			if !ok || len(languages) == 0 {
				serenaImportCodemodLog.Print("Found tools.serena but languages configuration is invalid or empty - skipping migration; verify tools.serena languages are set")
				return content, false, nil
			}

			alreadyImported := hasSerenaSharedImport(frontmatter)

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, "serena", "tools")
				if !modified {
					return lines, false
				}

				result = removeTopLevelBlockIfEmpty(result, "tools")

				if alreadyImported {
					return result, true
				}

				return addSerenaImport(result, languages), true
			})
			if applied {
				if alreadyImported {
					serenaImportCodemodLog.Print("Removed tools.serena (shared/mcp/serena.md import already present)")
				} else {
					serenaImportCodemodLog.Printf("Migrated tools.serena to shared/mcp/serena.md import with %d language(s)", len(languages))
				}
			}
			return newContent, applied, err
		},
	}
}

func extractSerenaLanguages(serenaAny any) ([]string, bool) {
	switch serena := serenaAny.(type) {
	case []string:
		return sliceutil.Deduplicate(serena), len(serena) > 0
	case []any:
		var languages []string
		for _, item := range serena {
			lang, ok := item.(string)
			if ok && strings.TrimSpace(lang) != "" {
				languages = append(languages, lang)
			}
		}
		return sliceutil.Deduplicate(languages), len(languages) > 0
	case string:
		trimmed := strings.TrimSpace(serena)
		if trimmed == "" {
			return nil, false
		}
		return []string{trimmed}, true
	case map[string]any:
		languagesAny, hasLanguages := serena["languages"]
		if !hasLanguages {
			return nil, false
		}
		return extractSerenaLanguagesFromLanguagesField(languagesAny)
	default:
		return nil, false
	}
}

func extractSerenaLanguagesFromLanguagesField(languagesAny any) ([]string, bool) {
	switch languages := languagesAny.(type) {
	case []string:
		return sliceutil.Deduplicate(languages), len(languages) > 0
	case []any:
		var result []string
		for _, item := range languages {
			lang, ok := item.(string)
			if ok && strings.TrimSpace(lang) != "" {
				result = append(result, lang)
			}
		}
		return sliceutil.Deduplicate(result), len(result) > 0
	case string:
		trimmed := strings.TrimSpace(languages)
		if trimmed == "" {
			return nil, false
		}
		return []string{trimmed}, true
	case map[string]any:
		var result []string
		for language := range languages {
			if strings.TrimSpace(language) != "" {
				result = append(result, language)
			}
		}
		sort.Strings(result)
		return sliceutil.Deduplicate(result), len(result) > 0
	default:
		return nil, false
	}
}

func hasSerenaSharedImport(frontmatter map[string]any) bool {
	importsAny, hasImports := frontmatter["imports"]
	if !hasImports {
		return false
	}

	switch imports := importsAny.(type) {
	case []string:
		return slices.ContainsFunc(imports, isSerenaImportPath)
	case []any:
		for _, entry := range imports {
			switch typed := entry.(type) {
			case string:
				if isSerenaImportPath(typed) {
					return true
				}
			case map[string]any:
				usesAny, hasUses := typed["uses"]
				if !hasUses {
					continue
				}
				uses, ok := usesAny.(string)
				if ok && isSerenaImportPath(uses) {
					return true
				}
			}
		}
	}

	return false
}

func isSerenaImportPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return trimmed == "shared/mcp/serena.md" || trimmed == "shared/mcp/serena"
}

func addSerenaImport(lines []string, languages []string) []string {
	entry := []string{
		"  - uses: shared/mcp/serena.md",
		"    with:",
		"      languages: " + formatStringArrayInline(languages),
	}

	importsIdx := -1
	importsEnd := len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "imports:") {
			importsIdx = i
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					importsEnd = j
					break
				}
			}
			break
		}
	}

	if importsIdx >= 0 {
		result := make([]string, 0, len(lines)+len(entry))
		result = append(result, lines[:importsEnd]...)
		result = append(result, entry...)
		result = append(result, lines[importsEnd:]...)
		return result
	}

	insertAt := 0
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "engine:") {
			insertAt = i + 1
			break
		}
	}

	importBlock := make([]string, 0, 1+len(entry))
	importBlock = append(importBlock, "imports:")
	importBlock = append(importBlock, entry...)

	result := make([]string, 0, len(lines)+len(importBlock))
	result = append(result, lines[:insertAt]...)
	result = append(result, importBlock...)
	result = append(result, lines[insertAt:]...)
	return result
}

func formatStringArrayInline(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func removeTopLevelBlockIfEmpty(lines []string, blockName string) []string {
	blockIdx := -1
	blockEnd := len(lines)
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), blockName+":") {
			blockIdx = i
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					blockEnd = j
					break
				}
			}
			break
		}
	}

	if blockIdx == -1 {
		return lines
	}

	hasMeaningfulNestedContent := false
	for _, line := range lines[blockIdx+1 : blockEnd] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		hasMeaningfulNestedContent = true
		break
	}

	if hasMeaningfulNestedContent {
		return lines
	}

	result := make([]string, 0, len(lines)-(blockEnd-blockIdx))
	result = append(result, lines[:blockIdx]...)
	result = append(result, lines[blockEnd:]...)
	return result
}
