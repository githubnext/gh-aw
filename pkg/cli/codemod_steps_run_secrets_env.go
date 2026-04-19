package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var stepsRunSecretsEnvCodemodLog = logger.New("cli:codemod_steps_run_secrets_env")

var stepsSecretExprRe = regexp.MustCompile(`\$\{\{\s*secrets\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// getStepsRunSecretsToEnvCodemod creates a codemod that moves secrets interpolated directly
// in run fields to step-level env bindings in steps-like sections.
func getStepsRunSecretsToEnvCodemod() Codemod {
	return Codemod{
		ID:           "steps-run-secrets-to-env",
		Name:         "Move step run secrets to env bindings",
		Description:  "Rewrites secrets interpolated directly in run commands to $VARS and adds step-level env bindings for strict-mode compatibility.",
		IntroducedIn: "0.26.0",
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
					current, sectionChanged = transformSectionStepsRunSecrets(current, section)
					modified = modified || sectionChanged
				}
				return current, modified
			})
			if applied {
				stepsRunSecretsEnvCodemodLog.Print("Moved inline step run secrets to step-level env bindings")
			}
			return newContent, applied, err
		},
	}
}

func transformSectionStepsRunSecrets(lines []string, sectionName string) ([]string, bool) {
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

	stepsRunSecretsEnvCodemodLog.Printf("Transforming section '%s': lines %d-%d", sectionName, sectionStart, sectionEnd)

	sectionLines := lines[sectionStart : sectionEnd+1]
	updatedSection, changed := transformStepsWithinSection(sectionLines, sectionIndent)
	if !changed {
		return lines, false
	}

	result := make([]string, 0, len(lines)-(len(sectionLines)-len(updatedSection)))
	result = append(result, lines[:sectionStart]...)
	result = append(result, updatedSection...)
	result = append(result, lines[sectionEnd+1:]...)
	return result, true
}

func transformStepsWithinSection(sectionLines []string, sectionIndent string) ([]string, bool) {
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
			updatedChunk, changed := rewriteStepRunSecretsToEnv(chunk, stepIndent)
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

func rewriteStepRunSecretsToEnv(stepLines []string, stepIndent string) ([]string, bool) {
	modified := false
	seen := make(map[string]bool)
	orderedSecrets := make([]string, 0)
	firstRunLine := -1
	envStart := -1
	envEnd := -1
	envIndent := ""
	var envKeyIndentLen int
	existingEnvKeys := make(map[string]bool)

	for i := 0; i < len(stepLines); i++ {
		line := stepLines[i]
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		envMatch, envValue, currentEnvKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "env")
		if envMatch && envValue == "" {
			envStart = i
			envIndent = indent
			envKeyIndentLen = currentEnvKeyIndentLen
			envEnd = i
			for j := i + 1; j < len(stepLines); j++ {
				t := strings.TrimSpace(stepLines[j])
				if len(t) == 0 {
					envEnd = j
					continue
				}
				if effectiveStepLineIndentLen(t, getIndentation(stepLines[j]), stepIndent) <= envKeyIndentLen {
					break
				}
				envEnd = j
				key := parseYAMLMapKey(t)
				if key != "" {
					existingEnvKeys[key] = true
				}
			}
		}

		runMatch, runValue, runKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "run")
		if !runMatch {
			continue
		}
		if firstRunLine == -1 {
			firstRunLine = i
		}

		if runValue == "|" || runValue == "|-" || runValue == ">" || runValue == ">-" {
			for j := i + 1; j < len(stepLines); j++ {
				t := strings.TrimSpace(stepLines[j])
				if len(t) == 0 {
					continue
				}
				if effectiveStepLineIndentLen(t, getIndentation(stepLines[j]), stepIndent) <= runKeyIndentLen {
					break
				}
				updatedLine, names := replaceStepSecretRefs(stepLines[j])
				if len(names) > 0 {
					stepLines[j] = updatedLine
					modified = true
				}
				for _, name := range names {
					if !seen[name] {
						seen[name] = true
						orderedSecrets = append(orderedSecrets, name)
					}
				}
			}
			continue
		}

		newLine, names := replaceStepSecretRefs(line)
		if len(names) > 0 {
			stepLines[i] = newLine
			modified = true
		}
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				orderedSecrets = append(orderedSecrets, name)
			}
		}
	}

	if len(orderedSecrets) == 0 {
		return stepLines, modified
	}

	stepsRunSecretsEnvCodemodLog.Printf("Found %d unique secret references in step run commands", len(orderedSecrets))

	missingSecrets := make([]string, 0, len(orderedSecrets))
	for _, name := range orderedSecrets {
		if !existingEnvKeys[name] {
			missingSecrets = append(missingSecrets, name)
		}
	}
	if len(missingSecrets) == 0 {
		return stepLines, true
	}

	stepsRunSecretsEnvCodemodLog.Printf("Adding env bindings for %d missing secrets: %v", len(missingSecrets), missingSecrets)

	if envStart != -1 {
		insertAt := envEnd + 1
		envValueIndent := envIndent + "  "
		insertLines := make([]string, 0, len(missingSecrets))
		for _, name := range missingSecrets {
			insertLines = append(insertLines, fmt.Sprintf("%s%s: ${{ secrets.%s }}", envValueIndent, name, name))
		}
		stepLines = append(stepLines[:insertAt], append(insertLines, stepLines[insertAt:]...)...)
		return stepLines, true
	}

	if firstRunLine == -1 {
		return stepLines, modified
	}

	insertIndent := stepIndent + "  "
	insertLines := []string{insertIndent + "env:"}
	for _, name := range missingSecrets {
		insertLines = append(insertLines, fmt.Sprintf("%s  %s: ${{ secrets.%s }}", insertIndent, name, name))
	}
	stepLines = append(stepLines[:firstRunLine], append(insertLines, stepLines[firstRunLine:]...)...)
	return stepLines, true
}

func replaceStepSecretRefs(line string) (string, []string) {
	matches := stepsSecretExprRe.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return line, nil
	}
	seen := make(map[string]bool)
	ordered := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := match[1]
		if !seen[name] {
			seen[name] = true
			ordered = append(ordered, name)
		}
	}
	// In Go regexp replacement syntax, "$$$1" means:
	// "$$" -> literal "$" in output, then "$1" -> capture group 1 (secret name),
	// resulting in "$SECRET_NAME" shell env references.
	updated := stepsSecretExprRe.ReplaceAllString(line, `$$$1`)
	return updated, ordered
}

// parseStepKeyLine detects a YAML step key in both standard form ("key: value")
// and list-item-inline form ("- key: value").
//
// Parameters:
//   - trimmed: current line with surrounding whitespace trimmed
//   - indent: raw indentation of the current line
//   - stepIndent: indentation of the step list item line
//   - key: YAML key name to match (for example "run" or "env")
//
// Returns:
//   - matched: whether the line contains the requested key in either supported form
//   - value: trimmed value after the key (empty for block-style keys)
//   - keyIndentLen: effective indentation length for block-boundary checks
func parseStepKeyLine(trimmed, indent, stepIndent, key string) (bool, string, int) {
	if strings.HasPrefix(trimmed, key+":") && len(indent) > len(stepIndent) {
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, key+":"))
		return true, value, len(indent)
	}
	listKeyPrefix := "- " + key + ":"
	if strings.HasPrefix(trimmed, listKeyPrefix) && len(indent) == len(stepIndent) {
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, listKeyPrefix))
		return true, value, len(stepIndent) + 2
	}
	return false, "", 0
}

// effectiveStepLineIndentLen returns the logical indentation length for a line
// within a step block.
//
// For list-item-inline lines like "- run: ...", the "- " marker contributes two
// characters to the effective YAML nesting level, so this function adds 2 to the
// physical step indentation when computing boundary comparisons.
func effectiveStepLineIndentLen(trimmed, indent, stepIndent string) int {
	if strings.HasPrefix(trimmed, "- ") && len(indent) == len(stepIndent) {
		return len(stepIndent) + 2
	}
	return len(indent)
}

func parseYAMLMapKey(trimmedLine string) string {
	if strings.HasPrefix(trimmedLine, "- ") {
		return ""
	}
	parts := strings.SplitN(trimmedLine, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
