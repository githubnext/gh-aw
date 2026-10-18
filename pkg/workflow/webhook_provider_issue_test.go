//go:build !integration

package workflow

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func providerIssueOn(alias string, config any) map[string]any {
	return map[string]any{"webhook-gateway": true, alias: config}
}

func TestProviderIssueSubscriptions(t *testing.T) {
	for alias, event := range providerIssueTriggers {
		t.Run(alias, func(t *testing.T) {
			config := map[string]any{"types": []any{"labeled", "opened", "opened"}}
			on := providerIssueOn(alias, config)
			before, err := json.Marshal(on)
			require.NoError(t, err)
			subscriptions, err := ParseWebhookSubscriptions(on, "provider-agent")
			require.NoError(t, err)
			require.Len(t, subscriptions, 1)
			sub := subscriptions[0]
			assert.Equal(t, event, sub.Event)
			assert.Equal(t, []string{"labeled", "opened"}, sub.Types)
			require.NotEmpty(t, sub.Provider)
			require.NotNil(t, sub.ActivityNormalization)
			assert.Equal(t, sub.Provider+"-issue/v1", sub.ActivityNormalization.Policy)
			assert.Len(t, sub.NativeEvents, 2)
			assert.Equal(t, 2, sub.SchemaVersion)
			assert.Equal(t, WebhookDispatchEventType("provider-agent"), sub.Dispatch.EventType)
			assert.Nil(t, sub.Dispatch.PayloadProjection)
			assert.Empty(t, sub.Repos)
			assert.Empty(t, NormalizeGitHubAppWebhookEvents(on), "external providers must not become GitHub App webhook requirements")
			after, err := json.Marshal(on)
			require.NoError(t, err)
			assert.JSONEq(t, string(before), string(after), "subscription parsing must not mutate source configuration")
		})
	}
}

func TestProviderIssueMixedGitHubRequirements(t *testing.T) {
	on := providerIssueOn("jira-issue", map[string]any{})
	on["linear-issue"] = map[string]any{}
	on["azure-devops-issue"] = map[string]any{}
	on["issues"] = map[string]any{"types": []any{"opened"}}
	on["security-alerts"] = map[string]any{"sources": "dependabot"}
	on["projects-items"] = map[string]any{"project_id": "PVT_example"}
	assert.Equal(t, []string{"dependabot_alert", "issues", "projects_v2_item"}, NormalizeGitHubAppWebhookEvents(on))
	assert.Equal(t, map[string]string{"organization_projects": "read", "vulnerability_alerts": "read"}, ComputeGitHubAppManifestWebhookPermissions(on))
}

func TestProviderIssueExperimentalNotice(t *testing.T) {
	for alias, event := range providerIssueTriggers {
		t.Run(alias, func(t *testing.T) {
			subscriptions, err := ParseWebhookSubscriptions(providerIssueOn(alias, map[string]any{}), "notice")
			require.NoError(t, err)
			var output bytes.Buffer
			NewCompiler().emitExperimentalFeatureWarningsTo(&WorkflowData{WebhookSubscriptions: subscriptions}, &output)
			assert.Contains(t, output.String(), "Using experimental feature: on."+alias+" (approved webhook bridge required)")
			assert.NotContains(t, output.String(), "on.security-alerts")
			assert.Equal(t, event, subscriptions[0].Event)
		})
	}
	on := map[string]any{"webhook-gateway": true}
	for alias := range providerIssueTriggers {
		on[alias] = providerIssueOn(alias, map[string]any{})[alias]
	}
	subscriptions, err := ParseWebhookSubscriptions(on, "all-providers")
	require.NoError(t, err)
	var output bytes.Buffer
	NewCompiler().emitExperimentalFeatureWarningsTo(&WorkflowData{WebhookSubscriptions: subscriptions}, &output)
	for alias := range providerIssueTriggers {
		assert.Equal(t, 1, strings.Count(output.String(), "Using experimental feature: on."+alias+" ("))
	}
}

func TestProviderIssueValidation(t *testing.T) {
	for alias := range providerIssueTriggers {
		for name, config := range map[string]any{
			"null":            nil,
			"scalar":          true,
			"unknown key":     map[string]any{"typo": true},
			"native activity": map[string]any{"types": []any{"created"}},
			"misspelling":     map[string]any{"types": []any{"labelled"}},
			"empty types":     map[string]any{"types": []any{}},
			"GitHub repos":    map[string]any{"repos": []any{"owner/repo"}},
			"unknown filter":  map[string]any{"filters": map[string]any{"repository": "owner/repo"}},
			"expression":      map[string]any{"filters": map[string]any{"sender": "${{ github.actor }}"}},
		} {
			t.Run(alias+"/"+name, func(t *testing.T) {
				_, err := ParseWebhookSubscriptions(providerIssueOn(alias, config), "invalid")
				require.Error(t, err)
			})
		}
		for _, on := range []any{alias, []any{"push", alias}, map[string]any{alias: map[string]any{}}} {
			_, err := ParseWebhookSubscriptions(on, "ungated")
			require.Error(t, err)
		}
	}
}

