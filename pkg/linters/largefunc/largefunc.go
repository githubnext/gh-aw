// Package largefunc implements a Go analysis linter that flags functions
// whose body exceeds a configurable line threshold.
package largefunc

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// DefaultMaxLines is the default maximum number of lines allowed in a function body.
const DefaultMaxLines = 60

// Analyzer is the large-function analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "largefunc",
	Doc:      "reports functions whose body exceeds the line limit (default 60 lines)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/largefunc",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// maxLines is the configurable threshold.  It is set via the -largefunc.max-lines flag.
var maxLines int

func init() {
	Analyzer.Flags.IntVar(&maxLines, "max-lines", DefaultMaxLines,
		"maximum number of lines permitted in a function body")
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		var name string
		var reportNode ast.Node

		switch fn := n.(type) {
		case *ast.FuncDecl:
			body = fn.Body
			name = fn.Name.Name
			reportNode = fn.Name
		case *ast.FuncLit:
			body = fn.Body
			name = "func literal"
			reportNode = body
		}

		if body == nil {
			return
		}

		start := pass.Fset.Position(body.Lbrace)
		end := pass.Fset.Position(body.Rbrace)
		// Subtract 1 to exclude the closing brace line itself, counting only body lines.
		lines := end.Line - start.Line - 1

		if lines > maxLines {
			pass.ReportRangef(
				reportNode,
				"%s is %d lines long (limit: %d); consider breaking it up",
				name, lines, maxLines,
			)
		}
	})

	return nil, nil
}
