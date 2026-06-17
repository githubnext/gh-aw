package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var effectiveTokensToAICreditsCodemodLog = logger.New("cli:codemod_effective_tokens_to_ai_credits")

const effectiveTokensPerAICredit = 10000

// getEffectiveTokensToAICreditsCodemod migrates obsolete ET-based budget fields
// to AI Credits equivalents:
//   - max-effective-tokens -> max-ai-credits
//   - max-daily-effective-tokens -> max-daily-ai-credits
//
// Only numeric values are migrated (and normalized to canonical base-10 strings).
// Expression values are intentionally not converted.
func getEffectiveTokensToAICreditsCodemod() Codemod {
	return Codemod{
		ID:           "effective-tokens-to-ai-credits",
		Name:         "Migrate obsolete effective-token limits to AI credits",
		Description:  "Migrates obsolete 'max-effective-tokens' and 'max-daily-effective-tokens' to AI Credits equivalents, normalizing numeric values and skipping expressions.",
		IntroducedIn: "1.0.47",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if frontmatter == nil {
				return content, false, nil
			}

			_, hasMaxAICredits := frontmatter["max-ai-credits"]
			_, hasMaxDailyAICredits := frontmatter["max-daily-ai-credits"]

			var maxAICreditsNormalized string
			migrateMaxAICredits := false
			if !hasMaxAICredits {
				if raw, exists := frontmatter["max-effective-tokens"]; exists {
					if normalized, ok := normalizeLegacyBudgetValue(raw, true); ok {
						maxAICreditsNormalized = normalized
						migrateMaxAICredits = true
					}
				}
			}

			var maxDailyAICreditsNormalized string
			migrateMaxDailyAICredits := false
			if !hasMaxDailyAICredits {
				if raw, exists := frontmatter["max-daily-effective-tokens"]; exists {
					if normalized, ok := normalizeLegacyBudgetValue(raw, true); ok {
						maxDailyAICreditsNormalized = normalized
						migrateMaxDailyAICredits = true
					}
				}
			}

			if !migrateMaxAICredits && !migrateMaxDailyAICredits {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				modified := false
				result := make([]string, 0, len(lines))

				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if isTopLevelKey(line) {
						if migrateMaxAICredits && strings.HasPrefix(trimmed, "max-effective-tokens:") {
							result = append(result, rewriteTopLevelScalarLine(line, "max-ai-credits", maxAICreditsNormalized))
							modified = true
							continue
						}
						if migrateMaxDailyAICredits && strings.HasPrefix(trimmed, "max-daily-effective-tokens:") {
							result = append(result, rewriteTopLevelScalarLine(line, "max-daily-ai-credits", maxDailyAICreditsNormalized))
							modified = true
							continue
						}
					}
					result = append(result, line)
				}

				return result, modified
			})
			if applied {
				effectiveTokensToAICreditsCodemodLog.Printf(
					"Migrated effective-token legacy fields (max-ai-credits=%t max-daily-ai-credits=%t)",
					migrateMaxAICredits,
					migrateMaxDailyAICredits,
				)
			}
			return newContent, applied, err
		},
	}
}

func normalizeLegacyBudgetValue(raw any, allowNegativeOne bool) (string, bool) {
	if value, ok := typeutil.ParseIntValue(raw); ok {
		if value > 0 {
			return convertEffectiveTokensToAICredits(value)
		}
		if allowNegativeOne && value == -1 {
			return "-1", true
		}
		return "", false
	}

	rawStr, ok := raw.(string)
	if !ok {
		return "", false
	}

	trimmed := strings.TrimSpace(rawStr)
	if trimmed == "" || isExpressionValue(trimmed) {
		return "", false
	}
	if allowNegativeOne && trimmed == "-1" {
		return "-1", true
	}
	if normalized, ok := typeutil.NormalizeInt64KMSuffix(trimmed); ok {
		parsed, err := strconv.Atoi(normalized)
		if err != nil {
			return "", false
		}
		return convertEffectiveTokensToAICredits(parsed)
	}

	return "", false
}

func convertEffectiveTokensToAICredits(effectiveTokens int) (string, bool) {
	aiCredits := effectiveTokens / effectiveTokensPerAICredit
	if aiCredits == 0 {
		return "", false
	}
	return strconv.Itoa(aiCredits), true
}

func isExpressionValue(value string) bool {
	return strings.Contains(value, "${{") && strings.Contains(value, "}}")
}

func rewriteTopLevelScalarLine(line, newKey, normalizedValue string) string {
	indent := getIndentation(line)
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return line
	}

	remainder := parts[1]
	comment := ""
	if idx := strings.Index(remainder, "#"); idx >= 0 {
		comment = remainder[idx:]
		remainder = remainder[:idx]
	}

	if comment == "" {
		return fmt.Sprintf("%s%s: %s", indent, newKey, normalizedValue)
	}

	trailingWSBeforeComment := remainder[len(strings.TrimRight(remainder, " \t")):]
	return fmt.Sprintf("%s%s: %s%s%s", indent, newKey, normalizedValue, trailingWSBeforeComment, comment)
}
