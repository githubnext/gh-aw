// Package hasprefixtrim implements a Go analysis linter that flags
// `if strings.HasPrefix(s, p) { s = s[len(p):] }` (and the HasSuffix
// equivalent) which should use strings.TrimPrefix / strings.TrimSuffix.
package hasprefixtrim

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the hasprefixtrim analysis pass.
var Analyzer = analyzerutil.New("hasprefixtrim", "reports `if strings.HasPrefix(s, p) { s = s[len(p):] }` (and the HasSuffix equivalent) that should use strings.TrimPrefix or strings.TrimSuffix", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	// else-if branches are conditional on the previous condition failing, so
	// they cannot be rewritten to a bare Trim call; track and skip them.
	elseIfs := map[*ast.IfStmt]bool{}
	nodeFilter := []ast.Node{(*ast.IfStmt)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		stmt := n.(*ast.IfStmt)
		if next, ok := stmt.Else.(*ast.IfStmt); ok {
			elseIfs[next] = true
		}
		if elseIfs[stmt] {
			return
		}
		analyzeIf(pass, stmt, generatedFiles, noLintIndex)
	})
}

func analyzeIf(pass *analysis.Pass, stmt *ast.IfStmt, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	if stmt.Init != nil || stmt.Else != nil || len(stmt.Body.List) != 1 {
		return
	}
	pos := pass.Fset.PositionFor(stmt.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "hasprefixtrim") {
		return
	}

	call, ok := stmt.Cond.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || call.Ellipsis.IsValid() {
		return
	}
	pkgPath, fn, ok := astutil.PackageCall(pass, call)
	if !ok || pkgPath != "strings" || (fn != "HasPrefix" && fn != "HasSuffix") {
		return
	}
	s, p := call.Args[0], call.Args[1]
	if !isSimple(s) {
		return
	}

	assign, ok := stmt.Body.List[0].(*ast.AssignStmt)
	if !ok || assign.Tok != token.ASSIGN || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return
	}
	if !same(assign.Lhs[0], s) {
		return
	}
	slice, ok := assign.Rhs[0].(*ast.SliceExpr)
	if !ok || slice.Slice3 || !same(slice.X, s) {
		return
	}

	switch fn {
	case "HasPrefix":
		if slice.High != nil || slice.Low == nil || !isLenOf(pass, slice.Low, p) {
			return
		}
		pass.Reportf(stmt.Pos(), "use strings.TrimPrefix(%s, %s) instead of HasPrefix check followed by slicing", types.ExprString(s), types.ExprString(p))
	case "HasSuffix":
		if slice.Low != nil || slice.High == nil {
			return
		}
		bin, ok := slice.High.(*ast.BinaryExpr)
		if !ok || bin.Op != token.SUB || !isLenOf(pass, bin.X, s) || !isLenOf(pass, bin.Y, p) {
			return
		}
		pass.Reportf(stmt.Pos(), "use strings.TrimSuffix(%s, %s) instead of HasSuffix check followed by slicing", types.ExprString(s), types.ExprString(p))
	}
}

// isSimple reports whether expr is an identifier or a selector chain of identifiers.
func isSimple(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name != "_"
	case *ast.SelectorExpr:
		return isSimple(e.X)
	}
	return false
}

// same reports whether a and b are the identical simple expression.
func same(a, b ast.Expr) bool {
	if !isSimple(a) || !isSimple(b) {
		return false
	}
	return types.ExprString(a) == types.ExprString(b)
}

// isLenOf reports whether expr is the builtin len(arg), or (when arg is a
// string literal) an integer literal equal to the literal's byte length.
func isLenOf(pass *analysis.Pass, expr, arg ast.Expr) bool {
	expr = astutil.UnwrapParenExpr(expr)
	if call, ok := expr.(*ast.CallExpr); ok {
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Name != "len" || len(call.Args) != 1 {
			return false
		}
		if _, isBuiltin := pass.TypesInfo.Uses[id].(*types.Builtin); !isBuiltin {
			return false
		}
		if lit, ok := astutil.StringLitValue(arg); ok {
			if inner, ok := astutil.StringLitValue(call.Args[0]); ok {
				return inner == lit
			}
			return false
		}
		return same(call.Args[0], arg)
	}
	lit, ok := astutil.StringLitValue(arg)
	if !ok {
		return false
	}
	bl, ok := expr.(*ast.BasicLit)
	if !ok || bl.Kind != token.INT {
		return false
	}
	tv, ok := pass.TypesInfo.Types[bl]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.Int {
		return false
	}
	n, exact := constant.Int64Val(tv.Value)
	return exact && n == int64(len(lit))
}
