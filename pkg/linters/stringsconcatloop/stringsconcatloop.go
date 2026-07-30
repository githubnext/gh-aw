// Package stringsconcatloop implements a Go analysis linter that flags
// string concatenation inside for/range loop bodies using += or the equivalent
// x = x + y form, which allocates a new string on every iteration and can lead
// to O(n²) total allocated bytes. The idiomatic fix is to use strings.Builder.
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
	Doc:      "reports string += or x = x + y concatenation inside for/range loops that should use strings.Builder",
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

		// Match both `x += y` (ADD_ASSIGN) and `x = x + y` (ASSIGN with a
		// self-referential binary addition). For the latter, also capture the
		// LHS identifier so the loop-scope guard can be applied.
		var lhsExpr ast.Expr
		var assignLhsName string // non-empty only for the token.ASSIGN form

		switch assign.Tok {
		case token.ADD_ASSIGN:
			lhsExpr = assign.Lhs[0]
		case token.ASSIGN:
			if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				continue
			}
			lhsIdent, ok := assign.Lhs[0].(*ast.Ident)
			if !ok {
				continue
			}
			binExpr, ok := assign.Rhs[0].(*ast.BinaryExpr)
			if !ok || binExpr.Op != token.ADD {
				continue
			}
			// Only match the direct self-referential form: x = x + rhs,
			// where x is the same identifier on both sides.
			rhsLeft, ok := binExpr.X.(*ast.Ident)
			if !ok || rhsLeft.Name != lhsIdent.Name {
				continue
			}
			lhsExpr = lhsIdent
			assignLhsName = lhsIdent.Name
		default:
			continue
		}

		pos := pass.Fset.PositionFor(assign.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}

		loopPos, loopNode, inLoop := enclosingLoop(pass, cur)
		if !inLoop {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringsconcatloop") {
			continue
		}
		if nolint.HasDirectiveForLinter(loopPos, noLintIndex, "stringsconcatloop") {
			continue
		}

		if !astutil.IsStringType(pass, lhsExpr) {
			continue
		}

		// For the x = x + y form, skip variables declared by the enclosing
		// loop itself (range key/value or for-init short-decl): those are
		// per-iteration rebinds, not cross-iteration accumulators.
		if assignLhsName != "" && isLoopScopedIdent(loopNode, assignLhsName) {
			continue
		}

		pass.ReportRangef(assign,
			"string concatenation with += inside a loop allocates O(n) strings and O(n²) total bytes; use strings.Builder instead")
	}

	return nil, nil
}

// enclosingLoop returns the nearest enclosing for/range statement, its source
// position, and true for cur (an AssignStmt), without crossing a function
// literal boundary. Assignments inside func literals are intentionally exempt.
func enclosingLoop(pass *analysis.Pass, cur inspector.Cursor) (token.Position, ast.Node, bool) {
	for encl := range cur.Enclosing(
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.FuncLit)(nil),
	) {
		switch encl.Node().(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			return pass.Fset.PositionFor(encl.Node().Pos(), false), encl.Node(), true
		case *ast.FuncLit:
			return token.Position{}, nil, false
		}
	}
	return token.Position{}, nil, false
}

// isLoopScopedIdent reports whether name is declared by loopNode as a loop
// variable: the Key or Value identifier of a RangeStmt, or a variable in the
// short-declaration Init of a ForStmt. Such variables are per-iteration
// rebinds, not cross-iteration accumulators.
func isLoopScopedIdent(loopNode ast.Node, name string) bool {
	switch n := loopNode.(type) {
	case *ast.RangeStmt:
		if id, ok := n.Key.(*ast.Ident); ok && id.Name == name {
			return true
		}
		if id, ok := n.Value.(*ast.Ident); ok && id.Name == name {
			return true
		}
	case *ast.ForStmt:
		init, ok := n.Init.(*ast.AssignStmt)
		if !ok || init.Tok != token.DEFINE {
			return false
		}
		for _, lhs := range init.Lhs {
			if id, ok := lhs.(*ast.Ident); ok && id.Name == name {
				return true
			}
		}
	}
	return false
}
