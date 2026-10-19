package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func compileSubscriptions(t *testing.T, triggers map[string]any) []workflow.WebhookSubscription {
	t.Helper()
	subscriptions, err := workflow.ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": triggers}, "notify")
	require.NoError(t, err)
	// Match the JSON representation received from compiled discovery metadata.
	raw, err := json.Marshal(subscriptions)
	require.NoError(t, err)
	require.NoError(t, parseJSON(raw, &subscriptions))
	return subscriptions
}

func object(t *testing.T, raw string) map[string]any {
	t.Helper()
	var result map[string]any
	require.NoError(t, parseJSON([]byte(raw), &result))
	return result
}

func githubFixture(t *testing.T, family, action string) map[string]any {
	t.Helper()
	p := object(t, `{
		"repository":{"id":10,"full_name":"source/repo","owner":{"id":42,"type":"Organization"}},
		"organization":{"id":42,"login":"source"},
		"sender":{"login":"human"}
	}`)
	p["action"] = action
	d := notificationFamilies[family]
	if d.Parent != "" {
		p[d.Parent] = object(t, `{"number":7,"body":"/review private arguments","labels":[{"name":"triage"},{"name":"unselected"}]}`)
	}
	if family == "pull_request_comment" {
		p["issue"].(map[string]any)["pull_request"] = map[string]any{}
	}
	resource := object(t, `{"id":9,"body":"/review private arguments","user":{"login":"reviewer"}}`)
	number := "7"
	if family == "commit_comment" {
		number = strings.Repeat("a", 40)
		resource["commit_id"] = number
	}
	raw := "https://github.example/source/repo/" + d.Path + "/" + number
	if d.Anchor != "" {
		raw += "#" + d.Anchor + "9"
	}
	resource["html_url"] = raw
	if d.Anchor == "" {
		p[d.Parent].(map[string]any)["html_url"] = raw
	} else if family == "pull_request_review_thread" {
		p["thread"] = map[string]any{"comments": []any{resource}}
	} else {
		p[d.Resource] = resource
	}
	return p
}

func sourceFixture() Source {
	return Source{Approved: true, Provider: "github", SecretEnv: "FIXTURE_GITHUB_SECRET",
		Repository: "source/repo", RepositoryID: "10", OwnerID: "42", Organization: "source", OrganizationID: "42"}
}

func TestAllGitHubNotificationFamilies(t *testing.T) {
	c := &Config{ServerURL: "https://github.example", APIURL: "https://api.github.example"}
	catalog, err := workflow.WebhookCatalog()
	require.NoError(t, err)
	aliases := map[string]string{"issues": "issue", "pull_request": "pull-request", "discussion": "discussion",
		"issue_comment": "issue-comment", "pull_request_comment": "pull-request-comment", "pull_request_review": "pull-request-review",
		"pull_request_review_comment": "pull-request-review-comment", "pull_request_review_thread": "pull-request-review-thread",
		"discussion_comment": "discussion-comment", "commit_comment": "commit-comment"}
	for family, def := range notificationFamilies {
		t.Run(family, func(t *testing.T) {
			sub := compileSubscriptions(t, map[string]any{aliases[family]: nil})[0]
			for _, action := range catalog[family].Actions {
				p := githubFixture(t, family, action)
				projected, err := project(c, sourceFixture(), sub, def.Native, p)
				require.NoError(t, err)
				require.NotNil(t, projected)
				require.Equal(t, []string{action}, projected.Type)
				raw, err := json.Marshal(projected)
				require.NoError(t, err)
				require.NotContains(t, string(raw), "private arguments")
				require.NotContains(t, string(raw), "unselected")
				resource := at(p, def.Resource)
				if def.Anchor == "" {
					resource = at(p, def.Parent)
				}
				if family == "pull_request_review_thread" {
					resource = at(p, "thread", "comments", "0")
				}
				resource.(map[string]any)["html_url"] = projected.URL + "?injected=1"
				_, err = project(c, sourceFixture(), sub, def.Native, p)
				require.ErrorContains(t, err, "canonical URL")
			}
		})
	}
}

