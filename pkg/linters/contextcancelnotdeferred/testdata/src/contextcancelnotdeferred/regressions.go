package contextcancelnotdeferred

import "context"

func parenthesizedAcquisition() {
	_, cancel := ((context.WithCancel)(context.Background())) // want `context cancel function should be deferred`
	(cancel)()
}

func parenthesizedDeferredCleanup() {
	_, cancel := (context.WithCancel(context.Background()))
	defer (cancel)()
}

func cancelContext(cancel context.CancelFunc) { cancel() }

// Helper cleanup is intentionally outside the direct-call analysis.
func helperCleanup() {
	_, cancel := context.WithCancel(context.Background())
	cancelContext(cancel)
}

func useCancel(cancel context.CancelFunc) {}

func helperUseWithLocalDefer() {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	useCancel(cancel)
}
