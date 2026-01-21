package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpUtilitiesLog = logger.New("workflow:mcp_utilities")

// shellQuote adds shell quoting to a string if needed
func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t\n'\"\\$`") {
		// Escape single quotes and wrap in single quotes
		s = strings.ReplaceAll(s, "'", "'\\''")
		return "'" + s + "'"
	}
	return s
}

// buildDockerCommandWithExpandableVars builds a properly quoted docker command
// that allows ${GITHUB_WORKSPACE} and $GITHUB_WORKSPACE to be expanded at runtime
func buildDockerCommandWithExpandableVars(cmd string) string {
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
		return result.String()
	}

	// No GITHUB_WORKSPACE variable, use normal quoting
	return shellQuote(cmd)
}
