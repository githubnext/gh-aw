package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputAddReviewerAllowlistsCodemodLog = logger.New("cli:codemod_safe_output_add_reviewer_allowlists")

func getSafeOutputAddReviewerAllowlistsCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-add-reviewer-allowlists",
		Name:         "Rename deprecated add-reviewer allowlist fields",
		Description:  "Renames reviewers to allowed-reviewers and team-reviewers to allowed-team-reviewers in safe-outputs.add-reviewer.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !addReviewerAllowlistsNeedsMigration(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return renameAddReviewerAllowlists(lines)
			})
			if applied {
				safeOutputAddReviewerAllowlistsCodemodLog.Print("Renamed deprecated add-reviewer allowlist keys to allowed-reviewers/allowed-team-reviewers")
			}
			return newContent, applied, err
		},
	}
}

func addReviewerAllowlistsNeedsMigration(frontmatter map[string]any) bool {
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return false
	}
	handlerAny, ok := safeOutputsMap["add-reviewer"]
	if !ok {
		return false
	}
	handlerMap, ok := handlerAny.(map[string]any)
	if !ok {
		return false
	}

	if _, hasNew := handlerMap["allowed-reviewers"]; !hasNew {
		if _, hasOld := handlerMap["reviewers"]; hasOld {
			return true
		}
	}
	if _, hasNew := handlerMap["allowed-team-reviewers"]; !hasNew {
		if _, hasOld := handlerMap["team-reviewers"]; hasOld {
			return true
		}
	}
	return false
}

func renameAddReviewerAllowlists(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""
	inAddReviewer := false
	addReviewerIndent := ""
	addReviewerChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if !strings.HasPrefix(trimmed, "#") {
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsChildIndent = ""
				inAddReviewer = false
				addReviewerIndent = ""
				addReviewerChildIndent = ""
			}
			if inAddReviewer && hasExitedBlock(line, addReviewerIndent) {
				inAddReviewer = false
				addReviewerIndent = ""
				addReviewerChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			inAddReviewer = false
			addReviewerIndent = ""
			addReviewerChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			if safeOutputsChildIndent == "" {
				safeOutputsChildIndent = indent
			}
			if indent == safeOutputsChildIndent {
				key := strings.TrimSuffix(trimmed, ":")
				if key == "add-reviewer" {
					inAddReviewer = true
					addReviewerIndent = indent
					addReviewerChildIndent = ""
				} else {
					inAddReviewer = false
					addReviewerIndent = ""
					addReviewerChildIndent = ""
				}
			}
			result = append(result, line)
			continue
		}

		if inAddReviewer && addReviewerChildIndent == "" && isDescendant(indent, addReviewerIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			addReviewerChildIndent = indent
		}

		// Rename "reviewers:" but not "team-reviewers:" on the same pass — check full prefix
		if inAddReviewer && indent == addReviewerChildIndent {
			if strings.HasPrefix(trimmed, "reviewers:") && !strings.HasPrefix(trimmed, "team-reviewers:") {
				newLine, replaced := findAndReplaceInLine(line, "reviewers", "allowed-reviewers")
				if replaced {
					result = append(result, newLine)
					modified = true
					safeOutputAddReviewerAllowlistsCodemodLog.Printf("Renamed reviewers to allowed-reviewers in safe-outputs.add-reviewer on line %d", i+1)
					continue
				}
			}
			if strings.HasPrefix(trimmed, "team-reviewers:") {
				newLine, replaced := findAndReplaceInLine(line, "team-reviewers", "allowed-team-reviewers")
				if replaced {
					result = append(result, newLine)
					modified = true
					safeOutputAddReviewerAllowlistsCodemodLog.Printf("Renamed team-reviewers to allowed-team-reviewers in safe-outputs.add-reviewer on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
