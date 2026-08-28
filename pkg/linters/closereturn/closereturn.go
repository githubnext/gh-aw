// Package closereturn implements a Go analysis linter that flags functions
// that return immediately after a successful resource acquisition without
// deferring the matching Close() call.
package closereturn

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the close-return analysis pass.
var Analyzer = analyzerutil.New("closereturn", "reports functions that return immediately after a successful resource acquisition without deferring the matching Close() call", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}
		pos := pass.Fset.PositionFor(fn.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		analyzeFunc(pass, fn, noLintIndex)
	})
}

func analyzeFunc(pass *analysis.Pass, fn *ast.FuncDecl, noLintIndex nolint.DirectiveIndex) {
	stmts := fn.Body.List
	for i := 0; i+1 < len(stmts); i++ {
		assign, ok := stmts[i].(*ast.AssignStmt)
		if !ok {
			continue
		}
		resource, ok := acquiredResource(pass, assign)
		if !ok {
			continue
		}
		if hasDeferredClose(stmts[i+1:], resource.obj, pass) {
			continue
		}
		nextIf, ok := stmts[i+1].(*ast.IfStmt)
		if !ok || nextIf.Else != nil {
			continue
		}
		if !isErrReturnGuard(pass, nextIf, resource.errObj) {
			continue
		}
		if !bodyReturnsOrFallsThrough(nextIf.Body) {
			continue
		}
		position := pass.Fset.PositionFor(assign.Pos(), false)
		if nolint.HasDirectiveForLinter(position, noLintIndex, "closereturn") {
			continue
		}
		pass.Report(analysis.Diagnostic{
			Pos:     assign.Pos(),
			End:     assign.End(),
			Message: "resource Close() should be deferred immediately after successful open before any early-return error guard",
		})
	}
}

type resourceBinding struct {
	obj    types.Object
	errObj types.Object
}

func acquiredResource(pass *analysis.Pass, assign *ast.AssignStmt) (resourceBinding, bool) {
	if assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return resourceBinding{}, false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || !isOpenLikeCall(pass, call) {
		return resourceBinding{}, false
	}
	resIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || resIdent.Name == "_" {
		return resourceBinding{}, false
	}
	errIdent, ok := assign.Lhs[1].(*ast.Ident)
	if !ok || errIdent.Name == "_" {
		return resourceBinding{}, false
	}
	resObj := pass.TypesInfo.ObjectOf(resIdent)
	errObj := pass.TypesInfo.ObjectOf(errIdent)
	if resObj == nil || errObj == nil {
		return resourceBinding{}, false
	}
	if !hasCloseMethod(resObj.Type()) || !isErrorType(errObj.Type()) {
		return resourceBinding{}, false
	}
	return resourceBinding{obj: resObj, errObj: errObj}, true
}

func isOpenLikeCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !astutil.IsPkgSelector(pass, sel, "os") {
		return false
	}
	switch sel.Sel.Name {
	case "Open", "Create", "OpenFile":
		return true
	default:
		return false
	}
}

func hasCloseMethod(t types.Type) bool {
	if t == nil {
		return false
	}
	if _, _, ok := types.LookupFieldOrMethod(t, true, nil, "Close"); ok {
		return true
	}
	return false
}

func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() == nil && obj.Name() == "error"
}

func isErrReturnGuard(pass *analysis.Pass, ifStmt *ast.IfStmt, errObj types.Object) bool {
	bin, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	left, ok := bin.X.(*ast.Ident)
	if !ok || pass.TypesInfo.ObjectOf(left) != errObj {
		return false
	}
	right, ok := bin.Y.(*ast.Ident)
	return ok && right.Name == "nil"
}

func bodyReturnsOrFallsThrough(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) == 0 {
		return false
	}
	_, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	return ok
}

func hasDeferredClose(stmts []ast.Stmt, obj types.Object, pass *analysis.Pass) bool {
	for _, stmt := range stmts {
		deferStmt, ok := stmt.(*ast.DeferStmt)
		if !ok {
			continue
		}
		if closeReceiver(pass, deferStmt.Call) == obj {
			return true
		}
	}
	return false
}

func closeReceiver(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" {
		return nil
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}
	return pass.TypesInfo.ObjectOf(ident)
}
