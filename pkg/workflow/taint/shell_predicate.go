package taint

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Numeric comparisons and malformed tests can print their operands on stderr.
// Only model arities/operators whose result does not disclose operand values.
func safeShellPredicate(name string, args []*syntax.Word) bool {
	for _, arg := range args {
		if !singlePredicateWord(arg) {
			return false
		}
	}
	if name == "[" {
		last, ok := sliceAt(args, len(args)-1)
		if !ok || literalShellWord(last) != "]" {
			return false
		}
		args = args[:len(args)-1]
	}
	first, ok := sliceAt(args, 0)
	if !ok {
		return true
	}
	if len(args) > 1 && literalShellWord(first) == "!" {
		return safeShellPredicate("test", args[1:])
	}
	switch len(args) {
	case 1:
		return true
	case 2:
		switch literalShellWord(first) {
		case "-n", "-z", "-a", "-b", "-c", "-d", "-e", "-f", "-g", "-h",
			"-k", "-L", "-N", "-O", "-G", "-p", "-r", "-s", "-S", "-u", "-w", "-x":
			return true
		}
	case 3:
		operator, ok := sliceAt(args, 1)
		if !ok {
			return false
		}
		switch literalShellWord(operator) {
		case "=", "==", "!=", "<", ">", "-ef", "-nt", "-ot":
			return true
		}
	}
	return false
}

func literalShellWord(word *syntax.Word) string {
	texts, known := wordTexts(word, nil, nil, nil)
	text, ok := sliceAt(texts, 0)
	if known && ok && len(texts) == 1 {
		return text
	}
	return ""
}

func singlePredicateWord(word *syntax.Word) bool {
	for _, part := range word.Parts {
		switch part := part.(type) {
		case *syntax.Lit:
			if strings.ContainsAny(part.Value, "*?[") {
				return false
			}
		case *syntax.SglQuoted:
		case *syntax.DblQuoted:
			if !singleQuotedPredicate(part) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func singleQuotedPredicate(word *syntax.DblQuoted) bool {
	single := true
	syntax.Walk(word, func(node syntax.Node) bool {
		if _, ok := node.(*syntax.CmdSubst); ok {
			return false
		}
		if param, ok := node.(*syntax.ParamExp); ok {
			if param.Param != nil && param.Param.Value == "@" || param.Names != 0 {
				single = false
			}
			if index, ok := param.Index.(*syntax.Word); ok && index.Lit() == "@" {
				single = false
			}
		}
		return single
	})
	return single
}
