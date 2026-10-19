package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJiraEvidenceAndScope(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"jira-issue": map[string]any{
		"project": "APP", "types": []any{"closed", "edited", "labeled", "unlabeled", "reopened"},
	}})[0]
	s := Source{Provider: "jira", Project: "APP", Origin: "https://fixture.atlassian.net"}
	p := object(t, `{"webhookEvent":"jira:issue_updated",
		"issue":{"id":"10001","self":"https://fixture.atlassian.net/rest/api/3/issue/10001",
		"fields":{"project":{"key":"APP"},"labels":["triage"],"description":"private"}},
		"changelog":{"items":[{"fieldtype":"jira","field":"resolution","from":null,"to":"1"},
		{"fieldtype":"jira","field":"labels","fromString":null,"toString":"triage"}]}}`)
	payload, err := project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"closed", "edited", "labeled"}, payload.Type)
	require.Equal(t, map[string][]any{"project": {"APP"}}, payload.Filters)
	sub.Types = []string{"closed"}
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"closed"}, payload.Type)
	delete(p, "changelog")
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Nil(t, payload)
	p["issue"].(map[string]any)["self"] = "https://attacker.example/rest/api/3/issue/10001"
	_, err = project(&Config{}, s, sub, "", p)
	require.Error(t, err)
}

func TestJiraAmbiguousLabelsAndMissingResolution(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"jira-issue": map[string]any{"project": "APP"}})[0]
	p := object(t, `{"webhookEvent":"jira:issue_updated",
		"issue":{"id":"1","self":"https://jira.example/rest/api/2/issue/1","fields":{"project":{"key":"APP"},"labels":["a","b"]}},
		"changelog":{"items":[{"fieldtype":"jira","field":"status","fromString":"Todo","toString":"Done"},
		{"fieldtype":"jira","field":"labels","fromString":"a","toString":"a,b"},
		{"fieldtype":"jira","field":"resolution","to":"1"}]}}`)
	payload, err := project(&Config{}, Source{Provider: "jira", Project: "APP", Origin: "https://jira.example"}, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"edited"}, payload.Type)
}

func TestLinearNativeTopLevelURLAndCompleteTerminalEvidence(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"linear-issue": map[string]any{
		"organization-id": "org", "team-id": "team",
	}})[0]
	s := Source{Provider: "linear", OrganizationID: "org", Team: "team", Workspace: "workspace"}
	p := object(t, `{"type":"Issue","action":"update","organizationId":"org",
		"url":"https://linear.app/workspace/issue/APP-42/example",
		"data":{"id":"immutable-uuid","identifier":"APP-42","teamId":"team",
		"completedAt":"2026-10-19T00:00:00Z","canceledAt":null,"labelIds":["new","new"]},
		"updatedFrom":{"completedAt":null,"labelIds":["old"]}}`)
	payload, err := project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"closed", "edited", "labeled", "unlabeled"}, payload.Type)
	require.Len(t, payload.Filters, 2)
	delete(p["data"].(map[string]any), "canceledAt")
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"edited", "labeled", "unlabeled"}, payload.Type)
	p["data"].(map[string]any)["url"] = "https://linear.app/different/issue/APP-42"
	_, err = project(&Config{}, s, sub, "", p)
	require.ErrorContains(t, err, "not enrolled")
	delete(p["data"].(map[string]any), "url")
	p["type"] = "IssueLabel"
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Nil(t, payload)
}

