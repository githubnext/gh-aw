package nilctxpassed

import (
	"context"
)

func takesCtx(ctx context.Context) {}

func takesCtxAndOther(ctx context.Context, n int) {}

func takesOtherAndCtx(n int, ctx context.Context) {}

func takesVariadicCtx(args ...context.Context) {}

type myKey struct{}

// flagged: nil passed as context.Context
func BadNilContext() {
	takesCtx(nil) // want `nil passed as context\.Context; use context\.Background\(\) or context\.TODO\(\) instead`
}

// flagged: nil in first of two params
func BadNilFirstParam() {
	takesCtxAndOther(nil, 42) // want `nil passed as context\.Context; use context\.Background\(\) or context\.TODO\(\) instead`
}

// flagged: nil in second positional context param
func BadNilSecondParam() {
	takesOtherAndCtx(42, nil) // want `nil passed as context\.Context; use context\.Background\(\) or context\.TODO\(\) instead`
}

// flagged: nil as the only variadic context arg
func BadNilVariadic() {
	takesVariadicCtx(nil) // want `nil passed as context\.Context; use context\.Background\(\) or context\.TODO\(\) instead`
}

// not flagged: proper context values
func GoodContextBackground() {
	takesCtx(context.Background())
}

func GoodContextTODO() {
	takesCtx(context.TODO())
}

// not flagged: non-context nil (e.g. error interface)
func takesError(err error) {}

func GoodNilError() {
	takesError(nil)
}

// not flagged: nil variable named "nil" would still be caught, but
// a local context variable passed normally is fine
func GoodLocalCtx() {
	ctx := context.Background()
	takesCtx(ctx)
}

// not flagged: nolint suppresses the diagnostic
func NolintSuppressed() {
	takesCtx(nil) //nolint:nilctxpassed
}
