package cli

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var serenaImportCodemodLog = logger.New("cli:codemod_serena_import")

// getSerenaToSharedImportCodemod creates a codemod that migrates removed tools.serena
// or engine.tools.serena configuration to an equivalent imports entry using
// shared/mcp/serena.md. The existing source: pin is preserved unchanged.
func getSerenaToSharedImportCodemod() Codemod {
	return Codemod{
		ID:           "serena-tools-to-shared-import",
		Name:         "Migrate tools.serena or engine.tools.serena to shared Serena import",
		Description:  "Removes 'tools.serena' or 'engine.tools.serena' and adds an equivalent 'imports' entry using shared/mcp/serena.md with languages. The existing 'source:' pin is preserved.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			languages, ok := findSerenaLanguagesForMigration(frontmatter)
			isListForm := false
			if !ok || len(languages) == 0 {
				// Check if tools is a list containing "serena" (no languages specified)
				if !isSerenaInToolsList(frontmatter) {
					return content, false, nil
				}
				// List form detected — migrate with empty placeholder (languages stays nil/empty)
				isListForm = true
			}

			alreadyImported := hasSerenaSharedImport(frontmatter)

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if err := removeSerenaTools(f, isListForm); err != nil {
					return err
				}
				if alreadyImported {
					return nil
				}
				return addSerenaSharedImport(f, languages)
			})
			if applied {
				if alreadyImported {
					serenaImportCodemodLog.Print("Removed tools.serena (shared/mcp/serena.md import already present)")
				} else if isListForm {
					serenaImportCodemodLog.Print("Migrated tools list entry 'serena' to shared/mcp/serena.md import (no languages specified — placeholder added)")
				} else {
					serenaImportCodemodLog.Printf("Migrated tools.serena to shared/mcp/serena.md import with %d language(s)", len(languages))
				}
			}
			return newContent, applied, err
		},
	}
}

func removeSerenaTools(f *codemod.Frontmatter, listForm bool) error {
	if listForm {
		tools, ok := f.Document.Frontmatter["tools"].([]any)
		if !ok {
			return errors.New("cannot migrate Serena: tools must be a sequence")
		}
		remaining := slices.DeleteFunc(slices.Clone(tools), func(value any) bool {
			text, ok := value.(string)
			return ok && strings.EqualFold(strings.TrimSpace(text), "serena")
		})
		if len(remaining) == 0 {
			return f.Remove("tools")
		}
		return f.Set([]string{"tools"}, remaining)
	}
	for _, path := range [][]string{{"tools"}, {"engine", "tools"}} {
		if err := f.Remove(append(slices.Clone(path), "serena")...); err != nil {
			return err
		}
		if err := f.PruneEmpty(path...); err != nil {
			return err
		}
	}
	return f.PruneEmpty("engine")
}

func addSerenaSharedImport(f *codemod.Frontmatter, languages []string) error {
	if languages == nil {
		languages = []string{}
	}
	entry := map[string]any{
		"uses": "shared/mcp/serena.md",
		"with": map[string]any{"languages": languages},
	}
	imports, exists := f.Lookup("imports")
	var entries []any
	if exists {
		var ok bool
		entries, ok = imports.([]any)
		if !ok {
			return errors.New("cannot migrate Serena: imports must be a sequence")
		}
	}
	if err := f.Set([]string{"imports"}, append(slices.Clone(entries), entry)); err != nil {
		return err
	}
	if len(languages) == 0 {
		return f.Comment([]string{"imports"}, `TODO: specify Serena languages, e.g. ["TypeScript", "JavaScript"]`)
	}
	return nil
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

// isSerenaInToolsList reports whether the "tools" frontmatter key is a YAML list that
// contains the plain string "serena". This covers the shorthand syntax:
//
//	tools:
//	  - serena
//
// which does not carry a languages specification.
func isSerenaInToolsList(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	switch tools := toolsAny.(type) {
	case []string:
		for _, item := range tools {
			if strings.EqualFold(strings.TrimSpace(item), "serena") {
				return true
			}
		}
	case []any:
		for _, item := range tools {
			s, ok := item.(string)
			if ok && strings.EqualFold(strings.TrimSpace(s), "serena") {
				return true
			}
		}
	}
	return false
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