func TestAzureWorkItemIdentityAndSelectedTerminalProjection(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"azure-devops-issue": map[string]any{
		"project_id": "project", "closed-states": []any{"Done", "Closed"},
	}})[0]
	s := Source{Provider: "azure-devops", Project: "project", CollectionID: "collection", CollectionURL: "https://dev.azure.com/org"}
	p := object(t, `{"eventType":"workitem.updated","resourceVersion":"2.0",
		"resourceContainers":{"project":{"id":"project"},"collection":{"id":"collection"}},
		"resource":{"id":999,"workItemId":42,
			"url":"https://dev.azure.com/org/_apis/wit/workItems/42/updates/999",
			"revision":{"url":"https://dev.azure.com/org/_apis/wit/workItems/42"},
			"fields":{"System.State":{"oldValue":"Active","newValue":"Done"},
				"System.Tags":{"oldValue":"old; common","newValue":"new; common"}}}}`)
	payload, err := project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"closed", "edited", "labeled", "unlabeled"}, payload.Type)
	require.Equal(t, []any{"Done"}, payload.Filters["closed-states"])
	require.Equal(t, "https://dev.azure.com/org/_apis/wit/workItems/42", payload.URL)
	sub.Types = []string{"edited"}
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.NotContains(t, payload.Filters, "closed-states")
	delete(p["resource"].(map[string]any), "workItemId")
	_, err = project(&Config{}, s, sub, "", p)
	require.ErrorContains(t, err, "canonical work item URL")
}

func TestAzureTerminalToTerminalAndSnapshotEvidence(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"azure-devops-issue": map[string]any{
		"project_id": "p", "closed-states": []any{"Done", "Closed"},
	}})[0]
	s := Source{Provider: "azure-devops", Project: "p", CollectionID: "c", CollectionURL: "https://org.visualstudio.com/DefaultCollection"}
	p := object(t, `{"eventType":"workitem.updated","resourceVersion":"1.0",
		"resourceContainers":{"project":{"id":"p"},"collection":{"id":"c"}},
		"resource":{"id":42,"url":"https://org.visualstudio.com/DefaultCollection/_apis/wit/workitems/42",
		"fields":{"System.State":{"oldValue":"Done","newValue":"Closed"}}}}`)
	payload, err := project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"edited"}, payload.Type)
	state := p["resource"].(map[string]any)["fields"].(map[string]any)["System.State"].(map[string]any)
	state["newValue"] = "Active"
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"edited", "reopened"}, payload.Type)
	require.Equal(t, []any{"Done"}, payload.Filters["closed-states"])
	delete(state, "oldValue")
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, []string{"edited"}, payload.Type)
}

func TestAzureProtectedConstructionAndContradictoryNativeEvidence(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"azure-devops-issue": map[string]any{
		"project_id": "project", "types": []any{"edited"},
	}})[0]
	s := Source{Provider: "azure-devops", Project: "project", CollectionID: "collection", CollectionURL: "https://dev.azure.com/org"}
	raw := `{"eventType":"workitem.updated","resourceVersion":"2.0",
		"resourceContainers":{"project":{"id":"project"},"collection":{"id":"collection"}},
		"resource":{"id":999,"workItemId":42,"revision":{"id":42},
			"url":"https://dev.azure.com/org/_apis/wit/workItems/42/updates/999"}}`
	payload, err := project(&Config{}, s, sub, "", object(t, raw))
	require.NoError(t, err)
	require.Equal(t, "https://dev.azure.com/org/_apis/wit/workItems/42", payload.URL)
	for _, field := range []string{"url", "revision", "_links"} {
		t.Run(field, func(t *testing.T) {
			p := object(t, raw)
			resource := p["resource"].(map[string]any)
			switch field {
			case "url":
				resource["url"] = "https://attacker.example/org/_apis/wit/workItems/42/updates/999"
			case "revision":
				resource["revision"] = map[string]any{"id": 43}
			case "_links":
				resource["_links"] = map[string]any{"workItem": map[string]any{
					"href": "https://dev.azure.com/org/_apis/wit/workItems/43",
				}}
			}
			_, err := project(&Config{}, s, sub, "", p)
			require.Error(t, err)
		})
	}
	p := object(t, raw)
	delete(p["resource"].(map[string]any), "url")
	payload, err = project(&Config{}, s, sub, "", p)
	require.NoError(t, err)
	require.Equal(t, "https://dev.azure.com/org/_apis/wit/workItems/42", payload.URL)
}
