package timesleepnocontext

import (
	"context"
	"time"
)

// Bad: time.Sleep inside a function that receives a context.
func BadSleep(ctx context.Context, d time.Duration) {
	time.Sleep(d) // want `use select with ctx\.Done\(\) instead of time\.Sleep to allow context cancellation`
}

// Bad: time.Sleep inside a method that receives a context.
type Worker struct{}

func (w *Worker) Wait(ctx context.Context) {
	time.Sleep(time.Second) // want `use select with ctx\.Done\(\) instead of time\.Sleep to allow context cancellation`
}

// Good: no context parameter — time.Sleep is acceptable.
func GoodNoContext(d time.Duration) {
	time.Sleep(d)
}

// Good: context parameter is blank — time.Sleep is acceptable.
func GoodBlankContext(_ context.Context, d time.Duration) {
	time.Sleep(d)
}

// Good: uses a context-aware select instead of time.Sleep.
func GoodSelectSleep(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Bad: time.Sleep in a goroutine closure that can close over the context.
func BadGoroutineWithCtx(ctx context.Context) {
	go func() {
		time.Sleep(time.Second) // want `use select with ctx\.Done\(\) instead of time\.Sleep to allow context cancellation`
	}()
}

// Good: function literal with its own context parameter already handles it.
func GoodFuncLitWithOwnCtx() {
	doWork(func(ctx context.Context, d time.Duration) {
		select {
		case <-time.After(d):
		case <-ctx.Done():
		}
	})
}

func doWork(fn func(context.Context, time.Duration)) {
	fn(context.Background(), time.Second)
}

// Good: inline nolint suppresses intentional sleep.
func GoodNoLint(ctx context.Context, d time.Duration) {
	_ = ctx
	time.Sleep(d) //nolint:timesleepnocontext
}
