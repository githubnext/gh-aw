// Package largefunc implements a Go analysis linter that flags functions
// whose body exceeds a configurable line threshold.
package largefunc

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:largefunc")

// DefaultMaxLines is the default maximum number of lines allowed in a function body.
const DefaultMaxLines = 60

// Analyzer is the large-function analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "largefunc",
	Doc:      "reports functions whose body exceeds the line limit (default 60 lines)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/largefunc",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

// maxLines is the configurable threshold.  It is set via the -largefunc.max-lines flag.
var maxLines int

func init() {
	Analyzer.Flags.IntVar(&maxLines, "max-lines", DefaultMaxLines,
		"maximum number of lines permitted in a function body")
}

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s (max-lines=%d)", pass.Pkg.Path(), maxLines)

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

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		checkFuncBodyLength(pass, n, generatedFiles, noLintIndex)
	})
	return nil, nil
}

// checkFuncBodyLength reports a diagnostic when the body of a function
// declaration or literal exceeds maxLines.
func checkFuncBodyLength(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
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

	position := pass.Fset.PositionFor(reportNode.Pos(), false)
	if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
		return
	}

	start := pass.Fset.Position(body.Lbrace)
	end := pass.Fset.Position(body.Rbrace)
	lines := end.Line - start.Line - 1 // subtract 1: exclude the closing brace line, count only body lines

	if lines > maxLines {
		if nolint.HasDirectiveForLinter(position, noLintIndex, "largefunc") {
			return
		}
		pkgLog.Printf("flagging %s: %d lines exceeds limit %d", name, lines, maxLines)
		pass.ReportRangef(
			reportNode,
			"%s is %d lines long (limit: %d); consider breaking it up",
			name, lines, maxLines,
		)
	}
}
