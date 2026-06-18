package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var shellLog = logger.New("workflow:shell")

// shellJoinArgs joins command arguments with proper shell escaping.
// Arguments containing ${{ }} GitHub Actions expressions are double-quoted;
// other arguments with special shell characters are single-quoted.
func shellJoinArgs(args []string) string {
	shellLog.Printf("Joining %d shell arguments with escaping", len(args))
	var escapedArgs []string
	for _, arg := range args {
		escapedArgs = append(escapedArgs, shellEscapeArg(arg))
	}
	result := strings.Join(escapedArgs, " ")
	shellLog.Print("Shell arguments joined successfully")
	return result
}

// shellEscapeArg escapes a single argument for safe use in shell commands.
// Arguments containing ${{ }} GitHub Actions expressions are double-quoted;
// other arguments with special shell characters are single-quoted.
func shellEscapeArg(arg string) string {
	// If the argument contains GitHub Actions expressions (${{ }}), use double-quote
	// wrapping. GitHub Actions evaluates ${{ }} at the YAML level before the shell runs,
	// so single-quoting would mangle the expression syntax (e.g., 'staging' inside
	// ${{ env.X == 'staging' }} becomes '\''staging'\'' which GA cannot parse).
	// Double-quoting preserves the expression for GA evaluation.
	if containsExpression(arg) {
		shellLog.Print("Argument contains GitHub Actions expression, using double-quote wrapping")
		escaped := strings.ReplaceAll(arg, `"`, `\"`)
		// Escape bare $ signs (those not part of a ${{ }} expression) so that bash
		// does not perform variable expansion inside the double-quoted string.
		// For example, the JSON key "$schema" must become "\$schema" so bash writes
		// the literal dollar sign rather than expanding the (unset) shell variable
		// $schema to an empty string. ${{ … }} expressions are left untouched because
		// GitHub Actions resolves them before the shell ever runs.
		escaped = escapeBareShellDollarSigns(escaped)
		return `"` + escaped + `"`
	}

	// Check if the argument contains special shell characters that need escaping
	if strings.ContainsAny(arg, "()[]{}*?$`\"'\\|&;<> \t\n") {
		shellLog.Print("Argument contains special characters, applying escaping")
		// Handle single quotes in the argument by escaping them
		// Use '\'' instead of '\"'\"' to avoid creating double-quoted contexts
		// that would interpret backslash escape sequences
		escaped := strings.ReplaceAll(arg, "'", "'\\''")
		return "'" + escaped + "'"
	}
	return arg
}

// escapeBareShellDollarSigns replaces every $ that is NOT the start of a ${{ }}
// GitHub Actions expression with \$. This prevents bash from performing variable
// expansion when the string is embedded inside a double-quoted shell argument.
//
// For example, the JSON key "$schema" would be mis-expanded by bash as the (usually
// unset) variable $schema, producing an empty string. Writing \$schema instead causes
// bash to treat the dollar sign as a literal character.
//
// ${{ }} expressions are intentionally left untouched: GitHub Actions resolves them
// at the YAML evaluation layer, before the shell runs, so they must remain verbatim
// in the script text. Any other $ — including $varname, ${varname}, and $0-$9
// positional parameters — is escaped.
func escapeBareShellDollarSigns(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	for i := range len(s) {
		if s[i] != '$' {
			result.WriteByte(s[i])
			continue
		}
		// It is a $; check whether it opens a ${{ }} GitHub Actions expression.
		if i+2 < len(s) && s[i+1] == '{' && s[i+2] == '{' {
			// Start of ${{ }}: leave as-is so GitHub Actions can evaluate it.
			result.WriteByte(s[i])
		} else {
			// Bare $: escape to \$ so bash treats it as a literal dollar sign.
			result.WriteString(`\$`)
		}
	}
	return result.String()
}

