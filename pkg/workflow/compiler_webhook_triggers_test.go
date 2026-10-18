//go:build !integration

package workflow

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeWebhookTriggerConfig(t *testing.T, root, content string) {
	t.Helper()
	if content == "" {
		return
	}
	path := filepath.Join(root, RepoConfigFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestWebhookRequiresTriggerOptIn(t *testing.T) {
	for name, config := range map[string]string{
		"missing file":   "",
		"missing option": `{}`,
		"explicit false": `{"webhook-triggers":false}`,
		"explicit true":  `{"webhook-triggers":true}`,
	} {
		for trigger, on := range map[string]string{
			"project":  "on:\n  projects-item: {project_id: PVT_example}\n",
			"security": "on:\n  security-alert: {sources: [dependabot]}\n",
		} {
			t.Run(name+"/"+trigger, func(t *testing.T) {
				compiler := NewCompiler()
				compiler.gitRoot = t.TempDir()
				writeWebhookTriggerConfig(t, compiler.gitRoot, config)
				data, err := compiler.ParseWorkflowString(webhookWorkflow(on), "cao-opt-in.md")
				if name == "explicit true" {
					require.NoError(t, err)
					_, err = compiler.CompileToYAML(data, "cao-opt-in.md")
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, "hosted CAO server")
					assert.Contains(t, err.Error(), `{"webhook-triggers":true}`)
					assert.Contains(t, err.Error(), RepoConfigFileName)
				}
				_, err = compiler.ParseWorkflowString(webhookWorkflow("on: workflow_dispatch\n"), "native.md")
				require.NoError(t, err, "native triggers do not require the opt-in")
			})
		}
	}
}

func TestWebhookOptInCannotBeBypassedByPreparsedData(t *testing.T) {
	data, err := newWebhookCompiler().ParseWorkflowString(webhookWorkflow("on:\n  projects-item: {project_id: PVT_example}\n"), "preparsed.md")
	require.NoError(t, err)
	compiler := NewCompiler()
	compiler.gitRoot = t.TempDir()
	_, err = compiler.CompileToYAML(data, "preparsed.md")
	require.ErrorContains(t, err, "hosted CAO server")
}

func TestWebhookRepoConfigValidation(t *testing.T) {
	for _, content := range []string{
		`{"webhook-triggers":null}`, `{"webhook-triggers":{}}`,
		`{"webhook-triggers":"true"}`, `{"webhook-triggers":1}`,
		`{"webhook-triggers":[]}`, `{"webhook-triggers":true`,
		`{"cao":{"triggers":true}}`,
		`{"repository-dispatch-triggers":true}`,
		`{"webhook-triggers":true,"cao":{"triggers":true}}`,
	} {
		t.Run(content, func(t *testing.T) {
			root := t.TempDir()
			writeWebhookTriggerConfig(t, root, content)
			_, err := LoadRepoConfig(root)
			require.Error(t, err)
			compiler := NewCompiler()
			compiler.gitRoot = root
			_, err = compiler.ParseWorkflowString(webhookWorkflow("on:\n  security-alert: {sources: [dependabot]}\n"), "invalid-cao.md")
			require.ErrorContains(t, err, "cannot enable hosted CAO triggers")
		})
	}
}

func TestWebhookFrontmatterCannotEnableTriggers(t *testing.T) {
	for _, declaration := range []string{"webhook-triggers: true\n", "features: {webhook-triggers: true}\n"} {
		compiler := NewCompiler()
		compiler.gitRoot = t.TempDir()
		_, err := compiler.ParseWorkflowString(webhookWorkflow(declaration+"on:\n  projects-item: {project_id: PVT_example}\n"), "frontmatter-cao.md")
		require.Error(t, err)
	}
}

func TestWebhookExperimentalNotices(t *testing.T) {
	for name, events := range map[string][]string{
		"native":   nil,
		"projects": {"projects_v2_item"},
		"security": {"dependabot_alert", "secret_scanning_alert"},
		"both":     {"projects_v2_item", "code_scanning_alert"},
	} {
		t.Run(name, func(t *testing.T) {
			data := &WorkflowData{}
			for _, event := range events {
				data.WebhookSubscriptions = append(data.WebhookSubscriptions, WebhookSubscription{Event: event})
			}
			compiler := NewCompiler()
			var output bytes.Buffer
			compiler.emitExperimentalFeatureWarningsTo(data, &output)
			expected := len(events)
			if name == "security" {
				expected = 1
			}
			assert.Equal(t, expected, compiler.GetExperimentalWarningCount())
			if expected > 0 {
				assert.Contains(t, output.String(), "Using experimental feature: on.")
				assert.Contains(t, output.String(), "hosted CAO triggers")
			}
		})
	}
}