func TestOriginalIdentityAndMinimalFilters(t *testing.T) {
	c := &Config{ServerURL: "https://github.example"}
	sub := compileSubscriptions(t, map[string]any{"issue": map[string]any{
		"types": []any{"opened"}, "filters": map[string]any{"labels": []any{"triage", "security"}, "number": 7},
	}})[0]
	p := githubFixture(t, "issues", "opened")
	payload, err := project(c, sourceFixture(), sub, "issues", p)
	require.NoError(t, err)
	require.Equal(t, []any{"triage"}, payload.Filters["labels"])
	require.Equal(t, []any{json.Number("7")}, payload.Filters["number"])
	p["issue"].(map[string]any)["pull_request"] = nil
	_, err = project(c, sourceFixture(), sub, "issues", p)
	require.ErrorContains(t, err, "conflict")
	delete(p["issue"].(map[string]any), "pull_request")
	p["repository"].(map[string]any)["id"] = json.Number("11")
	_, err = project(c, sourceFixture(), sub, "issues", p)
	require.ErrorContains(t, err, "not enrolled")
}

func TestProjectsCompleteTransitionsAndEnrollment(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"projects-items": map[string]any{
		"project_id": "PVT_one", "status-transitions": []any{
			map[string]any{"from": "Todo", "to": "Ready"},
			map[string]any{"from": "Review", "to": "Done"},
		},
	}})[0]
	p := object(t, `{"action":"edited","organization":{"id":42,"login":"source"},
		"projects_v2_item":{"id":5,"project_node_id":"PVT_one"},
		"changes":{"field_value":{"field_name":"Status","field_type":"single_select","from":{"name":"Todo"},"to":{"name":"Ready"}}}}`)
	c := &Config{APIURL: "https://git.example/api/v3"}
	s := sourceFixture()
	s.Projects = map[string]Project{"PVT_one": {Kind: "orgs", Owner: "source", Number: "3"}}
	payload, err := project(c, s, sub, "projects_v2_item", p)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Equal(t, "https://git.example/api/v3/orgs/source/projectsV2/3/items/5", payload.URL)
	require.Equal(t, []any{"Todo"}, payload.Filters["field.from.name"])
	p["changes"].(map[string]any)["field_value"].(map[string]any)["to"] = map[string]any{"name": "Done"}
	payload, err = project(c, s, sub, "projects_v2_item", p)
	require.NoError(t, err)
	require.Nil(t, payload, "cross-combined alternatives must not match")
	delete(s.Projects, "PVT_one")
	_, err = project(c, s, sub, "projects_v2_item", p)
	require.ErrorContains(t, err, "not enrolled")
}

func TestSecurityAndOrganizationSelectors(t *testing.T) {
	subs := compileSubscriptions(t, map[string]any{
		"repos": []any{"source/*"}, "orgs": []any{"sou*"},
		"security-alerts": map[string]any{"sources": []any{"dependabot"}, "types": []any{"created"}, "severity": "high"},
	})
	p := githubFixture(t, "issues", "created")
	p["alert"] = object(t, `{"number":8,"url":"https://git.example/api/v3/repos/source/repo/dependabot/alerts/8",
		"security_advisory":{"severity":"high","description":"private"}}`)
	c := &Config{APIURL: "https://git.example/api/v3"}
	payload, err := project(c, sourceFixture(), subs[0], "dependabot_alert", p)
	require.NoError(t, err)
	require.Equal(t, map[string][]any{"severity": {"high"}}, payload.Filters)
	s := sourceFixture()
	s.Organization = ""
	payload, err = project(c, s, subs[0], "dependabot_alert", p)
	require.NoError(t, err)
	require.Nil(t, payload)
}

