package syncutil

import "sync"

// OnceLoader caches the result of a fallible, expensive one-shot fetch.
// Safe for concurrent use; loader is invoked at most once.
type OnceLoader[T any] struct {
	mu     sync.Mutex
	result T
	err    error
	done   bool
}

// Get returns the cached result, invoking loader exactly once.
func (o *OnceLoader[T]) Get(loader func() (T, error)) (T, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.done {
		o.result, o.err = loader()
		o.done = true
	}

	return o.result, o.err
}

// Reset clears cached state.
func (o *OnceLoader[T]) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()

	var zero T
	o.result = zero
	o.err = nil
	o.done = false
}

// Override stores result and err as the cached value without invoking the
// loader. Subsequent calls to Get will return this value without invoking the
// loader. Safe for concurrent use.
func (o *OnceLoader[T]) Override(result T, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.result = result
	o.err = err
	o.done = true
}
