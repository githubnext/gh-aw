package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var discussionFlagCodemodLog = logger.New("cli:codemod_discussion_flag")

type getDiscussionFlagRemovalCodemodTransformState struct {
	inSafeOutputsBlock bool
	safeOutputsIndent  string
	inAddCommentBlock  bool
	addCommentIndent   string
	inDiscussionField  bool
}

// getDiscussionFlagRemovalCodemod creates a codemod for converting the deprecated discussion field in add-comment
func getDiscussionFlagRemovalCodemod() Codemod {
	return Codemod{
		ID:           "add-comment-discussion-removal",
		Name:         "Remove deprecated add-comment.discussion field",
		Description:  "Removes the deprecated 'safe-outputs.add-comment.discussion' field. Discussion targeting is now automatic based on context. Use 'discussions: false' to opt out of discussions:write permission.",
		IntroducedIn: "0.3.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return getDiscussionFlagRemovalCodemodApply(content, frontmatter)
		},
	}
}

func getDiscussionFlagRemovalCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	// Check if safe-outputs exists
	safeOutputsValue, hasSafeOutputs := frontmatter["safe-outputs"]
	if !hasSafeOutputs {
		return content, false, nil
	}

	safeOutputsMap, ok := safeOutputsValue.(map[string]any)
	if !ok {
		return content, false, nil
	}

	// Check if add-comment exists in safe-outputs
	addCommentValue, hasAddComment := safeOutputsMap["add-comment"]
	if !hasAddComment {
		return content, false, nil
	}

	addCommentMap, ok := addCommentValue.(map[string]any)
	if !ok {
		return content, false, nil
	}

	// Check if discussion field exists in add-comment
	_, hasDiscussion := addCommentMap["discussion"]
	if !hasDiscussion {
		return content, false, nil
	}

	newContent, applied, err := applyFrontmatterLineTransform(content, getDiscussionFlagRemovalCodemodTransformLines)
	if applied {
		discussionFlagCodemodLog.Print("Applied add-comment.discussion removal")
	}
	return newContent, applied, err
}

func getDiscussionFlagRemovalCodemodTransformLines(lines []string) ([]string, bool) {
	var result []string
	var modified bool
	var state getDiscussionFlagRemovalCodemodTransformState
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if getDiscussionFlagRemovalCodemodStartSafeOutputs(line, trimmedLine, &state, &result) {
			continue
		}
		getDiscussionFlagRemovalCodemodUpdateBlockExits(line, trimmedLine, &state)
		if getDiscussionFlagRemovalCodemodStartAddComment(line, trimmedLine, &state, &result) {
			continue
		}
		if state.inAddCommentBlock && strings.HasPrefix(trimmedLine, "discussion:") {
			modified = true
			state.inDiscussionField = true
			discussionFlagCodemodLog.Printf("Removed safe-outputs.add-comment.discussion on line %d", i+1)
			continue
		}
		if getDiscussionFlagRemovalCodemodSkipDiscussionField(line, trimmedLine, i, &state) {
			continue
		}
		result = append(result, line)
	}
	return result, modified
}

func getDiscussionFlagRemovalCodemodStartSafeOutputs(line string, trimmedLine string, state *getDiscussionFlagRemovalCodemodTransformState, result *[]string) bool {
	if strings.HasPrefix(trimmedLine, "safe-outputs:") {
		state.inSafeOutputsBlock = true
		state.safeOutputsIndent = getIndentation(line)
		*result = append(*result, line)
		return true
	}
	return false
}

func getDiscussionFlagRemovalCodemodStartAddComment(line string, trimmedLine string, state *getDiscussionFlagRemovalCodemodTransformState, result *[]string) bool {
	if state.inSafeOutputsBlock && strings.HasPrefix(trimmedLine, "add-comment:") {
		state.inAddCommentBlock = true
		state.addCommentIndent = getIndentation(line)
		*result = append(*result, line)
		return true
	}
	return false
}

func getDiscussionFlagRemovalCodemodUpdateBlockExits(line string, trimmedLine string, state *getDiscussionFlagRemovalCodemodTransformState) {
	if state.inSafeOutputsBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
		if hasExitedBlock(line, state.safeOutputsIndent) {
			state.inSafeOutputsBlock = false
			state.inAddCommentBlock = false
		}
	}
	if state.inAddCommentBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
		if hasExitedBlock(line, state.addCommentIndent) {
			state.inAddCommentBlock = false
		}
	}
}

func getDiscussionFlagRemovalCodemodSkipDiscussionField(line string, trimmedLine string, index int, state *getDiscussionFlagRemovalCodemodTransformState) bool {
	if !state.inDiscussionField {
		return false
	}
	if trimmedLine == "" {
		return true
	}

	currentIndent := getIndentation(line)
	discussionIndent := state.addCommentIndent + "  "
	if len(currentIndent) > len(discussionIndent) {
		discussionFlagCodemodLog.Printf("Removed nested discussion property on line %d: %s", index+1, trimmedLine)
		return true
	}
	state.inDiscussionField = false
	return false
}
