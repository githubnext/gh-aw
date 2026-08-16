//go:build !integration

// Package syncutil tests validate that OnceLoader permits concurrent Get calls
// while invoking its loader at most once.
package syncutil

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, "ok", got1)
	assert.Equal(t, "ok", got2)
	assert.Equal(t, int32(1), calls.Load())
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

	assert.Empty(t, got1)
	assert.Empty(t, got2)
	require.ErrorIs(t, err1, expectedErr)
	require.ErrorIs(t, err2, expectedErr)
	assert.Equal(t, int32(1), calls.Load())
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
			assert.NoError(t, err)
			assert.Equal(t, "value", got)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load())
}

func TestOnceLoaderOverride(t *testing.T) {
	overrideErr := errors.New("override-err")
	tests := []struct {
		name    string
		value   string
		err     error
		wantErr error
	}{
		{name: "value", value: "forced"},
		{name: "error", err: overrideErr, wantErr: overrideErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loader OnceLoader[string]
			var calls atomic.Int32

			loader.Override(tt.value, tt.err)

			got, err := loader.Get(func() (string, error) {
				calls.Add(1)
				return "should-not-run", nil
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.value, got)
			assert.Equal(t, int32(0), calls.Load())
		})
	}
}

func TestOnceLoaderReset(t *testing.T) {
	t.Run("after successful load", func(t *testing.T) {
		var loader OnceLoader[string]
		var calls atomic.Int32

		load := func() (string, error) {
			call := calls.Add(1)
			return "value-" + strconv.Itoa(int(call)), nil
		}

		got1, err1 := loader.Get(load)
		require.NoError(t, err1)
		assert.Equal(t, "value-1", got1)

		loader.Reset()

		got2, err2 := loader.Get(load)
		require.NoError(t, err2)
		assert.Equal(t, "value-2", got2)
		assert.Equal(t, int32(2), calls.Load())
	})

	t.Run("after cached error", func(t *testing.T) {
		var loader OnceLoader[string]
		var calls atomic.Int32
		expectedErr := errors.New("boom")

		got, err := loader.Get(func() (string, error) {
			calls.Add(1)
			return "", expectedErr
		})
		assert.Empty(t, got)
		require.ErrorIs(t, err, expectedErr)

		loader.Reset()

		got, err = loader.Get(func() (string, error) {
			calls.Add(1)
			return "recovered", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "recovered", got)
		assert.Equal(t, int32(2), calls.Load())
	})

	t.Run("after override", func(t *testing.T) {
		var loader OnceLoader[string]
		var calls atomic.Int32
		loader.Override("forced", nil)

		loader.Reset()

		got, err := loader.Get(func() (string, error) {
			calls.Add(1)
			return "loaded", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "loaded", got)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("on zero-value loader", func(t *testing.T) {
		var loader OnceLoader[string]
		var calls atomic.Int32

		loader.Reset()

		got, err := loader.Get(func() (string, error) {
			calls.Add(1)
			return "loaded", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "loaded", got)
		assert.Equal(t, int32(1), calls.Load())
	})
}
