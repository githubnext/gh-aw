//go:build !integration

package cli

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPSubprocessGuardrailLimitsConcurrentAcquisitions(t *testing.T) {
	guardrail := newMCPSubprocessGuardrail(maxActiveMCPChildProcesses)

	var current atomic.Int32
	var maxObserved atomic.Int32
	var wg sync.WaitGroup
	errCh := make(chan error, maxActiveMCPChildProcesses+2)
	acquiredCh := make(chan struct{}, maxActiveMCPChildProcesses+2)
	releaseCh := make(chan struct{})

	for range maxActiveMCPChildProcesses + 2 {
		wg.Go(func() {
			if err := guardrail.acquire(context.Background()); err != nil {
				errCh <- err
				return
			}
			defer guardrail.release()

			active := current.Add(1)
			defer current.Add(-1)

			for {
				previousMax := maxObserved.Load()
				if active <= previousMax || maxObserved.CompareAndSwap(previousMax, active) {
					break
				}
			}

			acquiredCh <- struct{}{}
			<-releaseCh
		})
	}

	for range maxActiveMCPChildProcesses {
		select {
		case <-acquiredCh:
		case err := <-errCh:
			t.Fatalf("unexpected acquisition error: %v", err)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for initial guardrail acquisitions")
		}
	}

	select {
	case <-acquiredCh:
		t.Fatal("guardrail allowed more than the configured number of active subprocesses")
	case err := <-errCh:
		t.Fatalf("unexpected acquisition error: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseCh)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err, "goroutine should not receive an acquisition error")
	}

	assert.Equal(t, int32(maxActiveMCPChildProcesses), maxObserved.Load(), "peak concurrent acquisitions should not exceed the guardrail limit")
}

func TestMCPSubprocessGuardrailAcquireHonorsContextCancellation(t *testing.T) {
	guardrail := newMCPSubprocessGuardrail(1)
	require.NoError(t, guardrail.acquire(context.Background()), "initial acquire on empty guardrail should succeed")
	defer guardrail.release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := guardrail.acquire(ctx)
	require.ErrorIs(t, err, context.Canceled, "acquire with a cancelled context should return context.Canceled")
}

func TestMCPSubprocessGuardrailAcquireHonorsCanceledContextWithAvailableSlot(t *testing.T) {
	guardrail := newMCPSubprocessGuardrail(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := guardrail.acquire(ctx)
	require.ErrorIs(t, err, context.Canceled, "acquire with a cancelled context should fail even when a slot is available")
}

func TestMCPSubprocessGuardrailAcquireCancelsWhileBlocked(t *testing.T) {
	guardrail := newMCPSubprocessGuardrail(1)
	require.NoError(t, guardrail.acquire(context.Background()), "initial acquire on empty guardrail should succeed")
	defer guardrail.release()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startedCh := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		close(startedCh)
		errCh <- guardrail.acquire(ctx)
	}()

	<-startedCh
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled, "blocked acquire should return context.Canceled after cancellation")
	case <-time.After(time.Second):
		t.Fatal("blocked acquire did not unblock after context cancellation")
	}
}
