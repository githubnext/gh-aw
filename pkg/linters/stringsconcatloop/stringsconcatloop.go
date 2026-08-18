// Package stringsconcatloop implements a Go analysis linter that flags
// string concatenation inside for/range loop bodies using += or the equivalent
// x = x + y form, which allocates a new string on every iteration and can lead
// to O(n²) total allocated bytes. The idiomatic fix is to use strings.Builder.
package stringsconcatloop

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/coverage"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:stringsconcatloop")

// Analyzer is the string-concat-in-loop analysis pass.
var Analyzer = analyzerutil.New("stringsconcatloop", "reports string concatenation (+=  or x = x + y) inside for/range loops that should use strings.Builder", run)

// hotThreshold gates findings on coverage data; see coverage package docs.
// String-concatenation-in-loop is the canonical example of a perf rule that
// only matters on hot paths: the O(n²) cost is only worth paying attention to
// when the loop actually executes during tests.
var hotThreshold *int

func init() {
	hotThreshold = coverage.RegisterHotThresholdFlag(Analyzer)
}

// concatLoopMatch holds the components of a string-concatenation-in-loop
// assignment identified by collectConcatLoopAssignment.
type concatLoopMatch struct {
	assign   *ast.AssignStmt
	lhsExpr  ast.Expr
	loopNode ast.Node
	pos      token.Position
}

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())
	root, err := astutil.Root(pass)
	if err != nil {
		return nil, err
	}
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	for cur := range root.Preorder((*ast.AssignStmt)(nil)) {
		m, ok := collectConcatLoopAssignment(pass, cur, noLintIndex, generatedFiles)
		if !ok {
			continue
		}
		if !shouldReportLoopConcat(pass, m.loopNode, m.lhsExpr) {
			continue
		}
		if !coverage.ShouldApply(pass, m.assign.Pos(), *hotThreshold) {
			continue
		}
		pkgLog.Printf("flagging string concatenation in loop at %s", m.pos)
		pass.ReportRangef(m.assign, "string concatenation inside a loop allocates O(n) strings and O(n²) total bytes; use strings.Builder instead")
	}

	return nil, nil
}

func collectConcatLoopAssignment(
	pass *analysis.Pass,
	cur inspector.Cursor,
	noLintIndex nolint.DirectiveIndex,
	generatedFiles filecheck.GeneratedIndex,
) (*concatLoopMatch, bool) {
	assign, ok := cur.Node().(*ast.AssignStmt)
	if !ok {
		return nil, false
	}
	lhsExpr, ok := concatAssignmentLHS(assign)
	if !ok {
		return nil, false
	}
	pos := pass.Fset.PositionFor(assign.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return nil, false
	}
	loopPos, loopNode, inLoop := enclosingLoop(pass, cur)
	if !inLoop {
		return nil, false
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringsconcatloop") || nolint.HasDirectiveForLinter(loopPos, noLintIndex, "stringsconcatloop") {
		return nil, false
	}
	return &concatLoopMatch{assign: assign, lhsExpr: lhsExpr, loopNode: loopNode, pos: pos}, true
}

func concatAssignmentLHS(assign *ast.AssignStmt) (ast.Expr, bool) {
	switch assign.Tok {
	case token.ADD_ASSIGN:
		if len(assign.Lhs) != 1 {
			return nil, false
		}
		return assign.Lhs[0], true
	case token.ASSIGN:
		return selfReferentialConcatLHS(assign)
	default:
		return nil, false
	}
}

func selfReferentialConcatLHS(assign *ast.AssignStmt) (ast.Expr, bool) {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return nil, false
	}
	lhsIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	binExpr, ok := assign.Rhs[0].(*ast.BinaryExpr)
	if !ok || binExpr.Op != token.ADD {
		return nil, false
	}
	rhsLeft, ok := binExpr.X.(*ast.Ident)
	if !ok || rhsLeft.Name != lhsIdent.Name {
		return nil, false
	}
	return lhsIdent, true
}

func shouldReportLoopConcat(pass *analysis.Pass, loopNode ast.Node, lhsExpr ast.Expr) bool {
	if !astutil.IsStringType(pass, lhsExpr) {
		return false
	}
	lhsIdent, ok := lhsExpr.(*ast.Ident)
	if !ok {
		return true
	}
	if isLoopScopedIdent(loopNode, lhsIdent.Name) {
		return false
	}
	return !isLoopBodyLocal(pass, loopNode, lhsIdent)
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
// variable: the Key or Value identifier of a RangeStmt. Such variables are
// per-iteration rebinds, not cross-iteration accumulators.
//
// Note: ForStmt init variables (e.g. for s := ""; ...) are intentionally NOT
// exempted — the init clause runs only once, so the variable carries state
// across all iterations and is a genuine accumulator.
func isLoopScopedIdent(loopNode ast.Node, name string) bool {
	n, ok := loopNode.(*ast.RangeStmt)
	if !ok {
		return false
	}
	if id, ok := n.Key.(*ast.Ident); ok && id.Name == name {
		return true
	}
	if id, ok := n.Value.(*ast.Ident); ok && id.Name == name {
		return true
	}
	return false
}

// isLoopBodyLocal reports whether ident is declared inside the loop body
// (rather than before the loop). Such variables are freshly created on every
// iteration and are therefore not cross-iteration accumulators.
func isLoopBodyLocal(pass *analysis.Pass, loopNode ast.Node, ident *ast.Ident) bool {
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return false
	}
	var body *ast.BlockStmt
	switch n := loopNode.(type) {
	case *ast.ForStmt:
		body = n.Body
	case *ast.RangeStmt:
		body = n.Body
	}
	if body == nil {
		return false
	}
	pos := obj.Pos()
	return pos >= body.Lbrace && pos < body.Rbrace
}
