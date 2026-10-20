package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivationPromptUsesLocalOutputBindings(t *testing.T) {
	for _, removeLabel := range []bool{false, true} {
		t.Run(map[bool]string{false: "retain label", true: "remove label"}[removeLabel], func(t *testing.T) {
			data := &WorkflowData{
				Name: "Label prompt", MainWorkflowMarkdown: "${{ needs.activation.outputs.label_command }}",
				LabelCommand: []string{"review"}, LabelCommandRemoveLabel: removeLabel,
			}
			job, err := NewCompiler().buildActivationJob(data, false, "", "test.lock.yml")
			require.NoError(t, err)
			steps := strings.Join(job.Steps, "")
			stepID := constants.GetTriggerLabelStepID
			if removeLabel {
				stepID = constants.RemoveTriggerLabelStepID
			}
			assert.Contains(t, steps, "GH_AW_NEEDS_ACTIVATION_OUTPUTS_LABEL_COMMAND: ${{ (steps."+stepID+".outputs.label_name) }}")
			assert.NotContains(t, steps, "${{ needs.activation.outputs.label_command }}")
		})
	}
}

func TestActivationPromptOutputRewritePreservesOtherReferences(t *testing.T) {
	data := &WorkflowData{
		Name: "Prompt output bindings",
		MainWorkflowMarkdown: "${{ format('needs.activation.outputs.label_command', needs.activation.outputs.label_command) }}\n" +
			"${{ needs.pre_activation.outputs.activated }}\n" +
			"${{ needs.prepare.outputs.value }}\n",
	}
	var steps []string
	err := NewCompiler().generatePromptInActivationJob(&steps, data, true, []string{"prepare"}, map[string]string{
		"label_command": "${{ steps.get_trigger_label.outputs.label_name || 'fallback' }}",
	})
	require.NoError(t, err)
	got := strings.Join(steps, "")
	assert.Contains(t, got, "format('needs.activation.outputs.label_command', (steps.get_trigger_label.outputs.label_name || 'fallback'))")
	assert.Contains(t, got, "${{ needs.pre_activation.outputs.activated }}")
	assert.Contains(t, got, "${{ needs.prepare.outputs.value }}")
}

func TestActivationPromptOutputBindingForms(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{"${{ needs.activation.outputs.comment_id }}", "${{ '' }}"},
		{"${{ needs.activation.outputs.setup-trace-id }}", "${{ (steps.setup.outputs.trace-id) }}"},
		{"${{ needs.activation.outputs.unknown }}", "${{ needs.activation.outputs.unknown }}"},
		{"${{ 'needs.activation.outputs.comment_id' }}", "${{ 'needs.activation.outputs.comment_id' }}"},
		{"needs.activation.outputs.comment_id", "needs.activation.outputs.comment_id"},
	} {
		got, err := rewriteActivationPromptOutputs(tt.source, map[string]string{
			"comment_id": `""`, "setup-trace-id": "${{ steps.setup.outputs.trace-id }}",
		})
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}