func TestAzureIssueClosedStates(t *testing.T) {
	for name, states := range map[string]any{
		"missing": nil, "empty": []any{}, "blank": " ", "numeric": 1, "expression": "${{ github.actor }}",
	} {
		t.Run(name, func(t *testing.T) {
			config := map[string]any{"types": []any{"closed"}, "closed-states": states}
			_, err := ParseWebhookSubscriptions(providerIssueOn("azure-devops-issue", config), "azure")
			require.ErrorContains(t, err, "closed-states")
		})
	}
	config := map[string]any{"types": []any{"closed", "reopened"}, "closed-states": []any{"Done", "Closed", "Done"}}
	subscriptions, err := ParseWebhookSubscriptions(providerIssueOn("azure-devops-issue", config), "azure")
	require.NoError(t, err)
	assert.Equal(t, []string{"Closed", "Done"}, subscriptions[0].ActivityNormalization.ClosedStates)
	assert.Equal(t, []string{"workitem.updated"}, subscriptions[0].NativeEvents)

	subscriptions, err = ParseWebhookSubscriptions(providerIssueOn("azure-devops-issue", map[string]any{}), "azure")
	require.NoError(t, err)
	assert.NotContains(t, subscriptions[0].Types, "closed")
	assert.NotContains(t, subscriptions[0].Types, "reopened")
	assert.Contains(t, subscriptions[0].Types, "labeled")
	assert.Contains(t, subscriptions[0].Types, "restored")
	for _, alias := range []string{"jira-issue", "linear-issue"} {
		_, err := ParseWebhookSubscriptions(providerIssueOn(alias, map[string]any{"closed-states": "Done"}), "invalid")
		require.Error(t, err)
	}
}

func TestProviderIssuePayloadFilters(t *testing.T) {
	for alias, test := range map[string]struct {
		filters map[string]any
		path    []string
	}{
		"jira-issue":         {map[string]any{"labels": []any{"bug", "urgent"}}, []string{"issue", "fields", "labels", "*"}},
		"linear-issue":       {map[string]any{"labels": "label-uuid"}, []string{"data", "labelIds", "*"}},
		"azure-devops-issue": {map[string]any{"state": map[string]any{"to": "Done"}}, []string{"resource", "fields", "System.State", "newValue"}},
	} {
		t.Run(alias, func(t *testing.T) {
			subscriptions, err := ParseWebhookSubscriptions(providerIssueOn(alias, map[string]any{"types": []any{"edited"}, "filters": test.filters}), "provider")
			require.NoError(t, err)
			require.Len(t, subscriptions[0].Conditions, 1)
			assert.Equal(t, test.path, subscriptions[0].Conditions[0].Path)
			assert.Equal(t, "string", subscriptions[0].Conditions[0].ValueType)
		})
	}
}

func TestCompileProviderIssueManifest(t *testing.T) {
	for alias, event := range providerIssueTriggers {
		t.Run(alias, func(t *testing.T) {
			source := webhookWorkflow("on:\n  webhook-gateway: true\n  workflow_dispatch:\n  bots: [cao-gateway]\n  " + alias + ":\n    types: [opened, labeled]\n")
			compiler := newWebhookCompiler()
			path := filepath.Join(t.TempDir(), alias+".md")
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			data, err := compiler.ParseWorkflowString(source, path)
			require.NoError(t, err)
			assert.Empty(t, ComputeGitHubAppManifestWebhookPermissions(providerIssueOn(alias, map[string]any{})), "provider events do not add GitHub App permissions")
			var triggers map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(data.On), &triggers))
			on := triggers["on"].(map[string]any)
			assert.NotContains(t, on, alias)
			assert.NotContains(t, on, "webhook-gateway")
			assert.Contains(t, on, "workflow_dispatch")
			assert.Contains(t, on, "repository_dispatch")
			require.NoError(t, compiler.CompileWorkflow(path))
			lock, err := os.ReadFile(stringutil.MarkdownToLockFile(path))
			require.NoError(t, err)
			output := string(lock)
			assert.Contains(t, output, "check_membership.cjs")
			assert.Contains(t, output, "validate_webhook.cjs")
			manifest, err := ExtractGHAWManifestFromLockFile(output)
			require.NoError(t, err)
			require.NotNil(t, manifest)
			require.Len(t, manifest.WebhookSubscriptions, 1)
			assert.Equal(t, event, manifest.WebhookSubscriptions[0].Event)
			assert.Equal(t, data.WebhookSubscriptions, manifest.WebhookSubscriptions)
		})
	}
}

func TestProviderIssueAuthorizationAndGates(t *testing.T) {
	for alias := range providerIssueTriggers {
		for name, test := range map[string]struct {
			enabled bool
			flag    string
			roles   string
			want    string
		}{
			"repository opt-in": {false, "  webhook-gateway: true\n", "", `{"webhook-gateway":true}`},
			"workflow opt-in":   {true, "", "", "webhook-gateway"},
			"false workflow":    {true, "  webhook-gateway: false\n", "", "webhook-gateway"},
			"weak role":         {true, "  webhook-gateway: true\n", "  roles: [read]\n", "write access"},
			"mixed weak role":   {true, "  webhook-gateway: true\n", "  roles: [write, triage]\n", "write access"},
		} {
			t.Run(alias+"/"+name, func(t *testing.T) {
				compiler := newWebhookCompiler()
				compiler.repoConfig.WebhookGateway = test.enabled
				source := webhookWorkflow("strict: false\non:\n  workflow_dispatch:\n" + test.flag + test.roles + "  " + alias + ":\n    types: [opened]\n")
				_, err := compiler.ParseWorkflowString(source, "provider-gate.md")
				require.ErrorContains(t, err, test.want)
			})
		}
	}
}

func TestProviderIssueCatalogNativeMappings(t *testing.T) {
	catalog, err := WebhookCatalog()
	require.NoError(t, err)
	for alias, event := range providerIssueTriggers {
		t.Run(alias, func(t *testing.T) {
			definition := catalog[event]
			require.NotEmpty(t, definition.Provider)
			assert.True(t, strings.HasPrefix(definition.Description, "Experimental. "), "every provider catalog description must mark its trigger experimental")
			assert.Len(t, definition.NativeEvents, len(definition.Actions))
			for _, activity := range definition.Actions {
				require.NotEmpty(t, definition.NativeEvents[activity], "every activity must identify a real provider event")
			}
		})
	}
}
