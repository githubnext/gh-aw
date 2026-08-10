package cli

import "strings"

func shellJoinArgs(args []string) string {
	escapedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		escapedArgs = append(escapedArgs, shellEscapeArg(arg))
	}
	return strings.Join(escapedArgs, " ")
}

func shellEscapeArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if !strings.ContainsAny(arg, "!()[]{}*?$`\"'\\|&;<> \t\r\n") {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}
