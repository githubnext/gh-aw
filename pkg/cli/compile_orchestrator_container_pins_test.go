//go:build !integration

package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMaybeForceRefreshContainerPins_PropagatesRefreshError(t *testing.T) {
	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })

	compileUpdateContainerPins = func(_ context.Context, _ containerPinUpdateDeps, _ string, _ bool, _ containerPinUpdateOptions) (bool, error) {
		return false, errors.New("refresh failed")
	}

	err := maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
	}, ".github/workflows")
	if err == nil {
		t.Fatal("expected error when refresh fails")
	}
	if !strings.Contains(err.Error(), "failed to refresh container pins") {
		t.Fatalf("expected wrapped refresh error, got: %v", err)
	}
}

func TestMaybeForceRefreshContainerPins_SkipsWhenNoEmit(t *testing.T) {
	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })

	called := false
	compileUpdateContainerPins = func(_ context.Context, _ containerPinUpdateDeps, _ string, _ bool, _ containerPinUpdateOptions) (bool, error) {
		called = true
		return false, nil
	}

	err := maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
		NoEmit:                    true,
	}, ".github/workflows")
	if err != nil {
		t.Fatalf("expected no error when --no-emit skips refresh, got: %v", err)
	}
	if called {
		t.Fatal("expected refresh to be skipped when --no-emit is set")
	}
}

func TestMaybeForceRefreshContainerPins_UsesRefreshExistingMode(t *testing.T) {
	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })

	called := false
	compileUpdateContainerPins = func(_ context.Context, _ containerPinUpdateDeps, _ string, _ bool, opts containerPinUpdateOptions) (bool, error) {
		called = true
		if !opts.refreshExisting {
			t.Fatal("expected refreshExisting=true when forcing container pin refresh")
		}
		return false, nil
	}

	err := maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
	}, ".github/workflows")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected refresh function to be called")
	}
}
