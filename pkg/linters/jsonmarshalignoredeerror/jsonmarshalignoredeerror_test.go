//go:build !integration

package jsonmarshalignoredeerror_test

import (
	"go/ast"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/jsonmarshalignoredeerror"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, jsonmarshalignoredeerror.Analyzer, "jsonmarshalignoredeerror")
}

func TestAnalyzerNestedParentheses(t *testing.T) {
	t.Parallel()
	analyzer := *jsonmarshalignoredeerror.Analyzer
	analyzer.Run = func(pass *analysis.Pass) (any, error) {
		wrapped := 0
		wrap := func(expr ast.Expr) ast.Expr {
			if paren, ok := expr.(*ast.ParenExpr); ok {
				wrapped++
				// gofmt collapses nested parentheses in source fixtures.
				return &ast.ParenExpr{Lparen: paren.Lparen, X: paren, Rparen: paren.Rparen}
			}
			return expr
		}
		for _, file := range pass.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch stmt := node.(type) {
				case *ast.ExprStmt:
					stmt.X = wrap(stmt.X)
				case *ast.AssignStmt:
					for i, expr := range stmt.Rhs {
						stmt.Rhs[i] = wrap(expr)
					}
				}
				return true
			})
		}
		if wrapped == 0 {
			t.Fatal("fixture must contain parenthesized expressions")
		}
		return jsonmarshalignoredeerror.Analyzer.Run(pass)
	}
	analysistest.Run(t, analysistest.TestData(), &analyzer, "jsonmarshalignoredeerror")
}
