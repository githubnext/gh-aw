package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var discussionTriggerCategoriesCodemodLog = logger.New("cli:codemod_discussion_trigger_categories")

type lowercaseDiscussionTriggerTypesInLinesState struct {
	inOn           bool
	onIndent       string
	currentTrigger string
	triggerIndent  string
	inTypes        bool
	typesIndent    string
}

// getDiscussionTriggerCategoriesLowercaseCodemod lowercases discussion trigger category values
// so source matches compile-time normalized values.
func getDiscussionTriggerCategoriesLowercaseCodemod() Codemod {
	return Codemod{
		ID:           "discussion-trigger-categories-lowercase",
		Name:         "Lowercase discussion trigger category values",
		Description:  "Lowercases mixed-case category strings in on.discussion.types and on.discussion_comment.types.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			onValue, hasOn := frontmatter["on"]
			if !hasOn {
				return content, false, nil
			}

			onMap, ok := onValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			if !hasMixedCaseDiscussionTypeValues(onMap) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, lowercaseDiscussionTriggerTypesInLines)
			if applied {
				discussionTriggerCategoriesCodemodLog.Print("Applied discussion trigger category lowercase codemod")
			}
			return newContent, applied, err
		},
	}
}

func hasMixedCaseDiscussionTypeValues(onMap map[string]any) bool {
	for _, trigger := range []string{"discussion", "discussion_comment"} {
		triggerValue, hasTrigger := onMap[trigger]
		if !hasTrigger {
			continue
		}
		triggerMap, ok := triggerValue.(map[string]any)
		if !ok {
			continue
		}

		typesValue, hasTypes := triggerMap["types"]
		if !hasTypes {
			continue
		}
		types, ok := typesValue.([]any)
		if !ok {
			continue
		}

		for _, typeValue := range types {
			typeString, ok := typeValue.(string)
			if !ok {
				continue
			}
			if typeString != strings.ToLower(typeString) {
				return true
			}
		}
	}
	return false
}

func lowercaseDiscussionTriggerTypesInLines(lines []string) ([]string, bool) {
	result := make([]string, len(lines))
	copy(result, lines)

	var modified bool
	var state lowercaseDiscussionTriggerTypesInLinesState

	for i, line := range result {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if lowercaseDiscussionTriggerTypesInLinesStartOn(line, trimmed, indent, &state) {
			continue
		}

		lowercaseDiscussionTriggerTypesInLinesUpdateExits(line, trimmed, indent, &state)
		if !state.inOn || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if lowercaseDiscussionTriggerTypesInLinesStartTrigger(trimmed, indent, &state) {
			continue
		}
		if state.currentTrigger == "" {
			continue
		}

		updatedLine, changed, handled := lowercaseDiscussionTriggerTypesInLinesUpdateTypeLine(line, trimmed, indent, &state)
		if changed {
			result[i] = updatedLine
			modified = true
		}
		if handled {
			continue
		}

		if state.inTypes && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			updatedLine, changed := lowercaseYAMLListItemLine(line)
			if changed {
				result[i] = updatedLine
				modified = true
			}
		}
	}

	return result, modified
}

func lowercaseDiscussionTriggerTypesInLinesStartOn(line string, trimmed string, indent string, state *lowercaseDiscussionTriggerTypesInLinesState) bool {
	if isTopLevelKey(line) && isOnBlockStartLine(trimmed) {
		state.inOn = true
		state.onIndent = indent
		state.currentTrigger = ""
		state.inTypes = false
		return true
	}
	return false
}

func lowercaseDiscussionTriggerTypesInLinesUpdateExits(line string, trimmed string, indent string, state *lowercaseDiscussionTriggerTypesInLinesState) {
	if state.inOn && isTopLevelKey(line) && len(indent) <= len(state.onIndent) && !isOnBlockStartLine(trimmed) {
		state.inOn = false
		state.currentTrigger = ""
		state.inTypes = false
	}
	if state.currentTrigger != "" && len(indent) <= len(state.triggerIndent) {
		state.currentTrigger = ""
		state.inTypes = false
	}
	if state.inTypes && len(indent) <= len(state.typesIndent) {
		state.inTypes = false
	}
}

func lowercaseDiscussionTriggerTypesInLinesStartTrigger(trimmed string, indent string, state *lowercaseDiscussionTriggerTypesInLinesState) bool {
	if len(indent) <= len(state.onIndent) {
		return false
	}
	trigger, isTrigger := getDiscussionTriggerFromLine(trimmed)
	if !isTrigger {
		return false
	}
	state.currentTrigger = trigger
	state.triggerIndent = indent
	state.inTypes = false
	return true
}

