package manualmutexunlock

import (
	"sync"
)

// Correct: defer unlock immediately after lock
func GoodMutexPattern() {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	
	// ... do work ...
}

// Wrong: manual unlock without defer - should be flagged
func BadMutexPattern() {
	var mu sync.Mutex
	mu.Lock() // want `mutex Unlock\(\) should be deferred immediately after Lock\(\) to prevent deadlocks on panic or early return`
	
	// ... do work ...
	
	mu.Unlock()
}

// Correct: RWMutex with defer
func GoodRWMutexPattern() {
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock()
	
	// ... do work ...
}

// Wrong: RWMutex without defer - should be flagged
func BadRWMutexPattern() {
	var mu sync.RWMutex
	mu.RLock() // want `mutex Unlock\(\) should be deferred immediately after Lock\(\) to prevent deadlocks on panic or early return`
	
	// ... do work ...
	
	mu.RUnlock()
}

// Wrong: write lock without defer - should be flagged
func BadRWMutexWriteLock() {
	var mu sync.RWMutex
	mu.Lock() // want `mutex Unlock\(\) should be deferred immediately after Lock\(\) to prevent deadlocks on panic or early return`
	
	// ... do work ...
	
	mu.Unlock()
}

// Correct: nested function with defer
func GoodNestedPattern() {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	
	func() {
		// This is a closure, analyzed separately
	}()
}

// Correct: multiple locks with defers
func GoodMultipleMutexes() {
	var mu1 sync.Mutex
	var mu2 sync.Mutex
	
	mu1.Lock()
	defer mu1.Unlock()
	
	mu2.Lock()
	defer mu2.Unlock()
	
	// ... do work ...
}

// Wrong: multiple locks, one without defer
func BadMultipleMutexes() {
	var mu1 sync.Mutex
	var mu2 sync.Mutex
	
	mu1.Lock()
	defer mu1.Unlock()
	
	mu2.Lock() // want `mutex Unlock\(\) should be deferred immediately after Lock\(\) to prevent deadlocks on panic or early return`
	
	// ... do work ...

	mu2.Unlock()
}

type guarded struct {
	mu sync.Mutex
}

// Wrong: selector-based mutex receiver without defer - should be flagged
func BadSelectorPattern() {
	var g guarded
	g.mu.Lock() // want `mutex Unlock\(\) should be deferred immediately after Lock\(\) to prevent deadlocks on panic or early return`
	g.mu.Unlock()
}

// Wrong: repeated lock on same mutex should still report earlier unresolved violation
func BadRepeatedLockBeforeGood() {
	var mu sync.Mutex
	mu.Lock() // want `mutex Unlock\(\) should be deferred immediately after Lock\(\) to prevent deadlocks on panic or early return`
	mu.Unlock()

	mu.Lock()
	defer mu.Unlock()
}
