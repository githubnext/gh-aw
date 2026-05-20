// Package excessivefuncparams implements a Go analysis linter that flags
// functions with too many positional parameters.
package excessivefuncparams

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// DefaultMaxParams is the default maximum number of parameters allowed in a function declaration.
const DefaultMaxParams = 8

// Analyzer is the excessive-function-parameters analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "excessivefuncparams",
	Doc:      "reports functions whose parameter count exceeds the limit (default 8 params)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/excessivefuncparams",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// maxParams is the configurable threshold. It is set via the -excessivefuncparams.max-params flag.
var maxParams int

func init() {
	Analyzer.Flags.IntVar(&maxParams, "max-params", DefaultMaxParams,
		"maximum number of parameters permitted in a function declaration")
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Type == nil || fn.Type.Params == nil {
			return
		}

		params := 0
		for _, field := range fn.Type.Params.List {
			if len(field.Names) == 0 {
				params++
				continue
			}
			params += len(field.Names)
		}

		if params > maxParams {
			pass.ReportRangef(
				fn.Name,
				"%s has %d parameters (limit: %d); consider using an options struct",
				fn.Name.Name, params, maxParams,
			)
		}
	})

	return nil, nil
}
