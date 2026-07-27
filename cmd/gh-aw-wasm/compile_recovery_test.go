package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRunCompileWithRecoverySuccess(t *testing.T) {
	t.Parallel()
	var resolved any
	var rejected error

	runCompileWithRecovery(
		func() (any, error) { return "ok", nil },
		func(result any) { resolved = result },
		func(err error) { rejected = err },
	)

	if rejected != nil {
		t.Fatalf("expected no rejection, got %v", rejected)
	}
	if resolved != "ok" {
		t.Fatalf("expected resolve value %q, got %v", "ok", resolved)
	}
}

func TestRunCompileWithRecoveryError(t *testing.T) {
	t.Parallel()
	want := errors.New("compile failed")
	var resolved any
	var rejected error

	runCompileWithRecovery(
		func() (any, error) { return nil, want },
		func(result any) { resolved = result },
		func(err error) { rejected = err },
	)

	if resolved != nil {
		t.Fatalf("expected no resolve value, got %v", resolved)
	}
	if !errors.Is(rejected, want) {
		t.Fatalf("expected rejection %v, got %v", want, rejected)
	}
}

func TestRunCompileWithRecoveryPanic(t *testing.T) {
	t.Parallel()
	var resolved any
	var rejected error

	runCompileWithRecovery(
		func() (any, error) {
			panic("boom")
		},
		func(result any) { resolved = result },
		func(err error) { rejected = err },
	)

	if resolved != nil {
		t.Fatalf("expected no resolve value, got %v", resolved)
	}
	if rejected == nil {
		t.Fatal("expected rejection from panic, got nil")
	}
	message := rejected.Error()
	if !strings.Contains(message, "compileWorkflow panic: boom") {
		t.Fatalf("expected panic prefix in rejection, got %q", message)
	}
	if !strings.Contains(message, "TestRunCompileWithRecoveryPanic") {
		t.Fatalf("expected stack trace in rejection, got %q", message)
	}
}
