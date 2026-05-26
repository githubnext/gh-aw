package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOnSection_LabelDispatchWithOtherEvents(t *testing.T) {
	c := &Compiler{}
	workflowData := &WorkflowData{WorkflowID: "central"}
	frontmatter := map[string]any{
		"on": map[string]any{
			"label_dispatch": map[string]any{
				"label":         "ns-pattern-review",
				"allowed-repos": []any{"my-org/*"},
			},
			"push": map[string]any{
				"branches": []any{"main"},
			},
		},
	}

	err := c.parseOnSection(frontmatter, workflowData, "/tmp/central.md")
	require.NoError(t, err)
	require.NotNil(t, workflowData.LabelDispatch)
	assert.Equal(t, "ns-pattern-review", workflowData.LabelDispatch.Label)
	assert.Equal(t, []string{"my-org/*"}, workflowData.LabelDispatch.AllowedRepos)
	require.NotNil(t, workflowData.LabelDispatchOtherEvents)
	assert.Contains(t, workflowData.LabelDispatchOtherEvents, "push")
}

func TestApplyDefaults_LabelDispatch(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name:       "Central Label Dispatch",
		WorkflowID: "central",
		On:         "",
		LabelDispatch: &LabelDispatchConfig{
			Label:        "ns-pattern-review",
			AllowedRepos: []string{"my-org/*"},
		},
	}

	mdPath := filepath.Join(t.TempDir(), "central.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("---\nname: test\n---\n"), 0644))
	require.NoError(t, c.applyDefaults(data, mdPath))
	assert.Contains(t, data.On, "repository_dispatch:")
	assert.Contains(t, data.On, "gh_aw_label_dispatch__central")
	assert.Contains(t, data.On, "workflow_dispatch:")
	assert.Contains(t, data.If, "github.event.client_payload.trigger_label == 'ns-pattern-review'")
	assert.Contains(t, data.If, "startsWith(github.event.client_payload.target_repo, 'my-org/')")
}

func TestGenerateLabelDispatchRemoteWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	stalePath := filepath.Join(tmpDir, "without-label.remote.yml")
	require.NoError(t, os.WriteFile(stalePath, []byte("stale"), 0644))

	workflows := []*WorkflowData{
		{
			Name:       "Central Label Dispatch",
			WorkflowID: "central",
			LabelDispatch: &LabelDispatchConfig{
				Label:        "ns-pattern-review",
				AllowedRepos: []string{"my-org/*"},
			},
		},
		{
			Name:       "Without Label",
			WorkflowID: "without-label",
		},
	}

	require.NoError(t, GenerateLabelDispatchRemoteWorkflows(workflows, tmpDir, "my-org/central-agents"))

	content, err := os.ReadFile(filepath.Join(tmpDir, "central.remote.yml"))
	require.NoError(t, err)
	text := string(content)
	assert.Contains(t, text, "types: [labeled]")
	assert.Contains(t, text, "github.event.label.name == 'ns-pattern-review'")
	assert.Contains(t, text, "my-org/central-agents")
	assert.Contains(t, text, "gh_aw_label_dispatch__central")
	assert.Contains(t, text, "target_repo")

	_, err = os.Stat(stalePath)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	assert.False(t, strings.Contains(text, "label_dispatch:"), "generated file should be plain GitHub Actions YAML")
}
