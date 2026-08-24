package cli

import (
	"context"
	"fmt"

	"github.com/github/gh-aw/pkg/workflow"
)

func resolveAddGhAwRef(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	resolvedRef, err := workflow.ResolveGhAwRef(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("--gh-aw-ref: %w", err)
	}
	return resolvedRef, nil
}
