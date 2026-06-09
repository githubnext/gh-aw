// Package execcommand implements a Go analysis linter that flags
// calls to exec.Command() inside functions that already receive a
// context.Context parameter, suggesting exec.CommandContext(ctx, ...) instead.
package execcommand

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the exec-command analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "execcommand",
	Doc:      "reports exec.Command() calls inside functions that already receive a context.Context parameter; use exec.CommandContext(ctx, ...) instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/execcommand",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok || !isExecCommandCall(pass, call) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}

		for encl := range cur.Enclosing((*ast.FuncDecl)(nil)) {
			fn, ok := encl.Node().(*ast.FuncDecl)
			if !ok {
				continue
			}
			ctxParamName, ok := contextParamName(pass, fn)
			if !ok {
				break
			}

			sel := call.Fun.(*ast.SelectorExpr)
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: "use exec.CommandContext with the context.Context parameter instead of exec.Command()",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message:   "Replace exec.Command() with exec.CommandContext(ctx, ...)",
						TextEdits: suggestedFix(sel, call, ctxParamName),
					},
				},
			})
			break
		}
	}

	return nil, nil
}

// suggestedFix returns the text edits needed to replace exec.Command(args...)
// with exec.CommandContext(ctx, args...).
func suggestedFix(sel *ast.SelectorExpr, call *ast.CallExpr, ctxParamName string) []analysis.TextEdit {
	// Edit 1: rename "Command" to "CommandContext" in the selector.
	renameEdit := analysis.TextEdit{
		Pos:     sel.Sel.Pos(),
		End:     sel.Sel.End(),
		NewText: []byte("CommandContext"),
	}

	// Edit 2: insert the context parameter as the first argument.
	var insertPos token.Pos
	var insertText string
	if len(call.Args) > 0 {
		insertPos = call.Args[0].Pos()
		insertText = ctxParamName + ", "
	} else {
		// exec.Command() with no args is unusual but handle it gracefully.
		insertPos = call.Lparen + 1
		insertText = ctxParamName
	}
	insertEdit := analysis.TextEdit{
		Pos:     insertPos,
		End:     insertPos,
		NewText: []byte(insertText),
	}

	return []analysis.TextEdit{renameEdit, insertEdit}
}

// isExecCommandCall reports whether call is a call to os/exec.Command.
// It uses pass.TypesInfo for type-accurate package matching.
func isExecCommandCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Command" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.Uses[ident]
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "os/exec"
}

// contextParamName returns the first non-blank context.Context parameter name.
func contextParamName(pass *analysis.Pass, fn *ast.FuncDecl) (string, bool) {
	if fn.Type.Params == nil {
		return "", false
	}
	ctxType := contextType(pass)
	if ctxType == nil {
		return "", false
	}
	for _, field := range fn.Type.Params.List {
		t := pass.TypesInfo.TypeOf(field.Type)
		if t == nil {
			continue
		}
		if !types.Identical(t, ctxType) {
			continue
		}
		// At least one name must not be blank.
		for _, name := range field.Names {
			if name.Name != "_" {
				return name.Name, true
			}
		}
	}
	return "", false
}

// contextType returns the types.Type for context.Context, or nil if the
// package is not imported.
func contextType(pass *analysis.Pass) types.Type {
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "context" {
			obj := pkg.Scope().Lookup("Context")
			if obj != nil {
				return obj.Type()
			}
		}
	}
	return nil
}
