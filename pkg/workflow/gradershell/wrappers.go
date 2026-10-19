package gradershell

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func unwrapCommand(name string, args []*syntax.Word) (tail []*syntax.Word, query, known bool) {
	for i := 0; i < len(args); i++ {
		value, static := literalWord(args[i]) //nolint:uncheckedsliceindex // The nonnegative cursor is rechecked by the loop after option-operand skips.
		if !static {
			return args[i:], false, true // The command checker reports the dynamic name.
		}
		if value == "--" {
			return args[i+1:], false, true
		}
		if name == "env" && strings.Contains(value, "=") && !strings.HasPrefix(value, "-") {
			continue
		}
		if !strings.HasPrefix(value, "-") || value == "-" {
			return args[i:], false, true
		}
		switch name {
		case "command":
			if value == "-v" || value == "-V" {
				return nil, true, true
			}
			if value != "-p" {
				return nil, false, false
			}
		case "exec":
			switch value {
			case "-a":
				i++
			case "-c", "-l", "-cl", "-lc":
			default:
				return nil, false, false
			}
		case "env":
			switch {
			case value == "-i", value == "--ignore-environment":
			case value == "-u", value == "--unset", value == "-C", value == "--chdir":
				i++
			case strings.HasPrefix(value, "--unset="), strings.HasPrefix(value, "--chdir="):
			default:
				return nil, false, false
			}
		case "sudo":
			switch value {
			case "-u", "--user", "-g", "--group", "-h", "--host":
				i++
			case "-n", "-E", "-H", "--non-interactive", "--preserve-env", "--set-home":
			default:
				return nil, false, false
			}
		case "nice":
			if value == "-n" || value == "--adjustment" {
				i++
			} else if !strings.HasPrefix(value, "--adjustment=") {
				return nil, false, false
			}
		default:
			return nil, false, false
		}
	}
	return nil, false, true
}
