//go:build !integration

package workflow

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		"explicit false": `{"webhook-gateway":false}`,
		"explicit true":  `{"webhook-gateway":true}`,
	} {
		for trigger, on := range map[string]string{
			"project":    "on:\n  gateway:\n    projects-items: {project_id: PVT_example}\n",
			"security":   "on:\n  gateway:\n    security-alerts: {sources: [dependabot]}\n",
			"multi-repo": "on:\n  gateway:\n    security-alerts: {repos: [bar/baz, \"foo/*\"]}\n",
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
					assert.Contains(t, err.Error(), `{"webhook-gateway":true}`)
					assert.Contains(t, err.Error(), RepoConfigFileName)
				}
				_, err = compiler.ParseWorkflowString(webhookWorkflow("on: workflow_dispatch\n"), "native.md")
				require.NoError(t, err, "native triggers do not require the opt-in")
			})
		}
	}
}

func TestWebhookOptInCannotBeBypassedByPreparsedData(t *testing.T) {
	data, err := newWebhookCompiler().ParseWorkflowString(webhookWorkflow("on:\n  gateway:\n    projects-items: {project_id: PVT_example}\n"), "preparsed.md")
	require.NoError(t, err)
	compiler := NewCompiler()
	compiler.gitRoot = t.TempDir()
	_, err = compiler.CompileToYAML(data, "preparsed.md")
	require.ErrorContains(t, err, "hosted CAO server")

	compiler = NewCompiler(WithNoEmit(true))
	compiler.gitRoot = t.TempDir()
	err = compiler.CompileWorkflowData(data, filepath.Join(t.TempDir(), "preparsed.md"))
	require.ErrorContains(t, err, "hosted CAO server")
}

func TestWebhookMinimumRoles(t *testing.T) {
	for name, trigger := range map[string]string{
		"projects":   "gateway: {projects-items: {project_id: PVT_example}}",
		"code":       "gateway: {security-alerts: {sources: [code-scanning]}}",
		"dependabot": "gateway: {security-alerts: {sources: [dependabot]}}",
		"multi-repo": "gateway: {security-alerts: {repos: [bar/baz, \"foo/*\"]}}",
	} {
		for _, tt := range []struct {
			roles string
			valid bool
		}{
			{"", true},
			{"write", true},
			{"[write]", true},
			{"maintain", true},
			{"maintainer", true},
			{"[admin]", true},
			{"[admin, maintain, write]", true},
			{"all", false},
			{"read", false},
			{"triage", false},
			{"[admin, write, read]", false},
			{"[write, triage]", false},
		} {
			for _, strict := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%s/strict=%t", name, tt.roles, strict), func(t *testing.T) {
					roleConfig := ""
					if tt.roles != "" {
						roleConfig = "  roles: " + tt.roles + "\n"
					}
					source := webhookWorkflow(fmt.Sprintf("strict: %t\non:\n  workflow_dispatch:\n  repository_dispatch:\n    types: [manual]\n  bots: [cao-gateway]\n%s  %s\n", strict, roleConfig, trigger))
					compiler := newWebhookCompiler()
					data, err := compiler.ParseWorkflowString(source, "webhook-roles.md")
					if !tt.valid {
						require.ErrorContains(t, err, "require on.roles with write access or higher")
						assert.Contains(t, err.Error(), "[admin, maintainer, write]")
						return
					}
					require.NoError(t, err)
					output, err := compiler.CompileToYAML(data, "webhook-roles.md")
					require.NoError(t, err)
					preActivation := extractJobSection(output, "pre_activation")
					assert.Contains(t, preActivation, "check_membership.cjs")
					assert.Contains(t, preActivation, fmt.Sprintf("GH_AW_REQUIRED_ROLES: %q", strings.Join(data.Roles, ",")))
					assert.Contains(t, preActivation, "steps.check_membership.outputs.is_team_member")
					assert.True(t, compiler.needsRoleCheck(data, map[string]any{"on": map[string]any{"workflow_dispatch": nil}}))
				})
			}
		}
	}
}

