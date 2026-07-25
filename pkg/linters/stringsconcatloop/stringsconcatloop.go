// Package stringsconcatloop implements a Go analysis linter that flags
// string += concatenation inside for/range loop bodies, which allocates a new
// string on every iteration and can lead to O(n²) total allocated bytes. The
// idiomatic fix is to use strings.Builder.
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

		pos := pass.Fset.PositionFor(assign.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}

		loopPos, inLoop := enclosingLoopPosition(pass, cur)
		if !inLoop {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringsconcatloop") {
			continue
		}
		if nolint.HasDirectiveForLinter(loopPos, noLintIndex, "stringsconcatloop") {
			continue
		}

		if !astutil.IsStringType(pass, assign.Lhs[0]) {
			continue
		}

		pass.ReportRangef(assign,
			"string concatenation with += inside a loop allocates O(n) strings and O(n²) total bytes; use strings.Builder instead")
	}

	return nil, nil
}

// enclosingLoopPosition returns the nearest enclosing for/range statement
// position for cur (an AssignStmt), without crossing a function literal
// boundary. Assignments inside func literals are intentionally exempt.
func enclosingLoopPosition(pass *analysis.Pass, cur inspector.Cursor) (token.Position, bool) {
	for encl := range cur.Enclosing(
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.FuncLit)(nil),
	) {
		switch encl.Node().(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			return pass.Fset.PositionFor(encl.Node().Pos(), false), true
		case *ast.FuncLit:
			return token.Position{}, false
		}
	}
	return token.Position{}, false
}
