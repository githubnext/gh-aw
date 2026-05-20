//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLooksLikeGUID(t *testing.T) {
	t.Parallel()

	assert.True(t, looksLikeGUID("b5a3f76a-3d8f-4790-b7e2-f2886f784345"))
	assert.True(t, looksLikeGUID("{B5A3F76A-3D8F-4790-B7E2-F2886F784345}"))
	assert.False(t, looksLikeGUID("weekly-research"))
	assert.False(t, looksLikeGUID("12345678-1234-1234-1234-123456789ab"))
}

func TestSelectJSONImportNameOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		currentName string
		workflow    *JSONWorkflow
		want        string
	}{
		{
			name:        "keeps non-guid current name",
			currentName: "weekly-research",
			workflow: &JSONWorkflow{
				Name: "Workflow Title",
			},
			want: "weekly-research",
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
			name:        "keeps guid when no json name or title",
			currentName: "0be2cc4b-de12-43fe-ada7-55ef6dc8f3ba",
			workflow:    &JSONWorkflow{},
			want:        "0be2cc4b-de12-43fe-ada7-55ef6dc8f3ba",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, selectJSONImportNameOverride(tt.currentName, tt.workflow))
		})
	}
}
