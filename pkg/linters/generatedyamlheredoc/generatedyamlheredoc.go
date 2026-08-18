// Package generatedyamlheredoc implements a Go analysis linter that flags
// shell heredocs embedded in generated workflow YAML.
package generatedyamlheredoc

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:generatedyamlheredoc")

const linterName = "generatedyamlheredoc"

// Analyzer is the generated-YAML-heredoc analysis pass.
var Analyzer = analyzerutil.New(
	linterName,
	"reports shell heredocs embedded in generated workflow YAML",
	run,
)

const diagnosticMessage = "generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead (do not base64 encode it)"

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())
	for cur := range insp.Root().Preorder((*ast.BasicLit)(nil)) {
		lit, ok := cur.Node().(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil || !containsShellHeredoc(value) {
			continue
		}

		pos := pass.Fset.PositionFor(lit.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) ||
			nolint.HasDirectiveForLinter(pos, noLintIndex, linterName) {
			continue
		}
		pkgLog.Printf("flagging generated heredoc at %s", pos)
		pass.Report(analysis.Diagnostic{
			Pos:     lit.Pos(),
			End:     lit.End(),
			Message: diagnosticMessage,
		})
	}
	return nil, nil
}

func containsShellHeredoc(value string) bool {
	for line := range strings.SplitSeq(value, "\n") {
		if lineContainsShellHeredoc(line) {
			return true
		}
	}
	return false
}

func lineContainsShellHeredoc(line string) bool {
	for {
		operatorIndex := strings.Index(line, "<<")
		if operatorIndex < 0 {
			return false
		}
		afterIndex := operatorIndex + len("<<")
		if afterIndex < len(line) && line[afterIndex] == '<' {
			line = line[afterIndex+1:]
			continue
		}
		if strings.LastIndex(line[:operatorIndex], "$((") > strings.LastIndex(line[:operatorIndex], "))") {
			line = line[afterIndex:]
			continue
		}
		afterOperator := strings.TrimLeft(line[afterIndex:], " \t")
		if afterOperator == "" {
			return false
		}
		delimiterStart := afterOperator[0]
		return delimiterStart == '-' ||
			delimiterStart == '\'' ||
			delimiterStart == '"' ||
			isShellWordByte(delimiterStart)
	}
}

func isShellWordByte(value byte) bool {
	return value == '_' ||
		value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9'
}
