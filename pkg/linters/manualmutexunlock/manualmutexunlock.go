// Package manualmutexunlock implements a Go analysis linter that flags
// mutex Unlock() calls that are not deferred, which can lead to deadlocks
// if a panic or early return occurs between Lock() and Unlock().
package manualmutexunlock

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/resourcetracker"
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
var Analyzer = resourcetracker.NewAnalyzer(resourcetracker.Config[mutexKey]{
	Name:         "manualmutexunlock",
	Doc:          "reports mutex Unlock() calls that are not deferred",
	Message:      "mutex Unlock() should be deferred immediately after Lock() to prevent deadlocks on panic or early return",
	Acquisitions: mutexLockAcquisitions,
	CleanupKey:   unlockCallKey,
})

// mutexLockAcquisitions reports mutexes locked by statements such as mu.Lock().
func mutexLockAcquisitions(pass *analysis.Pass, node ast.Node) []resourcetracker.Acquisition[mutexKey] {
	exprStmt, ok := node.(*ast.ExprStmt)
	if !ok {
		return nil
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return nil
	}
	key, ok := lockCallKey(pass, call)
	if !ok {
		return nil
	}
	return []resourcetracker.Acquisition[mutexKey]{{Key: key, Pos: call.Pos()}}
}

// lockCallKey returns the mutexKey for the receiver if call is like mu.Lock() or mu.RLock()
func lockCallKey(pass *analysis.Pass, call *ast.CallExpr) (mutexKey, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return mutexKey{}, false
	}
	if sel.Sel.Name != "Lock" && sel.Sel.Name != "RLock" {
		return mutexKey{}, false
	}
	return getMutexReceiverKey(pass, sel.X)
}

// unlockCallKey returns the mutexKey for the receiver if call is like mu.Unlock() or mu.RUnlock()
func unlockCallKey(pass *analysis.Pass, call *ast.CallExpr) (mutexKey, bool) {
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
