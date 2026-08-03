// Package ctxutil provides small helpers for consistent context.Context handling.
package ctxutil

import "context"

// OrBackground centralizes the repo's nil-context fallback convention: it
// returns context.Background() when ctx is nil, otherwise it returns ctx
// unchanged. Use this instead of duplicating the inline
//
//	if ctx == nil {
//	    ctx = context.Background()
//	}
//
// pattern at call sites.
func OrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
