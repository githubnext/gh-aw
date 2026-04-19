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

// buildDockerCommandWithExpandableVars builds a properly quoted docker command
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
func findExpandableVars(s string) []string {
	var vars []string
	seen := make(map[string]bool)
	for {
		start := strings.Index(s, "${")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], "}")
		if end < 0 {
			break
		}
		varRef := s[start : start+end+1]
		if !seen[varRef] {
			seen[varRef] = true
			vars = append(vars, varRef)
		}
		s = s[start+end+1:]
	}
	return vars
}
