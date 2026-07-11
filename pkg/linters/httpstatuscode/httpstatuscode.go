// Package httpstatuscode implements a Go analysis linter that reports
// integer HTTP status code literals used in comparisons that should use
// http.Status* named constants.
package httpstatuscode

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("linters:httpstatuscode")

var Analyzer = &analysis.Analyzer{
	Name:     "httpstatuscode",
	Doc:      "reports integer HTTP status code literals used in comparisons that should use http.Status* named constants",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/httpstatuscode",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var httpStatusNames = map[int]string{
	100: "http.StatusContinue",
	101: "http.StatusSwitchingProtocols",
	102: "http.StatusProcessing",
	103: "http.StatusEarlyHints",
	200: "http.StatusOK",
	201: "http.StatusCreated",
	202: "http.StatusAccepted",
	203: "http.StatusNonAuthoritativeInfo",
	204: "http.StatusNoContent",
	205: "http.StatusResetContent",
	206: "http.StatusPartialContent",
	207: "http.StatusMultiStatus",
	208: "http.StatusAlreadyReported",
	226: "http.StatusIMUsed",
	300: "http.StatusMultipleChoices",
	301: "http.StatusMovedPermanently",
	302: "http.StatusFound",
	303: "http.StatusSeeOther",
	304: "http.StatusNotModified",
	307: "http.StatusTemporaryRedirect",
	308: "http.StatusPermanentRedirect",
	400: "http.StatusBadRequest",
	401: "http.StatusUnauthorized",
	402: "http.StatusPaymentRequired",
	403: "http.StatusForbidden",
	404: "http.StatusNotFound",
	405: "http.StatusMethodNotAllowed",
	406: "http.StatusNotAcceptable",
	407: "http.StatusProxyAuthRequired",
	408: "http.StatusRequestTimeout",
	409: "http.StatusConflict",
	410: "http.StatusGone",
	411: "http.StatusLengthRequired",
	412: "http.StatusPreconditionFailed",
	413: "http.StatusRequestEntityTooLarge",
	414: "http.StatusRequestURITooLong",
	415: "http.StatusUnsupportedMediaType",
	416: "http.StatusRequestedRangeNotSatisfiable",
	417: "http.StatusExpectationFailed",
	418: "http.StatusTeapot",
	421: "http.StatusMisdirectedRequest",
	422: "http.StatusUnprocessableEntity",
	423: "http.StatusLocked",
	424: "http.StatusFailedDependency",
	425: "http.StatusTooEarly",
	426: "http.StatusUpgradeRequired",
	428: "http.StatusPreconditionRequired",
	429: "http.StatusTooManyRequests",
	431: "http.StatusRequestHeaderFieldsTooLarge",
	451: "http.StatusUnavailableForLegalReasons",
	500: "http.StatusInternalServerError",
	501: "http.StatusNotImplemented",
	502: "http.StatusBadGateway",
	503: "http.StatusServiceUnavailable",
	504: "http.StatusGatewayTimeout",
	505: "http.StatusHTTPVersionNotSupported",
	506: "http.StatusVariantAlsoNegotiates",
	507: "http.StatusInsufficientStorage",
	508: "http.StatusLoopDetected",
	510: "http.StatusNotExtended",
	511: "http.StatusNetworkAuthenticationRequired",
}

func run(pass *analysis.Pass) (any, error) {
	log.Printf("analyzing package %s", pass.Pkg.Path())

	root, err := astutil.Root(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "httpstatuscode")

	for cur := range root.Preorder((*ast.BinaryExpr)(nil), (*ast.SwitchStmt)(nil)) {
		switch node := cur.Node().(type) {
		case *ast.BinaryExpr:
			if node.Op != token.EQL && node.Op != token.NEQ {
				continue
			}
			lit, other := extractStatusLiteral(node)
			if lit == nil {
				continue
			}
			if !isHTTPStatusContext(pass, other) {
				continue
			}
			checkAndReport(pass, lit, noLintLinesByFile)
		case *ast.SwitchStmt:
			if node.Tag == nil || !isHTTPStatusContext(pass, node.Tag) {
				continue
			}
			for _, s := range node.Body.List {
				cc, ok := s.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, caseExpr := range cc.List {
					lit, ok := caseExpr.(*ast.BasicLit)
					if !ok || lit.Kind != token.INT {
						continue
					}
					checkAndReport(pass, lit, noLintLinesByFile)
				}
			}
		}
	}

	return nil, nil
}

