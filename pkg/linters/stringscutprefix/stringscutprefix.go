// Package stringscutprefix implements a Go analysis linter that flags
// if-blocks that use strings.HasPrefix followed by strings.TrimPrefix
// and suggests strings.CutPrefix instead.
package stringscutprefix

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the strings-cut-prefix analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "stringscutprefix",
	Doc:      "reports if-blocks that check strings.HasPrefix then call strings.TrimPrefix on the same args, suggesting strings.CutPrefix",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/stringscutprefix",
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

	for cur := range root.Preorder((*ast.IfStmt)(nil)) {
		ifStmt, ok := cur.Node().(*ast.IfStmt)
		if !ok {
			continue
		}

		pos := pass.Fset.PositionFor(ifStmt.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringscutprefix") {
			continue
		}

		// Condition must be strings.HasPrefix(s, prefix)
		hasPrefixCall, s, prefix := extractHasPrefixCall(pass, ifStmt.Cond)
		if hasPrefixCall == nil {
			continue
		}

		// Body must contain strings.TrimPrefix(s, prefix) with same args
		if bodyContainsTrimPrefix(pass, ifStmt.Body, s, prefix) {
			pass.ReportRangef(hasPrefixCall,
				"strings.HasPrefix + strings.TrimPrefix can be replaced with strings.CutPrefix")
		}
	}

	return nil, nil
}

// extractHasPrefixCall returns the call expression, the s argument, and the
// prefix argument if the expression is strings.HasPrefix(s, prefix).
func extractHasPrefixCall(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, ast.Expr, ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return nil, nil, nil
	}
	if !isStringsFunc(pass, call, "HasPrefix") {
		return nil, nil, nil
	}
	return call, call.Args[0], call.Args[1]
}

// bodyContainsTrimPrefix returns true if the block contains a direct statement
// that calls strings.TrimPrefix with the same arguments, before either argument
// is reassigned in the enclosing if body.
func bodyContainsTrimPrefix(pass *analysis.Pass, body *ast.BlockStmt, s, prefix ast.Expr) bool {
	for _, stmt := range body.List {
		if stmtContainsTrimPrefix(pass, stmt, s, prefix) {
			return true
		}
		if stmtReassignsExpr(pass, stmt, s) || stmtReassignsExpr(pass, stmt, prefix) {
			return false
		}
	}
	return false
}

func stmtContainsTrimPrefix(pass *analysis.Pass, stmt ast.Stmt, s, prefix ast.Expr) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		if n != stmt && shouldSkipNestedFlow(n) {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		if !isStringsFunc(pass, call, "TrimPrefix") {
			return true
		}
		if sameExpr(pass, call.Args[0], s) && sameExpr(pass, call.Args[1], prefix) {
			found = true
		}
		return true
	})
	return found
}

func stmtReassignsExpr(pass *analysis.Pass, stmt ast.Stmt, target ast.Expr) bool {
	reassigned := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		if reassigned || n == nil {
			return false
		}
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		switch node := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if sameExpr(pass, lhs, target) {
					reassigned = true
					return false
				}
			}
		case *ast.IncDecStmt:
			if sameExpr(pass, node.X, target) {
				reassigned = true
				return false
			}
		}
		return true
	})
	return reassigned
}

func shouldSkipNestedFlow(n ast.Node) bool {
	switch n.(type) {
	case *ast.BlockStmt, *ast.FuncLit, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	default:
		return false
	}
}

// isStringsFunc reports whether call invokes strings.<name>.
func isStringsFunc(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return false
	}
	pkgName, ok := obj.(*types.PkgName)
	return ok && pkgName.Imported().Path() == "strings"
}

// sameExpr returns true when a and b are syntactically equivalent simple
// expressions (identifiers or selector expressions) referring to the same
// object, or are equal basic literals.
func sameExpr(pass *analysis.Pass, a, b ast.Expr) bool {
	switch av := a.(type) {
	case *ast.Ident:
		bv, ok := b.(*ast.Ident)
		if !ok {
			return false
		}
		ao := pass.TypesInfo.ObjectOf(av)
		bo := pass.TypesInfo.ObjectOf(bv)
		return ao != nil && ao == bo
	case *ast.BasicLit:
		bv, ok := b.(*ast.BasicLit)
		return ok && av.Kind == bv.Kind && av.Value == bv.Value
	case *ast.SelectorExpr:
		bv, ok := b.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		return sameExpr(pass, av.X, bv.X) && av.Sel.Name == bv.Sel.Name
	case *ast.IndexExpr:
		bv, ok := b.(*ast.IndexExpr)
		if !ok {
			return false
		}
		return sameExpr(pass, av.X, bv.X) && sameExpr(pass, av.Index, bv.Index)
	}
	// Complex expressions are not structurally compared; treat them as unequal
	// to avoid false positives.
	return false
}
