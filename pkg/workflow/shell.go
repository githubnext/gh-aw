package workflow

import "strings"

// shellJoinArgs joins command arguments with proper shell escaping
// Arguments containing special characters are wrapped in single quotes
func shellJoinArgs(args []string) string {
	var escapedArgs []string
	for _, arg := range args {
		escapedArgs = append(escapedArgs, shellEscapeArg(arg))
	}
	return strings.Join(escapedArgs, " ")
}

// shellEscapeArg escapes a single argument for safe use in shell commands
// Arguments containing special characters are wrapped in single quotes
func shellEscapeArg(arg string) string {
	// If the argument is already properly quoted with double quotes, leave it as-is
	if len(arg) >= 2 && arg[0] == '"' && arg[len(arg)-1] == '"' {
		return arg
	}

	// If the argument is already properly quoted with single quotes, leave it as-is
	if len(arg) >= 2 && arg[0] == '\'' && arg[len(arg)-1] == '\'' {
		return arg
	}

	// Check if the argument contains special shell characters that need escaping
	if strings.ContainsAny(arg, "()[]{}*?$`\"'\\|&;<> \t\n") {
		// Handle single quotes in the argument by escaping them
		escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
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
	// Escape backslashes first (must be done before other escapes)
	escaped := strings.ReplaceAll(cmd, "\\", "\\\\")
	// Escape double quotes
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	// Escape dollar signs (to prevent variable expansion)
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	// Escape backticks (to prevent command substitution)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")

	// Wrap in double quotes
	return "\"" + escaped + "\""
}
