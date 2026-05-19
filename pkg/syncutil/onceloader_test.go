package syncutil

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestOnceLoaderGetCachesSuccess(t *testing.T) {
	var loader OnceLoader[string]
	var calls atomic.Int32

	load := func() (string, error) {
		calls.Add(1)
		return "ok", nil
	}

	got1, err1 := loader.Get(load)
	got2, err2 := loader.Get(load)

	if err1 != nil || err2 != nil {
		t.Fatalf("expected nil errors, got err1=%v err2=%v", err1, err2)
	}
	if got1 != "ok" || got2 != "ok" {
		t.Fatalf("expected cached value 'ok', got %q and %q", got1, got2)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected loader to be called once, got %d", calls.Load())
	}
}

func TestOnceLoaderGetCachesError(t *testing.T) {
	var loader OnceLoader[string]
	var calls atomic.Int32
	expectedErr := errors.New("boom")

	load := func() (string, error) {
		calls.Add(1)
		return "", expectedErr
	}

	got1, err1 := loader.Get(load)
	got2, err2 := loader.Get(load)

	if got1 != "" || got2 != "" {
		t.Fatalf("expected empty cached values, got %q and %q", got1, got2)
	}
	if !errors.Is(err1, expectedErr) || !errors.Is(err2, expectedErr) {
		t.Fatalf("expected cached errors to wrap %v, got err1=%v err2=%v", expectedErr, err1, err2)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected loader to be called once, got %d", calls.Load())
	}
}

func TestOnceLoaderGetConcurrentSingleInvoke(t *testing.T) {
	var loader OnceLoader[string]
	var calls atomic.Int32
	const workers = 50

	load := func() (string, error) {
		calls.Add(1)
		return "value", nil
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			got, err := loader.Get(load)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
				return
			}
			if got != "value" {
				t.Errorf("expected value, got %q", got)
			}
		}()
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("expected loader to be called once under concurrency, got %d", calls.Load())
	}
}

func TestOnceLoaderReset(t *testing.T) {
	var loader OnceLoader[string]
	var calls atomic.Int32

	load := func() (string, error) {
		n := calls.Add(1)
		return fmt.Sprintf("v%d", n), nil
	}

	got1, err1 := loader.Get(load)
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	if got1 != "v1" {
		t.Fatalf("expected first value v1, got %q", got1)
	}

	loader.Reset()

	got2, err2 := loader.Get(load)
	if err2 != nil {
		t.Fatalf("unexpected error after reset: %v", err2)
	}
	if got2 != "v2" {
		t.Fatalf("expected second value v2 after reset, got %q", got2)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected loader to run twice with reset, got %d", calls.Load())
	}
}
