//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAuditDataIncludesAmbientContext(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:   1,
			WorkflowName: "test",
			Status:       "completed",
			Conclusion:   "success",
			CreatedAt:    time.Now(),
		},
		TokenUsage: &TokenUsageSummary{
			AmbientContext: &AmbientContextMetrics{
				InputTokens:     1200,
				CachedTokens:    300,
				EffectiveTokens: 1500,
			},
		},
	}

	auditData := buildAuditData(processedRun, workflow.LogMetrics{}, nil)
	require.NotNil(t, auditData.Metrics.AmbientContext, "ambient context should be populated")
	assert.Equal(t, 1200, auditData.Metrics.AmbientContext.InputTokens, "input tokens should match")
	assert.Equal(t, 300, auditData.Metrics.AmbientContext.CachedTokens, "cached tokens should match")
	assert.Equal(t, 1500, auditData.Metrics.AmbientContext.EffectiveTokens, "effective tokens should match")
}
