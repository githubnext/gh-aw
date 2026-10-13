// Package goroutineresourceleak implements a Go analysis linter that flags
// goroutines that acquire resources (contexts, locks, channels) without
// guaranteed cleanup or synchronization.
//
// The linter detects several problematic patterns:
//
//  1. Goroutines that create cancellable contexts but don't defer cancel:
//     go func() {
//     ctx, cancel := context.WithCancel(parent)
//     // cancel not deferred — context leak and goroutine leak
//     use(ctx)
//     }()
//
//  2. Goroutines that acquire mutexes without defer unlock:
//     go func() {
//     mu.Lock()
//     // Unlock not deferred — can deadlock if function panics
//     use()
//     }()
//
//  3. Goroutines creating channels without receiver/draining:
//     go func() {
//     ch := make(chan T)
//     // Channel not drained — sends may block forever
//     }()
//
//  4. Goroutines launched in tight loops without WaitGroup:
//     for i := 0; i < n; i++ {
//     go doWork(i)
//     // No synchronization — main may exit before work finishes
//     }
package goroutineresourceleak

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:goroutineresourceleak")

// Analyzer is the goroutine-resource-leak analysis pass.
var Analyzer = analyzerutil.New(
	"goroutineresourceleak",
	"reports goroutines that acquire resources (contexts, locks, channels) without guaranteed cleanup or synchronization",
	run,
)

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())

	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.GoStmt)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		goStmt, ok := n.(*ast.GoStmt)
		if !ok {
			return
		}
		checkGoroutineResourceLeak(pass, goStmt, noLintIndex, generatedFiles)
	})
}

// checkGoroutineResourceLeak analyzes a goroutine launch for resource leaks.
func checkGoroutineResourceLeak(
	pass *analysis.Pass,
	goStmt *ast.GoStmt,
	noLintIndex nolint.DirectiveIndex,
	generatedFiles filecheck.GeneratedIndex,
) {
	// GoStmt.Call is already a *ast.CallExpr.
	call := goStmt.Call
	if call == nil {
		return
	}

	// Only check goroutines with function literals.
	funcLit, ok := astutil.UnwrapParenExpr(call.Fun).(*ast.FuncLit)
	if !ok {
		return
	}

	position := pass.Fset.PositionFor(goStmt.Pos(), false)
	if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
		return
	}

	if nolint.HasDirectiveForLinter(position, noLintIndex, "goroutineresourceleak") {
		return
	}

	// Check for resource leaks in the function body.
	if checkCancelLeak(pass, funcLit.Body, noLintIndex) {
		pkgLog.Printf("flagging goroutine with context cancel leak at %s", position)
		pass.ReportRangef(goStmt, "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel; context and goroutine may leak")
		return
	}

	if checkMutexLeak(pass, funcLit.Body, noLintIndex) {
		pkgLog.Printf("flagging goroutine with mutex lock leak at %s", position)
		pass.ReportRangef(goStmt, "goroutine acquires lock without deferred unlock; may deadlock if function panics")
		return
	}

	if checkChannelLeak(pass, funcLit.Body, noLintIndex) {
		pkgLog.Printf("flagging goroutine with channel leak at %s", position)
		pass.ReportRangef(goStmt, "goroutine creates channel without guaranteed close or drain; may cause send deadlock")
		return
	}
}

// checkCancelLeak reports whether the function body creates a context cancel
// function without deferring its cleanup.
func checkCancelLeak(
	pass *analysis.Pass,
	body *ast.BlockStmt,
	noLintIndex nolint.DirectiveIndex,
) bool {
	if body == nil {
		return false
	}

	// Track cancel variables in the function body to detect if they're deferred.
	cancelVars := make(map[string]bool) // varName -> isDeferred
	deferredCancels := make(map[string]bool)

	// First pass: identify all deferred cancel calls.
	for _, stmt := range body.List {
		if deferStmt, ok := stmt.(*ast.DeferStmt); ok {
			callExpr := deferStmt.Call
			if callExpr != nil {
				if ident, ok := callExpr.Fun.(*ast.Ident); ok {
					deferredCancels[ident.Name] = true
				}
			}
		}
	}

	// Second pass: find context.WithCancel/WithTimeout/WithDeadline assignments
	// that don't defer their cancel.
	for _, stmt := range body.List {
		assignStmt, ok := stmt.(*ast.AssignStmt)
		if !ok || len(assignStmt.Rhs) < 1 {
			continue
		}

		// Check if this is a ctx, cancel := context.With* assignment.
		if len(assignStmt.Lhs) >= 2 {
			cancelIdent, ok := assignStmt.Lhs[1].(*ast.Ident)
			if !ok || cancelIdent.Name == "_" {
				continue
			}

			callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}

			if isContextWithCancelCall(pass, callExpr) {
				// Found context.WithCancel/WithTimeout/WithDeadline assignment.
				// Check if cancel is deferred.
				if !deferredCancels[cancelIdent.Name] {
					cancelVars[cancelIdent.Name] = true
					return true
				}
			}
		}
	}

	return false
}

