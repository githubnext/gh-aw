// Package manualmutexunlock implements a Go analysis linter that flags
// mutex Unlock() calls that are not deferred, which can lead to deadlocks
// if a panic or early return occurs between Lock() and Unlock().
package manualmutexunlock

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// mutexKey uniquely identifies a mutex receiver so that distinct struct
// instances holding the same field type are tracked independently.
//
// For a direct local/parameter variable (e.g. `mu`), base is the variable's
// types.Object and field is nil.
//
// For a field selector (e.g. `a.mu`), base is the types.Object of the
// receiver variable `a` and field is the types.Object of the field `mu`.
// This prevents `a.mu` and `b.mu` from collapsing to the same key even
// though both resolve to the same field declaration.
//
// When the base expression is not a simple identifier (e.g. `getGuard().mu`),
// base is set to the field's types.Object and field is nil, matching the
// pre-existing behaviour for non-addressable expressions.
type mutexKey struct {
	base  types.Object
	field types.Object
}

// Analyzer is the manual-mutex-unlock analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "manualmutexunlock",
	Doc:      "reports mutex Unlock() calls that are not deferred",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/manualmutexunlock",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "manualmutexunlock")

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		inspectMutexFuncDecl(pass, noLintLinesByFile, n)
	})

	return nil, nil
}

func inspectMutexFuncDecl(pass *analysis.Pass, noLintLinesByFile map[string]map[int]struct{}, n ast.Node) {
	fn, ok := n.(*ast.FuncDecl)
	if !ok || fn.Body == nil {
		return
	}

	pos := pass.Fset.PositionFor(fn.Pos(), false)
	if filecheck.IsTestFile(pos.Filename) {
		return
	}

	// Track mutex variables: mutexKey -> *mutexVarState (lock position, hasDefer, hasManualUnlock)
	mutexVars := make(map[mutexKey]*mutexVarState)

	// Walk all statements in the function body
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		return inspectMutexNode(pass, noLintLinesByFile, mutexVars, node)
	})

	// Report mutexes with manual unlock but no defer
	for _, state := range mutexVars {
		if state.hasManualUnlock && !state.hasDefer {
			position := pass.Fset.PositionFor(state.lockPos, false)
			if nolint.HasDirective(position, noLintLinesByFile) {
				continue
			}
			pass.Report(analysis.Diagnostic{
				Pos:     state.lockPos,
				Message: "mutex Unlock() should be deferred immediately after Lock() to prevent deadlocks on panic or early return",
			})
		}
	}
}

func inspectMutexNode(pass *analysis.Pass, noLintLinesByFile map[string]map[int]struct{}, mutexVars map[mutexKey]*mutexVarState, node ast.Node) bool {
	if node == nil {
		return false
	}

	// Do not descend into function literals — closures are independent
	if _, ok := node.(*ast.FuncLit); ok {
		return false
	}

	// Look for mutex Lock() calls
	if exprStmt, ok := node.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			if key, ok := getLockCallKey(pass, call); ok {
				// If this mutex was already tracked from a prior lock on the same
				// binding, report any unresolved violation before overwriting state.
				if prev, exists := mutexVars[key]; exists && prev.hasManualUnlock && !prev.hasDefer {
					position := pass.Fset.PositionFor(prev.lockPos, false)
					if nolint.HasDirective(position, noLintLinesByFile) {
						mutexVars[key] = &mutexVarState{
							lockPos: call.Pos(),
						}
						return true
					}
					pass.Report(analysis.Diagnostic{
						Pos:     prev.lockPos,
						Message: "mutex Unlock() should be deferred immediately after Lock() to prevent deadlocks on panic or early return",
					})
				}
				mutexVars[key] = &mutexVarState{
					lockPos: call.Pos(),
				}
			}
		}
	}

	// Look for defer mu.Unlock()
	if deferStmt, ok := node.(*ast.DeferStmt); ok {
		if key, ok := getUnlockCallKey(pass, deferStmt.Call); ok {
			if state, found := mutexVars[key]; found {
				state.hasDefer = true
			}
		}
	}

	// Look for non-deferred mu.Unlock() in expression statements
	if exprStmt, ok := node.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			if key, ok := getUnlockCallKey(pass, call); ok {
				if state, found := mutexVars[key]; found {
					state.hasManualUnlock = true
				}
			}
		}
	}

	return true
}

type mutexVarState struct {
	lockPos         token.Pos
	hasDefer        bool
	hasManualUnlock bool
}

// getLockCallKey returns the mutexKey for the receiver if call is like mu.Lock() or mu.RLock()
func getLockCallKey(pass *analysis.Pass, call *ast.CallExpr) (mutexKey, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return mutexKey{}, false
	}
	if sel.Sel.Name != "Lock" && sel.Sel.Name != "RLock" {
		return mutexKey{}, false
	}
	return getMutexReceiverKey(pass, sel.X)
}

// getUnlockCallKey returns the mutexKey for the receiver if call is like mu.Unlock() or mu.RUnlock()
func getUnlockCallKey(pass *analysis.Pass, call *ast.CallExpr) (mutexKey, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return mutexKey{}, false
	}
	if sel.Sel.Name != "Unlock" && sel.Sel.Name != "RUnlock" {
		return mutexKey{}, false
	}
	return getMutexReceiverKey(pass, sel.X)
}

func getMutexReceiverKey(pass *analysis.Pass, recv ast.Expr) (mutexKey, bool) {
	if !isMutexType(pass.TypesInfo.TypeOf(recv)) {
		return mutexKey{}, false
	}

	switch r := recv.(type) {
	case *ast.Ident:
		obj := pass.TypesInfo.ObjectOf(r)
		if obj == nil {
			return mutexKey{}, false
		}
		return mutexKey{base: obj}, true
	case *ast.SelectorExpr:
		if sel := pass.TypesInfo.Selections[r]; sel != nil {
			fieldObj := sel.Obj()
			// When the base is a plain identifier (the common case: `a.mu`),
			// build a composite key (base var, field) so that distinct
			// instances of the same struct type are tracked independently.
			baseIdent, ok := r.X.(*ast.Ident)
			if !ok {
				// Fall back for non-ident base expressions (e.g. `getGuard().mu`):
				// use the field object alone as the key, matching prior behaviour.
				return mutexKey{base: fieldObj}, true
			}
			baseObj := pass.TypesInfo.ObjectOf(baseIdent)
			if baseObj == nil {
				return mutexKey{base: fieldObj}, true
			}
			return mutexKey{base: baseObj, field: fieldObj}, true
		}
	}
	return mutexKey{}, false
}

// isMutexType returns true if t is sync.Mutex, sync.RWMutex, or a pointer to one
func isMutexType(t types.Type) bool {
	if t == nil {
		return false
	}

	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Pkg().Path() == "sync" && (obj.Name() == "Mutex" || obj.Name() == "RWMutex")
}
