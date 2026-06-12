package ctxbackground

import ctxpkg "context"

// flagged: aliased context import still resolves to context package
func DoWorkAliasedImport(ctx ctxpkg.Context) {
	_ = ctxpkg.Background() // want `use the context.Context parameter instead of context.Background\(\)`
}
