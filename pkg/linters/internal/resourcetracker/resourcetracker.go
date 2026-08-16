// Package resourcetracker provides a shared state machine for linters that
// track a resource or synchronization primitive acquired in a function body and
// report when its cleanup call is made manually instead of being deferred.
//
// Linters supply the resource-specific matching rules (how a resource is
// acquired and how its cleanup call is recognised) while this package owns the
// common bookkeeping: AST traversal, closure boundaries, reassignment handling,
// suppression-directive filtering and diagnostic reporting.
package resourcetracker

import (
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:resourcetracker")

// Acquisition describes a resource acquired at Pos and tracked under Key.
type Acquisition[K comparable] struct {
	Key K
	Pos token.Pos
}

// Config describes a deferred-cleanup linter.
//
// Acquisitions reports the resources acquired by a statement, and CleanupKey
// reports the tracked resource targeted by a cleanup call. Cleanup calls found
// inside a defer statement mark the resource as correctly released; cleanup
// calls found in plain expression or assignment statements mark it as manually
// released and are reported at the acquisition position.
type Config[K comparable] struct {
	// Name is the analyzer name, also used for nolint directive matching.
	Name string
	// Doc is the analyzer documentation string.
	Doc string
	// Message is the diagnostic message reported at the acquisition position.
	Message string
	// Acquisitions returns the resources acquired by node, if any.
	Acquisitions func(pass *analysis.Pass, node ast.Node) []Acquisition[K]
	// CleanupKey returns the tracked resource released by call, if any.
	CleanupKey func(pass *analysis.Pass, call *ast.CallExpr) (K, bool)
}

// NewAnalyzer builds an analysis pass implementing the deferred-cleanup check
// described by cfg.
func NewAnalyzer[K comparable](cfg Config[K]) *analysis.Analyzer {
	return analyzerutil.New(cfg.Name, cfg.Doc, cfg.run)
}

type state struct {
	acquirePos token.Pos
	hasDefer   bool
	hasManual  bool
}

func (c Config[K]) run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("%s: analyzing package %s", c.Name, pass.Pkg.Path())

	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}
		pos := pass.Fset.PositionFor(fn.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		c.inspectBody(pass, noLintIndex, fn.Body)
	})
}

func (c Config[K]) inspectBody(pass *analysis.Pass, noLintIndex nolint.DirectiveIndex, body *ast.BlockStmt) {
	// Track resources keyed by the linter-provided key so that variable
	// shadowing and distinct receivers are handled correctly.
	tracked := make(map[K]*state)

	ast.Inspect(body, func(node ast.Node) bool {
		return c.inspectNode(pass, noLintIndex, tracked, node)
	})

	// Report resources cleaned up manually without a matching defer, ordered by
	// acquisition position so diagnostics are deterministic.
	pending := make([]*state, 0, len(tracked))
	for _, st := range tracked {
		if st.hasManual && !st.hasDefer {
			pending = append(pending, st)
		}
	}
	slices.SortFunc(pending, func(a, b *state) int {
		return cmpPos(a.acquirePos, b.acquirePos)
	})
	for _, st := range pending {
		c.report(pass, noLintIndex, st)
	}
}

func cmpPos(a, b token.Pos) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func (c Config[K]) inspectNode(pass *analysis.Pass, noLintIndex nolint.DirectiveIndex, tracked map[K]*state, node ast.Node) bool {
	if node == nil {
		return false
	}

	// Do not descend into function literals — closures are independent
	// execution contexts and are analyzed separately to avoid false positives.
	if _, ok := node.(*ast.FuncLit); ok {
		return false
	}

	for _, acquired := range c.Acquisitions(pass, node) {
		// If this key was already tracked from a prior acquisition on the same
		// binding, report any unresolved violation before overwriting the state.
		if prev, exists := tracked[acquired.Key]; exists && prev.hasManual && !prev.hasDefer {
			c.report(pass, noLintIndex, prev)
		}
		tracked[acquired.Key] = &state{acquirePos: acquired.Pos}
	}

	// A cleanup call inside defer resolves the resource.
	if deferStmt, ok := node.(*ast.DeferStmt); ok {
		if key, ok := c.CleanupKey(pass, deferStmt.Call); ok {
			if st, found := tracked[key]; found {
				st.hasDefer = true
			}
		}
	}

	// Cleanup calls used as plain statements, or whose results are assigned
	// (e.g. closeErr := f.Close()), are manual cleanups.
	switch stmt := node.(type) {
	case *ast.ExprStmt:
		if call, ok := stmt.X.(*ast.CallExpr); ok {
			c.markManual(pass, tracked, call)
		}
	case *ast.AssignStmt:
		for _, rhs := range stmt.Rhs {
			if call, ok := rhs.(*ast.CallExpr); ok {
				c.markManual(pass, tracked, call)
			}
		}
	}

	return true
}

func (c Config[K]) markManual(pass *analysis.Pass, tracked map[K]*state, call *ast.CallExpr) {
	key, ok := c.CleanupKey(pass, call)
	if !ok {
		return
	}
	if st, found := tracked[key]; found {
		st.hasManual = true
	}
}

func (c Config[K]) report(pass *analysis.Pass, noLintIndex nolint.DirectiveIndex, st *state) {
	position := pass.Fset.PositionFor(st.acquirePos, false)
	if nolint.HasDirectiveForLinter(position, noLintIndex, c.Name) {
		return
	}
	pkgLog.Printf("%s: flagging non-deferred cleanup at %s", c.Name, position)
	pass.Report(analysis.Diagnostic{
		Pos:     st.acquirePos,
		Message: c.Message,
	})
}
