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

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""
	inMessages := false
	messagesIndent := ""
	messagesChildIndent := ""
	inBlockScalar := false
	blockScalarIndent := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if inBlockScalar {
			if trimmed == "" || len(indent) > len(blockScalarIndent) {
				updated := strings.ReplaceAll(line, effectiveTokensSuffixPlaceholder, aiCreditsSuffixPlaceholder)
				if updated != line {
					modified = true
				}
				result = append(result, updated)
				continue
			}
			inBlockScalar = false
			blockScalarIndent = ""
		}

		if !strings.HasPrefix(trimmed, "#") {
			if inMessages && hasExitedBlock(line, messagesIndent) {
				inMessages = false
				messagesIndent = ""
				messagesChildIndent = ""
			}
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsIndent = ""
				safeOutputsChildIndent = ""
				inMessages = false
				messagesIndent = ""
				messagesChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			inMessages = false
			messagesIndent = ""
			messagesChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			if safeOutputsChildIndent == "" {
				safeOutputsChildIndent = indent
			}
			if indent == safeOutputsChildIndent && trimmed == "messages:" {
				inMessages = true
				messagesIndent = indent
				messagesChildIndent = ""
			}
			result = append(result, line)
			continue
		}

		if inMessages && isDescendant(indent, messagesIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if messagesChildIndent == "" {
				messagesChildIndent = indent
			}
			if indent == messagesChildIndent && strings.Contains(trimmed, ":") {
				updated := strings.ReplaceAll(line, effectiveTokensSuffixPlaceholder, aiCreditsSuffixPlaceholder)
				if updated != line {
					modified = true
				}
				result = append(result, updated)

				parts := strings.SplitN(updated, ":", 2)
				if len(parts) == 2 && isBlockScalarIndicator(parts[1]) {
					inBlockScalar = true
					blockScalarIndent = indent
				}
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
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
