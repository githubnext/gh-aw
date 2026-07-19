package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var messagesEffectiveTokensSuffixToAICreditsSuffixCodemodLog = logger.New("cli:codemod_messages_effective_tokens_suffix_to_ai_credits_suffix")

const (
	effectiveTokensSuffixPlaceholder = "{effective_tokens_suffix}"
	aiCreditsSuffixPlaceholder       = "{ai_credits_suffix}"
)

func getMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod() Codemod {
	return Codemod{
		ID:           "messages-effective-tokens-suffix-to-ai-credits-suffix",
		Name:         "Migrate safe-outputs messages ET suffix placeholder to AI credits suffix",
		Description:  "Rewrites safe-outputs.messages templates from '{effective_tokens_suffix}' to '{ai_credits_suffix}' so custom message footers render AI Credits (AIC) instead of Effective Tokens (ET).",
		IntroducedIn: "1.0.48",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			messagesMap, ok := getSafeOutputsMessagesMap(frontmatter)
			if !ok || !messagesNeedsAICreditsSuffixMigration(messagesMap) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, migrateMessagesEffectiveTokensSuffixToAICreditsSuffix)
			if applied {
				messagesEffectiveTokensSuffixToAICreditsSuffixCodemodLog.Print("Migrated safe-outputs.messages placeholders from effective_tokens_suffix to ai_credits_suffix")
			}
			return newContent, applied, err
		},
	}
}

func getSafeOutputsMessagesMap(frontmatter map[string]any) (map[string]any, bool) {
	if frontmatter == nil {
		return nil, false
	}
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return nil, false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return nil, false
	}
	messagesAny, ok := safeOutputsMap["messages"]
	if !ok {
		return nil, false
	}
	messagesMap, ok := messagesAny.(map[string]any)
	return messagesMap, ok
}

func messagesNeedsAICreditsSuffixMigration(messagesMap map[string]any) bool {
	for _, value := range messagesMap {
		text, ok := value.(string)
		if !ok {
			continue
		}
		if strings.Contains(text, effectiveTokensSuffixPlaceholder) {
			return true
		}
	}
	return false
}

func migrateMessagesEffectiveTokensSuffixToAICreditsSuffix(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	state := migrateMessagesEffectiveTokensSuffixToAICreditsSuffixState{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if updated, changed, handled := migrateMessagesEffectiveTokensSuffixToAICreditsSuffixBlockScalarLine(line, trimmed, indent, &state); handled {
			if changed {
				modified = true
			}
			result = append(result, updated)
			continue
		}

		migrateMessagesEffectiveTokensSuffixToAICreditsSuffixExitBlocks(line, trimmed, &state)

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			state.inSafeOutputs = true
			state.safeOutputsIndent = indent
			state.safeOutputsChildIndent = ""
			state.inMessages = false
			state.messagesIndent = ""
			state.messagesChildIndent = ""
			result = append(result, line)
			continue
		}

		if migrateMessagesEffectiveTokensSuffixToAICreditsSuffixEnterMessages(trimmed, indent, &state) {
			result = append(result, line)
			continue
		}

		if updated, changed, handled := migrateMessagesEffectiveTokensSuffixToAICreditsSuffixMessageLine(line, trimmed, indent, &state); handled {
			if changed {
				modified = true
			}
			result = append(result, updated)
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type migrateMessagesEffectiveTokensSuffixToAICreditsSuffixState struct {
	inSafeOutputs          bool
	safeOutputsIndent      string
	safeOutputsChildIndent string
	inMessages             bool
	messagesIndent         string
	messagesChildIndent    string
	inBlockScalar          bool
	blockScalarIndent      string
}

func migrateMessagesEffectiveTokensSuffixToAICreditsSuffixBlockScalarLine(
	line string,
	trimmed string,
	indent string,
	state *migrateMessagesEffectiveTokensSuffixToAICreditsSuffixState,
) (string, bool, bool) {
	if !state.inBlockScalar {
		return line, false, false
	}
	if trimmed == "" || len(indent) > len(state.blockScalarIndent) {
		updated := strings.ReplaceAll(line, effectiveTokensSuffixPlaceholder, aiCreditsSuffixPlaceholder)
		return updated, updated != line, true
	}
	state.inBlockScalar = false
	state.blockScalarIndent = ""
	return line, false, false
}

func migrateMessagesEffectiveTokensSuffixToAICreditsSuffixExitBlocks(
	line string,
	trimmed string,
	state *migrateMessagesEffectiveTokensSuffixToAICreditsSuffixState,
) {
	if strings.HasPrefix(trimmed, "#") {
		return
	}
	if state.inMessages && hasExitedBlock(line, state.messagesIndent) {
		state.inMessages = false
		state.messagesIndent = ""
		state.messagesChildIndent = ""
	}
	if state.inSafeOutputs && hasExitedBlock(line, state.safeOutputsIndent) {
		state.inSafeOutputs = false
		state.safeOutputsIndent = ""
		state.safeOutputsChildIndent = ""
		state.inMessages = false
		state.messagesIndent = ""
		state.messagesChildIndent = ""
	}
}

func migrateMessagesEffectiveTokensSuffixToAICreditsSuffixEnterMessages(
	trimmed string,
	indent string,
	state *migrateMessagesEffectiveTokensSuffixToAICreditsSuffixState,
) bool {
	if !state.inSafeOutputs || !isDescendant(indent, state.safeOutputsIndent) || !strings.HasSuffix(trimmed, ":") || strings.HasPrefix(trimmed, "#") {
		return false
	}
	if state.safeOutputsChildIndent == "" {
		state.safeOutputsChildIndent = indent
	}
	if indent == state.safeOutputsChildIndent && trimmed == "messages:" {
		state.inMessages = true
		state.messagesIndent = indent
		state.messagesChildIndent = ""
	}
	return true
}

func migrateMessagesEffectiveTokensSuffixToAICreditsSuffixMessageLine(
	line string,
	trimmed string,
	indent string,
	state *migrateMessagesEffectiveTokensSuffixToAICreditsSuffixState,
) (string, bool, bool) {
	if !state.inMessages || !isDescendant(indent, state.messagesIndent) || trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return line, false, false
	}
	if state.messagesChildIndent == "" {
		state.messagesChildIndent = indent
	}
	if indent != state.messagesChildIndent || !strings.Contains(trimmed, ":") {
		return line, false, false
	}
	updated := strings.ReplaceAll(line, effectiveTokensSuffixPlaceholder, aiCreditsSuffixPlaceholder)
	parts := strings.SplitN(updated, ":", 2)
	if len(parts) == 2 && isBlockScalarIndicator(parts[1]) {
		state.inBlockScalar = true
		state.blockScalarIndent = indent
	}
	return updated, updated != line, true
}

func isBlockScalarIndicator(valueSegment string) bool {
	valueWithoutComment := valueSegment
	if before, _, found := strings.Cut(valueSegment, "#"); found {
		valueWithoutComment = before
	}
	trimmed := strings.TrimSpace(valueWithoutComment)
	switch trimmed {
	case "|", "|-", "|+", ">", ">-", ">+":
		return true
	default:
		return false
	}
}
