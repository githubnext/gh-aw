// Package packagelevelmutableslicemap implements a Go analysis linter that
// flags package-level (file/package-scope) var declarations of slices or maps
// that are mutated from inside a function body via append re-assignment,
// index assignment, or delete().
//
// Package-level mutable slices/maps are shared across every goroutine and
// every call into the package for the lifetime of the process. Mutating one
// from inside a function — rather than storing the state on a struct or
// returning fresh values — risks data races under concurrent access and can
// leak state between unrelated calls.
//
// Mutations inside a top-level init() function are not reported: init runs
// exactly once before any other code, so it is the idiomatic place to
// populate package-level state.
package packagelevelmutableslicemap

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the package-level-mutable-slice-map analysis pass.
var Analyzer = analyzerutil.New("packagelevelmutableslicemap", "reports mutation of package-level slice/map variables from inside function bodies, which risks data races and cross-call state leaks", run)

func run(pass *analysis.Pass) (any, error) {
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

	targets := collectPackageLevelSliceMapVars(pass)
	if len(targets) == 0 {
		return nil, nil
	}

	for cur := range insp.Root().Preorder((*ast.AssignStmt)(nil), (*ast.ExprStmt)(nil)) {
		if isInInitFunction(cur) {
			continue
		}
		analyzeNode(pass, cur.Node(), targets, generatedFiles, noLintIndex)
	}
	return nil, nil
}

// isInInitFunction reports whether cur is inside a top-level init() function.
// Only top-level (no receiver) init functions are recognized; methods named
// init are ordinary methods and are not exempt.
func isInInitFunction(cur inspector.Cursor) bool {
	for encl := range cur.Enclosing((*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)) {
		decl, ok := encl.Node().(*ast.FuncDecl)
		if !ok {
			// Innermost enclosing function is a literal (e.g. a goroutine
			// started from init), which is not exempt.
			return false
		}
		return decl.Recv == nil && decl.Name != nil && decl.Name.Name == "init"
	}
	return false
}

// collectPackageLevelSliceMapVars scans the top-level declarations of every
// file in the package and returns the set of package-scope var objects whose
// underlying type is a slice or a map (including named wrapper types such as
// `type registry map[string]int`), keyed by their declared name.
func collectPackageLevelSliceMapVars(pass *analysis.Pass) map[types.Object]string {
	targets := make(map[types.Object]string)
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range valueSpec.Names {
					if name.Name == "_" {
						continue
					}
					obj := pass.TypesInfo.Defs[name]
					if obj == nil {
						continue
					}
					t := obj.Type()
					if t == nil {
						continue
					}
					switch t.Underlying().(type) {
					case *types.Slice, *types.Map:
						targets[obj] = name.Name
					}
				}
			}
		}
	}
	return targets
}

func analyzeNode(pass *analysis.Pass, n ast.Node, targets map[types.Object]string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	switch stmt := n.(type) {
	case *ast.AssignStmt:
		analyzeAssignStmt(pass, stmt, targets, generatedFiles, noLintIndex)
	case *ast.ExprStmt:
		analyzeExprStmt(pass, stmt, targets, generatedFiles, noLintIndex)
	}
}

func analyzeAssignStmt(pass *analysis.Pass, stmt *ast.AssignStmt, targets map[types.Object]string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	if name, ok := matchAppendReassign(pass, stmt, targets); ok {
		report(pass, stmt.Pos(), name, "append() re-assignment", generatedFiles, noLintIndex)
		return
	}
	if name, ok := matchIndexAssign(pass, stmt, targets); ok {
		report(pass, stmt.Pos(), name, "index assignment", generatedFiles, noLintIndex)
	}
}

func analyzeExprStmt(pass *analysis.Pass, stmt *ast.ExprStmt, targets map[types.Object]string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return
	}
	if !isBuiltinCall(pass, call, "delete") || len(call.Args) != 2 {
		return
	}
	name, ok := targetBaseName(pass, call.Args[0], targets)
	if !ok {
		return
	}
	report(pass, stmt.Pos(), name, "delete()", generatedFiles, noLintIndex)
}

// matchAppendReassign reports whether stmt re-assigns a tracked package-level
// target from an append() call, returning its declared name. Every LHS/RHS
// pair is inspected so parallel assignments such as
// `globalSlice, err = append(globalSlice, v), nil` are matched too, and the
// append arguments are not required to reference the target itself, so
// `globalSlice = append(otherSlice, v)` is matched as well.
func matchAppendReassign(pass *analysis.Pass, stmt *ast.AssignStmt, targets map[types.Object]string) (string, bool) {
	for i, lhs := range stmt.Lhs {
		lhsIdent, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		name, tracked := targets[pass.TypesInfo.Uses[lhsIdent]]
		if !tracked {
			continue
		}
		rhs, ok := astutil.RhsExprForIndex(stmt.Rhs, i)
		if !ok {
			continue
		}
		call, ok := rhs.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			continue
		}
		if isBuiltinCall(pass, call, "append") {
			return name, true
		}
	}
	return "", false
}

// matchIndexAssign reports whether stmt assigns into m[k] for a tracked
// package-level map/slice target m.
func matchIndexAssign(pass *analysis.Pass, stmt *ast.AssignStmt, targets map[types.Object]string) (string, bool) {
	for _, lhs := range stmt.Lhs {
		idxExpr, ok := lhs.(*ast.IndexExpr)
		if !ok {
			continue
		}
		if name, ok := targetBaseName(pass, idxExpr, targets); ok {
			return name, true
		}
	}
	return "", false
}

// isBuiltinCall reports whether call invokes the named Go builtin.
func isBuiltinCall(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != name {
		return false
	}
	builtin, ok := pass.TypesInfo.Uses[ident].(*types.Builtin)
	return ok && builtin.Name() == name
}

// targetBaseName reports whether expr is rooted at an identifier referring to
// a tracked package-level target, returning its declared name. Nested index
// expressions such as nested[a][b] resolve to their base identifier so
// mutations of nested collections are reported too.
func targetBaseName(pass *analysis.Pass, expr ast.Expr, targets map[types.Object]string) (string, bool) {
	for {
		switch e := expr.(type) {
		case *ast.Ident:
			obj := pass.TypesInfo.Uses[e]
			if obj == nil {
				return "", false
			}
			name, ok := targets[obj]
			return name, ok
		case *ast.IndexExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		default:
			return "", false
		}
	}
}

func report(pass *analysis.Pass, pos token.Pos, varName, kind string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	position := pass.Fset.PositionFor(pos, false)
	if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(position, noLintIndex, "packagelevelmutableslicemap") {
		return
	}
	pass.Report(analysis.Diagnostic{
		Pos:     pos,
		Message: "package-level slice/map variable " + varName + " is mutated via " + kind + "; mutating shared package state risks data races and can leak state across calls",
	})
}
