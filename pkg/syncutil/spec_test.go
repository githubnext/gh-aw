//go:build !integration

package syncutil_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/syncutil"
)

// TestSpec_Types_OnceLoader validates the documented contract of the
// OnceLoader[T] type as described in the syncutil README.md.
//
// Specification:
//   - OnceLoader[T] is a struct caching the result of an expensive, fallible
//     one-shot fetch; safe for concurrent use.
//   - The zero value of OnceLoader[T] is ready to use; no constructor needed.
func TestSpec_Types_OnceLoader(t *testing.T) {
	t.Run("documented: zero value is ready to use", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]

		value, err := loader.Get(func() (string, error) {
			return "ready", nil
		})

		require.NoError(t, err, "zero value Get should not error")
		assert.Equal(t, "ready", value, "zero value Get should return loader result")
	})

	t.Run("documented: usable with different generic type parameter", func(t *testing.T) {
		var loader syncutil.OnceLoader[int]

		value, err := loader.Get(func() (int, error) {
			return 42, nil
		})

		require.NoError(t, err, "OnceLoader[int] should work")
		assert.Equal(t, 42, value, "OnceLoader[int] should return loader result")
	})
}

// TestSpec_PublicAPI_OnceLoader_Get validates the documented behavior of the
// OnceLoader.Get method as described in the syncutil README.md.
//
// Specification:
//   - Returns the cached result, invoking loader exactly once.
//   - If loader returns an error, the error is cached alongside the zero
//     value of T; subsequent calls return the same error without
//     re-invoking loader.
func TestSpec_PublicAPI_OnceLoader_Get(t *testing.T) {
	t.Run("documented: invokes loader exactly once across multiple Get calls", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		var calls atomic.Int32

		load := func() (string, error) {
			calls.Add(1)
			return "cached", nil
		}

		v1, err1 := loader.Get(load)
		v2, err2 := loader.Get(load)
		v3, err3 := loader.Get(load)

		require.NoError(t, err1, "first Get should not error")
		require.NoError(t, err2, "second Get should not error")
		require.NoError(t, err3, "third Get should not error")

		assert.Equal(t, "cached", v1, "first Get should return loader result")
		assert.Equal(t, "cached", v2, "second Get should return cached result")
		assert.Equal(t, "cached", v3, "third Get should return cached result")
		assert.Equal(t, int32(1), calls.Load(), "loader must be invoked exactly once")
	})

	t.Run("documented: caches error alongside zero value of T", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		var calls atomic.Int32
		boom := errors.New("boom")

		load := func() (string, error) {
			calls.Add(1)
			return "", boom
		}

		v1, err1 := loader.Get(load)
		v2, err2 := loader.Get(load)

		require.Error(t, err1, "first Get should return loader error")
		require.Error(t, err2, "second Get should return cached error")
		require.ErrorIs(t, err1, boom, "first Get error should match loader error")
		require.ErrorIs(t, err2, boom, "subsequent Get error should match cached loader error")
		assert.Empty(t, v1, "documented: zero value of T is returned alongside error")
		assert.Empty(t, v2, "documented: zero value of T is returned alongside cached error")
		assert.Equal(t, int32(1), calls.Load(), "loader must not be re-invoked after error")
	})

	t.Run("documented: caches error alongside zero value of T for non-string types", func(t *testing.T) {
		var loader syncutil.OnceLoader[int]
		var calls atomic.Int32
		boom := errors.New("boom")

		load := func() (int, error) {
			calls.Add(1)
			return 0, boom
		}

		v1, err1 := loader.Get(load)
		v2, err2 := loader.Get(load)

		require.ErrorIs(t, err1, boom, "first Get error should match loader error")
		require.ErrorIs(t, err2, boom, "subsequent Get error should match cached error")
		assert.Equal(t, 0, v1, "documented: zero value of int (0) returned with error")
		assert.Equal(t, 0, v2, "documented: zero value of int (0) cached with error")
		assert.Equal(t, int32(1), calls.Load(), "loader must not be re-invoked after error")
	})
}

// TestSpec_PublicAPI_OnceLoader_Reset validates the documented behavior of
// the OnceLoader.Reset method as described in the syncutil README.md.
//
// Specification:
//   - Clears the cached result and error so that the next Get call re-invokes
//     loader.
func TestSpec_PublicAPI_OnceLoader_Reset(t *testing.T) {
	t.Run("documented: next Get after Reset re-invokes loader", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		var calls atomic.Int32

		load := func() (string, error) {
			n := calls.Add(1)
			if n == 1 {
				return "first", nil
			}
			return "second", nil
		}

		v1, err1 := loader.Get(load)
		require.NoError(t, err1, "first Get should succeed")
		assert.Equal(t, "first", v1, "first Get returns initial loader result")

		loader.Reset()

		v2, err2 := loader.Get(load)
		require.NoError(t, err2, "Get after Reset should succeed")
		assert.Equal(t, "second", v2, "documented: next Get after Reset re-invokes loader")
		assert.Equal(t, int32(2), calls.Load(), "loader must be invoked again after Reset")
	})

	t.Run("documented: Reset clears cached error", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		var calls atomic.Int32

		load := func() (string, error) {
			n := calls.Add(1)
			if n == 1 {
				return "", errors.New("first failure")
			}
			return "recovered", nil
		}

		_, err1 := loader.Get(load)
		require.Error(t, err1, "first Get should return loader error")

		loader.Reset()

		v2, err2 := loader.Get(load)
		require.NoError(t, err2, "documented: Reset clears cached error")
		assert.Equal(t, "recovered", v2, "documented: loader is re-invoked after Reset")
	})
}

