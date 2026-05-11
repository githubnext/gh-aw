package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var checkoutPersistCredentialsFalseCodemodLog = logger.New("cli:codemod_checkout_persist_credentials_false")

// getCheckoutPersistCredentialsFalseCodemod ensures checkout steps set with.persist-credentials: false.
func getCheckoutPersistCredentialsFalseCodemod() Codemod {
	return Codemod{
		ID:           "checkout-persist-credentials-false",
		Name:         "Add persist-credentials: false to checkout steps",
		Description:  "Ensures actions/checkout steps set with.persist-credentials: false in steps-like sections for strict-mode safety.",
		IntroducedIn: "1.0.44",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			sections := []string{"pre-steps", "steps", "post-steps", "pre-agent-steps"}
			hasTargetSection := false
			for _, section := range sections {
				if _, ok := frontmatter[section]; ok {
					hasTargetSection = true
					break
				}
			}
			if !hasTargetSection {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				modified := false
				current := lines
				for _, section := range sections {
					var sectionChanged bool
					current, sectionChanged = transformSectionCheckoutPersistCredentials(current, section)
					modified = modified || sectionChanged
				}
				return current, modified
			})
			if applied {
				checkoutPersistCredentialsFalseCodemodLog.Print("Added persist-credentials: false to actions/checkout step with blocks")
			}
			return newContent, applied, err
		},
	}
}

func transformSectionCheckoutPersistCredentials(lines []string, sectionName string) ([]string, bool) {
	sectionStart := -1
	sectionIndent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, sectionName+":") {
			sectionStart = i
			sectionIndent = getIndentation(line)
			break
		}
	}
	if sectionStart == -1 {
		return lines, false
	}

	sectionEnd := len(lines) - 1
	for i := sectionStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if len(getIndentation(lines[i])) <= len(sectionIndent) {
			sectionEnd = i - 1
			break
		}
	}

	sectionLines := lines[sectionStart : sectionEnd+1]
	updatedSection, changed := transformCheckoutWithinSection(sectionLines, sectionIndent)
	if !changed {
		return lines, false
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:sectionStart]...)
	result = append(result, updatedSection...)
	result = append(result, lines[sectionEnd+1:]...)
	return result, true
}

func transformCheckoutWithinSection(sectionLines []string, sectionIndent string) ([]string, bool) {
	result := make([]string, 0, len(sectionLines))
	modified := false

	for i := 0; i < len(sectionLines); {
		line := sectionLines[i]
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if strings.HasPrefix(trimmed, "- ") && len(indent) > len(sectionIndent) {
			stepStart := i
			stepIndent := indent
			stepEnd := len(sectionLines) - 1
			for j := i + 1; j < len(sectionLines); j++ {
				t := strings.TrimSpace(sectionLines[j])
				if len(t) == 0 {
					continue
				}
				jIndent := getIndentation(sectionLines[j])
				if strings.HasPrefix(t, "- ") && len(jIndent) == len(stepIndent) {
					stepEnd = j - 1
					break
				}
			}

			chunk := append([]string(nil), sectionLines[stepStart:stepEnd+1]...)
			updatedChunk, changed := ensureStepCheckoutPersistCredentials(chunk, stepIndent)
			modified = modified || changed
			result = append(result, updatedChunk...)
			i = stepEnd + 1
			continue
		}

		result = append(result, line)
		i++
	}

	return result, modified
}

func ensureStepCheckoutPersistCredentials(stepLines []string, stepIndent string) ([]string, bool) {
	usesIdx := -1
	usesIndent := ""
	isUsesInline := false
	withStart := -1
	withEnd := -1
	withIndent := ""
	withKeyIndentLen := 0
	persistIdx := -1

	for i := 0; i < len(stepLines); i++ {
		line := stepLines[i]
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		usesMatch, usesValue, _ := parseStepKeyLine(trimmed, indent, stepIndent, "uses")
		if usesMatch && isCheckoutUsesValue(usesValue) {
			usesIdx = i
			isUsesInline = strings.HasPrefix(trimmed, "- uses:") && len(indent) == len(stepIndent)
			if isUsesInline {
				usesIndent = stepIndent + "  "
			} else {
				usesIndent = indent
			}
		}

		withMatch, withValue, currentWithKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "with")
		if withMatch {
			if withValue != "" && hasPersistKey(withValue) {
				if persistExplicitTrue(withValue) {
					checkoutPersistCredentialsFalseCodemodLog.Print("Skipping checkout step update: explicit with.persist-credentials: true found")
				}
				return stepLines, false
			}
			withStart = i
			withEnd = i
			withIndent = indent
			withKeyIndentLen = currentWithKeyIndentLen
			for j := i + 1; j < len(stepLines); j++ {
				t := strings.TrimSpace(stepLines[j])
				if len(t) == 0 {
					withEnd = j
					continue
				}
				if effectiveStepLineIndentLen(t, getIndentation(stepLines[j]), stepIndent) <= withKeyIndentLen {
					break
				}
				withEnd = j
				if parseYAMLMapKey(t) == "persist-credentials" {
					persistIdx = j
				}
			}
		}
	}

	if usesIdx == -1 {
		return stepLines, false
	}

	if persistIdx != -1 {
		persistLine := strings.TrimSpace(stepLines[persistIdx])
		if persistExplicitTrue(persistLine) {
			checkoutPersistCredentialsFalseCodemodLog.Print("Skipping checkout step update: explicit with.persist-credentials: true found")
		}
		return stepLines, false
	}

	if withStart != -1 {
		insertAt := withEnd + 1
		insertLine := fmt.Sprintf("%spersist-credentials: false", withIndent+"  ")
		updated := append([]string{}, stepLines[:insertAt]...)
		updated = append(updated, insertLine)
		updated = append(updated, stepLines[insertAt:]...)
		return updated, true
	}

	if usesIndent == "" {
		usesIndent = stepIndent + "  "
	}
	insertLines := []string{
		usesIndent + "with:",
		usesIndent + "  persist-credentials: false",
	}
	insertAt := usesIdx + 1
	updated := append([]string{}, stepLines[:insertAt]...)
	updated = append(updated, insertLines...)
	updated = append(updated, stepLines[insertAt:]...)
	return updated, true
}

func isCheckoutUsesValue(raw string) bool {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "\"'")
	value = strings.ToLower(value)
	return strings.HasPrefix(value, "actions/checkout@") || value == "actions/checkout"
}

func hasPersistKey(raw string) bool {
	return extractPersistCredentialsValue(raw) != ""
}

func persistExplicitTrue(raw string) bool {
	return strings.EqualFold(extractPersistCredentialsValue(raw), "true")
}

func extractPersistCredentialsValue(raw string) string {
	lower := strings.ToLower(raw)
	idx := strings.Index(lower, "persist-credentials:")
	if idx == -1 {
		return ""
	}
	rest := strings.TrimSpace(raw[idx+len("persist-credentials:"):])
	if rest == "" {
		return ""
	}

	rest = strings.SplitN(rest, "#", 2)[0]
	rest = strings.SplitN(rest, ",", 2)[0]
	rest = strings.SplitN(rest, "}", 2)[0]
	return strings.TrimSpace(strings.Trim(rest, `"'`))
}