func checkAndReport(pass *analysis.Pass, lit *ast.BasicLit, noLintLinesByFile map[string]map[int]struct{}) {
	code64, err := strconv.ParseInt(lit.Value, 0, 64)
	if err != nil || code64 < 100 || code64 > 599 {
		return
	}
	code := int(code64)

	pos := pass.Fset.PositionFor(lit.Pos(), false)
	if filecheck.IsTestFile(pos.Filename) {
		return
	}
	if nolint.HasDirective(pos, noLintLinesByFile) {
		return
	}

	log.Printf("flagging magic HTTP status code %d at %s", code, pos)

	if name, ok := httpStatusNames[code]; ok {
		pass.Reportf(lit.Pos(), "use %s instead of magic HTTP status code %d", name, code)
		return
	}
	pass.Reportf(lit.Pos(), "use http.Status* constant instead of magic HTTP status code %d", code)
}

func extractStatusLiteral(expr *ast.BinaryExpr) (*ast.BasicLit, ast.Expr) {
	if lit, ok := expr.X.(*ast.BasicLit); ok && lit.Kind == token.INT {
		return lit, expr.Y
	}
	if lit, ok := expr.Y.(*ast.BasicLit); ok && lit.Kind == token.INT {
		return lit, expr.X
	}
	return nil, nil
}

func isHTTPStatusContext(pass *analysis.Pass, expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		obj, ok := pass.TypesInfo.Uses[e]
		if !ok {
			return false
		}
		t := obj.Type()
		if !isIntegerType(t) {
			return false
		}
		// For named integer types (custom enums/aliases), check whether the type
		// name itself indicates HTTP status to avoid false positives on non-HTTP
		// integer types (e.g. type JobState int).
		if named, isNamed := t.(*types.Named); isNamed {
			return isHTTPStatusTypeName(named.Obj().Name())
		}
		// For plain integer types, fall back to variable name heuristic.
		return isHTTPStatusVarName(e.Name)
	case *ast.SelectorExpr:
		if !isHTTPStatusFieldName(e.Sel.Name) {
			return false
		}
		if sel, ok := pass.TypesInfo.Selections[e]; ok {
			return isIntegerType(sel.Type())
		}
		obj, ok := pass.TypesInfo.Uses[e.Sel]
		if !ok {
			return false
		}
		field, ok := obj.(*types.Var)
		if !ok || !field.IsField() {
			return false
		}
		return isIntegerType(field.Type())
	}
	return false
}

// isHTTPStatusVarName returns true if a plain-integer variable/parameter name
// suggests it holds an HTTP status code.
func isHTTPStatusVarName(name string) bool {
	switch name {
	case "status", "statusCode", "httpStatus":
		return true
	}
	return false
}

// isHTTPStatusFieldName returns true if a struct field name suggests HTTP status.
// Accepts StatusCode, Status, and HTTPStatus to cover common response field spellings.
func isHTTPStatusFieldName(name string) bool {
	switch name {
	case "StatusCode", "Status", "HTTPStatus":
		return true
	}
	return false
}

// isHTTPStatusTypeName returns true if a named integer type's name indicates that
// it represents an HTTP status code (e.g. HTTPStatusCode, HTTPStatus).
// Both "http" and "status" must appear in the name (case-insensitive) to avoid
// matching unrelated HTTP types such as HTTPVersion or HTTPMethod.
func isHTTPStatusTypeName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "http") && strings.Contains(lower, "status")
}

func isIntegerType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0
}
