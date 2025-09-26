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
	// Check if the argument contains special shell characters that need escaping
	if strings.ContainsAny(arg, "()[]{}*?$`\"'\\|&;<> \t\n") {
		// Handle single quotes in the argument by escaping them
		escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
		return "'" + escaped + "'"
	}
	return arg
}