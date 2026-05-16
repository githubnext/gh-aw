package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var autoHoistRunExpressionsLog = logger.New("cli:codemod_auto_hoist_run_expressions")

// autoHoistSimpleExprBodyRe matches simple JavaScript property-access chains such as
// "github.token", "env.FOO", "inputs.my-input", "steps.my-step.outputs.result".
// Only word characters and hyphens separated by dots are allowed; any expression
// containing spaces, operators, or other punctuation falls through to the hash-based
// name generator.
var autoHoistSimpleExprBodyRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*(\.[a-zA-Z_][a-zA-Z0-9_-]*)*$`)

// getAutoHoistRunExpressionsCodemod creates a codemod that hoists ALL ${{ ... }}
// expressions from run: blocks into step-level env bindings.
//
// Unlike steps-run-secrets-to-env (which only handles secrets.*, env.*, and
// github.token), this codemod covers every expression that would otherwise trigger
// the "compiler regression detected" hard error.
//
// Naming convention: EXPR_ + sanitised uppercase expression body.
//   - Simple chains: github.token → EXPR_GITHUB_TOKEN
//   - Complex expressions: ${{ secrets.TOKEN || '' }} → EXPR_<8-char-hash>
//
// PowerShell awareness: steps with shell: pwsh or shell: powershell receive
// $env:VARNAME references in the run script instead of $VARNAME.
func getAutoHoistRunExpressionsCodemod() Codemod {
	return Codemod{
		ID:           "auto-hoist-run-expressions",
		Name:         "Auto-hoist run-block expressions to env bindings",
		Description:  "Rewrites all ${{ ... }} expressions in run: scripts to $VARNAME references (or $env:VARNAME for PowerShell steps) and adds EXPR_* step-level env bindings.",
		IntroducedIn: "1.0.45",
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
					current, sectionChanged = transformSectionAutoHoistExpressions(current, section)
					modified = modified || sectionChanged
				}
				return current, modified
			})
			if applied {
				autoHoistRunExpressionsLog.Print("Auto-hoisted inline run expressions to step-level env bindings")
			}
			return newContent, applied, err
		},
	}
}

func transformSectionAutoHoistExpressions(lines []string, sectionName string) ([]string, bool) {
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

	autoHoistRunExpressionsLog.Printf("Transforming section '%s': lines %d-%d", sectionName, sectionStart, sectionEnd)

	sectionLines := lines[sectionStart : sectionEnd+1]
	updatedSection, changed := transformStepsWithinSectionAutoHoist(sectionLines, sectionIndent)
	if !changed {
		return lines, false
	}

	result := make([]string, 0, len(lines)-(len(sectionLines)-len(updatedSection)))
	result = append(result, lines[:sectionStart]...)
	result = append(result, updatedSection...)
	result = append(result, lines[sectionEnd+1:]...)
	return result, true
}

func transformStepsWithinSectionAutoHoist(sectionLines []string, sectionIndent string) ([]string, bool) {
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
			updatedChunk, changed := rewriteStepAutoHoistExpressions(chunk, stepIndent)
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

func rewriteStepAutoHoistExpressions(stepLines []string, stepIndent string) ([]string, bool) {
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
	shellIsPowerShell := false

	// First pass: detect shell type so PowerShell steps get $env:VARNAME syntax.
	for _, line := range stepLines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)
		shellMatch, shellValue, _ := parseStepKeyLine(trimmed, indent, stepIndent, "shell")
		if shellMatch {
			v := strings.ToLower(strings.TrimSpace(shellValue))
			if v == "pwsh" || v == "powershell" {
				shellIsPowerShell = true
			}
			break
		}
	}

	// Second pass: scan env block and rewrite run block.
	for i := 0; i < len(stepLines); i++ {
		line := stepLines[i]
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		// Track the existing env: block so new bindings can be appended to it.
		envMatchKey, envValue, currentEnvKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "env")
		if envMatchKey && envValue == "" {
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
				updatedLine, bindings := replaceAutoHoistExpressionRefs(stepLines[j], shellIsPowerShell)
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

		newLine, bindings := replaceAutoHoistExpressionRefs(line, shellIsPowerShell)
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

	autoHoistRunExpressionsLog.Printf("Found %d unique run expression references in step run commands", len(orderedBindings))

	missingBindings := make([]string, 0, len(orderedBindings))
	for _, name := range orderedBindings {
		if !existingEnvKeys[name] {
			missingBindings = append(missingBindings, name)
		}
	}
	if len(missingBindings) == 0 {
		return stepLines, true
	}

	autoHoistRunExpressionsLog.Printf("Adding env bindings for %d missing expressions: %v", len(missingBindings), missingBindings)

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

// replaceAutoHoistExpressionRefs replaces every ${{ ... }} expression in line
// with either $VARNAME (bash) or $env:VARNAME (PowerShell) and returns the
// corresponding env bindings to add to the step's env: block.
func replaceAutoHoistExpressionRefs(line string, shellIsPowerShell bool) (string, []stepExpressionBinding) {
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

		envName, canonicalExpression, ok := mapAutoHoistExpressionToEnvBinding(body)
		if !ok {
			result.WriteString(fullExpression)
			last = fullEnd
			continue
		}

		if shellIsPowerShell {
			result.WriteString("$env:" + envName)
		} else {
			result.WriteString("$" + envName)
		}
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

// mapAutoHoistExpressionToEnvBinding maps any ${{ body }} expression to a
// deterministic EXPR_* environment variable name and the canonical expression
// string to bind as its value.
//
// Simple property-access chains (e.g. "github.token", "env.FOO",
// "inputs.my-input") produce pretty names:
//
//	github.token    → EXPR_GITHUB_TOKEN
//	env.FOO         → EXPR_ENV_FOO
//	inputs.my-input → EXPR_INPUTS_MY_INPUT
//
// Complex expressions (anything containing spaces, operators, function calls,
// etc.) fall back to a hash-based name to guarantee uniqueness:
//
//	secrets.TOKEN || '' → EXPR_<8-char-fnv32-hex>
func mapAutoHoistExpressionToEnvBinding(body string) (string, string, bool) {
	if autoHoistSimpleExprBodyRe.MatchString(body) {
		replacer := strings.NewReplacer(".", "_", "-", "_")
		name := "EXPR_" + strings.ToUpper(replacer.Replace(body))
		return name, fmt.Sprintf("${{ %s }}", body), true
	}
	// Complex expression: include a hash suffix to prevent name collisions
	// between different complex expressions in the same step.
	name := hashedBindingName("EXPR", body)
	return name, fmt.Sprintf("${{ %s }}", body), true
}
