package a

import (
	"context"
	"sync"
)

func parenthesizedCancelLeak() {
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		_, cancel := ((context.WithCancel)(context.Background()))
		_ = cancel
	}()
}

func parenthesizedMutexLeak(mu *sync.Mutex) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		(((mu).Lock)())
	}()
}

func cleanupAfterBookkeeping(mu *sync.Mutex) {
	go func() {
		ctx, cancel := (context.WithCancel(context.Background()))
		println("starting")
		_ = ctx
		defer (cancel)()
		((mu).Lock)()
		println("locked")
		defer ((mu).Unlock)()
	}()
}

func mutexAfterConditionalReturn(mu *sync.Mutex, exit bool) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		if exit {
			return
		}
		defer mu.Unlock()
	}()
}

func multipleAcquisitions(mu *sync.Mutex, rw *sync.RWMutex) {
	go func() {
		_, first := context.WithCancel(context.Background())
		_, second := context.WithCancel(context.Background())
		defer second()
		defer first()
		mu.Lock()
		rw.RLock()
		defer rw.RUnlock()
		defer mu.Unlock()
	}()
}

func reassignedCancel() {
	go func() { // want "goroutine launches context.WithCancel/WithTimeout/WithDeadline without deferred cancel"
		_, cancel := context.WithCancel(context.Background())
		_, cancel = context.WithCancel(context.Background())
		defer cancel()
	}()
}

func parenthesizedChannels() {
	go func() {
		ch := (make(chan int, 1))
		(ch) <- 1
		_ = <-(ch)
	}()
	go func() {
		ch := make(chan int)
		(close)((ch))
	}()
}

func processChannel(ch chan int) { close(ch) }

func channelPassedToHelper() {
	go func() {
		ch := make(chan int)
		processChannel((ch))
	}()
}

func channelStoredInStruct() {
	go func() {
		ch := make(chan int)
		holder := struct{ ch chan int }{ch: ch}
		processChannel(holder.ch)
	}()
}

func channelCapturedByClosure() {
	go func() {
		ch := make(chan int)
		cleanup := func() { close(ch) }
		cleanup()
	}()
}

func channelStoredInField(holder *struct{ ch chan int }) {
	go func() {
		ch := make(chan int)
		holder.ch = ch
	}()
}

func receivePassedToHelper() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		println(<-ch)
	}()
}

func builtinLenIsNotHandoff() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		_ = len(ch)
	}()
}

func channelReturned() {
	go func() chan int {
		ch := make(chan int)
		return ch
	}()
}

func channelLengthPassedToHelper() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		doSomeWorkWithID(len(ch))
	}()
}

func reassignMutexField(w *withMu) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		w.mu.Lock()
		w.mu = sync.Mutex{}
		defer w.mu.Unlock()
	}()
}

func doubleLockBeforeDefer(mu *sync.Mutex) {
	go func() { // want "goroutine acquires lock without deferred unlock"
		mu.Lock()
		mu.Lock()
		defer mu.Unlock()
	}()
}

func unrelatedGuardBeforeDefer(mu *sync.Mutex, log bool) {
	go func() {
		mu.Lock()
		if log {
			println("locked")
		}
		defer mu.Unlock()
	}()
}
