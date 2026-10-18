package taint

import (
	"fmt"
	"maps"
	"path/filepath"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type shell struct {
	a         *analyzer
	path      string
	step      string
	bindings  scope
	persisted scope
	files     scope
	vars      scope
	markers   scope
	functions map[string]*syntax.Stmt
	active    map[string]bool
	trace     bool
	constants map[string]string
}

func (a *analyzer) script(text string, fields map[string]any, path string, bindings, persisted, files scope) error {
	s := &shell{
		a: a, path: path, bindings: bindings, persisted: persisted,
		files: files, vars: make(scope), markers: make(scope),
		functions: make(map[string]*syntax.Stmt), active: make(map[string]bool),
		constants: make(map[string]string),
	}
	id, err := optionalField[string](fields, "id", path)
	if err != nil {
		return err
	}
	s.step = id
	for key, sources := range bindings {
		if name, ok := strings.CutPrefix(key, "shell."); ok {
			s.vars[name] = sources
		}
	}
	rendered, _, err := a.interpolate(text, path, bindings, func(i int, sources value) string {
		marker := fmt.Sprintf("GH_AW_TAINT_PLACEHOLDER_%d_", i)
		// A unique marker remains tainted even inside a single-quoted string:
		// Actions interpolation happens before the shell parses its quotes.
		for suffix := 0; strings.Contains(text, marker); suffix++ {
			marker = fmt.Sprintf("GH_AW_TAINT_PLACEHOLDER_%d_%d_", i, suffix)
		}
		s.markers[marker] = sources
		return marker
	})
	if err != nil {
		return err
	}
	shellName, err := optionalField[string](fields, "shell", path)
	if err != nil {
		return err
	}
	if shellName != "" && shellName != "bash" && shellName != "sh" &&
		!strings.HasPrefix(shellName, "bash ") && !strings.HasPrefix(shellName, "sh ") {
		a.gap(path, "only bash and sh run scripts are modeled")
		return nil
	}
	tree, parseErr := syntax.NewParser().Parse(strings.NewReader(rendered), "")
	if parseErr != nil {
		a.gap(path, "shell syntax could not be analyzed")
		return nil
	}
	// Monotone local assignments keep both sides of branches and loop-carried
	// values. Iterate to a fixed point; do not clear taint on reassignment.
	for {
		before := s.size()
		s.stmts(tree.Stmts, false, nil)
		if s.size() == before {
			break
		}
	}
	return nil
}

func (s *shell) size() int {
	n := 0
	for _, values := range []scope{s.vars, s.files, s.persisted, s.bindings} {
		for _, sources := range values {
			n += len(sources)
		}
	}
	return n
}

func (s *shell) word(word *syntax.Word) value {
	if word == nil {
		return nil
	}
	var result value
	syntax.Walk(word, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.Lit:
			for marker, sources := range s.markers {
				if strings.Contains(n.Value, marker) {
					result = union(result, sources)
				}
			}
		case *syntax.SglQuoted:
			for marker, sources := range s.markers {
				if strings.Contains(n.Value, marker) {
					result = union(result, sources)
				}
			}
		case *syntax.ParamExp:
			if n.Param != nil {
				result = union(result, s.vars[n.Param.Value])
				if n.Excl {
					// Dynamic indirection can select any in-scope value.
					for _, sources := range s.vars {
						result = union(result, sources)
					}
				}
			}
		case *syntax.CmdSubst:
			child := *s
			child.vars = maps.Clone(s.vars)
			child.constants = maps.Clone(s.constants)
			result = union(result, child.stmts(n.Stmts, true, nil))
			return false
		}
		return true
	})
	return result
}

