//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLogsDataIncludesAmbientContext(t *testing.T) {
	tmpDir := testutil.TempDir(t, "logs-ambient-context")
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   1,
				WorkflowName: "test",
				Status:       "completed",
				Conclusion:   "success",
				CreatedAt:    time.Now(),
				LogsPath:     tmpDir,
			},
			TokenUsage: &TokenUsageSummary{
				AmbientContext: &AmbientContextMetrics{
					InputTokens:     800,
					CachedTokens:    200,
					EffectiveTokens: 1000,
				},
			},
		},
	}

	data := buildLogsData(processedRuns, tmpDir, nil)
	require.Len(t, data.Runs, 1, "should produce a single run")
	require.NotNil(t, data.Runs[0].AmbientContext, "ambient context should be included")
	assert.Equal(t, 800, data.Runs[0].AmbientContext.InputTokens, "input tokens should match")
	assert.Equal(t, 200, data.Runs[0].AmbientContext.CachedTokens, "cached tokens should match")
	assert.Equal(t, 1000, data.Runs[0].AmbientContext.EffectiveTokens, "effective tokens should match")
}
