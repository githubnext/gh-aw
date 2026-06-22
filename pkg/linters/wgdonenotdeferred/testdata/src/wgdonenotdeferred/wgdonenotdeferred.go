package wgdonenotdeferred

import "sync"

// Bad: Done() not deferred in a goroutine — if the goroutine panics the WaitGroup will deadlock.
func BadGoroutine(wg *sync.WaitGroup) {
	go func() {
		wg.Done() // want `sync.WaitGroup Done\(\) should be deferred to prevent deadlock if the function panics`
		doWork()
	}()
}

// Bad: Done() not deferred in a regular function that receives the WaitGroup pointer.
func BadHelper(wg *sync.WaitGroup) {
	doWork()
	wg.Done() // want `sync.WaitGroup Done\(\) should be deferred to prevent deadlock if the function panics`
}

// Bad: Done() not deferred on a value receiver WaitGroup field.
type Worker struct {
	wg sync.WaitGroup
}

func (w *Worker) BadMethod() {
	w.wg.Done() // want `sync.WaitGroup Done\(\) should be deferred to prevent deadlock if the function panics`
}

// Good: Done() is properly deferred in a goroutine.
func GoodGoroutine(wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		doWork()
	}()
}

// Good: Done() is properly deferred in a regular function.
func GoodHelper(wg *sync.WaitGroup) {
	defer wg.Done()
	doWork()
}

// Good: Done() is properly deferred on a struct field.
func (w *Worker) GoodMethod() {
	defer w.wg.Done()
	doWork()
}

// Good: a different type has a Done() method — must not be flagged.
type Finisher struct{}

func (f *Finisher) Done() {}

func GoodOtherDone() {
	f := &Finisher{}
	f.Done()
}

func doWork() {}
