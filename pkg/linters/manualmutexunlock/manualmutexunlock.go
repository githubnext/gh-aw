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
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the manual-mutex-unlock analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "manualmutexunlock",
	Doc:      "reports mutex Unlock() calls that are not deferred",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/manualmutexunlock",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		pos := pass.Fset.PositionFor(fn.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		// Track mutex variables: types.Object -> *mutexVarState (lock position, hasDefer, hasManualUnlock)
		mutexVars := make(map[types.Object]*mutexVarState)

		// Walk all statements in the function body
		ast.Inspect(fn.Body, func(node ast.Node) bool {
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
					if obj := getLockCallObj(pass, call); obj != nil {
						// If this mutex was already tracked from a prior lock on the same
						// binding, report any unresolved violation before overwriting state.
						if prev, exists := mutexVars[obj]; exists && prev.hasManualUnlock && !prev.hasDefer {
							pass.Report(analysis.Diagnostic{
								Pos:     prev.lockPos,
								Message: "mutex Unlock() should be deferred immediately after Lock() to prevent deadlocks on panic or early return",
							})
						}
						mutexVars[obj] = &mutexVarState{
							lockPos: call.Pos(),
						}
					}
				}
			}

			// Look for defer mu.Unlock()
			if deferStmt, ok := node.(*ast.DeferStmt); ok {
				if obj := getUnlockCallObj(pass, deferStmt.Call); obj != nil {
					if state, found := mutexVars[obj]; found {
						state.hasDefer = true
					}
				}
			}

			// Look for non-deferred mu.Unlock() in expression statements
			if exprStmt, ok := node.(*ast.ExprStmt); ok {
				if call, ok := exprStmt.X.(*ast.CallExpr); ok {
					if obj := getUnlockCallObj(pass, call); obj != nil {
						if state, found := mutexVars[obj]; found {
							state.hasManualUnlock = true
						}
					}
				}
			}

			return true
		})

		// Report mutexes with manual unlock but no defer
		for _, state := range mutexVars {
			if state.hasManualUnlock && !state.hasDefer {
				pass.Report(analysis.Diagnostic{
					Pos:     state.lockPos,
					Message: "mutex Unlock() should be deferred immediately after Lock() to prevent deadlocks on panic or early return",
				})
			}
		}
	})

	return nil, nil
}

type mutexVarState struct {
	lockPos         token.Pos
	hasDefer        bool
	hasManualUnlock bool
}

// getLockCallObj returns the types.Object for the receiver if call is like mu.Lock() or mu.RLock()
func getLockCallObj(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if sel.Sel.Name != "Lock" && sel.Sel.Name != "RLock" {
		return nil
	}
	return getMutexReceiverObj(pass, sel.X)
}

// getUnlockCallObj returns the types.Object for the receiver if call is like mu.Unlock() or mu.RUnlock()
func getUnlockCallObj(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if sel.Sel.Name != "Unlock" && sel.Sel.Name != "RUnlock" {
		return nil
	}
	return getMutexReceiverObj(pass, sel.X)
}

func getMutexReceiverObj(pass *analysis.Pass, recv ast.Expr) types.Object {
	if !isMutexType(pass.TypesInfo.TypeOf(recv)) {
		return nil
	}

	switch r := recv.(type) {
	case *ast.Ident:
		return pass.TypesInfo.ObjectOf(r)
	case *ast.SelectorExpr:
		if sel := pass.TypesInfo.Selections[r]; sel != nil {
			return sel.Obj()
		}
	}
	return nil
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
