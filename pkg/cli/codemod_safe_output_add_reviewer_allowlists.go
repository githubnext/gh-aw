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

	state := renameAddReviewerAllowlistsState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		renameAddReviewerAllowlistsUpdateBlockState(line, trimmed, &state)

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			renameAddReviewerAllowlistsEnterSafeOutputs(indent, &state)
			result = append(result, line)
			continue
		}

		if renameAddReviewerAllowlistsEnterHandler(trimmed, indent, &state) {
			result = append(result, line)
			continue
		}

		renameAddReviewerAllowlistsSetChildIndent(trimmed, indent, &state)

		if newLine, replaced := renameAddReviewerAllowlistsReplaceLine(line, trimmed, indent, i, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameAddReviewerAllowlistsState struct {
	inSafeOutputs          bool
	safeOutputsIndent      string
	safeOutputsChildIndent string
	inAddReviewer          bool
	addReviewerIndent      string
	addReviewerChildIndent string
}

func renameAddReviewerAllowlistsUpdateBlockState(line, trimmed string, state *renameAddReviewerAllowlistsState) {
	if strings.HasPrefix(trimmed, "#") {
		return
	}
	if state.inSafeOutputs && hasExitedBlock(line, state.safeOutputsIndent) {
		state.inSafeOutputs = false
		state.safeOutputsChildIndent = ""
		state.inAddReviewer = false
		state.addReviewerIndent = ""
		state.addReviewerChildIndent = ""
	}
	if state.inAddReviewer && hasExitedBlock(line, state.addReviewerIndent) {
		state.inAddReviewer = false
		state.addReviewerIndent = ""
		state.addReviewerChildIndent = ""
	}
}

func renameAddReviewerAllowlistsEnterSafeOutputs(indent string, state *renameAddReviewerAllowlistsState) {
	state.inSafeOutputs = true
	state.safeOutputsIndent = indent
	state.safeOutputsChildIndent = ""
	state.inAddReviewer = false
	state.addReviewerIndent = ""
	state.addReviewerChildIndent = ""
}

func renameAddReviewerAllowlistsEnterHandler(trimmed, indent string, state *renameAddReviewerAllowlistsState) bool {
	if !state.inSafeOutputs || !isDescendant(indent, state.safeOutputsIndent) ||
		!strings.HasSuffix(trimmed, ":") || strings.HasPrefix(trimmed, "#") {
		return false
	}
	if state.safeOutputsChildIndent == "" {
		state.safeOutputsChildIndent = indent
	}
	if indent == state.safeOutputsChildIndent {
		key := strings.TrimSuffix(trimmed, ":")
		state.inAddReviewer = key == "add-reviewer"
		if state.inAddReviewer {
			state.addReviewerIndent = indent
			state.addReviewerChildIndent = ""
		} else {
			state.addReviewerIndent = ""
			state.addReviewerChildIndent = ""
		}
	}
	return true
}

func renameAddReviewerAllowlistsSetChildIndent(trimmed, indent string, state *renameAddReviewerAllowlistsState) {
	if state.inAddReviewer && state.addReviewerChildIndent == "" && isDescendant(indent, state.addReviewerIndent) &&
		trimmed != "" && !strings.HasPrefix(trimmed, "#") {
		state.addReviewerChildIndent = indent
	}
}

func renameAddReviewerAllowlistsReplaceLine(line, trimmed, indent string, lineIndex int, state *renameAddReviewerAllowlistsState) (string, bool) {
	if !state.inAddReviewer || indent != state.addReviewerChildIndent {
		return "", false
	}
	// Rename "reviewers:" but not "team-reviewers:" on the same pass — check full prefix
	if strings.HasPrefix(trimmed, "reviewers:") && !strings.HasPrefix(trimmed, "team-reviewers:") {
		newLine, replaced := findAndReplaceInLine(line, "reviewers", "allowed-reviewers")
		if replaced {
			safeOutputAddReviewerAllowlistsCodemodLog.Printf("Renamed reviewers to allowed-reviewers in safe-outputs.add-reviewer on line %d", lineIndex+1)
		}
		return newLine, replaced
	}
	if strings.HasPrefix(trimmed, "team-reviewers:") {
		newLine, replaced := findAndReplaceInLine(line, "team-reviewers", "allowed-team-reviewers")
		if replaced {
			safeOutputAddReviewerAllowlistsCodemodLog.Printf("Renamed team-reviewers to allowed-team-reviewers in safe-outputs.add-reviewer on line %d", lineIndex+1)
		}
		return newLine, replaced
	}
	return "", false
}
