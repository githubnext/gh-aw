// Package stringsconcatloop implements a Go analysis linter that flags
// string += concatenation inside for/range loop bodies, which allocates a
// new string copy on every iteration (O(n²) memory). The idiomatic fix is
// to use strings.Builder.
package stringsconcatloop

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the string-concat-in-loop analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "stringsconcatloop",
	Doc:      "reports string += concatenation inside for/range loops that should use strings.Builder",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/stringsconcatloop",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	root, err := astutil.Root(pass)
	if err != nil {
		return nil, err
	}
	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, err
	}

	for cur := range root.Preorder((*ast.AssignStmt)(nil)) {
		assign, ok := cur.Node().(*ast.AssignStmt)
		if !ok {
			continue
		}
		if assign.Tok != token.ADD_ASSIGN {
			continue
		}
		if len(assign.Lhs) == 0 {
			continue
		}

		pos := pass.Fset.PositionFor(assign.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringsconcatloop") {
			continue
		}

		if !astutil.IsStringType(pass, assign.Lhs[0]) {
			continue
		}

		if !isInsideLoop(cur) {
			continue
		}

		pass.ReportRangef(assign,
			"string concatenation with += inside a loop causes O(n²) allocations; use strings.Builder instead")
	}

	return nil, nil
}

// isInsideLoop reports whether cur (an AssignStmt) is enclosed within a
// for or range loop body, without crossing a function literal boundary.
// Assignments inside func literals are exempt because they form a new scope.
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