func lowercaseDiscussionTriggerTypesInLinesUpdateTypeLine(line string, trimmed string, indent string, state *lowercaseDiscussionTriggerTypesInLinesState) (string, bool, bool) {
	if !strings.HasPrefix(trimmed, "types:") {
		return line, false, false
	}
	state.typesIndent = indent
	afterColon := strings.TrimSpace(strings.TrimPrefix(trimmed, "types:"))
	if strings.HasPrefix(afterColon, "[") && strings.HasSuffix(afterColon, "]") {
		updatedLine, changed := lowercaseInlineTypesArrayLine(line)
		state.inTypes = false
		return updatedLine, changed, true
	}
	if afterColon == "" {
		state.inTypes = true
	}
	return line, false, true
}

func isOnBlockStartLine(trimmed string) bool {
	key, isBlockMappingKey := getBlockMappingKey(trimmed)
	return isBlockMappingKey && key == "on"
}

func getDiscussionTriggerFromLine(trimmed string) (string, bool) {
	key, isBlockMappingKey := getBlockMappingKey(trimmed)
	if !isBlockMappingKey {
		return "", false
	}
	return key, key == "discussion" || key == "discussion_comment"
}

func getBlockMappingKey(trimmed string) (string, bool) {
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "- ") {
		return "", false
	}

	keyPart, valuePart, hasColon := strings.Cut(trimmed, ":")
	if !hasColon {
		return "", false
	}

	key := strings.TrimSpace(keyPart)
	if key == "" {
		return "", false
	}

	valuePart = strings.TrimSpace(valuePart)
	if valuePart != "" && !strings.HasPrefix(valuePart, "#") {
		return "", false
	}

	if len(key) >= 2 {
		if (strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"")) ||
			(strings.HasPrefix(key, "'") && strings.HasSuffix(key, "'")) {
			key = key[1 : len(key)-1]
		}
	}

	return key, key != ""
}

func lowercaseInlineTypesArrayLine(line string) (string, bool) {
	commentIndex := strings.Index(line, "#")
	valuePart := line
	commentPart := ""
	if commentIndex >= 0 {
		valuePart = strings.TrimRight(line[:commentIndex], " ")
		commentPart = line[commentIndex:]
	}

	colonIndex := strings.Index(valuePart, ":")
	if colonIndex < 0 {
		return line, false
	}

	listValue := strings.TrimSpace(valuePart[colonIndex+1:])
	if !strings.HasPrefix(listValue, "[") || !strings.HasSuffix(listValue, "]") {
		return line, false
	}

	inner := strings.TrimSpace(listValue[1 : len(listValue)-1])
	if inner == "" {
		return line, false
	}

	parts := strings.Split(inner, ",")
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		unquoted := strings.Trim(trimmed, `"'`)
		lower := strings.ToLower(unquoted)

		var updatedValue string
		if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
			updatedValue = `"` + lower + `"`
		} else if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") {
			updatedValue = "'" + lower + "'"
		} else {
			updatedValue = lower
		}

		leftPaddingLen := len(part) - len(strings.TrimLeft(part, " \t"))
		rightPaddingLen := len(part) - len(strings.TrimRight(part, " \t"))
		leftPadding := part[:leftPaddingLen]
		rightPadding := part[len(part)-rightPaddingLen:]
		parts[i] = leftPadding + updatedValue + rightPadding
	}

	updatedInner := strings.Join(parts, ",")
	if updatedInner == inner {
		return line, false
	}

	prefix := valuePart[:colonIndex+1]
	updated := prefix + " [" + updatedInner + "]"
	if commentPart != "" {
		updated += " " + strings.TrimSpace(commentPart)
	}
	return updated, true
}

func lowercaseYAMLListItemLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "- ") {
		return line, false
	}

	lineWithoutIndent := strings.TrimLeft(line, " \t")
	_, valueAfterDash, hasDash := strings.Cut(lineWithoutIndent, "- ")
	if !hasDash {
		return line, false
	}

	indent := getIndentation(line)
	valueWithComment := strings.TrimSpace(valueAfterDash)

	commentIndex := strings.Index(valueWithComment, "#")
	valuePart := valueWithComment
	commentPart := ""
	if commentIndex >= 0 {
		valuePart = strings.TrimSpace(valueWithComment[:commentIndex])
		commentPart = strings.TrimSpace(valueWithComment[commentIndex:])
	}

	if strings.Contains(valuePart, "${{") {
		return line, false
	}

	unquoted := strings.Trim(valuePart, `"'`)
	lower := strings.ToLower(unquoted)
	if unquoted == lower {
		return line, false
	}

	var updatedValue string
	if strings.HasPrefix(valuePart, "\"") && strings.HasSuffix(valuePart, "\"") {
		updatedValue = `"` + lower + `"`
	} else if strings.HasPrefix(valuePart, "'") && strings.HasSuffix(valuePart, "'") {
		updatedValue = "'" + lower + "'"
	} else {
		updatedValue = lower
	}

	updated := indent + "- " + updatedValue
	if commentPart != "" {
		updated += " " + commentPart
	}
	return updated, true
}