func (s *shell) assign(assign *syntax.Assign) {
	if assign.Name == nil {
		return
	}
	assigned := s.word(assign.Value)
	s.vars[assign.Name.Value] = union(s.vars[assign.Name.Value], assigned)
	if _, exists := s.constants[assign.Name.Value]; !exists && len(assigned) == 0 {
		if text := wordText(assign.Value, s.constants); !strings.Contains(text, "$") {
			s.constants[assign.Name.Value] = text
		}
	}
	if assign.Array != nil {
		for _, elem := range assign.Array.Elems {
			s.vars[assign.Name.Value] = union(s.vars[assign.Name.Value], s.word(elem.Value))
		}
	}
}

func (s *shell) stmts(stmts []*syntax.Stmt, captured bool, stdin value) value {
	var result value
	for _, stmt := range stmts {
		result = union(result, s.stmt(stmt, captured, stdin))
	}
	return result
}

func (s *shell) stmt(stmt *syntax.Stmt, captured bool, stdin value) value {
	redirected := false
	for _, redir := range stmt.Redirs {
		switch redir.Op.String() {
		case ">", ">>", ">&", "&>", "&>>":
			redirected = redirected || redir.N == nil || redir.N.Value == "1" || strings.HasPrefix(redir.Op.String(), "&")
		case "<":
			stdin = union(stdin, s.files[wordText(redir.Word, s.constants)])
		case "<<", "<<-", "<<<": //nolint:generatedyamlheredoc // This analyzes authored heredocs; it does not generate shell.
			stdin = union(stdin, s.word(redir.Hdoc), s.word(redir.Word))
		}
	}
	out := s.command(stmt.Cmd, captured || redirected, stdin)
	for _, redir := range stmt.Redirs {
		switch redir.Op.String() {
		case ">", ">>", "&>", "&>>":
			if redir.N == nil || redir.N.Value == "1" || strings.HasPrefix(redir.Op.String(), "&") {
				s.write(wordText(redir.Word, s.constants), out, commandText(stmt.Cmd), int(stmt.Pos().Line()))
			}
		case ">&":
			if staticWord(redir.Word) == "2" {
				s.a.emit(Site{Kind: Log, Path: s.path, Line: int(stmt.Pos().Line())}, out)
			}
		}
	}
	if redirected {
		return nil
	}
	return out
}

func (s *shell) command(command syntax.Command, captured bool, stdin value) value { //nolint:largefunc // Shell node cases share one stream and scope model.
	switch cmd := command.(type) {
	case *syntax.CallExpr:
		return s.call(cmd, captured, stdin)
	case *syntax.DeclClause:
		for _, assign := range cmd.Args {
			s.assign(assign)
		}
	case *syntax.Block:
		return s.stmts(cmd.Stmts, captured, stdin)
	case *syntax.Subshell:
		child := *s
		child.vars = maps.Clone(s.vars)
		child.constants = maps.Clone(s.constants)
		return child.stmts(cmd.Stmts, captured, stdin)
	case *syntax.BinaryCmd:
		if cmd.Op.String() == "|" || cmd.Op.String() == "|&" {
			left := s.stmt(cmd.X, true, stdin)
			return s.stmt(cmd.Y, captured, left)
		}
		return union(s.stmt(cmd.X, captured, stdin), s.stmt(cmd.Y, captured, stdin))
	case *syntax.IfClause:
		result := union(s.stmts(cmd.Cond, captured, stdin), s.stmts(cmd.Then, captured, stdin))
		if cmd.Else != nil {
			result = union(result, s.command(cmd.Else, captured, stdin))
		}
		return result
	case *syntax.ForClause:
		if loop, ok := cmd.Loop.(*syntax.WordIter); ok {
			for _, item := range loop.Items {
				s.vars[loop.Name.Value] = union(s.vars[loop.Name.Value], s.word(item))
			}
		}
		return s.stmts(cmd.Do, captured, stdin)
	case *syntax.WhileClause:
		return union(s.stmts(cmd.Cond, captured, stdin), s.stmts(cmd.Do, captured, stdin))
	case *syntax.CaseClause:
		var result value
		for _, item := range cmd.Items {
			result = union(result, s.stmts(item.Stmts, captured, stdin))
		}
		return result
	case *syntax.FuncDecl:
		if cmd.Name != nil {
			s.functions[cmd.Name.Value] = cmd.Body
		}
	}
	return nil
}

