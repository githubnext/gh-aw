package cli

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var stepsRunSecretsEnvCodemodLog = logger.New("cli:codemod_steps_run_secrets_env")

var (
	stepsAnyExprRe        = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)
	stepsSecretBodyExprRe = regexp.MustCompile(`^secrets\.([A-Za-z_][A-Za-z0-9_]*)$`)
	stepsEnvBodyExprRe    = regexp.MustCompile(`^env\.([A-Za-z_][A-Za-z0-9_]*)$`)
	stepsSecretRefExprRe  = regexp.MustCompile(`\bsecrets\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	stepsEnvRefExprRe     = regexp.MustCompile(`\benv\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	stepsGitHubTokenRe    = regexp.MustCompile(`\bgithub\.token\b`)
)

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
	orderedBindings := make([]string, 0)
	bindingExprs := make(map[string]string)
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
				updatedLine, bindings := replaceStepExpressionRefs(stepLines[j])
				if len(bindings) > 0 {
					stepLines[j] = updatedLine
					modified = true
				}
				for _, binding := range bindings {
					if !seen[binding.Name] {
						seen[binding.Name] = true
						orderedBindings = append(orderedBindings, binding.Name)
						bindingExprs[binding.Name] = binding.Expression
					}
				}
			}
			continue
		}

		newLine, bindings := replaceStepExpressionRefs(line)
		if len(bindings) > 0 {
			stepLines[i] = newLine
			modified = true
		}
		for _, binding := range bindings {
			if !seen[binding.Name] {
				seen[binding.Name] = true
				orderedBindings = append(orderedBindings, binding.Name)
				bindingExprs[binding.Name] = binding.Expression
			}
		}
	}

	if len(orderedBindings) == 0 {
		return stepLines, modified
	}

	stepsRunSecretsEnvCodemodLog.Printf("Found %d unique run expression references in step run commands", len(orderedBindings))

	missingBindings := make([]string, 0, len(orderedBindings))
	for _, name := range orderedBindings {
		if !existingEnvKeys[name] {
			missingBindings = append(missingBindings, name)
		}
	}
	if len(missingBindings) == 0 {
		return stepLines, true
	}

	stepsRunSecretsEnvCodemodLog.Printf("Adding env bindings for %d missing expressions: %v", len(missingBindings), missingBindings)

	if envStart != -1 {
		insertAt := envEnd + 1
		envValueIndent := envIndent + "  "
		insertLines := make([]string, 0, len(missingBindings))
		for _, name := range missingBindings {
			insertLines = append(insertLines, fmt.Sprintf("%s%s: %s", envValueIndent, name, bindingExprs[name]))
		}
		stepLines = append(stepLines[:insertAt], append(insertLines, stepLines[insertAt:]...)...)
		return stepLines, true
	}

	if firstRunLine == -1 {
		return stepLines, modified
	}

	insertIndent := stepIndent + "  "
	insertLines := []string{insertIndent + "env:"}
	for _, name := range missingBindings {
		insertLines = append(insertLines, fmt.Sprintf("%s  %s: %s", insertIndent, name, bindingExprs[name]))
	}
	stepLines = append(stepLines[:firstRunLine], append(insertLines, stepLines[firstRunLine:]...)...)
	return stepLines, true
}

type stepExpressionBinding struct {
	Name       string
	Expression string
}

func replaceStepExpressionRefs(line string) (string, []stepExpressionBinding) {
	matches := stepsAnyExprRe.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return line, nil
	}

	var result strings.Builder
	last := 0
	seen := make(map[string]bool)
	ordered := make([]stepExpressionBinding, 0, len(matches))

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		fullStart, fullEnd := match[0], match[1]
		bodyStart, bodyEnd := match[2], match[3]
		fullExpression := line[fullStart:fullEnd]
		body := strings.TrimSpace(line[bodyStart:bodyEnd])

		result.WriteString(line[last:fullStart])

		envName, canonicalExpression, ok := mapRunExpressionToEnvBinding(body)
		if !ok {
			result.WriteString(fullExpression)
			last = fullEnd
			continue
		}

		result.WriteString("$" + envName)
		if !seen[envName] {
			seen[envName] = true
			ordered = append(ordered, stepExpressionBinding{
				Name:       envName,
				Expression: canonicalExpression,
			})
		}
		last = fullEnd
	}

	result.WriteString(line[last:])
	return result.String(), ordered
}

func mapRunExpressionToEnvBinding(body string) (string, string, bool) {
	if secretMatch := stepsSecretBodyExprRe.FindStringSubmatch(body); len(secretMatch) == 2 {
		secretName := secretMatch[1]
		return secretName, fmt.Sprintf("${{ secrets.%s }}", secretName), true
	}

	if envMatch := stepsEnvBodyExprRe.FindStringSubmatch(body); len(envMatch) == 2 {
		envName := envMatch[1]
		return "GH_AW_ENV_" + envName, fmt.Sprintf("${{ env.%s }}", envName), true
	}

	if body == "github.token" {
		return "GH_AW_GITHUB_TOKEN", "${{ github.token }}", true
	}

	if secretRef := stepsSecretRefExprRe.FindStringSubmatch(body); len(secretRef) == 2 {
		return hashedBindingName("GH_AW_SECRET_"+secretRef[1], body), fmt.Sprintf("${{ %s }}", body), true
	}

	if envRef := stepsEnvRefExprRe.FindStringSubmatch(body); len(envRef) == 2 {
		return hashedBindingName("GH_AW_ENV_"+envRef[1], body), fmt.Sprintf("${{ %s }}", body), true
	}

	if stepsGitHubTokenRe.MatchString(body) {
		return hashedBindingName("GH_AW_GITHUB_TOKEN", body), fmt.Sprintf("${{ %s }}", body), true
	}

	return "", "", false
}

// hashedBindingName returns a collision-resistant binding key by suffixing
// the caller-provided prefix with a stable hash of the expression body.
func hashedBindingName(prefix, body string) string {
	h := fnv.New32a()
	// fnv.Hash.Write on in-memory bytes is guaranteed not to return an error.
	_, _ = h.Write([]byte(body))
	return fmt.Sprintf("%s_%08x", prefix, h.Sum32())
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
