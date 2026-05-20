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

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""
	inMergePR := false
	mergePRIndent := ""
	mergePRChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if !strings.HasPrefix(trimmed, "#") {
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsChildIndent = ""
				inMergePR = false
				mergePRIndent = ""
				mergePRChildIndent = ""
			}
			if inMergePR && hasExitedBlock(line, mergePRIndent) {
				inMergePR = false
				mergePRIndent = ""
				mergePRChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			inMergePR = false
			mergePRIndent = ""
			mergePRChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			if safeOutputsChildIndent == "" {
				safeOutputsChildIndent = indent
			}
			if indent == safeOutputsChildIndent {
				key := strings.TrimSuffix(trimmed, ":")
				if key == "merge-pull-request" {
					inMergePR = true
					mergePRIndent = indent
					mergePRChildIndent = ""
				} else {
					inMergePR = false
					mergePRIndent = ""
					mergePRChildIndent = ""
				}
			}
			result = append(result, line)
			continue
		}

		if inMergePR && mergePRChildIndent == "" && isDescendant(indent, mergePRIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			mergePRChildIndent = indent
		}

		if inMergePR && indent == mergePRChildIndent && strings.HasPrefix(trimmed, "allowed-labels:") {
			newLine, replaced := findAndReplaceInLine(line, "allowed-labels", "required-labels")
			if replaced {
				result = append(result, newLine)
				modified = true
				safeOutputMergePRConstraintsCodemodLog.Printf("Renamed allowed-labels to required-labels in safe-outputs.merge-pull-request on line %d", i+1)
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}
