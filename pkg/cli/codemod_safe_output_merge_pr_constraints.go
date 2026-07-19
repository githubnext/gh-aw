package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputMergePRConstraintsCodemodLog = logger.New("cli:codemod_safe_output_merge_pr_constraints")

func getSafeOutputMergePRConstraintsCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-merge-pr-constraints",
		Name:         "Rename deprecated merge-pull-request constraint fields",
		Description:  "Renames allowed-labels to required-labels in safe-outputs.merge-pull-request.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !mergePRConstraintsNeedsMigration(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return renameMergePRConstraints(lines)
			})
			if applied {
				safeOutputMergePRConstraintsCodemodLog.Print("Renamed deprecated merge-pull-request constraint keys to required-labels")
			}
			return newContent, applied, err
		},
	}
}

func mergePRConstraintsNeedsMigration(frontmatter map[string]any) bool {
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return false
	}
	handlerAny, ok := safeOutputsMap["merge-pull-request"]
	if !ok {
		return false
	}
	handlerMap, ok := handlerAny.(map[string]any)
	if !ok {
		return false
	}

	if _, hasNew := handlerMap["required-labels"]; !hasNew {
		if _, hasOld := handlerMap["allowed-labels"]; hasOld {
			return true
		}
	}
	return false
}

func renameMergePRConstraints(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	state := renameMergePRConstraintsState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		renameMergePRConstraintsUpdateBlockState(line, trimmed, &state)

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			renameMergePRConstraintsEnterSafeOutputs(indent, &state)
			result = append(result, line)
			continue
		}

		if renameMergePRConstraintsEnterHandler(trimmed, indent, &state) {
			result = append(result, line)
			continue
		}

		renameMergePRConstraintsSetChildIndent(trimmed, indent, &state)

		if newLine, replaced := renameMergePRConstraintsReplaceLine(line, trimmed, indent, i, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameMergePRConstraintsState struct {
	inSafeOutputs          bool
	safeOutputsIndent      string
	safeOutputsChildIndent string
	inMergePR              bool
	mergePRIndent          string
	mergePRChildIndent     string
}

func renameMergePRConstraintsUpdateBlockState(line, trimmed string, state *renameMergePRConstraintsState) {
	if strings.HasPrefix(trimmed, "#") {
		return
	}
	if state.inSafeOutputs && hasExitedBlock(line, state.safeOutputsIndent) {
		state.inSafeOutputs = false
		state.safeOutputsChildIndent = ""
		state.inMergePR = false
		state.mergePRIndent = ""
		state.mergePRChildIndent = ""
	}
	if state.inMergePR && hasExitedBlock(line, state.mergePRIndent) {
		state.inMergePR = false
		state.mergePRIndent = ""
		state.mergePRChildIndent = ""
	}
}

func renameMergePRConstraintsEnterSafeOutputs(indent string, state *renameMergePRConstraintsState) {
	state.inSafeOutputs = true
	state.safeOutputsIndent = indent
	state.safeOutputsChildIndent = ""
	state.inMergePR = false
	state.mergePRIndent = ""
	state.mergePRChildIndent = ""
}

func renameMergePRConstraintsEnterHandler(trimmed, indent string, state *renameMergePRConstraintsState) bool {
	if !state.inSafeOutputs || !isDescendant(indent, state.safeOutputsIndent) ||
		!strings.HasSuffix(trimmed, ":") || strings.HasPrefix(trimmed, "#") {
		return false
	}
	if state.safeOutputsChildIndent == "" {
		state.safeOutputsChildIndent = indent
	}
	if indent == state.safeOutputsChildIndent {
		key := strings.TrimSuffix(trimmed, ":")
		state.inMergePR = key == "merge-pull-request"
		if state.inMergePR {
			state.mergePRIndent = indent
			state.mergePRChildIndent = ""
		} else {
			state.mergePRIndent = ""
			state.mergePRChildIndent = ""
		}
	}
	return true
}

func renameMergePRConstraintsSetChildIndent(trimmed, indent string, state *renameMergePRConstraintsState) {
	if state.inMergePR && state.mergePRChildIndent == "" && isDescendant(indent, state.mergePRIndent) &&
		trimmed != "" && !strings.HasPrefix(trimmed, "#") {
		state.mergePRChildIndent = indent
	}
}

func renameMergePRConstraintsReplaceLine(line, trimmed, indent string, lineIndex int, state *renameMergePRConstraintsState) (string, bool) {
	if !state.inMergePR || indent != state.mergePRChildIndent || !strings.HasPrefix(trimmed, "allowed-labels:") {
		return "", false
	}
	newLine, replaced := findAndReplaceInLine(line, "allowed-labels", "required-labels")
	if replaced {
		safeOutputMergePRConstraintsCodemodLog.Printf("Renamed allowed-labels to required-labels in safe-outputs.merge-pull-request on line %d", lineIndex+1)
	}
	return newLine, replaced
}
