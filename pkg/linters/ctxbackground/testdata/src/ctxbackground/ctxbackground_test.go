package ctxbackground

import "context"

func helperInTestFile(ctx context.Context) {
	_ = context.Background()
}
