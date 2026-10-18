//go:build !integration

package workflow

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExperimentalFeatureNoticeClassification(t *testing.T) {
	t.Parallel()
	for _, batch := range []bool{false, true} {
		compiler := NewCompiler()
		compiler.SetBatchMode(batch)
		data := &WorkflowData{
			ParsedTools: NewTools(map[string]any{"profile": "go"}),
			LSP:         map[string]LSPServerConfig{"typescript": {}},
			SafeOutputs: &SafeOutputsConfig{Steer: true},
		}
		var output bytes.Buffer
		compiler.emitExperimentalFeatureWarningsTo(data, &output)
		assert.Equal(t, 3, compiler.GetWarningCount())
		assert.Equal(t, 3, compiler.GetExperimentalWarningCount())
		if batch {
			assert.Equal(t, 1, compiler.GetExperimentalFeatureUsage()["Using experimental feature: tools.profile"])
			assert.Equal(t, 1, compiler.GetExperimentalFeatureUsage()["Using experimental feature: lsp"])
			assert.Equal(t, 1, compiler.GetExperimentalFeatureUsage()["Using experimental feature: safe-outputs steer"])
		} else {
			assert.Contains(t, output.String(), "Using experimental feature: tools.profile")
			assert.Contains(t, output.String(), "Using experimental feature: lsp")
			assert.Contains(t, output.String(), "Using experimental feature: safe-outputs steer")
		}
	}
}

func TestNoToolProfileExperimentalNotice(t *testing.T) {
	t.Parallel()
	for _, tools := range []*ToolsConfig{nil, NewTools(map[string]any{"bash": true})} {
		for _, batch := range []bool{false, true} {
			compiler := NewCompiler()
			compiler.SetBatchMode(batch)
			var output bytes.Buffer
			compiler.emitExperimentalFeatureWarningsTo(&WorkflowData{ParsedTools: tools}, &output)
			assert.Zero(t, compiler.GetWarningCount())
			assert.Zero(t, compiler.GetExperimentalWarningCount())
			assert.NotContains(t, output.String(), "Using experimental feature: tools.profile")
			assert.NotContains(t, compiler.GetExperimentalFeatureUsage(), "Using experimental feature: tools.profile")
		}
	}
}

func TestExperimentalWarningAccounting(t *testing.T) {
	t.Parallel()
	for _, batch := range []bool{false, true} {
		compiler := NewCompiler()
		compiler.SetBatchMode(batch)
		compiler.IncrementWarningCount()
		assert.Zero(t, compiler.GetExperimentalWarningCount())
		compiler.IncrementExperimentalWarningCount()
		assert.Equal(t, 2, compiler.GetWarningCount())
		assert.Equal(t, 1, compiler.GetExperimentalWarningCount())
		compiler.ResetWarningCount()
		assert.Zero(t, compiler.GetWarningCount())
		assert.Zero(t, compiler.GetExperimentalWarningCount())
	}

}
