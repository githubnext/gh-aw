//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectJSONImportNameOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		currentName string
		workflow    *JSONWorkflow
		want        string
	}{
		{
			name:        "uses json name when available, overrides current name",
			currentName: "weekly-research",
			workflow: &JSONWorkflow{
				Name: "Workflow Title",
			},
			want: "workflow-title",
		},
		{
			name:        "uses json name when current name is guid",
			currentName: "0be2cc4b-de12-43fe-ada7-55ef6dc8f3ba",
			workflow: &JSONWorkflow{
				Name: "Issue Triage",
			},
			want: "issue-triage",
		},
		{
			name:        "falls back to json title from extra when name missing",
			currentName: "0be2cc4b-de12-43fe-ada7-55ef6dc8f3ba",
			workflow: &JSONWorkflow{
				Extra: map[string]any{"title": "Title From JSON"},
			},
			want: "title-from-json",
		},
		{
			name:        "keeps current name when no json name or title",
			currentName: "0be2cc4b-de12-43fe-ada7-55ef6dc8f3ba",
			workflow:    &JSONWorkflow{},
			want:        "0be2cc4b-de12-43fe-ada7-55ef6dc8f3ba",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, selectJSONImportNameOverride(tt.currentName, tt.workflow))
		})
	}
}
