// Package analyzerutil provides shared analyzer setup for custom linters.
package analyzerutil

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

const repositoryURL = "https://github.com/github/gh-aw/tree/main/pkg/linters/"

// New creates an analyzer with the standard linter dependencies and URL.
func New(name, doc string, run func(*analysis.Pass) (any, error)) *analysis.Analyzer {
	return NewAtPath(name, doc, name, run)
}

// NewAtPath creates an analyzer with the standard linter dependencies and URL
// for packagePath.
func NewAtPath(name, doc, packagePath string, run func(*analysis.Pass) (any, error)) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     name,
		Doc:      doc,
		URL:      repositoryURL + packagePath,
		Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
		Run:      run,
	}
}

// Indexes builds the nolint directive index and the generated-file index that
// every linter needs before walking the AST.
func Indexes(pass *analysis.Pass) (nolint.DirectiveIndex, filecheck.GeneratedIndex, error) {
	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, nil, err
	}
	return noLintIndex, generatedFiles, nil
}

// PreorderIndexed builds the shared nolint and generated-file indexes and runs
// visit for each node matching nodeFilter.
func PreorderIndexed(pass *analysis.Pass, nodeFilter []ast.Node, visit func(*analysis.Pass, ast.Node, filecheck.GeneratedIndex, nolint.DirectiveIndex)) (any, error) {
	noLintIndex, generatedFiles, err := Indexes(pass)
	if err != nil {
		return nil, err
	}
	return Preorder(pass, nodeFilter, func(n ast.Node) {
		visit(pass, n, generatedFiles, noLintIndex)
	})
}

// Preorder runs fn for each node matching nodeFilter.
func Preorder(pass *analysis.Pass, nodeFilter []ast.Node, fn func(ast.Node)) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	for cur := range insp.Root().Preorder(nodeFilter...) {
		fn(cur.Node())
	}
	return nil, nil
}
