package cli

import (
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
)

var stepsRunSecretsEnvCodemodLog = logger.New("cli:codemod_steps_run_secrets_env")

var (
	stepsAnyExprRe        = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)
	stepsSecretBodyExprRe = regexp.MustCompile(`^secrets\.([A-Za-z_][A-Za-z0-9_]*)$`)
	stepsEnvBodyExprRe    = regexp.MustCompile(`^env\.([A-Za-z_][A-Za-z0-9_]*)$`)
	stepsSecretRefExprRe  = regexp.MustCompile(`\bsecrets\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	stepsEnvRefExprRe     = regexp.MustCompile(`\benv\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	stepsGitHubTokenRe    = regexp.MustCompile(`\bgithub\.token\b`)
	// stepsGenericExprRe matches simple GitHub Actions property-access chains such as
	// "github.repository", "inputs.my-input", "steps.my-step.outputs.result".
	// Only word characters and hyphens separated by dots are allowed; anything
	// containing spaces, operators, or other punctuation falls through to a
	// hash-based name.
	stepsGenericExprRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*(\.[a-zA-Z_][a-zA-Z0-9_-]*)*$`)
)

// getStepsRunSecretsToEnvCodemod creates a codemod that moves all ${{ ... }}
// expressions interpolated directly in run fields to step-level env bindings.
// Secrets, env refs, and github.token are given stable legacy names; all other
// expressions receive an EXPR_* name. PowerShell steps (shell: pwsh / powershell)
// receive $env:VARNAME references instead of $VARNAME.
func getStepsRunSecretsToEnvCodemod() Codemod {
	return Codemod{
		ID:           "steps-run-secrets-to-env",
		Name:         "Move step run expressions to env bindings",
		Description:  "Rewrites all ${{ ... }} expressions interpolated directly in run commands to $VARS (or $env:VARS for PowerShell steps) and adds step-level env bindings for strict-mode compatibility. Note: expressions inside single-quoted strings are rewritten too; since single quotes suppress shell variable expansion, those sections should be double-quoted if the substituted value is required.",
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
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
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
	state := rewriteStepRunSecretsToEnvState{
		seen:            make(map[string]struct{}),
		orderedBindings: make([]string, 0),
		bindingExprs:    make(map[string]string),
		firstRunLine:    -1,
		envStart:        -1,
		envEnd:          -1,
		existingEnvKeys: make(map[string]struct{}),
	}

	// First pass: detect shell type so PowerShell steps get $env:VARNAME syntax.
	// Restrict the scan to lines at the direct step-key indentation level so
	// that a run-block body line that happens to contain a literal substring
	// like "shell: pwsh" is not misclassified as PowerShell.
	shellIsPowerShell := rewriteStepRunSecretsToEnvDetectPowerShell(stepLines, stepIndent)

	for i := 0; i < len(stepLines); i++ {
		changed := rewriteStepRunSecretsToEnvProcessLine(stepLines, i, stepIndent, shellIsPowerShell, &state)
		modified = modified || changed
	}

	if len(state.orderedBindings) == 0 {
		return stepLines, modified
	}

	stepsRunSecretsEnvCodemodLog.Printf("Found %d unique run expression references in step run commands", len(state.orderedBindings))

	missingBindings := rewriteStepRunSecretsToEnvMissingBindings(&state)
	if len(missingBindings) == 0 {
		return stepLines, true
	}

	stepsRunSecretsEnvCodemodLog.Printf("Adding env bindings for %d missing expressions: %v", len(missingBindings), missingBindings)
	return rewriteStepRunSecretsToEnvInsertMissingBindings(stepLines, stepIndent, missingBindings, modified, &state)
}

type rewriteStepRunSecretsToEnvState struct {
	seen            map[string]struct{}
	orderedBindings []string
	bindingExprs    map[string]string
	firstRunLine    int
	envStart        int
	envEnd          int
	envIndent       string
	envKeyIndentLen int
	existingEnvKeys map[string]struct{}
}

func rewriteStepRunSecretsToEnvDetectPowerShell(stepLines []string, stepIndent string) bool {
	directKeyIndent := stepIndent + "  "
	for _, line := range stepLines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)
		// Accept only direct step-key lines: standard form at exactly stepIndent+"  ",
		// or list-item-inline form "- key:" at exactly stepIndent.
		if indent != directKeyIndent && (indent != stepIndent || !strings.HasPrefix(trimmed, "- ")) {
			continue
		}
		shellMatch, shellValue, _ := parseStepKeyLine(trimmed, indent, stepIndent, "shell")
		if shellMatch {
			v := strings.ToLower(strings.TrimSpace(shellValue))
			return v == "pwsh" || v == "powershell" //nolint:tolowerequalfold
		}
	}
	return false
}

func rewriteStepRunSecretsToEnvProcessLine(stepLines []string, i int, stepIndent string, shellIsPowerShell bool, state *rewriteStepRunSecretsToEnvState) bool {
	line := stepLines[i]
	trimmed := strings.TrimSpace(line)
	indent := getIndentation(line)

	rewriteStepRunSecretsToEnvTrackEnvBlock(stepLines, i, stepIndent, trimmed, indent, state)
	runMatch, runValue, runKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "run")
	if !runMatch {
		return false
	}
	if state.firstRunLine == -1 {
		state.firstRunLine = i
	}
	if runValue == "|" || runValue == "|-" || runValue == ">" || runValue == ">-" {
		return rewriteStepRunSecretsToEnvProcessRunBlock(stepLines, i, stepIndent, runKeyIndentLen, shellIsPowerShell, state)
	}

	newLine, bindings := replaceStepExpressionRefs(line, shellIsPowerShell, state.bindingExprs)
	if len(bindings) == 0 {
		return false
	}
	stepLines[i] = newLine
	rewriteStepRunSecretsToEnvRegisterBindings(bindings, state)
	return true
}

func rewriteStepRunSecretsToEnvTrackEnvBlock(stepLines []string, i int, stepIndent, trimmed, indent string, state *rewriteStepRunSecretsToEnvState) {
	envMatch, envValue, currentEnvKeyIndentLen := parseStepKeyLine(trimmed, indent, stepIndent, "env")
	if !envMatch || envValue != "" {
		return
	}
	state.envStart = i
	state.envIndent = indent
	state.envKeyIndentLen = currentEnvKeyIndentLen
	state.envEnd = i
	for j := i + 1; j < len(stepLines); j++ {
		t := strings.TrimSpace(stepLines[j])
		if t == "" {
			state.envEnd = j
			continue
		}
		if effectiveStepLineIndentLen(t, getIndentation(stepLines[j]), stepIndent) <= state.envKeyIndentLen {
			break
		}
		state.envEnd = j
		if key := parseYAMLMapKey(t); key != "" {
			state.existingEnvKeys[key] = struct{}{}
		}
	}
}

func rewriteStepRunSecretsToEnvProcessRunBlock(stepLines []string, i int, stepIndent string, runKeyIndentLen int, shellIsPowerShell bool, state *rewriteStepRunSecretsToEnvState) bool {
	modified := false
	for j := i + 1; j < len(stepLines); j++ {
		t := strings.TrimSpace(stepLines[j])
		if t == "" {
			continue
		}
		if effectiveStepLineIndentLen(t, getIndentation(stepLines[j]), stepIndent) <= runKeyIndentLen {
			break
		}
		// Skip shell comment lines – expressions inside # comments are
		// documentation-only and must not generate env bindings.
		// NOTE: heredoc boundaries are not tracked; lines starting with
		// '#' inside a heredoc body are also skipped (follow-up needed).
		if strings.HasPrefix(t, "#") {
			continue
		}
		updatedLine, bindings := replaceStepExpressionRefs(stepLines[j], shellIsPowerShell, state.bindingExprs)
		if len(bindings) > 0 {
			stepLines[j] = updatedLine
			modified = true
			rewriteStepRunSecretsToEnvRegisterBindings(bindings, state)
		}
	}
	return modified
}

func rewriteStepRunSecretsToEnvRegisterBindings(bindings []stepExpressionBinding, state *rewriteStepRunSecretsToEnvState) {
	for _, binding := range bindings {
		if !setutil.Contains(state.seen, binding.Name) {
			state.seen[binding.Name] = struct{}{}
			state.orderedBindings = append(state.orderedBindings, binding.Name)
			state.bindingExprs[binding.Name] = binding.Expression
		}
	}
}

func rewriteStepRunSecretsToEnvMissingBindings(state *rewriteStepRunSecretsToEnvState) []string {
	missingBindings := make([]string, 0, len(state.orderedBindings))
	for _, name := range state.orderedBindings {
		if !setutil.Contains(state.existingEnvKeys, name) {
			missingBindings = append(missingBindings, name)
		}
	}
	return missingBindings
}

func rewriteStepRunSecretsToEnvInsertMissingBindings(stepLines []string, stepIndent string, missingBindings []string, modified bool, state *rewriteStepRunSecretsToEnvState) ([]string, bool) {
	if state.envStart != -1 {
		insertAt := state.envEnd + 1
		envValueIndent := state.envIndent + "  "
		insertLines := make([]string, 0, len(missingBindings))
		for _, name := range missingBindings {
			insertLines = append(insertLines, fmt.Sprintf("%s%s: %s", envValueIndent, name, state.bindingExprs[name]))
		}
		stepLines = append(stepLines[:insertAt], append(insertLines, stepLines[insertAt:]...)...)
		return stepLines, true
	}

	if state.firstRunLine == -1 {
		return stepLines, modified
	}

	insertIndent := stepIndent + "  "
	insertLines := []string{insertIndent + "env:"}
	for _, name := range missingBindings {
		insertLines = append(insertLines, fmt.Sprintf("%s  %s: %s", insertIndent, name, state.bindingExprs[name]))
	}
	stepLines = append(stepLines[:state.firstRunLine], append(insertLines, stepLines[state.firstRunLine:]...)...)
	return stepLines, true
}

type stepExpressionBinding struct {
	Name       string
	Expression string
}

func replaceStepExpressionRefs(line string, shellIsPowerShell bool, existingBindings map[string]string) (string, []stepExpressionBinding) {
	matches := stepsAnyExprRe.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return line, nil
	}

	var result strings.Builder
	last := 0
	state := replaceStepExpressionRefsState{
		bodyToName:      make(map[string]string),
		localNames:      make(map[string]string),
		registeredNames: make(map[string]struct{}),
		ordered:         make([]stepExpressionBinding, 0, len(matches)),
	}

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		fullStart, fullEnd := match[0], match[1]
		bodyStart, bodyEnd := match[2], match[3]
		fullExpression := line[fullStart:fullEnd]
		body := strings.TrimSpace(line[bodyStart:bodyEnd])

		result.WriteString(line[last:fullStart])

		if cachedName, done := state.bodyToName[body]; done {
			result.WriteString(replaceStepExpressionRefsVarRef(cachedName, shellIsPowerShell))
			last = fullEnd
			continue
		}

		envName, canonicalExpression, ok := replaceStepExpressionRefsBinding(body, existingBindings, &state)
		if !ok {
			result.WriteString(fullExpression)
			last = fullEnd
			continue
		}

		state.bodyToName[body] = envName
		state.localNames[envName] = canonicalExpression
		result.WriteString(replaceStepExpressionRefsVarRef(envName, shellIsPowerShell))
		replaceStepExpressionRefsRegisterBinding(envName, canonicalExpression, &state)
		last = fullEnd
	}

	result.WriteString(line[last:])
	return result.String(), state.ordered
}

type replaceStepExpressionRefsState struct {
	// bodyToName maps expression body → assigned env-var name for same-body dedup
	// within this line (avoids re-computing the name for repeated occurrences).
	bodyToName map[string]string
	// localNames maps env-var name → canonical expression for within-line
	// collision detection (two different bodies that sanitize to the same name).
	localNames map[string]string
	// registeredNames tracks which names already appear in ordered, so we never
	// add a duplicate binding entry.
	registeredNames map[string]struct{}
	ordered         []stepExpressionBinding
}

func replaceStepExpressionRefsBinding(body string, existingBindings map[string]string, state *replaceStepExpressionRefsState) (string, string, bool) {
	envName, canonicalExpression, ok := mapRunExpressionToEnvBinding(body)
	if !ok {
		return "", "", false
	}

	// Collision guard: if this env-var name is already bound to a *different*
	// expression (from a previous line in this step via existingBindings, or
	// from an earlier occurrence within this line via localNames), fall back
	// to a hash-based name so both expressions receive unique bindings.
	if crossLine := existingBindings[envName]; (crossLine != "" && crossLine != canonicalExpression) ||
		(state.localNames[envName] != "" && state.localNames[envName] != canonicalExpression) {
		envName = hashedBindingName("EXPR", body)
		canonicalExpression = fmt.Sprintf("${{ %s }}", body)
	}
	return envName, canonicalExpression, true
}

func replaceStepExpressionRefsVarRef(envName string, shellIsPowerShell bool) string {
	if shellIsPowerShell {
		return "$env:" + envName
	}
	return "$" + envName
}

func replaceStepExpressionRefsRegisterBinding(envName, canonicalExpression string, state *replaceStepExpressionRefsState) {
	if setutil.Contains(state.registeredNames, envName) {
		return
	}
	state.registeredNames[envName] = struct{}{}
	state.ordered = append(state.ordered, stepExpressionBinding{
		Name:       envName,
		Expression: canonicalExpression,
	})
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

	// Catch-all: hoist any remaining expression using EXPR_ naming.
	if stepsGenericExprRe.MatchString(body) {
		replacer := strings.NewReplacer(".", "_", "-", "_")
		name := "EXPR_" + strings.ToUpper(replacer.Replace(body))
		return name, fmt.Sprintf("${{ %s }}", body), true
	}
	// Complex expression: use a hash suffix for collision safety.
	name := hashedBindingName("EXPR", body)
	return name, fmt.Sprintf("${{ %s }}", body), true
}

// hashedBindingName returns a collision-resistant binding key by suffixing
// the caller-provided prefix with a stable hash of the expression body.
func hashedBindingName(prefix, body string) string {
	h := fnv.New32a()
	// fnv.Hash.Write on in-memory bytes is guaranteed not to return an error.
	_, _ = io.WriteString(h, body)
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
