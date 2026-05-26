package syncutil

import (
	"sync"

	"github.com/github/gh-aw/pkg/logger"
)

var syncutilLog = logger.New("syncutil:onceloader")

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
		syncutilLog.Print("OnceLoader.Get: cache miss, invoking loader")
		o.result, o.err = loader()
		o.done = true
		if o.err != nil {
			syncutilLog.Printf("OnceLoader.Get: loader failed: %v", o.err)
		} else {
			syncutilLog.Print("OnceLoader.Get: loader succeeded, result cached")
		}
	}

	return o.result, o.err
}

// Override stores result and err as the cached value without invoking the
// loader. Subsequent calls to Get will return this value without invoking the
// loader. Safe for concurrent use.
func (o *OnceLoader[T]) Override(result T, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	syncutilLog.Printf("OnceLoader.Override: storing cached value (err=%v)", err)
	o.result = result
	o.err = err
	o.done = true
}
