package cli

import (
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
			agentSections := []string{"pre-steps", "steps", "post-steps", "pre-agent-steps"}
			if !hasTopLevelSection(frontmatter, agentSections) && !hasAgentJobSection(frontmatter, agentSections) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				modified := false
				current := lines
				// Top-level sections and jobs.agent sections are distinct config surfaces
				// for the same agent job and are transformed independently when present.
				for _, section := range agentSections {
					var sectionChanged bool
					current, sectionChanged = transformSectionCheckoutPersistCredentials(current, section)
					modified = modified || sectionChanged
				}
				current, appliedInAgentJob := transformAgentJobCheckoutPersistCredentials(current, agentSections)
				modified = modified || appliedInAgentJob
				return current, modified
			})
			if applied {
				checkoutPersistCredentialsFalseCodemodLog.Print("Added persist-credentials: false to actions/checkout step with blocks")
			}
			return newContent, applied, err
		},
	}
}

func hasTopLevelSection(frontmatter map[string]any, sections []string) bool {
	for _, section := range sections {
		if _, ok := frontmatter[section]; ok {
			return true
		}
	}
	return false
}

func hasAgentJobSection(frontmatter map[string]any, sections []string) bool {
	jobsValue, ok := frontmatter["jobs"]
	if !ok {
		return false
	}
	jobsMap, ok := jobsValue.(map[string]any)
	if !ok {
		return false
	}
	agentValue, ok := jobsMap["agent"]
	if !ok {
		return false
	}
	agentMap, ok := agentValue.(map[string]any)
	if !ok {
		return false
	}
	for _, section := range sections {
		if _, ok := agentMap[section]; ok {
			return true
		}
	}
	return false
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

func transformAgentJobCheckoutPersistCredentials(lines []string, sectionNames []string) ([]string, bool) {
	jobsStart := -1
	jobsIndent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "jobs:") {
			jobsStart = i
			jobsIndent = getIndentation(line)
			break
		}
	}
	if jobsStart == -1 {
		return lines, false
	}

	jobsEnd := len(lines) - 1
	for i := jobsStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if len(getIndentation(lines[i])) <= len(jobsIndent) {
			jobsEnd = i - 1
			break
		}
	}

	jobsLines := lines[jobsStart : jobsEnd+1]
	jobsChildIndentLen, hasJobsChild := findDirectChildIndentLen(jobsLines, 0, len(jobsIndent))
	if !hasJobsChild {
		return lines, false
	}

	agentStart := -1
	agentIndent := ""
	for i, line := range jobsLines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)
		if len(indent) == jobsChildIndentLen && parseYAMLMapKey(trimmed) == "agent" {
			agentStart = i
			agentIndent = indent
			break
		}
	}
	if agentStart == -1 {
		return lines, false
	}

	agentEnd := len(jobsLines) - 1
	for i := agentStart + 1; i < len(jobsLines); i++ {
		trimmed := strings.TrimSpace(jobsLines[i])
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if len(getIndentation(jobsLines[i])) <= len(agentIndent) {
			agentEnd = i - 1
			break
		}
	}

	agentLines := append([]string(nil), jobsLines[agentStart:agentEnd+1]...)
	modified := false
	for _, sectionName := range sectionNames {
		var sectionChanged bool
		agentLines, sectionChanged = transformNestedSectionCheckoutPersistCredentials(agentLines, sectionName, agentIndent)
		modified = modified || sectionChanged
	}
	if !modified {
		return lines, false
	}

	updatedJobsLines := make([]string, 0, len(jobsLines))
	updatedJobsLines = append(updatedJobsLines, jobsLines[:agentStart]...)
	updatedJobsLines = append(updatedJobsLines, agentLines...)
	updatedJobsLines = append(updatedJobsLines, jobsLines[agentEnd+1:]...)

	result := make([]string, 0, len(lines))
	result = append(result, lines[:jobsStart]...)
	result = append(result, updatedJobsLines...)
	result = append(result, lines[jobsEnd+1:]...)
	return result, true
}

func transformNestedSectionCheckoutPersistCredentials(lines []string, sectionName, parentIndent string) ([]string, bool) {
	childIndentLen, hasChild := findDirectChildIndentLen(lines, 0, len(parentIndent))
	if !hasChild {
		return lines, false
	}

	sectionStart := -1
	sectionIndent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)
		if len(indent) == childIndentLen && strings.HasPrefix(trimmed, sectionName+":") {
			sectionStart = i
			sectionIndent = indent
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

// findDirectChildIndentLen returns the indentation width of the first non-empty,
// non-comment line that is a direct child of the given parent block.
// It returns (0, false) when no such child line exists.
func findDirectChildIndentLen(lines []string, parentStart int, parentIndentLen int) (int, bool) {
	for i := parentStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indentLen := len(getIndentation(lines[i]))
		if indentLen <= parentIndentLen {
			return 0, false
		}
		return indentLen, true
	}

	return 0, false
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
	withStart := -1
	withEnd := -1
	withIndent := ""
	persistIdx := -1

	for i := range stepLines {
		line := stepLines[i]
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		usesMatch, usesValue, _ := parseStepKeyLine(trimmed, indent, stepIndent, "uses")
		if usesMatch && isCheckoutUsesValue(usesValue) {
			usesIdx = i
			isUsesInline := strings.HasPrefix(trimmed, "- uses:") && len(indent) == len(stepIndent)
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
			withKeyIndentLen := currentWithKeyIndentLen
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
		insertLine := withIndent + "  persist-credentials: false"
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
