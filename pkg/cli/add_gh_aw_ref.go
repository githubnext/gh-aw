package cli

import (
	"context"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var addGhAwRefLog = logger.New("cli:add_gh_aw_ref")

func resolveAddGhAwRef(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	addGhAwRefLog.Printf("Resolving --gh-aw-ref %q to a commit SHA", ref)
	resolvedRef, err := workflow.ResolveGhAwRef(ctx, ref)
	if err != nil {
		addGhAwRefLog.Printf("Failed to resolve --gh-aw-ref %q: %v", ref, err)
		return "", fmt.Errorf("--gh-aw-ref: %w", err)
	}
	addGhAwRefLog.Printf("Resolved --gh-aw-ref %q to %s", ref, resolvedRef)
	return resolvedRef, nil
}
