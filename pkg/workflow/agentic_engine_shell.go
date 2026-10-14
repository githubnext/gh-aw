package workflow

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

const maxAgenticEngineShellDepth = 8

var (
	agenticEngineBinaryPattern         = regexp.MustCompile(`(?i)^(?:copilot|claude|codex|gemini|agy|pi)(?:\.exe)?$`)
	agenticEnginePackagePattern        = regexp.MustCompile(`(?i)(?:^|=)@(?:github/copilot|anthropic-ai/claude-code|openai/codex|google/gemini-cli|earendil-works/pi-coding-agent)(?:@[^\s]+)?$`)
	agenticEnginePackageInstallPattern = regexp.MustCompile(`(?i)^(?:install|i|add|exec|dlx|x)$`)
	agenticEngineAssignmentPattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	agenticEngineInstallerURLPattern   = regexp.MustCompile(`(?i)(?:github\.com/github/copilot-cli/(?:releases|raw)|raw\.githubusercontent\.com/github/copilot-cli/|claude\.ai/install\.(?:sh|ps1))`)
	agenticEngineParserErrorPattern    = regexp.MustCompile(`(?i)\b(?:copilot|claude|codex|gemini|agy|pi)(?:\.exe)?\b`)
)

func agenticEngineWordValue(source string, word *syntax.Word) string {
	if value := word.Lit(); value != "" {
		return value
	}
	if len(word.Parts) == 1 {
		switch part := word.Parts[0].(type) { //nolint:uncheckedsliceindex // The single-element length is checked above.
		case *syntax.SglQuoted:
			return part.Value
		case *syntax.DblQuoted:
			var value strings.Builder
			for _, quotedPart := range part.Parts {
				switch quotedPart := quotedPart.(type) {
				case *syntax.Lit:
					value.WriteString(quotedPart.Value)
				case *syntax.SglQuoted:
					value.WriteString(quotedPart.Value)
				default:
					return agenticEngineWordSource(source, word)
				}
			}
			return value.String()
		}
	}
	return agenticEngineWordSource(source, word)
}

func agenticEngineWordSource(source string, word *syntax.Word) string {
	start, end := int(word.Pos().Offset()), int(word.End().Offset())
	if start < 0 || end < start || end > len(source) {
		return ""
	}
	return strings.Trim(source[start:end], `"'`)
}

func agenticEngineCommandName(word string) string {
	word = strings.Trim(strings.TrimSpace(word), `"'`)
	word = strings.ReplaceAll(word, `\`, "/")
	name := strings.ToLower(path.Base(word))
	return strings.TrimSuffix(name, ".exe")
}

func agenticEngineCommandWords(source string, call *syntax.CallExpr) []string {
	words := make([]string, 0, len(call.Args))
	for _, arg := range call.Args {
		words = append(words, agenticEngineWordValue(source, arg))
	}
	return words
}

func agenticEngineWrappedCommandIndex(command string, args []string) int {
	optionValues := map[string]map[string]struct{}{
		"sudo":    {"-u": {}, "--user": {}, "-g": {}, "--group": {}, "-h": {}, "--host": {}, "-p": {}, "--prompt": {}, "-C": {}, "--close-from": {}, "-D": {}, "--chdir": {}},
		"env":     {"-u": {}, "--unset": {}, "-C": {}, "--chdir": {}, "-S": {}, "--split-string": {}},
		"timeout": {"-k": {}, "--kill-after": {}, "-s": {}, "--signal": {}},
		"time":    {"-f": {}, "--format": {}},
		"exec":    {"-a": {}, "--argv0": {}},
		"command": {"-a": {}, "--argv0": {}},
	}
	options := optionValues[command]
	for i := 0; i < len(args); i++ {
		arg := strings.Trim(args[i], `"'`) //nolint:uncheckedsliceindex // The loop bounds i by len(args).
		if arg == "--" {
			return i + 1
		}
		if command == "env" && agenticEngineAssignmentPattern.MatchString(arg) {
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			if command == "timeout" {
				return i + 1
			}
			return i
		}
		if _, ok := options[arg]; ok && i+1 < len(args) {
			i++
		}
	}
	return len(args)
}