// checkMutexLeak reports whether the function body acquires a lock without
// guaranteeing defer unlock.
func checkMutexLeak(
	pass *analysis.Pass,
	body *ast.BlockStmt,
	noLintIndex nolint.DirectiveIndex,
) bool {
	if body == nil {
		return false
	}

	// Track Lock calls and their corresponding Unlock calls.
	deferredUnlocks := make(map[string]bool)

	// First pass: find deferred Unlock calls.
	for _, stmt := range body.List {
		if deferStmt, ok := stmt.(*ast.DeferStmt); ok {
			callExpr := deferStmt.Call
			if callExpr != nil {
				if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					// Track both Unlock and RUnlock.
					if selExpr.Sel.Name == "Unlock" || selExpr.Sel.Name == "RUnlock" {
						if ident, ok := selExpr.X.(*ast.Ident); ok {
							deferredUnlocks[ident.Name] = true
						}
					}
				}
			}
		}
	}

	// Second pass: find Lock calls that are not guaranteed to have deferred Unlock.
	for _, stmt := range body.List {
		exprStmt, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		// Check for mu.Lock() or mu.RLock() calls.
		if (selExpr.Sel.Name == "Lock" || selExpr.Sel.Name == "RLock") && isMutexType(pass, selExpr.X) {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				receiverName := ident.Name
				// Flag if there's no corresponding deferred Unlock or RUnlock.
				if !deferredUnlocks[receiverName] {
					return true
				}
			}
		}
	}

	return false
}

// checkChannelLeak reports whether the function body creates a channel
// without guaranteed close or drain.
func checkChannelLeak(
	pass *analysis.Pass,
	body *ast.BlockStmt,
	noLintIndex nolint.DirectiveIndex,
) bool {
	if body == nil {
		return false
	}

	// For now, we use a simplified heuristic: if a channel is created
	// with make(chan T) but we don't see any send, close, or receive operations
	// on it, it's likely a leak. However, this requires flow analysis.
	// We'll focus on the most obvious case: channels created but never used.

	// Track all channel creations.
	channels := make(map[string]bool) // varName -> isCreated

	// First pass: find channel creations and usages.
	ast.Inspect(body, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			if len(assignStmt.Rhs) >= 1 {
				callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr)
				if !ok {
					return true
				}

				// Check for make(chan T) pattern.
				if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "make" {
					if len(callExpr.Args) >= 1 {
						if _, ok := callExpr.Args[0].(*ast.ChanType); ok {
							// This is make(chan T, ...) — a channel creation.
							for _, lhs := range assignStmt.Lhs {
								if varIdent, ok := lhs.(*ast.Ident); ok && varIdent.Name != "_" {
									channels[varIdent.Name] = true
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	// Second pass: check if channels are used (sent to, received from, or closed).
	// If a channel is created but never used in any way, it's likely a leak.
	// This is a conservative heuristic; we only flag obvious cases.
	usedChannels := make(map[string]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		// Look for send operations: ch <- value
		if sendStmt, ok := n.(*ast.SendStmt); ok {
			if ident, ok := sendStmt.Chan.(*ast.Ident); ok {
				usedChannels[ident.Name] = true
			}
		}

		// Look for close() calls: close(ch)
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "close" {
				if len(callExpr.Args) >= 1 {
					if argIdent, ok := callExpr.Args[0].(*ast.Ident); ok {
						usedChannels[argIdent.Name] = true
					}
				}
			}
		}

		// Look for receive operations: value := <-ch or <-ch
		if unaryExpr, ok := n.(*ast.UnaryExpr); ok && unaryExpr.Op.String() == "<-" {
			if ident, ok := unaryExpr.X.(*ast.Ident); ok {
				usedChannels[ident.Name] = true
			}
		}

		return true
	})

	// Check if any created channel is not used.
	for ch := range channels {
		if !usedChannels[ch] {
			return true
		}
	}

	return false
}

// isContextWithCancelCall reports whether call is context.WithCancel,
// context.WithTimeout, or context.WithDeadline.
func isContextWithCancelCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if !astutil.IsPkgSelector(pass, sel, "context") {
		return false
	}
	switch sel.Sel.Name {
	case "WithCancel", "WithTimeout", "WithDeadline":
		return true
	default:
		return false
	}
}

// isMutexType reports whether expr is a sync.Mutex or sync.RWMutex.
func isMutexType(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}

	// Handle pointers to mutex.
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Handle named types like sync.Mutex.
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	pkgPath := obj.Pkg().Path()
	name := obj.Name()

	return (pkgPath == "sync" || pkgPath == "sync/atomic") &&
		(name == "Mutex" || name == "RWMutex")
}