func (s *shell) call(cmd *syntax.CallExpr, captured bool, stdin value) value { //nolint:largefunc // Keep command-specific stream semantics together.
	if len(cmd.Args) == 0 {
		for _, assign := range cmd.Assigns {
			if s.trace {
				s.a.emit(Site{Kind: Log, Path: s.path, Line: int(assign.Pos().Line())}, s.word(assign.Value))
			}
			s.assign(assign)
		}
		return nil
	}
	name := ""
	var args value
	for index, arg := range cmd.Args {
		if index == 0 {
			name = staticWord(arg)
		} else {
			args = union(args, s.word(arg))
		}
	}
	line := int(cmd.Pos().Line())
	site := Site{Kind: Log, Path: s.path, Line: line}
	if s.trace {
		s.a.emit(site, args)
		for _, assign := range cmd.Assigns {
			s.a.emit(site, s.word(assign.Value))
		}
	}
	if body := s.functions[name]; body != nil {
		return s.function(name, body, cmd.Args[1:], captured, stdin)
	}
	var out value
	switch name {
	case "echo", "printf":
		out = args
		for index, arg := range cmd.Args {
			if index == 1 && strings.HasPrefix(staticWord(arg), "::add-mask::") {
				return nil
			}
		}
	case "env", "printenv":
		if len(cmd.Args) == 1 || name == "env" {
			for _, sources := range s.vars {
				out = union(out, sources)
			}
			for _, assign := range cmd.Assigns {
				out = union(out, s.word(assign.Value))
			}
		} else {
			for _, arg := range cmd.Args[1:] {
				out = union(out, s.vars[staticWord(arg)])
			}
		}
	case "cat", "head", "tail", "base64", "openssl", "jq", "sed", "awk", "tr", "cut", "tee":
		out = union(args, stdin)
		for _, arg := range cmd.Args[1:] {
			out = union(out, s.files[wordText(arg, s.constants)])
		}
		if name == "tee" {
			for _, arg := range cmd.Args[1:] {
				if target := wordText(arg, s.constants); target != "" && !strings.HasPrefix(target, "-") {
					s.write(target, stdin, "", line)
				}
			}
		}
	case "curl", "wget":
		s.network(cmd.Args[1:], line)
	case "set":
		if len(cmd.Args) == 1 {
			for _, sources := range s.vars {
				out = union(out, sources)
			}
		}
		previous := ""
		for _, arg := range cmd.Args[1:] {
			flag := staticWord(arg)
			if strings.HasPrefix(flag, "-") && strings.Contains(flag, "x") ||
				previous == "-o" && flag == "xtrace" {
				s.trace = true
			}
			previous = flag
		}
	default:
		if len(args)+len(stdin) > 0 {
			s.a.gap(s.path, "external command data flow is not modeled")
		}
	}
	if !captured {
		s.a.emit(site, out)
	}
	return out
}

func (s *shell) network(args []*syntax.Word, line int) {
	credential, header := false, false
	for _, word := range args {
		arg := staticWord(word)
		if credential {
			credential = false
			continue
		}
		if header {
			header = false
			if !authenticationHeader(arg) {
				s.a.emit(Site{Kind: Network, Path: s.path, Line: line}, s.word(word))
			}
			continue
		}
		if arg == "-H" || arg == "--header" {
			header = true
			continue
		}
		if arg == "-u" || arg == "--user" || arg == "--oauth2-bearer" || arg == "--password" || arg == "--http-password" {
			credential = true
			continue
		}
		if text, ok := strings.CutPrefix(arg, "--header="); ok && authenticationHeader(text) {
			continue
		}
		if strings.HasPrefix(arg, "--user=") || strings.HasPrefix(arg, "--oauth2-bearer=") ||
			strings.HasPrefix(arg, "--password=") || strings.HasPrefix(arg, "--http-password=") {
			continue
		}
		s.a.emit(Site{Kind: Network, Path: s.path, Line: line}, s.word(word))
	}
}