func TestSlashPositionAndProjection(t *testing.T) {
	sub := compileSubscriptions(t, map[string]any{"slash-command": map[string]any{
		"name": []any{"review", "fix*"}, "events": []any{"issue_comment"},
	}})[0]
	c := &Config{ServerURL: "https://github.example"}
	for _, body := range []string{"/review arguments", "/fix", "/fix-this\narguments"} {
		p := githubFixture(t, "issue_comment", "created")
		p["comment"].(map[string]any)["body"] = body
		payload, err := project(c, sourceFixture(), sub, "issue_comment", p)
		require.NoError(t, err)
		require.NotNil(t, payload)
		require.Len(t, payload.Filters, 2)
	}
	for _, body := range []string{" /review", "prefix\n/review", "/Review", "/review/argument", "/review!"} {
		p := githubFixture(t, "issue_comment", "created")
		p["comment"].(map[string]any)["body"] = body
		payload, err := project(c, sourceFixture(), sub, "issue_comment", p)
		require.NoError(t, err)
		require.Nil(t, payload, body)
	}
}

func TestCatalogContentSelectorsProjectOnlyExactConfiguredLiterals(t *testing.T) {
	catalog, err := workflow.WebhookCatalog()
	require.NoError(t, err)
	subs := compileSubscriptions(t, map[string]any{"projects-items": map[string]any{
		"project_id": "PVT_one", "types": []any{"edited"}, "filters": map[string]any{"body": map[string]any{"from": "private draft body"}},
	}})
	require.NoError(t, validateSubscription("notify", subs[0], catalog))
	p := object(t, `{"projects_v2_item":{"project_node_id":"PVT_one"},
		"changes":{"body":{"from":"private draft body","to":"unselected new body"}}}`)
	filters, ok := matchedFilters(subs[0], p)
	require.True(t, ok)
	require.Equal(t, []any{"private draft body"}, filters["body.from"])
	require.NotContains(t, filters, "body.to")
	p = object(t, `{"projects_v2_item":{"project_node_id":"PVT_one"},
		"changes":{"body":{"from":"private draft body plus extra text"}}}`)
	_, ok = matchedFilters(subs[0], p)
	require.False(t, ok)
	subs = compileSubscriptions(t, map[string]any{"azure-devops-issue": map[string]any{
		"project_id": "project", "types": []any{"edited"}, "tags": map[string]any{"from": "triage; private-label"},
	}})
	require.NoError(t, validateSubscription("notify", subs[0], catalog))
	p = object(t, `{"resourceContainers":{"project":{"id":"project"}},
		"resource":{"fields":{"System.Tags":{"oldValue":"triage; private-label","newValue":"unselected"}}}}`)
	filters, ok = matchedFilters(subs[0], p)
	require.True(t, ok)
	require.Equal(t, []any{"triage; private-label"}, filters["tags.from"])
	require.NotContains(t, filters, "tags.to")
	p["resource"].(map[string]any)["fields"].(map[string]any)["System.Tags"].(map[string]any)["oldValue"] = "triage; private-label; extra"
	_, ok = matchedFilters(subs[0], p)
	require.False(t, ok)
}

func TestLabelCommandUnsupportedRatherThanUnfenced(t *testing.T) {
	catalog, err := workflow.WebhookCatalog()
	require.NoError(t, err)
	sub := compileSubscriptions(t, map[string]any{"label-command": "review"})[0]
	require.ErrorContains(t, validateSubscription("notify", sub, catalog), "source-write fencing")
}

func TestCanonicalURLConservatism(t *testing.T) {
	for _, raw := range []string{"https://github.example/x", "https://github.example/x#issuecomment-1"} {
		require.True(t, canonicalResource(raw), raw)
	}
	for _, raw := range []string{"https://github.example:443/x", "https://github.example:0443/x", "https://GITHUB.example/x",
		"HTTPS://github.example/x", "https://user@github.example/x", "https://github.example/a/./x",
		"https://github.example/a/../x", "https://github.example/%2e/x", "https://github.example/%2e%2e/x",
		"https://0x7f000001/x", "https://127.1/x", "https://127.0.0.01/x", "https://github.example:08080/x",
		"https://github.example/x?", "https://github.example/x#", "https://github.example/a%2fb", "https://github.example/é"} {
		require.False(t, canonicalResource(raw), raw)
	}
}
