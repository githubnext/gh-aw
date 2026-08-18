// Package excessivefuncparams implements a Go analysis linter that flags
// functions with too many positional parameters.
package excessivefuncparams

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:excessivefuncparams")

// DefaultMaxParams is the default maximum number of parameters allowed in a function declaration.
const DefaultMaxParams = 8

// Analyzer is the excessive-function-parameters analysis pass.
var Analyzer = analyzerutil.New("excessivefuncparams", "reports functions whose parameter count exceeds the limit (default 8 params)", run)

// maxParams is the configurable threshold. It is set via the -excessivefuncparams.max-params flag.
var maxParams int

func init() {
	Analyzer.Flags.IntVar(&maxParams, "max-params", DefaultMaxParams,
		"maximum number of parameters permitted in a function declaration")
}

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s (max-params=%d)", pass.Pkg.Path(), maxParams)

	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}
		if fn.Type == nil || fn.Type.Params == nil {
			return
		}
		position := pass.Fset.PositionFor(fn.Name.Pos(), false)
		if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
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
			if nolint.HasDirectiveForLinter(position, noLintIndex, "excessivefuncparams") {
				return
			}
			pkgLog.Printf("flagging %s: %d parameters exceeds limit %d", fn.Name.Name, params, maxParams)
			pass.ReportRangef(
				fn.Name,
				"%s has %d parameters (limit: %d); consider using an options struct",
				fn.Name.Name, params, maxParams,
			)
		}
	})
}
