package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var checkoutPersistCredentialsFalseCodemodLog = logger.New("cli:codemod_checkout_persist_credentials_false")

type checkoutPersistCredentialsBlock struct {
	start  int
	end    int
	indent string
}

type checkoutPersistCredentialsStepScan struct {
	usesIdx    int
	usesIndent string
	withStart  int
	withEnd    int
	withIndent string
	persistIdx int
}

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
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
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
	jobsBlock, ok := transformAgentJobCheckoutPersistCredentialsFindJobs(lines)
	if !ok {
		return lines, false
	}

	jobsLines := lines[jobsBlock.start : jobsBlock.end+1]
	agentBlock, ok := transformAgentJobCheckoutPersistCredentialsFindAgent(jobsLines, jobsBlock.indent)
	if !ok {
		return lines, false
	}

	agentLines := append([]string(nil), jobsLines[agentBlock.start:agentBlock.end+1]...)
	agentLines, modified := transformAgentJobCheckoutPersistCredentialsSections(agentLines, sectionNames, agentBlock.indent)
	if !modified {
		return lines, false
	}

	updatedJobsLines := make([]string, 0, len(jobsLines))
	updatedJobsLines = append(updatedJobsLines, jobsLines[:agentBlock.start]...)
	updatedJobsLines = append(updatedJobsLines, agentLines...)
	updatedJobsLines = append(updatedJobsLines, jobsLines[agentBlock.end+1:]...)

	result := make([]string, 0, len(lines))
	result = append(result, lines[:jobsBlock.start]...)
	result = append(result, updatedJobsLines...)
	result = append(result, lines[jobsBlock.end+1:]...)
	return result, true
}

func transformAgentJobCheckoutPersistCredentialsFindJobs(lines []string) (checkoutPersistCredentialsBlock, bool) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "jobs:") {
			return checkoutPersistCredentialsBlock{
				start:  i,
				end:    transformAgentJobCheckoutPersistCredentialsBlockEnd(lines, i, getIndentation(line)),
				indent: getIndentation(line),
			}, true
		}
	}
	return checkoutPersistCredentialsBlock{}, false
}

func transformAgentJobCheckoutPersistCredentialsBlockEnd(lines []string, start int, indent string) int {
	end := len(lines) - 1
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if len(getIndentation(lines[i])) <= len(indent) {
			return i - 1
		}
	}
	return end
}

func transformAgentJobCheckoutPersistCredentialsFindAgent(jobsLines []string, jobsIndent string) (checkoutPersistCredentialsBlock, bool) {
	jobsChildIndentLen, hasJobsChild := findDirectChildIndentLen(jobsLines, 0, len(jobsIndent))
	if !hasJobsChild {
		return checkoutPersistCredentialsBlock{}, false
	}
	for i, line := range jobsLines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)
		if len(indent) == jobsChildIndentLen && parseYAMLMapKey(trimmed) == "agent" {
			return checkoutPersistCredentialsBlock{
				start:  i,
				end:    transformAgentJobCheckoutPersistCredentialsBlockEnd(jobsLines, i, indent),
				indent: indent,
			}, true
		}
	}
	return checkoutPersistCredentialsBlock{}, false
}

func transformAgentJobCheckoutPersistCredentialsSections(agentLines []string, sectionNames []string, agentIndent string) ([]string, bool) {
	modified := false
	for _, sectionName := range sectionNames {
		var sectionChanged bool
		agentLines, sectionChanged = transformNestedSectionCheckoutPersistCredentials(agentLines, sectionName, agentIndent)
		modified = modified || sectionChanged
	}
	return agentLines, modified
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
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
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
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
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
				if t == "" {
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
	scan, skip := ensureStepCheckoutPersistCredentialsScan(stepLines, stepIndent)
	if skip || scan.usesIdx == -1 {
		return stepLines, false
	}

	if scan.persistIdx != -1 {
		persistLine := strings.TrimSpace(stepLines[scan.persistIdx])
		if persistExplicitTrue(persistLine) {
			checkoutPersistCredentialsFalseCodemodLog.Print("Skipping checkout step update: explicit with.persist-credentials: true found")
		}
		return stepLines, false
	}

	if scan.withStart != -1 {
		return ensureStepCheckoutPersistCredentialsInsertIntoWith(stepLines, scan)
	}

	return ensureStepCheckoutPersistCredentialsAddWith(stepLines, stepIndent, scan)
}

func ensureStepCheckoutPersistCredentialsScan(stepLines []string, stepIndent string) (checkoutPersistCredentialsStepScan, bool) {
	scan := checkoutPersistCredentialsStepScan{usesIdx: -1, withStart: -1, withEnd: -1, persistIdx: -1}
	for i := range stepLines {
		line := stepLines[i]
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		usesMatch, usesValue, _ := parseStepKeyLine(trimmed, indent, stepIndent, "uses")
		if usesMatch && isCheckoutUsesValue(usesValue) {
			scan.usesIdx = i
			scan.usesIndent = indent
			if strings.HasPrefix(trimmed, "- uses:") && len(indent) == len(stepIndent) {
				scan.usesIndent = stepIndent + "  "
			}
		}

		withMatch, withValue, currentWithKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "with")
		if withMatch && withValue != "" && hasPersistKey(withValue) {
			if persistExplicitTrue(withValue) {
				checkoutPersistCredentialsFalseCodemodLog.Print("Skipping checkout step update: explicit with.persist-credentials: true found")
			}
			return scan, true
		}
		if withMatch {
			scan.withStart = i
			scan.withEnd = i
			scan.withIndent = indent
			scan = ensureStepCheckoutPersistCredentialsScanWithBlock(stepLines, stepIndent, i, currentWithKeyIndentLen, scan)
		}
	}
	return scan, false
}

func ensureStepCheckoutPersistCredentialsScanWithBlock(stepLines []string, stepIndent string, start int, withKeyIndentLen int, scan checkoutPersistCredentialsStepScan) checkoutPersistCredentialsStepScan {
	for j := start + 1; j < len(stepLines); j++ {
		t := strings.TrimSpace(stepLines[j])
		if t == "" {
			scan.withEnd = j
			continue
		}
		if effectiveStepLineIndentLen(t, getIndentation(stepLines[j]), stepIndent) <= withKeyIndentLen {
			break
		}
		scan.withEnd = j
		if parseYAMLMapKey(t) == "persist-credentials" {
			scan.persistIdx = j
		}
	}
	return scan
}

func ensureStepCheckoutPersistCredentialsInsertIntoWith(stepLines []string, scan checkoutPersistCredentialsStepScan) ([]string, bool) {
	insertAt := scan.withEnd + 1
	insertLine := scan.withIndent + "  persist-credentials: false"
	updated := append([]string{}, stepLines[:insertAt]...)
	updated = append(updated, insertLine)
	updated = append(updated, stepLines[insertAt:]...)
	return updated, true
}

func ensureStepCheckoutPersistCredentialsAddWith(stepLines []string, stepIndent string, scan checkoutPersistCredentialsStepScan) ([]string, bool) {
	usesIndent := scan.usesIndent
	if usesIndent == "" {
		usesIndent = stepIndent + "  "
	}
	insertLines := []string{
		usesIndent + "with:",
		usesIndent + "  persist-credentials: false",
	}
	insertAt := scan.usesIdx + 1
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
