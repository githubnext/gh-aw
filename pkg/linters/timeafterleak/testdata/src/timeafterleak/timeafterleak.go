package timeafterleak

import (
	"context"
	"time"
)

// BadForLoop is the canonical timer-leak: time.After in a for+select Comm.
func BadForLoop(ctx context.Context) {
	for {
		select {
		case <-time.After(time.Second): // want `time\.After creates a new timer on each loop iteration`
			doWork()
		case <-ctx.Done():
			return
		}
	}
}

// BadForLoopAssign also leaks: the receive is the Comm of the case clause.
func BadForLoopAssign(ctx context.Context) {
	for {
		select {
		case t := <-time.After(time.Second): // want `time\.After creates a new timer on each loop iteration`
			_ = t
		case <-ctx.Done():
			return
		}
	}
}

// BadRangeLoop flags time.After in a range-based loop select.
func BadRangeLoop(items []string, ctx context.Context) {
	for range items {
		select {
		case <-time.After(time.Millisecond): // want `time\.After creates a new timer on each loop iteration`
		case <-ctx.Done():
			return
		}
	}
}

// GoodNoLoop is fine: time.After in a select that is not inside a loop.
func GoodNoLoop(ctx context.Context) {
	select {
	case <-time.After(time.Second):
		doWork()
	case <-ctx.Done():
		return
	}
}

// GoodNewTimer uses the correct pattern: a single timer reused each iteration.
func GoodNewTimer(ctx context.Context) {
	t := time.NewTimer(time.Second)
	defer t.Stop()
	for {
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
		t.Reset(time.Second)
		select {
		case <-t.C:
			doWork()
		case <-ctx.Done():
			return
		}
	}
}

// GoodTimeAfterInBody calls time.After inside the case body, not as the Comm.
// The timer fires exactly once here, so there is no accumulation leak.
func GoodTimeAfterInBody(ctx context.Context, ch <-chan struct{}) {
	for {
		select {
		case <-ch:
			<-time.After(time.Second) // in Body, not Comm — not flagged
		case <-ctx.Done():
			return
		}
	}
}

// GoodFuncLitInsideLoop: the select is inside a goroutine closure launched
// per iteration; the for loop does not directly enclose the CommClause.
func GoodFuncLitInsideLoop(ctx context.Context) {
	for {
		go func() {
			select {
			case <-time.After(time.Second): // FuncLit boundary — not flagged
			case <-ctx.Done():
			}
		}()
		<-ctx.Done()
		return
	}
}

func doWork() {}
