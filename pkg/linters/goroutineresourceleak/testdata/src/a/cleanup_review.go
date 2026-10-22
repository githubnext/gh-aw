package a

import (
	"context"
	"sync"
)

func cleanupAfterNestedLock(mu *sync.Mutex) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		{
			mu.Lock()
			defer mu.Unlock()
		}
		defer mu.Unlock()
	}()
}

func cleanupAfterConditionalLock(mu *sync.Mutex, again bool) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		if again {
			((mu).Lock)()
			defer mu.Unlock()
		}
		defer mu.Unlock()
	}()
}

func cleanupAfterMutexPanic(mu *sync.Mutex, failed bool) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		defer func() { _ = recover() }()
		mu.Lock()
		if failed {
			(panic)("stop")
		}
		defer mu.Unlock()
	}()
}

func cleanupAfterCancelPanic(failed bool) {
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		defer func() { _ = recover() }()
		_, cancel := context.WithCancel(context.Background())
		if failed {
			panic("stop")
		}
		defer cancel()
	}()
}

func cleanupAfterDirectPanic(mu *sync.Mutex) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		panic("stop")
		defer mu.Unlock()
	}()
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		_, cancel := context.WithCancel(context.Background())
		panic("stop")
		defer cancel()
	}()
}

func cleanupAfterDeferredPanic(mu *sync.Mutex) {
	go func() {
		mu.Lock()
		defer panic("later")
		defer mu.Unlock()
	}()
	go func() {
		_, cancel := context.WithCancel(context.Background())
		defer (panic)("later")
		defer cancel()
	}()
}

func cleanupAfterAsyncPanic(mu *sync.Mutex) {
	go func() {
		mu.Lock()
		go panic("elsewhere")
		defer mu.Unlock()
	}()
	go func() {
		_, cancel := context.WithCancel(context.Background())
		go (panic)("elsewhere")
		defer cancel()
	}()
}

func cleanupAfterShadowedPanic(mu *sync.Mutex) {
	panic := func(string) {}
	go func() {
		mu.Lock()
		panic("not a builtin")
		defer mu.Unlock()
	}()
	go func() {
		_, cancel := context.WithCancel(context.Background())
		(panic)("not a builtin")
		defer cancel()
	}()
}

func cleanupIgnoresUncalledClosure(mu *sync.Mutex) {
	go func() {
		mu.Lock()
		_ = func() {
			mu.Lock()
			defer mu.Unlock()
			panic("not executed")
		}
		defer mu.Unlock()
	}()
	go func() {
		_, cancel := context.WithCancel(context.Background())
		_ = func() { panic("not executed") }
		defer cancel()
	}()
}