func TestWebhookMinimumRolesCannotBeBypassedByPreparsedData(t *testing.T) {
	for _, roles := range [][]string{nil, {}, {"all"}, {"read"}, {"triage"}, {"write", "read"}, {"admin", "all"}, {"unknown"}} {
		t.Run(fmt.Sprint(roles), func(t *testing.T) {
			data, err := newWebhookCompiler().ParseWorkflowString(webhookWorkflow("strict: false\non:\n  gateway:\n    projects-items: {project_id: PVT_example}\n"), "preparsed-roles.md")
			require.NoError(t, err)
			data.Roles = roles
			_, err = newWebhookCompiler().CompileToYAML(data, "preparsed-roles.md")
			require.ErrorContains(t, err, "require on.roles with write access or higher")
			path := filepath.Join(t.TempDir(), "preparsed-roles.md")
			err = newWebhookCompiler().CompileWorkflowData(data, path)
			require.ErrorContains(t, err, "require on.roles with write access or higher")
			_, err = os.Stat(filepath.Join(filepath.Dir(path), "preparsed-roles.lock.yml"))
			require.True(t, os.IsNotExist(err), "invalid roles must not produce a lock file")
		})
	}
}

func TestWebhookMinimumRolesDoesNotRestrictNativeTriggers(t *testing.T) {
	for _, roles := range []string{"all", "read", "triage", "[write, read]"} {
		t.Run(roles, func(t *testing.T) {
			compiler := NewCompiler()
			source := webhookWorkflow("on:\n  issues: {types: [opened]}\n  roles: " + roles + "\n")
			data, err := compiler.ParseWorkflowString(source, "native-roles.md")
			require.NoError(t, err)
			_, err = compiler.CompileToYAML(data, "native-roles.md")
			require.NoError(t, err)
		})
	}
}

func TestWebhookRepoConfigValidation(t *testing.T) {
	for _, content := range []string{
		`{"webhook-gateway":null}`, `{"webhook-gateway":{}}`,
		`{"webhook-gateway":"true"}`, `{"webhook-gateway":1}`,
		`{"webhook-gateway":[]}`, `{"webhook-gateway":true`,
		`{"cao":{"triggers":true}}`,
		`{"repository-dispatch-triggers":true}`,
		`{"webhook-triggers":true}`,
		`{"webhook-triggers":false,"webhook-gateway":true}`,
		`{"webhook-gateway":true,"cao":{"triggers":true}}`,
	} {
		t.Run(content, func(t *testing.T) {
			root := t.TempDir()
			writeWebhookTriggerConfig(t, root, content)
			_, err := LoadRepoConfig(root)
			require.Error(t, err)
			compiler := NewCompiler()
			compiler.gitRoot = root
			_, err = compiler.ParseWorkflowString(webhookWorkflow("on:\n  gateway:\n    security-alerts: {sources: [dependabot]}\n"), "invalid-cao.md")
			require.ErrorContains(t, err, "cannot enable hosted CAO triggers")
		})
	}
}

func TestWebhookFrontmatterCannotEnableTriggers(t *testing.T) {
	for _, declaration := range []string{"webhook-gateway: true\n", "features: {webhook-gateway: true}\n"} {
		compiler := NewCompiler()
		compiler.gitRoot = t.TempDir()
		_, err := compiler.ParseWorkflowString(webhookWorkflow(declaration+"on:\n  gateway:\n    projects-items: {project_id: PVT_example}\n"), "frontmatter-cao.md")
		require.Error(t, err)
	}
}

func TestWebhookExperimentalNotices(t *testing.T) {
	for name, events := range map[string][]string{
		"native":   nil,
		"projects": {"projects_v2_item"},
		"security": {"dependabot_alert", "code_scanning_alert"},
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