// shellEscapeArgWithVarPreserved escapes arg for use as a double-quoted shell argument,
// preserving ${{ }} GitHub Actions expressions and a specific ${varName} shell variable
// reference intact while escaping all other bare $ signs.
//
// This is used for the AWF config JSON when a local shell variable (such as
// ${GH_AW_MAX_AI_CREDITS}) has been injected into the JSON: the variable reference
// must survive into the shell so bash can expand it, and any ${{ }} expressions in
// the JSON (e.g. from AllowedDomains) must remain verbatim for GitHub Actions to
// evaluate. All other bare $ signs (e.g. "$schema") are escaped as \$.
func shellEscapeArgWithVarPreserved(arg, varName string) string {
	varRef := "${" + varName + "}"
	escaped := strings.ReplaceAll(arg, `"`, `\"`)
	var result strings.Builder
	result.Grow(len(escaped) + 2)
	result.WriteByte('"')
	for i := range len(escaped) {
		if escaped[i] != '$' {
			result.WriteByte(escaped[i])
			continue
		}
		switch {
		case i+2 < len(escaped) && escaped[i+1] == '{' && escaped[i+2] == '{':
			// ${{ }}: GitHub Actions expression — leave as-is so GA can evaluate it.
			result.WriteByte(escaped[i])
		case strings.HasPrefix(escaped[i:], varRef):
			// ${varName}: shell variable reference — leave as-is so bash expands it.
			result.WriteByte(escaped[i])
		default:
			// Bare $: escape to \$ so bash treats it as a literal dollar sign.
			result.WriteString(`\$`)
		}
	}
	result.WriteByte('"')
	return result.String()
}

// that allows ${VAR_NAME} variables to be expanded at runtime.
func buildDockerCommandWithExpandableVars(cmd string) string {
	shellLog.Printf("Building docker command with expandable vars (length: %d)", len(cmd))
	// Find all ${VAR_NAME} patterns that need expansion outside of single quotes.
	// We want: 'docker run ... -v '"${GITHUB_WORKSPACE}"':'"${GITHUB_WORKSPACE}"':rw ...'
	// This closes the single quote, adds the variable in double quotes, then reopens single quote.

	// Collect all unique variable references
	expandableVars := findExpandableVars(cmd)

	if len(expandableVars) == 0 {
		shellLog.Print("No expandable variables found, using normal escaping")
		return shellEscapeArg(cmd)
	}

	shellLog.Printf("Docker command built with expandable variables: %v", expandableVars)

	// Process the command: wrap in single quotes, break out for each variable
	var result strings.Builder
	result.WriteString("'")
	remaining := cmd
	for len(remaining) > 0 {
		// Find the next variable reference
		nextIdx := -1
		nextVar := ""
		for _, v := range expandableVars {
			idx := strings.Index(remaining, v)
			if idx >= 0 && (nextIdx < 0 || idx < nextIdx) {
				nextIdx = idx
				nextVar = v
			}
		}
		if nextIdx < 0 {
			// No more variables, write the rest
			escapedPart := strings.ReplaceAll(remaining, "'", "'\\''")
			result.WriteString(escapedPart)
			break
		}
		// Write text before the variable
		before := remaining[:nextIdx]
		escapedBefore := strings.ReplaceAll(before, "'", "'\\''")
		result.WriteString(escapedBefore)
		// Break out of single quotes, add variable in double quotes, reopen single quotes
		result.WriteString("'\"" + nextVar + "\"'")
		remaining = remaining[nextIdx+len(nextVar):]
	}
	result.WriteString("'")
	return result.String()
}

// findExpandableVars returns all unique ${VAR_NAME} patterns in the string.
// It intentionally skips ${{ }} GitHub Actions expressions: those are evaluated
// by the GH Actions runner before the shell runs and must remain intact inside
// single-quoted strings — they should NOT be broken out as shell variables.
func findExpandableVars(s string) []string {
	var vars []string
	seen := make(map[string]struct {
	})
	for {
		start := strings.Index(s, "${")
		if start < 0 {
			break
		}
		// Skip GitHub Actions expressions (${{ ... }}): they start with ${{ and
		// must not be treated as shell variable references.
		if start+2 < len(s) && s[start+2] == '{' {
			s = s[start+3:]
			continue
		}
		end := strings.Index(s[start:], "}")
		if end < 0 {
			break
		}
		varRef := s[start : start+end+1]
		if !hasStringKey(seen, varRef) {
			seen[varRef] = struct {
			}{}
			vars = append(vars, varRef)
		}
		s = s[start+end+1:]
	}
	return vars
}
