// Package timeafterleak implements a Go analysis linter that flags
// time.After calls used as select case channel receives inside loops,
// which allocate a new timer on every iteration that is not garbage
// collected until it fires when another case is selected first.
package timeafterleak

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the time-after-leak analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "timeafterleak",
	Doc:      "reports time.After calls used as the channel-receive expression in a select CommClause that is enclosed by a for or range loop; does not flag receives inside case bodies, single-case selects without a default, or selects enclosed only by a function literal boundary",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/timeafterleak",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
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

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok {
			continue
		}
		if !isTimeAfterCall(pass, call) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "timeafterleak") {
			continue
		}

		if !isInsideLoopSelectComm(cur) {
			continue
		}

		pass.ReportRangef(call,
			"time.After creates a new timer on each loop iteration that is not garbage collected until it fires; use time.NewTimer with Reset and Stop instead")
	}

	return nil, nil
}

// isTimeAfterCall reports whether call is an invocation of time.After.
func isTimeAfterCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "After" {
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
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "time"
}

// isInsideLoopSelectComm reports whether cur is a time.After call used as the
// channel receive expression in the Comm field of a select CommClause that is
// enclosed by a for or range loop, without crossing a function literal boundary.
// Single-case selects without a default are not flagged because the timer must
// fire before the loop can continue — no timer accumulation is possible.
func isInsideLoopSelectComm(cur inspector.Cursor) bool {
	// The immediate parent of time.After(...) must be a channel-receive UnaryExpr.
	recvCur := cur.Parent()
	unary, ok := recvCur.Node().(*ast.UnaryExpr)
	if !ok || unary.Op != token.ARROW {
		return false
	}

	// The parent of the receive expression must be the Comm statement of a CommClause.
	// Comm is an ExprStmt (case <-ch:) or AssignStmt (case v := <-ch:).
	commStmtCur := recvCur.Parent()
	var commStmt ast.Stmt
	switch s := commStmtCur.Node().(type) {
	case *ast.ExprStmt:
		commStmt = s
	case *ast.AssignStmt:
		commStmt = s
	default:
		return false
	}

	// The parent of the Comm statement must be a CommClause, and commStmt must
	// be the Comm field (not a statement in the Body).
	clauseCur := commStmtCur.Parent()
	cc, ok := clauseCur.Node().(*ast.CommClause)
	if !ok || cc.Comm != commStmt {
		return false
	}

	if isSingleCaseSelect(clauseCur, cc) {
		return false
	}

	// Walk up from the CommClause to find an enclosing for or range loop,
	// stopping at any function literal boundary.
	for encl := range clauseCur.Enclosing(
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

// isSingleCaseSelect reports whether the CommClause cc is the only clause in
// its enclosing SelectStmt. Single-case selects are not flagged because the
// timer must fire — no accumulation is possible.
// A default clause (CommClause with nil Comm) is counted as another clause,
// so a select with a timer case plus a default returns false and is reportable.
func isSingleCaseSelect(clauseCur inspector.Cursor, cc *ast.CommClause) bool {
	for selCur := range clauseCur.Enclosing((*ast.SelectStmt)(nil)) {
		sel, ok := selCur.Node().(*ast.SelectStmt)
		if !ok {
			break
		}
		for _, stmt := range sel.Body.List {
			if other, isComm := stmt.(*ast.CommClause); isComm && other != cc {
				return false
			}
		}
		return true
	}
	return false
}