func agenticEngineScriptArgument(args []string) (int, bool) {
	for i := 1; i < len(args); i++ {
		option := strings.Trim(args[i], `"'`) //nolint:uncheckedsliceindex // The loop bounds i by len(args).
		if option == "--" {
			return -1, false
		}
		switch option {
		case "--rcfile", "--init-file", "-o", "-O":
			i++
			continue
		}
		if strings.HasPrefix(option, "--") {
			continue
		}
		if strings.HasPrefix(option, "-") {
			flags := option[1:]
			for flagIndex, flag := range flags {
				if flag == 'c' {
					script := i + 1
					if script < len(args) && strings.Trim(args[script], `"'`) == "--" { //nolint:uncheckedsliceindex // The condition checks script against len(args).
						script++
					}
					return script, script < len(args)
				}
				if flag == 'o' || flag == 'O' {
					if flagIndex == len(flags)-1 {
						i++
					}
					break
				}
			}
		}
	}
	return -1, false
}

func agenticEngineCommandInvokes(words []string, depth int) bool {
	if len(words) == 0 {
		return false
	}
	command := agenticEngineCommandName(words[0]) //nolint:uncheckedsliceindex // The empty slice is rejected above.
	args := words[1:]
	if agenticEngineBinaryPattern.MatchString(command) {
		return true
	}
	if command == "command" && len(args) > 0 && (args[0] == "-v" || args[0] == "-V") { //nolint:uncheckedsliceindex // The condition checks len(args).
		return false
	}
	switch command {
	case "env":
		for i, arg := range args {
			if (arg == "-S" || arg == "--split-string") && i+1 < len(args) {
				return agenticEngineScriptInvokes(args[i+1], depth+1) //nolint:uncheckedsliceindex // The condition checks i+1 against len(args).
			}
		}
		index := agenticEngineWrappedCommandIndex(command, args)
		return index < len(args) && agenticEngineCommandInvokes(args[index:], depth+1)
	case "sudo", "time", "timeout", "exec", "command":
		index := agenticEngineWrappedCommandIndex(command, args)
		if index < len(args) {
			return agenticEngineCommandInvokes(args[index:], depth+1)
		}
	case "bash", "sh":
		if depth >= maxAgenticEngineShellDepth {
			return false
		}
		index, ok := agenticEngineScriptArgument(words)
		return ok && agenticEngineScriptInvokes(words[index], depth+1) //nolint:uncheckedsliceindex // The helper reports ok only for a valid index.
	}
	return agenticEngineNamedCommandInvokes(command, args, depth)
}

func agenticEngineNamedCommandInvokes(command string, args []string, depth int) bool {
	switch command {
	case "npx":
		return slices.ContainsFunc(args, func(arg string) bool {
			arg = strings.Trim(arg, `"'`)
			return agenticEnginePackagePattern.MatchString(arg) ||
				agenticEngineBinaryPattern.MatchString(agenticEngineCommandName(arg))
		})
	case "npm", "pnpm", "yarn", "bun":
		return agenticEnginePackageManagerInvokes(command, args, depth)
	case "gh":
		return agenticEngineGHCommandInvokes(args)
	case "curl", "wget":
		return slices.ContainsFunc(args, func(arg string) bool {
			return agenticEngineInstallerURLPattern.MatchString(arg)
		})
	case "brew":
		if len(args) > 0 && strings.EqualFold(strings.Trim(args[0], `"'`), "install") { //nolint:uncheckedsliceindex // The condition checks len(args).
			return slices.ContainsFunc(args[1:], func(arg string) bool {
				switch strings.Trim(arg, `"'`) {
				case "copilot-cli", "claude-code", "codex":
					return true
				default:
					return false
				}
			})
		}
	}
	return false
}

func agenticEnginePackageManagerInvokes(command string, args []string, depth int) bool {
	installIndex := 0
	if command == "yarn" && len(args) > 0 && strings.EqualFold(strings.Trim(args[0], `"'`), "global") { //nolint:uncheckedsliceindex // The condition checks len(args).
		installIndex++
	}
	if installIndex >= len(args) {
		return false
	}
	operation := strings.Trim(args[installIndex], `"'`) //nolint:uncheckedsliceindex // installIndex is checked against len(args) above.
	if !agenticEnginePackageInstallPattern.MatchString(operation) {
		return false
	}
	if slices.ContainsFunc(args[installIndex+1:], func(arg string) bool {
		return agenticEnginePackagePattern.MatchString(strings.Trim(arg, `"'`))
	}) {
		return true
	}
	if operation == "exec" || operation == "dlx" || operation == "x" {
		commandArgs := args[installIndex+1:]
		index := agenticEngineWrappedCommandIndex("env", commandArgs)
		return index < len(commandArgs) && agenticEngineCommandInvokes(commandArgs[index:], depth+1)
	}
	return false
}

