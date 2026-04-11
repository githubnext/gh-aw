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
	if containsGitHubActionsExpression(arg) {
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

// containsGitHubActionsExpression checks if a string contains GitHub Actions
// expressions (${{ ... }}). It verifies that ${{ appears before }}.
func containsGitHubActionsExpression(s string) bool {
	openIdx := strings.Index(s, "${{")
	if openIdx < 0 {
		return false
	}
	return strings.Contains(s[openIdx:], "}}")
}

// buildDockerCommandWithExpandableVars builds a properly quoted docker command
// that allows ${GITHUB_WORKSPACE} and $GITHUB_WORKSPACE to be expanded at runtime
func buildDockerCommandWithExpandableVars(cmd string) string {
	shellLog.Printf("Building docker command with expandable vars (length: %d)", len(cmd))
	// Replace ${GITHUB_WORKSPACE} with a placeholder that we'll handle specially
	// We want: 'docker run ... -v '"${GITHUB_WORKSPACE}"':'"${GITHUB_WORKSPACE}"':rw ...'
	// This closes the single quote, adds the variable in double quotes, then reopens single quote

	// Split on ${GITHUB_WORKSPACE} to handle it specially
	if strings.Contains(cmd, "${GITHUB_WORKSPACE}") {
		parts := strings.Split(cmd, "${GITHUB_WORKSPACE}")
		var result strings.Builder
		result.WriteString("'")
		for i, part := range parts {
			if i > 0 {
				// Add the variable expansion outside of single quotes
				result.WriteString("'\"${GITHUB_WORKSPACE}\"'")
			}
			// Escape single quotes in the part
			escapedPart := strings.ReplaceAll(part, "'", "'\\''")
			result.WriteString(escapedPart)
		}
		result.WriteString("'")
		shellLog.Print("Docker command built with expandable GITHUB_WORKSPACE variables")
		return result.String()
	}

	// No GITHUB_WORKSPACE variable, use normal quoting
	shellLog.Print("No GITHUB_WORKSPACE variable found, using normal escaping")
	return shellEscapeArg(cmd)
}
