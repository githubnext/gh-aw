package manualmutexunlock

import "sync"

func parenthesizedAcquisition(mu *sync.Mutex) {
	(((mu).Lock)()) // want `mutex Unlock\(\) should be deferred`
	((mu).Unlock)()
}

func parenthesizedDeferredCleanup(mu *sync.RWMutex) {
	((mu).RLock)()
	defer ((mu).RUnlock)()
}

func releaseLock(mu *sync.Mutex) { mu.Unlock() }

// Helper cleanup is intentionally outside the direct-call analysis.
func helperCleanup(mu *sync.Mutex) {
	mu.Lock()
	releaseLock(mu)
}

func useLock(mu *sync.Mutex) {}

func helperUseWithLocalDefer(mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
	useLock(mu)
}
