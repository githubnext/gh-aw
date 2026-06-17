package regexpcompileinfunction

import . "regexp"

// NOT flagged: dot imports produce bare identifier calls, not selector expressions,
// so isRegexpCompileCall exits early at the *ast.SelectorExpr guard.
// This is a known limitation of the linter.
func DotImportExample() bool {
	r := MustCompile(`^[a-z]+$`) // not flagged: dot-import calls are not selector expressions
	return r.MatchString("abc")
}