func agenticEngineGHCommandInvokes(args []string) bool {
	if len(args) == 0 {
		return false
	}
	first := strings.Trim(args[0], `"'`) //nolint:uncheckedsliceindex // The empty slice is rejected above.
	if strings.EqualFold(first, "copilot") {
		return true
	}
	return len(args) > 2 && strings.EqualFold(first, "extension") &&
		strings.EqualFold(strings.Trim(args[1], `"'`), "install") && //nolint:uncheckedsliceindex // The condition checks len(args).
		strings.Contains(strings.ToLower(args[2]), "gh-copilot") //nolint:uncheckedsliceindex // The condition checks len(args).
}

func agenticEngineShellInterpreter(source string, node syntax.Node) bool {
	if node == nil {
		return false
	}
	found := false
	syntax.Walk(node, func(node syntax.Node) bool {
		call, ok := node.(*syntax.CallExpr)
		if !ok {
			return true
		}
		words := agenticEngineCommandWords(source, call)
		if len(words) > 0 {
			command := agenticEngineCommandName(words[0])
			if command == "bash" || command == "sh" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func agenticEngineHeredocs(node syntax.Node) []*syntax.Word {
	var heredocs []*syntax.Word
	syntax.Walk(node, func(node syntax.Node) bool {
		stmt, ok := node.(*syntax.Stmt)
		if ok {
			for _, redir := range stmt.Redirs {
				if redir.Hdoc != nil {
					heredocs = append(heredocs, redir.Hdoc)
				}
			}
		}
		return true
	})
	return heredocs
}

func agenticEngineHeredocInvokes(source string, node syntax.Node, depth int) bool {
	if depth >= maxAgenticEngineShellDepth {
		return false
	}
	invokes := false
	syntax.Walk(node, func(node syntax.Node) bool {
		if invokes {
			return false
		}
		switch node := node.(type) {
		case *syntax.Stmt:
			if !agenticEngineShellInterpreter(source, node.Cmd) {
				return true
			}
			for _, redir := range node.Redirs {
				if redir.Hdoc != nil && agenticEngineScriptInvokes(redir.Hdoc.Lit(), depth+1) {
					invokes = true
					return false
				}
			}
		case *syntax.BinaryCmd:
			if node.Op != syntax.Pipe && node.Op != syntax.PipeAll {
				return true
			}
			if !agenticEngineShellInterpreter(source, node.Y) {
				return true
			}
			for _, heredoc := range agenticEngineHeredocs(node.X) {
				if agenticEngineScriptInvokes(heredoc.Lit(), depth+1) {
					invokes = true
					return false
				}
			}
		}
		return true
	})
	return invokes
}

func agenticEngineScriptInvokes(source string, depth int) bool {
	if depth >= maxAgenticEngineShellDepth {
		return false
	}
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := parser.Parse(strings.NewReader(source), "")
	if err != nil {
		return agenticEngineParserErrorPattern.MatchString(source)
	}
	invokes := false
	syntax.Walk(file, func(node syntax.Node) bool {
		if invokes {
			return false
		}
		if call, ok := node.(*syntax.CallExpr); ok &&
			agenticEngineCommandInvokes(agenticEngineCommandWords(source, call), depth) {
			invokes = true
			return false
		}
		return true
	})
	return invokes || agenticEngineHeredocInvokes(source, file, depth)
}

func stepInvokesAgenticEngine(step map[string]any) bool {
	if uses, ok := step["uses"].(string); ok {
		action, _, _ := strings.Cut(strings.ToLower(uses), "@")
		switch action {
		case "anthropics/claude-code-action", "openai/codex-action", "github/copilot-cli-action", "google-github-actions/run-gemini-cli":
			return true
		}
	}
	run, ok := step["run"].(string)
	return ok && agenticEngineScriptInvokes(run, 0)
}
