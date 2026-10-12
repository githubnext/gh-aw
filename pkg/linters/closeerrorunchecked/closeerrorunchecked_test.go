//go:build !integration

package closeerrorunchecked_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/closeerrorunchecked"
	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil/analyzertest"
	unchecked_deferredclose "github.com/github/gh-aw/pkg/linters/unchecked-deferred-close"
)

func TestCloseErrorUnchecked(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, closeerrorunchecked.Analyzer, "closeerrorunchecked")
	analysistest.Run(t, testdata, closeerrorunchecked.Analyzer, "closeerrorunchecked/customerror")
}

func TestNestedParenthesizedClose(t *testing.T) {
	t.Parallel()
	// gofmt collapses nested parentheses in fixtures, but leaves raw strings intact.
	const source = `package p
type closer struct{}
func (closer) Close() error { return nil }
func discarded(c closer) { ((c.Close())) }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "nested.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	stmt := file.Decls[2].(*ast.FuncDecl).Body.List[0].(*ast.ExprStmt)
	outer, ok := stmt.X.(*ast.ParenExpr)
	if !ok {
		t.Fatal("expected outer parentheses")
	}
	inner, ok := outer.X.(*ast.ParenExpr)
	if !ok {
		t.Fatal("expected nested parentheses")
	}
	call, ok := inner.X.(*ast.CallExpr)
	if !ok {
		t.Fatal("expected Close call inside nested parentheses")
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	config := &types.Config{}
	pkg, err := config.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:  closeerrorunchecked.Analyzer,
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
		ResultOf:  make(map[*analysis.Analyzer]any),
		Report:    func(d analysis.Diagnostic) { diagnostics = append(diagnostics, d) },
	}
	for _, prerequisite := range closeerrorunchecked.Analyzer.Requires {
		result, err := prerequisite.Run(pass)
		if err != nil {
			t.Fatal(err)
		}
		pass.ResultOf[prerequisite] = result
	}
	if _, err := closeerrorunchecked.Analyzer.Run(pass); err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diagnostics)
	}
	if diagnostic := diagnostics[0]; diagnostic.Pos != call.Pos() ||
		diagnostic.Message != "Close() error is explicitly discarded; resource cleanup failures may be silently ignored" {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}
}

func TestComplementaryCloseErrorAnalyzers(t *testing.T) {
	t.Parallel()
	analyzertest.RunTogether(t, analysistest.TestData(), []*analysis.Analyzer{
		closeerrorunchecked.Analyzer,
		unchecked_deferredclose.Analyzer,
	}, "combined")
}
