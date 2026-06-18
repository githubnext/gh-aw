package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputRequireTitlePrefixCodemodLog = logger.New("cli:codemod_safe_output_require_title_prefix")

func getSafeOutputRequireTitlePrefixCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-title-prefix-to-required-title-prefix",
		Name:         "Rename deprecated safe-outputs title-prefix constraints",
		Description:  "Renames deprecated constraint fields to required-title-prefix/required-labels for applicable safe-outputs handlers.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			handlersToRename := safeOutputsHandlersNeedingTitlePrefixMigration(frontmatter)
			if len(handlersToRename) == 0 {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return renameSafeOutputTitlePrefixConstraints(lines, handlersToRename)
			})
			if applied {
				safeOutputRequireTitlePrefixCodemodLog.Print("Renamed deprecated safe-outputs constraint keys to required-title-prefix/required-labels")
			}
			return newContent, applied, err
		},
	}
}

func safeOutputsHandlersNeedingTitlePrefixMigration(frontmatter map[string]any) map[string]struct {
} {
	result := map[string]struct {
	}{}
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return result
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return result
	}

	handlers := []string{
		"close-issue",
		"close-pull-request",
		"close-discussion",
		"mark-pull-request-as-ready-for-review",
		"push-to-pull-request-branch",
	}

	for _, handler := range handlers {
		handlerAny, ok := safeOutputsMap[handler]
		if !ok {
			continue
		}
		handlerMap, ok := handlerAny.(map[string]any)
		if !ok {
			continue
		}
		needsTitlePrefixRename := false
		if _, hasRequired := handlerMap["required-title-prefix"]; !hasRequired {
			if _, hasDeprecated := handlerMap["title-prefix"]; hasDeprecated {
				needsTitlePrefixRename = true
			}
		}

		needsRequiredLabelsRename := false
		if handler == "push-to-pull-request-branch" {
			if _, hasRequired := handlerMap["required-labels"]; !hasRequired {
				if _, hasDeprecated := handlerMap["labels"]; hasDeprecated {
					needsRequiredLabelsRename = true
				}
			}
		}

		if needsTitlePrefixRename || needsRequiredLabelsRename {
			result[handler] = struct {
			}{}
		}
	}

	return result
}

func renameSafeOutputTitlePrefixConstraints(lines []string, handlersToRename map[string]struct {
}) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""
	activeHandler := ""
	activeHandlerIndent := ""
	handlerChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if !strings.HasPrefix(trimmed, "#") {
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsChildIndent = ""
				activeHandler = ""
				activeHandlerIndent = ""
				handlerChildIndent = ""
			}
			if activeHandler != "" && hasExitedBlock(line, activeHandlerIndent) {
				activeHandler = ""
				activeHandlerIndent = ""
				handlerChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			activeHandler = ""
			activeHandlerIndent = ""
			handlerChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			if activeHandler != "" && handlerChildIndent == "" && isDescendant(indent, activeHandlerIndent) {
				handlerChildIndent = indent
			}
			if safeOutputsChildIndent == "" {
				safeOutputsChildIndent = indent
			}
			if indent != safeOutputsChildIndent {
				result = append(result, line)
				continue
			}
			key := strings.TrimSuffix(trimmed, ":")
			if hasStringKey(handlersToRename, key) {
				activeHandler = key
				activeHandlerIndent = indent
				handlerChildIndent = ""
			} else {
				activeHandler = ""
				activeHandlerIndent = ""
				handlerChildIndent = ""
			}
			result = append(result, line)
			continue
		}

		if activeHandler != "" && handlerChildIndent == "" && isDescendant(indent, activeHandlerIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "- ") {
			handlerChildIndent = indent
		}

		if activeHandler != "" && indent == handlerChildIndent && strings.HasPrefix(trimmed, "title-prefix:") {
			newLine, replaced := findAndReplaceInLine(line, "title-prefix", "required-title-prefix")
			if replaced {
				result = append(result, newLine)
				modified = true
				safeOutputRequireTitlePrefixCodemodLog.Printf("Renamed title-prefix in safe-outputs.%s on line %d", activeHandler, i+1)
				continue
			}
		}
		if activeHandler == "push-to-pull-request-branch" && indent == handlerChildIndent && strings.HasPrefix(trimmed, "labels:") {
			newLine, replaced := findAndReplaceInLine(line, "labels", "required-labels")
			if replaced {
				result = append(result, newLine)
				modified = true
				safeOutputRequireTitlePrefixCodemodLog.Printf("Renamed labels in safe-outputs.%s on line %d", activeHandler, i+1)
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}
