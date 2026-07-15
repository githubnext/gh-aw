// Package ctxhelper provides helper functions used by nilctxpassed test
// fixtures to exercise calls from packages that do not directly import context.
package ctxhelper

import "context"

// TakesCtx accepts a context.Context parameter.
func TakesCtx(ctx context.Context) {}

// TakesOtherPointer accepts a *int parameter (not context.Context).
func TakesOtherPointer(p *int) {}
