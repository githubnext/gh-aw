//go:build !integration

package workflow

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomJobDetectionCompilation(t *testing.T) {
	for _, tt := range []struct {
		name, fields, condition string
		disabled                bool
	}{
		{"direct", "needs: agent", "", false},
		{"detection only", "needs: detection", "", false},
		{"always", "needs: agent\n    if: always()", "always()", false},
		{"or bypass", "needs: [agent, detection]\n    if: always() || needs.agent.result == 'failure'", "always() || needs.agent.result == 'failure'", false},
		{"false condition", "needs: agent\n    if: 'false'", "false", false},
		{"disabled detection", "needs: agent\n    if: always()", "always()", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			detection := ""
			if tt.disabled {
				detection = "  threat-detection: false\n"
			}
			document, err := aw.Parse("---\non: workflow_dispatch\nengine: copilot\npermissions: {contents: read}\nsafe-outputs:\n  create-issue:\n" + detection +
				"jobs:\n  report:\n    uses: ./.github/workflows/report.yml\n    " + tt.fields + "\n---\nPublic prompt.\n")
			require.NoError(t, err)
			compiler := NewCompiler(WithNoEmit(true))
			data, err := compiler.ParseWorkflowDocument(document, filepath.Join(t.TempDir(), "test.md"))
			require.NoError(t, err)
			content, err := compiler.CompileToYAML(data, "test.md")
			require.NoError(t, err)
			workflow, err := actions.Parse([]byte(content))
			require.NoError(t, err)
			job, exists := compiler.jobManager.GetJob("report")
			require.True(t, exists)
			if tt.disabled {
				assert.NotContains(t, job.Needs, "detection")
				assert.NotContains(t, job.If, "needs.detection")
			} else {
				assert.Contains(t, job.Needs, "detection")
				assert.Contains(t, job.If, "needs.detection.result == 'success'")
				assert.Contains(t, job.If, "needs.detection.outputs.detection_success == 'true'")
				if tt.condition != "" {
					assert.Contains(t, job.If, "("+tt.condition+")")
				}
				agent, _ := compiler.jobManager.GetJob("agent")
				assert.NotContains(t, agent.Needs, "report")
			}
			runner, err := workflowlint.Default().Select("custom-job-detection")
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), workflowlint.Input{Workflow: workflow, Document: document})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	}
}

func TestCustomJobDetectionGuard(t *testing.T) {
	for _, condition := range []string{"", "always()"} {
		job := &Job{Name: "report", Needs: []string{"agent"}, If: condition}
		require.NoError(t, addCustomJobDetectionGuard(job))
		node, err := ParseExpression(stripExpressionWrapper(job.If))
		require.NoError(t, err)
		for _, result := range []string{"success", "failure", "cancelled", "skipped", ""} {
			for _, detected := range []string{"true", "false", ""} {
				values := map[string]string{
					"needs.detection.result":                    result,
					"needs.detection.outputs.detection_success": detected,
				}
				assert.Equal(t, result == "success" && detected == "true", evaluateDetectionRedactionGate(t, node, values),
					"result=%s detection_success=%s", result, detected)
			}
		}
	}
	for _, condition := range []string{"${{ always() }} trailing", "always() &&", "${{ false }}${{ true }}"} {
		job := &Job{Name: "report", Needs: []string{"agent"}, If: condition}
		require.ErrorContains(t, addCustomJobDetectionGuard(job), "jobs.report.if")
	}
}

func TestCustomJobDetectionTransitiveAndPreDetection(t *testing.T) {
	for _, intermediate := range []bool{false, true} {
		compiler := NewCompiler()
		compiler.jobManager = NewJobManager()
		require.NoError(t, compiler.jobManager.AddJob(&Job{Name: "agent"}))
		detectionNeeds := []string{"agent"}
		if intermediate {
			detectionNeeds = append(detectionNeeds, "scan")
		}
		require.NoError(t, compiler.jobManager.AddJob(&Job{Name: "detection", Needs: detectionNeeds}))
		require.NoError(t, compiler.jobManager.AddJob(&Job{Name: "scan", Needs: []string{"agent"}, If: "always()"}))
		require.NoError(t, compiler.jobManager.AddJob(&Job{Name: "report", Needs: []string{"scan"}, If: "always()"}))
		data := &WorkflowData{Jobs: map[string]any{"scan": map[string]any{"needs": "agent"}, "report": map[string]any{"needs": "scan"}}}
		err := compiler.guardCustomJobDetection(data)
		if intermediate {
			require.ErrorContains(t, err, "not between them")
			continue
		}
		require.NoError(t, err)
		for _, id := range []string{"scan", "report"} {
			job, _ := compiler.jobManager.GetJob(id)
			assert.Contains(t, job.Needs, "detection")
			assert.Contains(t, job.If, "needs.detection.outputs.detection_success == 'true'")
		}
		dependencies, _ := compiler.buildMainJobDependencies(data, true)
		assert.NotContains(t, dependencies, "scan")
		assert.NotContains(t, dependencies, "report")
	}
}

func TestCustomJobDetectionOrdering(t *testing.T) {
	for _, beforeAgent := range []bool{false, true} {
		compiler := NewCompiler()
		compiler.jobManager = NewJobManager()
		agent := &Job{Name: "agent"}
		if beforeAgent {
			agent.Needs = []string{"prepare"}
		}
		require.NoError(t, compiler.jobManager.AddJob(agent))
		require.NoError(t, compiler.jobManager.AddJob(&Job{Name: "detection", Needs: []string{"agent"}}))
		prepare := &Job{Name: "prepare", If: "always()"}
		require.NoError(t, compiler.jobManager.AddJob(prepare))
		err := compiler.guardCustomJobDetection(&WorkflowData{Jobs: map[string]any{"prepare": map[string]any{}}})
		if beforeAgent {
			require.NoError(t, err)
			assert.Empty(t, prepare.Needs)
			assert.Equal(t, "always()", prepare.If)
		} else {
			require.ErrorContains(t, err, "custom jobs must run before agent or after detection")
		}
	}
}
