// Package panicinlibrarycode implements a Go analysis linter that flags
// panic() calls in library (pkg/) packages.
package panicinlibrarycode

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the panic-in-library-code analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "panicinlibrarycode",
	Doc:      "reports panic() calls in library code under pkg/ that should return errors instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/panic-in-library-code",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	// Skip packages under cmd/ entry-points — they are allowed to call panic.
	if strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		// Skip test files
		if strings.HasSuffix(pkgPath, ".test") || filecheck.IsTestFile(pass.Fset.Position(call.Pos()).Filename) {
			return
		}

		// Check if this is a call to the builtin panic function
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return
		}

		if ident.Name != "panic" {
			return
		}

		// Verify it's the builtin panic, not a user-defined function
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if _, ok := obj.(*types.Builtin); !ok {
				return // Not the builtin panic
			}
		}

		pass.ReportRangef(call, "avoid panic in library code; return an error instead")
	})

	return nil, nil
}
