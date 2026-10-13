// Package a is the test fixture for the goroutineresourceleak analyzer.
package a

import (
	"context"
	"sync"
	"time"
)

// ===== CANCEL CONTEXT LEAKS =====

// cancelWithoutDefer launches a goroutine that creates a cancellable context
// but doesn't defer the cancel function.
func cancelWithoutDefer() {
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		ctx, cancel := context.WithCancel(context.Background())
		_ = ctx
		_ = cancel
		// cancel is never called or deferred — leak!
	}()
}

// timeoutWithoutDefer launches a goroutine with context.WithTimeout without
// deferring the cancel.
func timeoutWithoutDefer() {
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		_ = ctx
		_ = cancel
	}()
}

// deadlineWithoutDefer launches a goroutine with context.WithDeadline without
// deferring the cancel.
func deadlineWithoutDefer() {
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
		_ = ctx
		_ = cancel
	}()
}

// cancelWithDefer launches a goroutine that properly defers cancel.
// This should NOT trigger a diagnostic.
func cancelWithDefer() {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = ctx
	}()
}

// timeoutWithDefer launches a goroutine with context.WithTimeout and properly
// defers the cancel. This should NOT trigger a diagnostic.
func timeoutWithDefer() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		_ = ctx
	}()
}

// noContextCreation launches a goroutine that doesn't create any cancellable
// context. This should NOT trigger a diagnostic.
func noContextCreation() {
	go func() {
		ctx := context.Background()
		_ = ctx
	}()
}

// ===== MUTEX LEAKS =====

// mutexWithoutDefer launches a goroutine that locks a mutex but doesn't
// defer the unlock.
func mutexWithoutDefer(mu *sync.Mutex) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		// Critical section
		// unlock is never called or deferred — deadlock risk!
		mu.Unlock()
	}()
}

// rwmutexWithoutDefer launches a goroutine that locks an RWMutex without
// deferring the unlock.
func rwmutexWithoutDefer(mu *sync.RWMutex) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		// Critical section
		mu.Unlock()
	}()
}

// mutexWithDefer launches a goroutine that properly defers the unlock.
// This should NOT trigger a diagnostic.
func mutexWithDefer(mu *sync.Mutex) {
	go func() {
		mu.Lock()
		defer mu.Unlock()
		// Critical section
	}()
}

// rwmutexReadWithDefer launches a goroutine that properly defers RUnlock.
// This should NOT trigger a diagnostic.
func rwmutexReadWithDefer(mu *sync.RWMutex) {
	go func() {
		mu.RLock()
		defer mu.RUnlock()
		// Read-only section
	}()
}

// noMutexAcquisition launches a goroutine without any mutex operations.
// This should NOT trigger a diagnostic.
func noMutexAcquisition() {
	go func() {
		// No locks acquired
	}()
}

// ===== CHANNEL LEAKS =====

// channelWithoutUse launches a goroutine that creates a channel but never
// uses it (never sends, closes, or receives).
func channelWithoutUse() {
	go func() { // want "goroutine creates channel without guaranteed close or drain"
		ch := make(chan int)
		_ = ch
		// Channel is created but never used — potential deadlock if sender expects receiver
	}()
}

// channelWithSend launches a goroutine that creates a channel and sends to it.
// This should NOT trigger a diagnostic.
func channelWithSend() {
	go func() {
		ch := make(chan int)
		defer close(ch)
		ch <- 42
	}()
}

// channelWithReceive launches a goroutine that creates a channel and receives from it.
// This should NOT trigger a diagnostic (assuming parent goroutine handles the send).
func channelWithReceive() {
	go func() {
		ch := make(chan int)
		_ = <-ch
	}()
}

// channelWithClose launches a goroutine that creates and closes a channel.
// This should NOT trigger a diagnostic.
func channelWithClose() {
	go func() {
		ch := make(chan int)
		close(ch)
	}()
}

// bufferedChannelWithoutUse launches a goroutine that creates a buffered channel
// but never uses it.
func bufferedChannelWithoutUse() {
	go func() { // want "goroutine creates channel without guaranteed close or drain"
		ch := make(chan int, 10)
		_ = ch
		// Buffered channel created but never used
	}()
}

// multipleChannelsWithoutUse launches a goroutine that creates multiple channels
// but uses only one. The unused one should trigger a diagnostic.
func multipleChannelsWithoutUse() {
	go func() { // want "goroutine creates channel without guaranteed close or drain"
		ch1 := make(chan int)
		ch2 := make(chan int)
		defer close(ch1)
		ch1 <- 42
		_ = ch2
		// ch2 is never used
	}()
}

// ===== SAFE PATTERNS (SHOULD NOT TRIGGER) =====

// namedFunctionGoroutine launches a goroutine with a named function.
// Named functions are out of scope for this linter.
func namedFunctionGoroutine() {
	go doSomeWork()
}

func doSomeWork() {
	// Any resource leaks here are the responsibility of doSomeWork itself.
}

// goroutineWithExplicitTimeout launches a goroutine with explicit timeout handling.
// This should NOT trigger a diagnostic.
func goroutineWithExplicitTimeout() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		// Safe: cancel is deferred
		_ = ctx
	}()
}

// goroutineWithExplicitWaitGroup launches a goroutine with proper WaitGroup synchronization.
// This should NOT trigger a diagnostic.
func goroutineWithExplicitWaitGroup() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Work here
	}()
	wg.Wait()
}

//nolint:goroutineresourceleak
// suppressedGoroutineResourceLeak launches a goroutine that would normally
// trigger a diagnostic but is suppressed with nolint.
func suppressedGoroutineResourceLeak() {
	go func() { // nolint:goroutineresourceleak
		ctx, cancel := context.WithCancel(context.Background())
		_ = ctx
		_ = cancel
		// This would normally trigger, but is suppressed by nolint.
	}()
}
