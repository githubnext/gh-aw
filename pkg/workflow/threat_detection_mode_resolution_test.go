//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreatDetectionModeResolution(t *testing.T) {
	runtimeExpr := "${{ steps.detection_mode.outputs.continue_on_error == 'true' }}"
	explicitExpr := "${{ inputs.warn }}"
	for _, tc := range []struct {
		name           string
		repo           *bool
		workflow       *bool
		expression     *string
		wantValue      bool
		wantExpression *string
		wantRuntime    bool
	}{
		{name: "runtime default", wantValue: true, wantExpression: &runtimeExpr, wantRuntime: true},
		{name: "repo false", repo: new(false)},
		{name: "repo true", repo: new(true), wantValue: true},
		{name: "workflow false overrides repo", repo: new(true), workflow: new(false)},
		{name: "workflow true overrides repo", repo: new(false), workflow: new(true), wantValue: true},
		{name: "workflow expression overrides repo", repo: new(false), expression: &explicitExpr, wantValue: true, wantExpression: &explicitExpr},
		{name: "explicit runtime expression is not a default", expression: &runtimeExpr, wantValue: true, wantExpression: &runtimeExpr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCompiler()
			c.repoConfigLoaded = true
			c.repoConfig = &RepoConfig{DefaultDetectionContinueOnError: tc.repo}
			data := &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						ContinueOnError:     tc.workflow,
						ContinueOnErrorExpr: tc.expression,
					},
				},
			}

			value, expression, runtimeDefault := c.resolveThreatDetectionContinueOnError(data)
			assert.Equal(t, tc.wantValue, value)
			assert.Equal(t, tc.wantExpression, expression)
			assert.Equal(t, tc.wantRuntime, runtimeDefault)
			step := strings.Join(c.buildDetectionDefaultStep(data), "")
			assert.Equal(t, tc.wantRuntime, strings.Contains(step, "id: detection_mode"))
			if !tc.wantRuntime {
				assert.Empty(t, step)
			}
		})
	}
}
