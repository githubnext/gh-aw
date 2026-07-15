// Package nilctxpassed implements a Go analysis linter that flags function
// calls where nil is passed as a context.Context argument.
package nilctxpassed

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the nil-context-passed analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "nilctxpassed",
	Doc:      "reports function calls where nil is passed as a context.Context argument",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/nilctxpassed",
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

		callPos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(callPos.Filename, generatedFiles) {
			continue
		}

		sig := calleeSignature(pass, call)
		if sig == nil {
			continue
		}

		params := sig.Params()
		for i, arg := range call.Args {
			var paramType types.Type
			if sig.Variadic() && params.Len() > 0 && i >= params.Len()-1 {
				// Variadic: the last param is a slice; check its element type.
				sliceType, ok := params.At(params.Len() - 1).Type().(*types.Slice)
				if !ok {
					continue
				}
				paramType = sliceType.Elem()
			} else if i < params.Len() {
				paramType = params.At(i).Type()
			} else {
				continue
			}

			if !isContextContext(paramType) {
				continue
			}

			if !isBuiltinNil(pass, arg) {
				continue
			}

			argPos := pass.Fset.PositionFor(arg.Pos(), false)
			if nolint.HasDirectiveForLinter(argPos, noLintIndex, "nilctxpassed") {
				continue
			}

			pass.Report(analysis.Diagnostic{
				Pos:     arg.Pos(),
				End:     arg.End(),
				Message: "nil passed as context.Context; use context.Background() or context.TODO() instead",
			})
		}
	}

	return nil, nil
}

// isContextContext reports whether t is the context.Context interface type,
// identified by inspecting the named type directly rather than relying on the
// current package importing "context". This catches cases where the analyzed
// package passes nil to a context.Context parameter from an external function
// without importing the context package itself.
func isContextContext(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

// calleeSignature returns the *types.Signature of the callee if available.
func calleeSignature(pass *analysis.Pass, call *ast.CallExpr) *types.Signature {
	if pass.TypesInfo == nil {
		return nil
	}
	t := pass.TypesInfo.TypeOf(call.Fun)
	if t == nil {
		return nil
	}
	sig, ok := t.Underlying().(*types.Signature)
	if !ok {
		return nil
	}
	return sig
}

// isBuiltinNil reports whether expr is the predeclared nil identifier.
func isBuiltinNil(pass *analysis.Pass, expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok || ident.Name != "nil" {
		return false
	}
	if pass.TypesInfo == nil {
		return false
	}
	obj := pass.TypesInfo.Uses[ident]
	_, ok = obj.(*types.Nil)
	return ok
}
