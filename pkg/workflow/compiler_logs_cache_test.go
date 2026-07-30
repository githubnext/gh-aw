package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsesSharedLogsCache(t *testing.T) {
	tests := []struct {
		name string
		data WorkflowData
		want bool
	}{
		{
			name: "scheduled custom logs command",
			data: WorkflowData{On: "schedule: daily", CustomSteps: "run: gh aw logs --json"},
			want: true,
		},
		{
			name: "scheduled prompt audit command",
			data: WorkflowData{On: "schedule: daily", MarkdownContent: "Run `gh aw audit 123`."},
			want: true,
		},
		{
			name: "non-scheduled logs command",
			data: WorkflowData{On: "workflow_dispatch:", CustomSteps: "run: gh aw logs"},
			want: false,
		},
		{
			name: "scheduled unrelated workflow",
			data: WorkflowData{On: "schedule: daily", MarkdownContent: "Review issues."},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, usesSharedLogsCache(&tt.data))
		})
	}
}

func TestGenerateSharedLogsCacheRestoreSteps(t *testing.T) {
	cache := NewActionCache(t.TempDir())
	data := &WorkflowData{
		On:             "schedule: daily",
		CustomSteps:    "run: ./gh-aw logs",
		ActionCache:    cache,
		ActionResolver: NewActionResolver(cache),
	}
	var yaml strings.Builder

	generateSharedLogsCacheRestoreSteps(&yaml, data)

	output := yaml.String()
	assert.Contains(t, output, "actions/cache/restore@")
	assert.Contains(t, output, "path: "+sharedLogsCachePath)
	assert.NotContains(t, output, "actions/cache/save@")
}

func TestSharedLogsCacheRestoreFollowsCustomCheckout(t *testing.T) {
	cache := NewActionCache(t.TempDir())
	data := &WorkflowData{
		On:             "schedule: daily",
		CustomSteps:    "steps:\n  - uses: actions/checkout@v4\n  - name: Download logs\n    run: gh aw logs",
		ActionCache:    cache,
		ActionResolver: NewActionResolver(cache),
	}
	var yaml strings.Builder
	compiler := &Compiler{}

	compiler.addCustomStepsWithRuntimeInsertion(&yaml, data.CustomSteps, sharedLogsCacheRestoreSteps(data), nil, false)

	output := yaml.String()
	checkoutIndex := strings.Index(output, "uses: actions/checkout@v4")
	restoreIndex := strings.Index(output, "Restore shared agentic logs cache")
	logsIndex := strings.Index(output, "run: gh aw logs")
	assert.Greater(t, restoreIndex, checkoutIndex)
	assert.Greater(t, logsIndex, restoreIndex)
}
