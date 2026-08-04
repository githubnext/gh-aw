package cli

import (
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var opencodeImportCodemodLog = logger.New("cli:codemod_opencode_import")

const openCodeSharedImport = "github/gh-aw/.github/workflows/shared/opencode.md@main"

// getOpenCodeSharedImportCodemod adds the shared OpenCode engine definition to
// workflows that select OpenCode as their engine.
func getOpenCodeSharedImportCodemod() Codemod {
	return Codemod{
		ID:           "opencode-engine-to-shared-import",
		Name:         "Add shared OpenCode import",
		Description:  "Adds the github/gh-aw shared OpenCode import when the workflow uses the OpenCode engine.",
		IntroducedIn: "0.40.1",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !usesOpenCodeEngine(frontmatter) || hasOpenCodeSharedImport(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, addOpenCodeSharedImport)
			if applied {
				opencodeImportCodemodLog.Print("Added github/gh-aw shared OpenCode import")
			}
			return newContent, applied, err
		},
	}
}

func usesOpenCodeEngine(frontmatter map[string]any) bool {
	engine, ok := frontmatter["engine"]
	if !ok {
		return false
	}

	switch value := engine.(type) {
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "opencode")
	case map[string]any:
		id, ok := value["id"].(string)
		return ok && strings.EqualFold(strings.TrimSpace(id), "opencode")
	default:
		return false
	}
}

func hasOpenCodeSharedImport(frontmatter map[string]any) bool {
	imports, ok := frontmatter["imports"]
	if !ok {
		return false
	}

	switch value := imports.(type) {
	case []string:
		return slices.ContainsFunc(value, isOpenCodeImportPath)
	case []any:
		for _, entry := range value {
			switch importEntry := entry.(type) {
			case string:
				if isOpenCodeImportPath(importEntry) {
					return true
				}
			case map[string]any:
				uses, ok := importEntry["uses"].(string)
				if ok && isOpenCodeImportPath(uses) {
					return true
				}
			}
		}
	}

	return false
}

func isOpenCodeImportPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return trimmed == openCodeSharedImport || trimmed == "shared/opencode.md" || trimmed == "shared/opencode"
}

func addOpenCodeSharedImport(lines []string) ([]string, bool) {
	entry := "  - " + openCodeSharedImport
	for i, line := range lines {
		if !isTopLevelKey(line) || !strings.HasPrefix(strings.TrimSpace(line), "imports:") {
			continue
		}

		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if isTopLevelKey(lines[j]) {
				end = j
				break
			}
		}
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:end]...)
		result = append(result, entry)
		result = append(result, lines[end:]...)
		return result, true
	}

	insertAt := len(lines)
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "engine:") {
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					insertAt = j
					break
				}
			}
			break
		}
	}

	result := make([]string, 0, len(lines)+2)
	result = append(result, lines[:insertAt]...)
	result = append(result, "imports:", entry)
	result = append(result, lines[insertAt:]...)
	return result, true
}
