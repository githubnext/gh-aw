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

	ctxType := astutil.ContextContextType(pass)
	if ctxType == nil {
		return nil, nil
	}

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "nilctxpassed") {
			continue
		}

		sig := calleeSignature(pass, call)
		if sig == nil {
			continue
		}

		params := sig.Params()
		for i, arg := range call.Args {
			var paramType types.Type
			if sig.Variadic() && i >= params.Len()-1 {
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

			if !types.Identical(paramType, ctxType) {
				continue
			}

			if !isBuiltinNil(pass, arg) {
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