// TestSpec_PublicAPI_OnceLoader_Override validates the documented behavior of
// the OnceLoader.Override method as described in the syncutil README.md.
//
// Specification:
//   - Stores result and err as the cached value without invoking loader.
//   - Subsequent Get calls return this value without invoking loader.
func TestSpec_PublicAPI_OnceLoader_Override(t *testing.T) {
	t.Run("documented: Get returns overridden value without calling loader", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		var calls atomic.Int32

		load := func() (string, error) {
			calls.Add(1)
			return "from-loader", nil
		}

		loader.Override("forced", nil)

		v, err := loader.Get(load)

		require.NoError(t, err, "Get after Override should not error")
		assert.Equal(t, "forced", v, "documented: Get returns the overridden value")
		assert.Equal(t, int32(0), calls.Load(), "documented: loader is never called after Override")
	})

	t.Run("documented: Override with error is returned by subsequent Get", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		boom := errors.New("override-err")

		loader.Override("", boom)

		v, err := loader.Get(func() (string, error) {
			return "should-not-run", nil
		})

		require.ErrorIs(t, err, boom, "documented: Get returns the overridden error")
		assert.Empty(t, v, "documented: empty string returned alongside overridden error")
	})
}

// TestSpec_ThreadSafety_OnceLoader validates the documented concurrency
// guarantees of OnceLoader as described in the syncutil README.md.
//
// Specification:
//   - OnceLoader[T] is safe for concurrent use.
//   - The internal mutex ensures loader is invoked at most once, even when
//     multiple goroutines call Get concurrently.
//   - Reset acquires the same mutex, making it safe to call concurrently
//     with Get.
func TestSpec_ThreadSafety_OnceLoader(t *testing.T) {
	t.Run("documented: loader invoked at most once under concurrent Get", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		var calls atomic.Int32
		const workers = 64

		load := func() (string, error) {
			calls.Add(1)
			return "result", nil
		}

		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				v, err := loader.Get(load)
				assert.NoError(t, err, "concurrent Get should not error")
				assert.Equal(t, "result", v, "concurrent Get should return cached value")
			}()
		}
		wg.Wait()

		assert.Equal(t, int32(1), calls.Load(), "documented: loader invoked at most once under concurrency")
	})

	t.Run("documented: Reset is safe to call concurrently with Get", func(t *testing.T) {
		var loader syncutil.OnceLoader[string]
		const workers = 32

		load := func() (string, error) {
			return "v", nil
		}

		var wg sync.WaitGroup
		wg.Add(workers * 2)
		for range workers {
			go func() {
				defer wg.Done()
				_, _ = loader.Get(load)
			}()
			go func() {
				defer wg.Done()
				loader.Reset()
			}()
		}
		wg.Wait()
		// Test passes if the race detector (-race) reports no data races.
	})
}

// TestSpec_UsageExample_OnceLoader validates that the documented usage
// example pattern compiles and runs as described in the README.md.
//
// Specification (Usage Examples):
//
//	var cache syncutil.OnceLoader[string]
//	value, err := cache.Get(func() (string, error) {
//	    return expensiveOperation()
//	})
//	cache.Reset()
func TestSpec_UsageExample_OnceLoader(t *testing.T) {
	var cache syncutil.OnceLoader[string]
	var calls atomic.Int32

	expensiveOperation := func() (string, error) {
		calls.Add(1)
		return "result", nil
	}

	value, err := cache.Get(func() (string, error) {
		return expensiveOperation()
	})
	require.NoError(t, err, "usage example: Get should succeed")
	assert.Equal(t, "result", value, "usage example: Get should return loader result")

	value2, err2 := cache.Get(func() (string, error) {
		return expensiveOperation()
	})
	require.NoError(t, err2, "usage example: subsequent Get should succeed")
	assert.Equal(t, "result", value2, "usage example: subsequent Get returns cached value")
	assert.Equal(t, int32(1), calls.Load(), "usage example: loader called only once before Reset")

	cache.Reset()

	value3, err3 := cache.Get(func() (string, error) {
		return expensiveOperation()
	})
	require.NoError(t, err3, "usage example: Get after Reset should succeed")
	assert.Equal(t, "result", value3, "usage example: Get after Reset returns fresh loader result")
	assert.Equal(t, int32(2), calls.Load(), "usage example: Reset allows re-fetching on the next Get call")
}
