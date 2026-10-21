// Package osisnotexist reports legacy os error predicates that do not inspect
// arbitrary wrapped errors.
package osisnotexist

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the osisnotexist analysis pass.
var Analyzer = analyzerutil.New("osisnotexist", "reports legacy os error predicates that should use wrapped-error-aware checks", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}
	return analyzerutil.Preorder(pass, []ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) ||
			nolint.HasDirectiveForLinter(pos, noLintIndex, "osisnotexist") {
			return
		}
		pkgPath, name, ok := astutil.PackageCall(pass, call)
		if !ok || pkgPath != "os" {
			return
		}
		var replacement string
		switch name {
		case "IsNotExist":
			replacement = "errors.Is(err, fs.ErrNotExist)"
		case "IsExist":
			replacement = "errors.Is(err, fs.ErrExist)"
		case "IsPermission":
			replacement = "errors.Is(err, fs.ErrPermission)"
		case "IsTimeout":
			pass.ReportRangef(call, "os.IsTimeout does not inspect arbitrary wrapped errors; use errors.As with interface{ Timeout() bool } and check Timeout(); errors.Is(err, os.ErrDeadlineExceeded) only covers deadline errors")
			return
		default:
			return
		}
		pass.ReportRangef(call, "os.%s does not inspect arbitrary wrapped errors; use %s instead", name, replacement)
	})
}
