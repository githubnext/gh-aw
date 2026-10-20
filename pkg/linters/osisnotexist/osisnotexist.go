// Package osisnotexist implements a Go analysis linter that flags calls to
// os.IsNotExist, os.IsExist, os.IsPermission, and os.IsTimeout, which do not
// unwrap errors, and suggests errors.Is with the matching io/fs sentinel.
package osisnotexist

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// replacements maps os error predicates to the errors.Is sentinel to use instead.
var replacements = map[string]string{
	"IsNotExist":   "fs.ErrNotExist",
	"IsExist":      "fs.ErrExist",
	"IsPermission": "fs.ErrPermission",
	"IsTimeout":    "os.ErrDeadlineExceeded",
}

// Analyzer is the os-is-not-exist analysis pass.
var Analyzer = analyzerutil.New("osisnotexist", "reports calls to os.IsNotExist, os.IsExist, os.IsPermission, and os.IsTimeout that do not unwrap errors and should use errors.Is instead", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "osisnotexist") {
			return
		}
		fn, ok := astutil.CalledOSFunc(pass, call, "IsNotExist", "IsExist", "IsPermission", "IsTimeout")
		if !ok {
			return
		}
		pass.ReportRangef(call, "os.%s does not unwrap wrapped errors; use errors.Is(err, %s) instead", fn.Name(), replacements[fn.Name()])
	})
}
