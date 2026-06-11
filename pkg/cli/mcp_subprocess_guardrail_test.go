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
		require.NoError(t, err)
	}

	assert.Equal(t, int32(maxActiveMCPChildProcesses), maxObserved.Load())
}

func TestMCPSubprocessGuardrailAcquireHonorsContextCancellation(t *testing.T) {
	guardrail := newMCPSubprocessGuardrail(1)
	require.NoError(t, guardrail.acquire(context.Background()))
	defer guardrail.release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := guardrail.acquire(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
