// Package deferinloop implements a Go analysis linter that flags defer
// statements placed directly inside for or range loop bodies. A defer inside
// a loop does not execute at the end of each iteration — it runs when the
// enclosing function returns, which can cause resource leaks and unexpected
// cleanup ordering.
package deferinloop

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the defer-in-loop analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "deferinloop",
	Doc:      "reports defer statements enclosed anywhere within a for or range loop body; a function literal between a defer and an enclosing loop is treated as a new scope boundary, making the defer exempt; test files are not checked",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/deferinloop",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "deferinloop")

	for cur := range insp.Root().Preorder((*ast.DeferStmt)(nil)) {
		deferStmt, ok := cur.Node().(*ast.DeferStmt)
		if !ok {
			continue
		}

		pos := pass.Fset.PositionFor(deferStmt.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			continue
		}

		if !isInsideLoop(cur) {
			continue
		}

		pass.ReportRangef(deferStmt,
			"defer inside a loop does not execute at the end of each iteration; it runs when the enclosing function returns, which can cause resource leaks")
	}

	return nil, nil
}

// isInsideLoop reports whether cur (a DeferStmt) is enclosed anywhere within a
// for or range loop body, without crossing a function literal boundary.
// Defers inside func literals are exempt because they form a new function scope
// and execute when the literal returns, not the outer function.
func isInsideLoop(cur inspector.Cursor) bool {
	for encl := range cur.Enclosing(
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.FuncLit)(nil),
	) {
		switch encl.Node().(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			return true
		case *ast.FuncLit:
			return false
		}
	}
	return false
}
