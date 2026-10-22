// Package stringshasprefixslice implements a Go analysis linter that flags
// `if strings.HasPrefix(s, p) { ... s[len(p):] ... }` patterns that should use
// strings.CutPrefix (or strings.TrimPrefix) instead.
package stringshasprefixslice

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the strings-has-prefix-slice analysis pass.
var Analyzer = analyzerutil.New("stringshasprefixslice", "reports if strings.HasPrefix(s, p) blocks that manually slice s[len(p):] and should use strings.CutPrefix instead", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.IfStmt)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeIf(pass, n, generatedFiles, noLintIndex)
	})
}

// analyzeIf checks whether an if statement guards a manual prefix slice with
// strings.HasPrefix on the same operands.
func analyzeIf(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	stmt, ok := n.(*ast.IfStmt)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(stmt.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringshasprefixslice") {
		return
	}

	call, ok := astutil.AsStringsMethodCall(pass, astutil.UnwrapParenExpr(stmt.Cond), "HasPrefix")
	if !ok || len(call.Args) != 2 {
		return
	}
	sText := astutil.NodeText(pass.Fset, call.Args[0])
	pText := astutil.NodeText(pass.Fset, call.Args[1])
	if sText == "" || pText == "" {
		return
	}

	found := false
	ast.Inspect(stmt.Body, func(node ast.Node) bool {
		if found {
			return false
		}
		slice, ok := node.(*ast.SliceExpr)
		if !ok || slice.High != nil || slice.Max != nil || slice.Low == nil {
			return true
		}
		lenCall, ok := astutil.UnwrapParenExpr(slice.Low).(*ast.CallExpr)
		if !ok || len(lenCall.Args) != 1 {
			return true
		}
		if id, ok := lenCall.Fun.(*ast.Ident); !ok || id.Name != "len" {
			return true
		}
		if astutil.NodeText(pass.Fset, slice.X) == sText && astutil.NodeText(pass.Fset, lenCall.Args[0]) == pText {
			found = true
		}
		return !found
	})
	if !found {
		return
	}

	pass.Report(analysis.Diagnostic{
		Pos:     stmt.Pos(),
		End:     stmt.Cond.End(),
		Message: "use strings.CutPrefix(" + sText + ", " + pText + ") instead of strings.HasPrefix followed by slicing with len(" + pText + ")",
	})
}
