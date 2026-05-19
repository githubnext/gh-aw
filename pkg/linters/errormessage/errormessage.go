// Package errormessage implements a Go analysis linter that enforces
// actionable error-message patterns in changed files.
package errormessage

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	// changedFilesCSV allows CI to scope linting to changed files only,
	// preventing legacy violations from blocking incremental adoption.
	changedFilesCSV string
)

// Analyzer is the errormessage analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "errormessage",
	Doc:      "reports non-actionable error message patterns in changed files",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/errormessage",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func init() {
	Analyzer.Flags.StringVar(&changedFilesCSV, "changed-files", "", "comma-separated list of changed file paths to lint (when empty, analyzer is a no-op)")
}

func run(pass *analysis.Pass) (any, error) {
	changed := parseChangedFiles(changedFilesCSV)
	if len(changed) == 0 {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	noLintLinesByFile := buildNoLintLineIndex(pass)

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if !shouldCheckFile(pos.Filename, changed) || strings.HasSuffix(pos.Filename, "_test.go") {
			return
		}
		if hasNoLintDirective(pos, noLintLinesByFile) {
			return
		}

		if msg, ok := extractLiteralErrorMessage(call); ok && returnsError(pass, call) {
			checkNegativeLanguage(pass, call, msg)
			checkGenericWrap(pass, call, msg)
			checkValidationFmtErrorf(pass, call, pos.Filename)
		}

		if !isNewValidationErrorCall(call) {
			return
		}

		checkNewValidationSuggestion(pass, call)
	})

	return nil, nil
}

func parseChangedFiles(csv string) map[string]struct{} {
	changed := map[string]struct{}{}
	for part := range strings.SplitSeq(csv, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		normalized := filepath.ToSlash(trimmed)
		changed[normalized] = struct{}{}
	}
	return changed
}

func shouldCheckFile(filename string, changed map[string]struct{}) bool {
	path := filepath.ToSlash(filename)
	for changedPath := range changed {
		if path == changedPath || strings.HasSuffix(path, "/"+changedPath) {
			return true
		}
	}
	return false
}

func extractLiteralErrorMessage(call *ast.CallExpr) (string, bool) {
	if len(call.Args) == 0 {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return unquoted, true
}

func isFmtErrorf(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "fmt" && sel.Sel.Name == "Errorf"
}

func isNewValidationErrorCall(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name == "NewValidationError"
	case *ast.SelectorExpr:
		return fun.Sel.Name == "NewValidationError"
	default:
		return false
	}
}

func checkValidationFmtErrorf(pass *analysis.Pass, call *ast.CallExpr, filename string) {
	if !strings.HasSuffix(filename, "_validation.go") || !isFmtErrorf(call) {
		return
	}
	pass.Reportf(call.Pos(), "use NewValidationError(...) instead of fmt.Errorf(...) in validation files")
}

func checkNegativeLanguage(pass *analysis.Pass, call *ast.CallExpr, msg string) {
	lower := strings.ToLower(msg)
	if !containsAnyKeyword(lower, "invalid", "cannot", "must", "failed") {
		return
	}
	if containsAnyKeyword(lower, "expected", "requires", "should", "example", "valid") {
		return
	}
	pass.Reportf(call.Pos(), "error message uses negative language without constructive guidance; include expected/requires/should/example details")
}

func checkNewValidationSuggestion(pass *analysis.Pass, call *ast.CallExpr) {
	if len(call.Args) < 4 {
		pass.Reportf(call.Pos(), "NewValidationError(...) should include a non-empty suggestion with an example")
		return
	}

	suggestion, ok := extractStringLiteral(call.Args[3])
	if !ok {
		return
	}

	if strings.TrimSpace(suggestion) == "" {
		pass.Reportf(call.Pos(), "NewValidationError(...) suggestion must not be empty")
		return
	}

	lower := strings.ToLower(suggestion)
	if !strings.Contains(lower, "example") && !looksLikeYAMLExample(suggestion) {
		pass.Reportf(call.Pos(), "NewValidationError(...) suggestion should include an example (for example: YAML snippet)")
	}
}

func checkGenericWrap(pass *analysis.Pass, call *ast.CallExpr, msg string) {
	if !isFmtErrorf(call) {
		return
	}
	if strings.HasPrefix(strings.ToLower(msg), "failed to ") && strings.Contains(msg, ": %w") {
		pass.Reportf(call.Pos(), "avoid generic 'failed to ...: %%w' wrapping; add specific recovery guidance")
	}
}

func extractStringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func looksLikeYAMLExample(s string) bool {
	trimmed := strings.TrimSpace(s)
	if strings.Contains(trimmed, "\n") && strings.Contains(trimmed, ":") {
		return true
	}
	return strings.Contains(trimmed, ":") && strings.Contains(trimmed, " ")
}

func containsAnyKeyword(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if containsKeyword(s, keyword) {
			return true
		}
	}
	return false
}

func containsKeyword(s, keyword string) bool {
	offset := 0
	for {
		i := strings.Index(s[offset:], keyword)
		if i < 0 {
			return false
		}
		start := offset + i
		end := start + len(keyword)
		if isWordBoundary(s, start-1) && isWordBoundary(s, end) {
			return true
		}
		offset = start + 1
	}
}

func isWordBoundary(s string, idx int) bool {
	if idx < 0 || idx >= len(s) {
		return true
	}
	ch := s[idx]
	return (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '_'
}

func returnsError(pass *analysis.Pass, call *ast.CallExpr) bool {
	t := pass.TypesInfo.TypeOf(call)
	if t == nil {
		return false
	}
	return implementsError(t)
}

func implementsError(t types.Type) bool {
	obj := types.Universe.Lookup("error")
	if obj == nil {
		return false
	}
	errIface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return false
	}

	if types.Implements(t, errIface) {
		return true
	}
	if p, ok := t.(*types.Pointer); ok {
		return types.Implements(p, errIface)
	}
	return types.Implements(types.NewPointer(t), errIface)
}

func hasNoLintDirective(position token.Position, noLintLinesByFile map[string]map[int]struct{}) bool {
	if position.Filename == "" {
		return false
	}

	noLintLines := noLintLinesByFile[position.Filename]
	if noLintLines == nil {
		return false
	}

	_, sameLine := noLintLines[position.Line]
	_, previousLine := noLintLines[position.Line-1]
	return sameLine || previousLine
}

func buildNoLintLineIndex(pass *analysis.Pass) map[string]map[int]struct{} {
	noLintLinesByFile := make(map[string]map[int]struct{}, len(pass.Files))
	for _, file := range pass.Files {
		filename := pass.Fset.PositionFor(file.Pos(), false).Filename
		if filename == "" {
			continue
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				text := strings.TrimPrefix(comment.Text, "//")
				if !strings.HasPrefix(text, "nolint:errormessage") && !strings.HasPrefix(text, "nolint:all") {
					continue
				}
				line := pass.Fset.PositionFor(comment.Slash, false).Line
				if noLintLinesByFile[filename] == nil {
					noLintLinesByFile[filename] = make(map[int]struct{})
				}
				noLintLinesByFile[filename][line] = struct{}{}
			}
		}
	}
	return noLintLinesByFile
}
