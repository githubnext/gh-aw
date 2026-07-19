package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
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

	state := renameSafeOutputTitlePrefixConstraintsState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		renameSafeOutputTitlePrefixConstraintsUpdateBlockState(line, trimmed, &state)

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			renameSafeOutputTitlePrefixConstraintsEnterSafeOutputs(indent, &state)
			result = append(result, line)
			continue
		}

		if renameSafeOutputTitlePrefixConstraintsEnterHandler(trimmed, indent, handlersToRename, &state) {
			result = append(result, line)
			continue
		}

		renameSafeOutputTitlePrefixConstraintsSetChildIndent(trimmed, indent, &state)

		if newLine, replaced := renameSafeOutputTitlePrefixConstraintsReplaceLine(line, trimmed, indent, i, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameSafeOutputTitlePrefixConstraintsState struct {
	inSafeOutputs          bool
	safeOutputsIndent      string
	safeOutputsChildIndent string
	activeHandler          string
	activeHandlerIndent    string
	handlerChildIndent     string
}

func renameSafeOutputTitlePrefixConstraintsUpdateBlockState(line, trimmed string, state *renameSafeOutputTitlePrefixConstraintsState) {
	if strings.HasPrefix(trimmed, "#") {
		return
	}
	if state.inSafeOutputs && hasExitedBlock(line, state.safeOutputsIndent) {
		state.inSafeOutputs = false
		state.safeOutputsChildIndent = ""
		state.activeHandler = ""
		state.activeHandlerIndent = ""
		state.handlerChildIndent = ""
	}
	if state.activeHandler != "" && hasExitedBlock(line, state.activeHandlerIndent) {
		state.activeHandler = ""
		state.activeHandlerIndent = ""
		state.handlerChildIndent = ""
	}
}

func renameSafeOutputTitlePrefixConstraintsEnterSafeOutputs(indent string, state *renameSafeOutputTitlePrefixConstraintsState) {
	state.inSafeOutputs = true
	state.safeOutputsIndent = indent
	state.safeOutputsChildIndent = ""
	state.activeHandler = ""
	state.activeHandlerIndent = ""
	state.handlerChildIndent = ""
}

func renameSafeOutputTitlePrefixConstraintsEnterHandler(trimmed, indent string, handlersToRename map[string]struct {
}, state *renameSafeOutputTitlePrefixConstraintsState) bool {
	if !state.inSafeOutputs || !isDescendant(indent, state.safeOutputsIndent) ||
		!strings.HasSuffix(trimmed, ":") || strings.HasPrefix(trimmed, "#") {
		return false
	}
	if state.activeHandler != "" && state.handlerChildIndent == "" && isDescendant(indent, state.activeHandlerIndent) {
		state.handlerChildIndent = indent
	}
	if state.safeOutputsChildIndent == "" {
		state.safeOutputsChildIndent = indent
	}
	if indent != state.safeOutputsChildIndent {
		return true
	}
	key := strings.TrimSuffix(trimmed, ":")
	if setutil.Contains(handlersToRename, key) {
		state.activeHandler = key
		state.activeHandlerIndent = indent
		state.handlerChildIndent = ""
	} else {
		state.activeHandler = ""
		state.activeHandlerIndent = ""
		state.handlerChildIndent = ""
	}
	return true
}

func renameSafeOutputTitlePrefixConstraintsSetChildIndent(trimmed, indent string, state *renameSafeOutputTitlePrefixConstraintsState) {
	if state.activeHandler != "" && state.handlerChildIndent == "" && isDescendant(indent, state.activeHandlerIndent) &&
		trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "- ") {
		state.handlerChildIndent = indent
	}
}

func renameSafeOutputTitlePrefixConstraintsReplaceLine(line, trimmed, indent string, lineIndex int, state *renameSafeOutputTitlePrefixConstraintsState) (string, bool) {
	if state.activeHandler != "" && indent == state.handlerChildIndent && strings.HasPrefix(trimmed, "title-prefix:") {
		newLine, replaced := findAndReplaceInLine(line, "title-prefix", "required-title-prefix")
		if replaced {
			safeOutputRequireTitlePrefixCodemodLog.Printf("Renamed title-prefix in safe-outputs.%s on line %d", state.activeHandler, lineIndex+1)
		}
		return newLine, replaced
	}
	if state.activeHandler == "push-to-pull-request-branch" && indent == state.handlerChildIndent && strings.HasPrefix(trimmed, "labels:") {
		newLine, replaced := findAndReplaceInLine(line, "labels", "required-labels")
		if replaced {
			safeOutputRequireTitlePrefixCodemodLog.Printf("Renamed labels in safe-outputs.%s on line %d", state.activeHandler, lineIndex+1)
		}
		return newLine, replaced
	}
	return "", false
}