func authenticationHeader(header string) bool {
	name, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(header)), ":")
	switch name {
	case "authorization", "proxy-authorization", "x-api-key", "api-key", "x-goog-api-key", "private-token":
		return true
	default:
		return false
	}
}

func (s *shell) function(name string, body *syntax.Stmt, args []*syntax.Word, captured bool, stdin value) value {
	if s.active[name] {
		s.a.gap(s.path, "recursive shell function calls are not modeled")
		return nil
	}
	previous := maps.Clone(s.vars)
	for i, arg := range args {
		s.vars[strconv.Itoa(i+1)] = s.word(arg)
	}
	s.active[name] = true
	out := s.stmt(body, captured, stdin)
	delete(s.active, name)
	for i := range args {
		s.vars[strconv.Itoa(i+1)] = previous[strconv.Itoa(i+1)]
	}
	return out
}

func staticWord(word *syntax.Word) string {
	return wordText(word, nil)
}

func wordText(word *syntax.Word, constants map[string]string) string {
	if word == nil {
		return ""
	}
	var text strings.Builder
	syntax.Walk(word, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.Lit:
			text.WriteString(n.Value)
		case *syntax.SglQuoted:
			text.WriteString(n.Value)
		case *syntax.ParamExp:
			if n.Param != nil {
				if literal, exists := constants[n.Param.Value]; exists {
					text.WriteString(literal)
				} else {
					text.WriteString("$" + n.Param.Value)
				}
			}
			return false
		case *syntax.CmdSubst:
			return false
		}
		return true
	})
	return text.String()
}

func commandText(command syntax.Command) string {
	if call, ok := command.(*syntax.CallExpr); ok {
		var parts []string
		for _, arg := range call.Args[1:] {
			parts = append(parts, staticWord(arg))
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func (s *shell) write(target string, sources value, payload string, line int) {
	site := Site{Path: s.path, Line: line}
	switch target {
	case "$GITHUB_STEP_SUMMARY":
		site.Kind = Summary
		s.a.emit(site, sources)
	case "$GITHUB_OUTPUT", "$GITHUB_ENV":
		name, _, found := strings.Cut(payload, "=")
		if !found {
			name, _, _ = strings.Cut(payload, "<<")
		}
		name = strings.TrimSpace(name)
		if target == "$GITHUB_OUTPUT" {
			site.Kind = Output
			s.a.emit(site, sources)
			if name == "" {
				name = "*"
			}
			if s.step != "" {
				key := "steps." + strings.ToLower(s.step) + ".outputs." + strings.ToLower(name)
				s.bindings[key] = union(s.bindings[key], sources)
			}
		} else if name != "" {
			key := "env." + strings.ToLower(name)
			s.persisted[key] = union(s.persisted[key], sources)
			s.persisted["shell."+name] = union(s.persisted["shell."+name], sources)
		} else if len(sources) > 0 {
			s.a.gap(s.path, "dynamic GITHUB_ENV names are not modeled")
		}
	case "", "/dev/null":
	default:
		s.files[target] = union(s.files[target], sources)
	}
}

func artifactContains(pattern, file string) bool {
	if pattern == "" || strings.HasPrefix(pattern, "!") {
		return false
	}
	if pattern == "." || pattern == "**" || pattern == "**/*" || pattern == file ||
		strings.HasPrefix(file, strings.TrimSuffix(pattern, "/")+"/") {
		return true
	}
	matched, err := filepath.Match(pattern, file)
	return err == nil && matched
}
