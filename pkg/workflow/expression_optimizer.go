// This file re-exports the GitHub Actions expression optimizer from pkg/ghexpr
// so that existing code in pkg/workflow continues to compile without modification.
// New code should import pkg/ghexpr directly.
package workflow

import "github.com/github/gh-aw/pkg/ghexpr"

// OptimizeExpression re-exports [ghexpr.OptimizeExpression].
var OptimizeExpression = ghexpr.OptimizeExpression
