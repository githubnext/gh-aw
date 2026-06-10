// Package tolowerequalfold implements a Go analysis linter that flags
// case-insensitive string comparisons performed via strings.ToLower (or
// strings.ToUpper) combined with == that should instead use strings.EqualFold.
package tolowerequalfold

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the tolower-equalfold analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "tolowerequalfold",
	Doc:      "reports case-insensitive string comparisons using strings.ToLower/ToUpper that should use strings.EqualFold",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/tolowerequalfold",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "tolowerequalfold")
	caseConvAliases := collectCaseConvAliases(pass)

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return
		}
		if expr.Op != token.EQL && expr.Op != token.NEQ {
			return
		}

		if filecheck.IsTestFile(pass.Fset.Position(expr.Pos()).Filename) {
			return
		}

		if arg, ok := caseConvArg(expr.X); ok && sameOperand(pass, arg, expr.Y) {
			return
		}
		if arg, ok := caseConvArg(expr.Y); ok && sameOperand(pass, expr.X, arg) {
			return
		}
		if arg, ok := caseConvAliasArg(pass, expr.X, caseConvAliases); ok && sameOperand(pass, arg, expr.Y) {
			return
		}
		if arg, ok := caseConvAliasArg(pass, expr.Y, caseConvAliases); ok && sameOperand(pass, expr.X, arg) {
			return
		}

		if isCaseConvCall(expr.X) || isCaseConvCall(expr.Y) ||
			(isCaseConvAlias(pass, expr.X, caseConvAliases) && astutil.IsStringLiteral(expr.Y)) ||
			(isCaseConvAlias(pass, expr.Y, caseConvAliases) && astutil.IsStringLiteral(expr.X)) {
			if nolint.HasDirective(pass.Fset.PositionFor(expr.Pos(), false), noLintLinesByFile) {
				return
			}
			pass.Report(analysis.Diagnostic{
				Pos:            expr.Pos(),
				End:            expr.End(),
				Message:        "use strings.EqualFold for case-insensitive comparison instead of strings.ToLower/ToUpper with ==",
				SuggestedFixes: buildEqualFoldFix(pass, expr),
			})
		}
	})

	return nil, nil
}

// buildEqualFoldFix returns a SuggestedFix that rewrites a direct
// strings.ToLower/ToUpper comparison to strings.EqualFold.
// A fix is only emitted when at least one side is a direct caseConvCall (not
// an alias variable), since alias variables may be defined at a different
// source location.
func buildEqualFoldFix(pass *analysis.Pass, expr *ast.BinaryExpr) []analysis.SuggestedFix {
	leftArg, leftOK := caseConvArg(expr.X)
	rightArg, rightOK := caseConvArg(expr.Y)
	if !leftOK && !rightOK {
		return nil
	}
	arg1 := expr.X
	if leftOK {
		arg1 = leftArg
	}
	arg2 := expr.Y
	if rightOK {
		arg2 = rightArg
	}
	text1 := astutil.NodeText(pass.Fset, arg1)
	text2 := astutil.NodeText(pass.Fset, arg2)
	if text1 == "" || text2 == "" {
		return nil
	}
	call := fmt.Sprintf("strings.EqualFold(%s, %s)", text1, text2)
	if expr.Op == token.NEQ {
		call = "!" + call
	}
	return []analysis.SuggestedFix{{
		Message: "Replace with strings.EqualFold",
		TextEdits: []analysis.TextEdit{{
			Pos:     expr.Pos(),
			End:     expr.End(),
			NewText: []byte(call),
		}},
	}}
}

func collectCaseConvAliases(pass *analysis.Pass) map[types.Object]ast.Expr {
	aliases := make(map[types.Object]ast.Expr)
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.AssignStmt:
				collectAliasesFromAssignStmt(pass, n, aliases)
			case *ast.ValueSpec:
				collectAliasesFromValueSpec(pass, n, aliases)
			case *ast.IncDecStmt:
				if ident, ok := n.X.(*ast.Ident); ok {
					delete(aliases, pass.TypesInfo.ObjectOf(ident))
				}
			case *ast.RangeStmt:
				if n.Tok == token.ASSIGN {
					deleteAliasForExpr(pass, aliases, n.Key)
					deleteAliasForExpr(pass, aliases, n.Value)
				}
			}
			return true
		})
	}
	return aliases
}

func collectAliasesFromAssignStmt(pass *analysis.Pass, stmt *ast.AssignStmt, aliases map[types.Object]ast.Expr) {
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
			if arg, ok := caseConvArg(rhs); ok {
				aliases[obj] = arg
			} else {
				delete(aliases, obj)
			}
		case token.ASSIGN:
			delete(aliases, obj)
		}
	}
}

func collectAliasesFromValueSpec(pass *analysis.Pass, spec *ast.ValueSpec, aliases map[types.Object]ast.Expr) {
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
		if arg, ok := caseConvArg(rhs); ok {
			aliases[obj] = arg
		} else {
			delete(aliases, obj)
		}
	}
}

func deleteAliasForExpr(pass *analysis.Pass, aliases map[types.Object]ast.Expr, expr ast.Expr) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return
	}
	delete(aliases, pass.TypesInfo.ObjectOf(ident))
}

// isCaseConvCall reports whether node is a call to strings.ToLower or strings.ToUpper.
func isCaseConvCall(n ast.Node) bool {
	_, ok := caseConvArg(n)
	return ok
}

func isCaseConvAlias(pass *analysis.Pass, expr ast.Expr, aliases map[types.Object]ast.Expr) bool {
	_, ok := caseConvAliasArg(pass, expr, aliases)
	return ok
}

func caseConvAliasArg(pass *analysis.Pass, expr ast.Expr, aliases map[types.Object]ast.Expr) (ast.Expr, bool) {
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

// caseConvArg returns the argument when n is strings.ToLower/ToUpper(<arg>).
func caseConvArg(n ast.Node) (ast.Expr, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	if len(call.Args) != 1 {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	if ident.Name != "strings" {
		return nil, false
	}
	if sel.Sel.Name != "ToLower" && sel.Sel.Name != "ToUpper" {
		return nil, false
	}
	return call.Args[0], true
}

func sameOperand(pass *analysis.Pass, left ast.Expr, right ast.Expr) bool {
	leftIdent, leftOK := left.(*ast.Ident)
	rightIdent, rightOK := right.(*ast.Ident)
	if !leftOK || !rightOK {
		return false
	}

	leftObj := pass.TypesInfo.ObjectOf(leftIdent)
	rightObj := pass.TypesInfo.ObjectOf(rightIdent)
	return leftObj != nil && rightObj != nil && leftObj == rightObj
}
