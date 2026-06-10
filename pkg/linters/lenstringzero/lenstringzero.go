// Package lenstringzero implements a Go analysis linter that flags len(s) == 0
// and len(s) != 0 comparisons on string values that should use == "" or != "" instead.
package lenstringzero

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

var Analyzer = &analysis.Analyzer{
	Name:     "lenstringzero",
	Doc:      "reports len(s) == 0 and len(s) != 0 comparisons on string values that should use == \"\" or != \"\" instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/lenstringzero",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	lenStringAliases := collectLenStringAliases(pass)

	nodeFilter := []ast.Node{(*ast.BinaryExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return
		}
		if expr.Op != token.EQL && expr.Op != token.NEQ {
			return
		}

		pos := pass.Fset.PositionFor(expr.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		var lenArg ast.Expr
		isDirect := false
		if isLenCall(expr.X) && isIntZero(expr.Y) {
			lenArg = lenCallArg(expr.X)
			isDirect = true
		} else if isIntZero(expr.X) && isLenCall(expr.Y) {
			lenArg = lenCallArg(expr.Y)
			isDirect = true
		} else if isIntZero(expr.Y) {
			if arg, ok := lenAliasArg(pass, expr.X, lenStringAliases); ok {
				lenArg = arg
			}
		} else if isIntZero(expr.X) {
			if arg, ok := lenAliasArg(pass, expr.Y, lenStringAliases); ok {
				lenArg = arg
			}
		}
		if lenArg == nil {
			return
		}

		t := pass.TypesInfo.TypeOf(lenArg)
		if t == nil {
			return
		}
		basic, ok := t.Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.String {
			return
		}

		op := expr.Op.String()
		var cmpVerb string
		if expr.Op == token.EQL {
			cmpVerb = "empty"
		} else {
			cmpVerb = "non-empty"
		}
		var fixes []analysis.SuggestedFix
		if isDirect {
			fixes = buildLenStringFix(pass, expr, lenArg)
		}
		pass.Report(analysis.Diagnostic{
			Pos:            expr.Pos(),
			End:            expr.End(),
			Message:        fmt.Sprintf(`use s %s "" to check for %s string instead of len(s) %s 0`, op, cmpVerb, op),
			SuggestedFixes: fixes,
		})
	})

	return nil, nil
}

// buildLenStringFix returns a SuggestedFix that rewrites a direct len(s) == 0
// or len(s) != 0 comparison to s == "" or s != "".
func buildLenStringFix(pass *analysis.Pass, expr *ast.BinaryExpr, lenArg ast.Expr) []analysis.SuggestedFix {
	text := astutil.NodeText(pass.Fset, lenArg)
	if text == "" {
		return nil
	}
	replacement := fmt.Sprintf(`%s %s ""`, text, expr.Op.String())
	return []analysis.SuggestedFix{{
		Message: "Replace with direct string comparison",
		TextEdits: []analysis.TextEdit{{
			Pos:     expr.Pos(),
			End:     expr.End(),
			NewText: []byte(replacement),
		}},
	}}
}

func isLenCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == "len"
}

func lenCallArg(expr ast.Expr) ast.Expr {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}
	return call.Args[0]
}

func isIntZero(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}

func collectLenStringAliases(pass *analysis.Pass) map[types.Object]ast.Expr {
	aliases := make(map[types.Object]ast.Expr)
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.AssignStmt:
				collectLenStringAliasesFromAssignStmt(pass, n, aliases)
			case *ast.ValueSpec:
				collectLenStringAliasesFromValueSpec(pass, n, aliases)
			case *ast.IncDecStmt:
				if ident, ok := n.X.(*ast.Ident); ok {
					delete(aliases, pass.TypesInfo.ObjectOf(ident))
				}
			case *ast.RangeStmt:
				if n.Tok == token.ASSIGN {
					deleteLenStringAliasForExpr(pass, aliases, n.Key)
					deleteLenStringAliasForExpr(pass, aliases, n.Value)
				}
			}
			return true
		})
	}
	return aliases
}

func collectLenStringAliasesFromAssignStmt(pass *analysis.Pass, stmt *ast.AssignStmt, aliases map[types.Object]ast.Expr) {
	for i, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj == nil || !astutil.IsLocalObject(obj) {
			continue
		}

		switch stmt.Tok {
		case token.DEFINE:
			if obj.Pos() != ident.Pos() {
				delete(aliases, obj)
				continue
			}
			rhs, ok := astutil.RhsExprForIndex(stmt.Rhs, i)
			if !ok {
				delete(aliases, obj)
				continue
			}
			if arg, ok := lenStringArg(pass, rhs); ok {
				aliases[obj] = arg
			} else {
				delete(aliases, obj)
			}
		case token.ASSIGN:
			delete(aliases, obj)
		}
	}
}

func collectLenStringAliasesFromValueSpec(pass *analysis.Pass, spec *ast.ValueSpec, aliases map[types.Object]ast.Expr) {
	for i, name := range spec.Names {
		if name.Name == "_" {
			continue
		}
		obj := pass.TypesInfo.ObjectOf(name)
		if obj == nil || !astutil.IsLocalObject(obj) {
			continue
		}
		rhs, ok := astutil.RhsExprForIndex(spec.Values, i)
		if !ok {
			delete(aliases, obj)
			continue
		}
		if arg, ok := lenStringArg(pass, rhs); ok {
			aliases[obj] = arg
		} else {
			delete(aliases, obj)
		}
	}
}

func lenAliasArg(pass *analysis.Pass, expr ast.Expr, aliases map[types.Object]ast.Expr) (ast.Expr, bool) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil, false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}
	arg, ok := aliases[obj]
	if !ok {
		return nil, false
	}
	return arg, true
}

func lenStringArg(pass *analysis.Pass, expr ast.Expr) (ast.Expr, bool) {
	if !isLenCall(expr) {
		return nil, false
	}
	arg := lenCallArg(expr)
	t := pass.TypesInfo.TypeOf(arg)
	if t == nil {
		return nil, false
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.String {
		return nil, false
	}
	return arg, true
}

func deleteLenStringAliasForExpr(pass *analysis.Pass, aliases map[types.Object]ast.Expr, expr ast.Expr) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return
	}
	delete(aliases, pass.TypesInfo.ObjectOf(ident))
}
