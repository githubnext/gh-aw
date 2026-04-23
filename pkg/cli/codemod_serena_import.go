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
// or engine.tools.serena configuration to an equivalent imports entry using
// shared/mcp/serena.md, and may normalize a pinned source ref to @main.
func getSerenaToSharedImportCodemod() Codemod {
	return Codemod{
		ID:           "serena-tools-to-shared-import",
		Name:         "Migrate tools.serena or engine.tools.serena to shared Serena import",
		Description:  "Removes 'tools.serena' or 'engine.tools.serena', adds an equivalent 'imports' entry using shared/mcp/serena.md with languages, and may rewrite a pinned 'source:' ref to '@main'.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			languages, ok := findSerenaLanguagesForMigration(frontmatter)
			if !ok || len(languages) == 0 {
				return content, false, nil
			}

			alreadyImported := hasSerenaSharedImport(frontmatter)

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, "serena", "tools")
				if !modified {
					return lines, false
				}

				result = removeBlockIfEmpty(result, "tools")
				result = removeBlockIfEmpty(result, "engine")

				if alreadyImported {
					return result, true
				}

				return addSerenaImport(result, languages), true
			})
			if applied {
				newContent = maybeUpdatePinnedSourceRef(newContent, frontmatter)
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

func findSerenaLanguagesForMigration(frontmatter map[string]any) ([]string, bool) {
	toolsAny, hasTools := frontmatter["tools"]
	if hasTools {
		if toolsMap, ok := toolsAny.(map[string]any); ok {
			if serenaAny, hasSerena := toolsMap["serena"]; hasSerena {
				languages, ok := extractSerenaLanguages(serenaAny)
				if ok && len(languages) > 0 {
					return languages, true
				}
			}
		}
	}

	engineAny, hasEngine := frontmatter["engine"]
	if !hasEngine {
		return nil, false
	}

	engineMap, ok := engineAny.(map[string]any)
	if !ok {
		return nil, false
	}

	engineToolsAny, hasEngineTools := engineMap["tools"]
	if !hasEngineTools {
		return nil, false
	}

	engineToolsMap, ok := engineToolsAny.(map[string]any)
	if !ok {
		return nil, false
	}

	serenaAny, hasSerena := engineToolsMap["serena"]
	if !hasSerena {
		return nil, false
	}

	languages, ok := extractSerenaLanguages(serenaAny)
	if !ok || len(languages) == 0 {
		return nil, false
	}

	return languages, true
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
			insertAt = len(lines)
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					insertAt = j
					break
				}
			}
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

func removeBlockIfEmpty(lines []string, blockName string) []string {
	result := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, blockName+":") {
			result = append(result, line)
			i++
			continue
		}

		valuePart := strings.TrimSpace(strings.TrimPrefix(trimmed, blockName+":"))
		if valuePart != "" && !strings.HasPrefix(valuePart, "#") {
			result = append(result, line)
			i++
			continue
		}

		hasMeaningfulNestedContent, blockEnd := hasNestedContent(lines, i+1, getIndentation(line))

		if hasMeaningfulNestedContent {
			result = append(result, line)
			i++
			continue
		}

		i = blockEnd
	}

	return result
}

func hasNestedContent(lines []string, startIndex int, blockIndent string) (bool, int) {
	for i := startIndex; i < len(lines); i++ {
		nestedLine := lines[i]
		nestedTrimmed := strings.TrimSpace(nestedLine)
		if nestedTrimmed == "" {
			continue
		}

		nestedIndent := getIndentation(nestedLine)
		if strings.HasPrefix(nestedTrimmed, "#") {
			if len(nestedIndent) <= len(blockIndent) {
				return false, i
			}
			continue
		}

		if len(nestedIndent) <= len(blockIndent) && strings.Contains(nestedLine, ":") {
			return false, i
		}

		return true, i
	}

	return false, len(lines)
}

func maybeUpdatePinnedSourceRef(content string, frontmatter map[string]any) string {
	sourceAny, hasSource := frontmatter["source"]
	if !hasSource {
		return content
	}

	source, ok := sourceAny.(string)
	if !ok || strings.TrimSpace(source) == "" {
		return content
	}

	sourceSpec, err := parseSourceSpec(source)
	if err != nil {
		return content
	}

	if sourceSpec.Repo != "github/gh-aw" || !IsCommitSHA(sourceSpec.Ref) {
		return content
	}

	updatedSource := sourceSpec.Repo + "/" + sourceSpec.Path + "@main"
	updatedContent, err := UpdateFieldInFrontmatter(content, "source", updatedSource)
	if err != nil {
		return content
	}
	return updatedContent
}
