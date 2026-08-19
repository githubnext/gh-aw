package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var minIntegrityNoneBashCodemodLog = logger.New("cli:codemod_min_integrity_none_bash")

// getMinIntegrityNoneRequiresBashCodemod creates a codemod that adds an explicit
// 'tools.bash: false' when 'tools.github.min-integrity' is set to 'none' and
// 'tools.bash' is not already specified.
//
// Strict mode requires bash access to be explicit whenever min-integrity is none, since
// any external user can trigger the workflow. No bash tool was configured before, so
// inserting 'bash: false' preserves the existing behavior while satisfying the new
// strict-mode requirement.
func getMinIntegrityNoneRequiresBashCodemod() Codemod {
	return Codemod{
		ID:           "min-integrity-none-requires-bash",
		Name:         "Add explicit 'tools.bash: false' when 'tools.github.min-integrity' is 'none'",
		Description:  "Inserts 'tools.bash: false' when 'tools.github.min-integrity' is set to 'none' and 'tools.bash' is not already specified, preserving current behavior while satisfying strict mode",
		IntroducedIn: "1.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsMap, ok := frontmatter["tools"].(map[string]any)
			if !ok {
				return content, false, nil
			}

			if _, hasBash := toolsMap["bash"]; hasBash {
				return content, false, nil
			}

			githubMap, ok := toolsMap["github"].(map[string]any)
			if !ok {
				return content, false, nil
			}

			minIntegrity, ok := githubMap["min-integrity"].(string)
			if !ok || minIntegrity != "none" {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, insertBashFalseIntoTopLevelTools)
			if applied {
				minIntegrityNoneBashCodemodLog.Print("Inserted 'tools.bash: false' because tools.github.min-integrity is 'none'")
			}
			return newContent, applied, err
		},
	}
}

// insertBashFalseIntoTopLevelTools inserts 'bash: false' as the first child of the
// top-level 'tools:' block, supporting both block mappings and inline flow mappings.
// It assumes the caller has already verified that 'tools' exists as a mapping and that
// 'tools.bash' is not already present.
func insertBashFalseIntoTopLevelTools(lines []string) ([]string, bool) {
	toolsLine := -1
	for i, line := range lines {
		if isTopLevelBlockKey(line, "tools") {
			toolsLine = i
			break
		}
	}
	if toolsLine == -1 {
		return insertEntryIntoInlineMapping(lines, "tools", "bash: false")
	}

	fieldIndent := "  "
	insertAt := toolsLine + 1

	for i := toolsLine + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if hasExitedBlock(line, "") {
			break
		}
		fieldIndent = getIndentation(line)
		insertAt = i
		break
	}

	result := insertLine(lines, insertAt, fieldIndent+"bash: false")
	return result, true
}
