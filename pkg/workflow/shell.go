package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var shellLog = logger.New("workflow:shell")

// shellJoinArgs joins command arguments with proper shell escaping
// Arguments containing special characters are wrapped in single quotes
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

// shellEscapeArg escapes a single argument for safe use in shell commands
// Arguments containing special characters are wrapped in single quotes
func shellEscapeArg(arg string) string {
	// If the argument is already properly quoted with double quotes, leave it as-is
	if len(arg) >= 2 && arg[0] == '"' && arg[len(arg)-1] == '"' {
		shellLog.Print("Argument already double-quoted, leaving as-is")
		return arg
	}

	// If the argument is already properly quoted with single quotes, leave it as-is
	if len(arg) >= 2 && arg[0] == '\'' && arg[len(arg)-1] == '\'' {
		shellLog.Print("Argument already single-quoted, leaving as-is")
		return arg
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

// shellEscapeCommandString escapes a complete command string (which may already contain
// quoted arguments) for passing as a single argument to another command.
// It wraps the command in double quotes and escapes any double quotes, dollar signs,
// backticks, and backslashes within the command.
// This is useful when passing a command to wrapper programs like awf that expect
// the command as a single quoted argument.
func shellEscapeCommandString(cmd string) string {
	shellLog.Printf("Escaping command string (length: %d)", len(cmd))
	// Escape backslashes first (must be done before other escapes)
	escaped := strings.ReplaceAll(cmd, "\\", "\\\\")
	// Escape double quotes
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	// Escape dollar signs (to prevent variable expansion)
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	// Escape backticks (to prevent command substitution)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")

	shellLog.Print("Command string escaped successfully")
	// Wrap in double quotes
	return "\"" + escaped + "\""
}
