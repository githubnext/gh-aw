//go:build !integration

package cli

import (
	"errors"
	"testing"
)

func TestHandleBatchToolErrorPreservesFatalFindingInNonStrictMode(t *testing.T) {
	t.Parallel()
	fatal := &fatalFindingError{err: errors.New("high severity finding")}

	err := handleBatchToolError("zizmor", fatal, false, false)
	if err == nil {
		t.Fatal("expected fatalFindingError to propagate even in non-strict mode, got nil")
	}
	var got *fatalFindingError
	if !errors.As(err, &got) {
		t.Fatalf("expected wrapped fatalFindingError, got %v", err)
	}
}

func TestHandleBatchToolErrorSuppressesRegularErrorInNonStrictMode(t *testing.T) {
	t.Parallel()
	err := handleBatchToolError("zizmor", errors.New("plain warning"), false, false)
	if err != nil {
		t.Fatalf("expected non-strict mode to suppress plain errors, got %v", err)
	}
}

func TestHandleBatchToolErrorPropagatesInStrictMode(t *testing.T) {
	t.Parallel()
	err := handleBatchToolError("zizmor", errors.New("plain warning"), true, false)
	if err == nil {
		t.Fatal("expected strict mode to propagate errors, got nil")
	}
}
