//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unifiedSteeringCacheEvent = `{"type":"firewall.steering","timestamp":"2026-10-10T10:00:00Z","data":{"event":"token_steering","message":"[AWF TOKEN WARNING] budget","requestId":"req-1","threshold":80},"provenance":{"component":"firewall","phase":"agent"}}` + "\n"

func unifiedSteeringCacheRun(t *testing.T) (string, string) {
	t.Helper()
	runDir := testutil.TempDir(t, "unified-steering-cache")
	t.Cleanup(func() {
		unifiedSteeringLogCache.Delete(filepath.Clean(runDir))
		apiProxySteeringLogCache.Delete(filepath.Clean(runDir))
	})
	return runDir, filepath.Join(runDir, "aw_session.jsonl")
}

func TestUnifiedSteeringLogCache(t *testing.T) {
	t.Run("summary and timeline share parsed result", func(t *testing.T) {
		runDir, path := unifiedSteeringCacheRun(t)
		require.NoError(t, os.WriteFile(path, []byte(unifiedSteeringCacheEvent), 0o644))

		summary := &TokenUsageSummary{}
		applyGatewaySteeringSummary(summary, runDir)
		require.Equal(t, 1, summary.TotalSteeringEvents)
		require.NoError(t, os.WriteFile(path, []byte("invalid json\n"), 0o644))

		events, err := extractGatewaySteeringEvents(runDir)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, GatewaySteeringEvent{
			Type: tokenSteeringEventName, Message: "[AWF TOKEN WARNING] budget",
			Timestamp: "2026-10-10T10:00:00Z", RequestID: "req-1", Threshold: 80,
		}, events[0])

		cached, err := readUnifiedSteeringLog(runDir + string(os.PathSeparator) + ".")
		require.NoError(t, err)
		require.NotNil(t, cached)
		assert.Equal(t, summary.SteeringEventCounts, cached.eventCounts)
	})

	t.Run("caches absent session", func(t *testing.T) {
		runDir, path := unifiedSteeringCacheRun(t)
		log, err := readUnifiedSteeringLog(runDir)
		require.NoError(t, err)
		require.Nil(t, log)
		require.NoError(t, os.WriteFile(path, []byte("invalid json\n"), 0o644))

		log, err = readUnifiedSteeringLog(runDir)
		require.NoError(t, err)
		assert.Nil(t, log)
	})

	t.Run("caches session without steering", func(t *testing.T) {
		runDir, path := unifiedSteeringCacheRun(t)
		require.NoError(t, os.WriteFile(path, []byte("{}\n"), 0o644))
		log, err := readUnifiedSteeringLog(runDir)
		require.NoError(t, err)
		require.Nil(t, log)
		require.NoError(t, os.WriteFile(path, []byte("invalid json\n"), 0o644))

		log, err = readUnifiedSteeringLog(runDir)
		require.NoError(t, err)
		assert.Nil(t, log)
	})

	t.Run("retries parse errors", func(t *testing.T) {
		runDir, path := unifiedSteeringCacheRun(t)
		require.NoError(t, os.WriteFile(path, []byte("invalid json\n"), 0o644))
		_, err := readUnifiedSteeringLog(runDir)
		require.ErrorContains(t, err, "invalid session event")
		require.NoError(t, os.WriteFile(path, []byte(unifiedSteeringCacheEvent), 0o644))

		log, err := readUnifiedSteeringLog(runDir)
		require.NoError(t, err)
		require.NotNil(t, log)
		assert.Equal(t, 1, log.eventCounts[tokenSteeringEventName])
	})

	t.Run("recovers from invalid cached value", func(t *testing.T) {
		runDir, path := unifiedSteeringCacheRun(t)
		require.NoError(t, os.WriteFile(path, []byte(unifiedSteeringCacheEvent), 0o644))
		unifiedSteeringLogCache.Store(filepath.Clean(runDir), "invalid")

		log, err := readUnifiedSteeringLog(runDir)
		require.NoError(t, err)
		require.NotNil(t, log)
		assert.Equal(t, 1, log.eventCounts[tokenSteeringEventName])
	})

	t.Run("concurrent readers share one result", func(t *testing.T) {
		runDir, path := unifiedSteeringCacheRun(t)
		require.NoError(t, os.WriteFile(path, []byte(unifiedSteeringCacheEvent), 0o644))
		const readers = 16
		logs := make([]*apiProxySteeringLog, readers)
		errs := make([]error, readers)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range readers {
			wg.Go(func() {
				<-start
				logs[i], errs[i] = readUnifiedSteeringLog(runDir)
			})
		}
		close(start)
		wg.Wait()

		require.NotNil(t, logs[0])
		for i := range readers {
			require.NoError(t, errs[i])
			assert.Same(t, logs[0], logs[i])
		}
	})
}
