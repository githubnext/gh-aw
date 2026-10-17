package workflow

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
)

func webhookTestOn(event string, config map[string]any) map[string]any {
	if event == "projects_v2_item" {
		return map[string]any{"projects-item": config}
	}
	for kind, name := range securityAlertEvents {
		if event == name {
			security := map[string]any{"type": kind}
			maps.Copy(security, config)
			return map[string]any{"security-alert": security}
		}
	}
	panic("unsupported test webhook event: " + event)
}

func TestWebhookSubscriptionValidation(t *testing.T) {
	tests := []struct {
		name   string
		event  string
		config map[string]any
		want   string
	}{
		{"unknown action", "projects_v2_item", map[string]any{"types": []any{"labeled"}}, "unsupported activity type"},
		{"empty types", "projects_v2_item", map[string]any{"types": []any{}}, "non-empty list"},
		{"scalar types", "projects_v2_item", map[string]any{"types": "edited"}, "non-empty list"},
		{"unsupported property", "projects_v2_item", map[string]any{"filter": map[string]any{}}, "unsupported on.projects-item"},
		{"labels require hydration", "projects_v2_item", map[string]any{"filters": map[string]any{"labels": []any{"bug"}}}, "unsupported filter"},
		{"wrong identifier type", "projects_v2_item", map[string]any{"filters": map[string]any{"project_id": 123}}, "expected string"},
		{"expression is not literal", "projects_v2_item", map[string]any{"filters": map[string]any{"project_id": "${{ github.repository }}"}}, "must be literals"},
		{"null value", "projects_v2_item", map[string]any{"filters": map[string]any{"project_id": nil}}, "expected string"},
		{"empty values", "projects_v2_item", map[string]any{"filters": map[string]any{"project_id": []any{}}}, "must not be empty"},
		{"mixed types", "projects_v2_item", map[string]any{"filters": map[string]any{"project_id": []any{"PVT_1", true}}}, "expected string"},
		{"empty nested filter", "projects_v2_item", map[string]any{"filters": map[string]any{"field": map[string]any{}}}, "empty object"},
		{"dotted filter", "projects_v2_item", map[string]any{"filters": map[string]any{"field.name": "Status"}}, "nested filters"},
		{"scalar filters", "projects_v2_item", map[string]any{"filters": true}, "filters must be an object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWebhookSubscriptions(webhookTestOn(tt.event, tt.config), "project-agent")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestWebhookRejectsRemovedFrontmatter(t *testing.T) {
	for _, field := range []string{"webhook", "webhook-auth"} {
		on := webhookTestOn("projects_v2_item", map[string]any{})
		on[field] = map[string]any{}
		for _, input := range []any{on, map[string]any{field: nil}, field, []any{"push", field}} {
			if _, err := ParseWebhookSubscriptions(input, "test"); err == nil || !strings.Contains(err.Error(), "on."+field+" is not supported") {
				t.Fatalf("expected removed on.%s to fail, got %v", field, err)
			}
		}
	}
	if got, err := ParseWebhookSubscriptions(map[string]any{"push": nil}, "test"); err != nil || len(got) != 0 {
		t.Fatalf("native events must remain unaffected: %v, %v", got, err)
	}
}

func TestWebhookContractDeterministicAndComplete(t *testing.T) {
	on := webhookTestOn("projects_v2_item", map[string]any{
		"types": []any{"edited", "created", "edited"},
		"filters": map[string]any{
			"project_id": []any{"PVT_b", "PVT_a", "PVT_a"},
		},
	})
	on["security-alert"] = map[string]any{"type": "dependabot"}
	first, err := ParseWebhookSubscriptions(on, "project-agent")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseWebhookSubscriptions(on, "project-agent")
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(first)
	right, _ := json.Marshal(second)
	if string(left) != string(right) {
		t.Fatal("contract is not deterministic")
	}
	if len(first) != 2 || first[0].Event != "dependabot_alert" {
		t.Fatalf("subscriptions not sorted: %+v", first)
	}
	sub := first[1]
	if !slices.Equal(sub.Types, []string{"created", "edited"}) || len(sub.Conditions[0].Values) != 2 {
		t.Fatalf("values not canonicalized: %+v", sub)
	}
	if sub.SchemaVersion != 1 || sub.Dispatch.Repository != "${{ github.repository }}" ||
		sub.Dispatch.Method != "repository_dispatch" || sub.Dispatch.MaxPayloadBytes != 65535 || sub.Dispatch.MaxRequestBytes != 65535 ||
		sub.Dispatch.Context.Payload != "github.event.client_payload.payload" ||
		sub.Dispatch.Workflow != ".github/workflows/project-agent.lock.yml" {
		t.Fatalf("incomplete dispatch contract: %+v", sub)
	}
	if sub.Dispatch.EventType != first[0].Dispatch.EventType ||
		sub.Dispatch.EventType == WebhookDispatchEventType("another-workflow") ||
		len(sub.Dispatch.EventType) > 100 {
		t.Fatal("dispatch identity does not scope subscriptions to one workflow")
	}
}

func TestWebhookCatalogIntegrity(t *testing.T) {
	catalog, err := WebhookCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"projects_v2", "projects_v2_item", "projects_v2_status_update", "issue_dependencies", "issue_relates_to", "sub_issues"} {
		if _, ok := catalog[required]; !ok {
			t.Errorf("missing documented webhook event %s", required)
		}
	}
	for _, native := range []string{"issues", "push", "pull_request", "discussion", "label", "release", "repository_dispatch", "workflow_dispatch"} {
		if _, ok := catalog[native]; ok {
			t.Errorf("unnecessarily duplicated native trigger %s", native)
		}
	}
	for name, event := range catalog {
		if !strings.HasPrefix(event.Source, "https://docs.github.com/") || event.Description == "" || len(event.Availability) == 0 {
			t.Errorf("missing provenance/availability for %s", name)
		}
		for filter, spec := range event.Filters {
			if len(spec.Path) == 0 || spec.Description == "" || !slices.Contains([]string{"string", "integer", "boolean"}, spec.Type) {
				t.Errorf("invalid filter schema %s.%s: %+v", name, filter, spec)
			}
			for _, action := range spec.Actions {
				if !slices.Contains(event.Actions, action) {
					t.Errorf("filter %s.%s references unknown action %s", name, filter, action)
				}
			}
		}
	}
	// Callers cannot mutate the authoritative catalog.
	delete(catalog, "projects_v2")
	again, _ := WebhookCatalog()
	if _, ok := again["projects_v2"]; !ok {
		t.Fatal("catalog mutation escaped its caller")
	}
}

func TestWebhookProjectTransitionConformance(t *testing.T) {
	subscriptions, err := ParseWebhookSubscriptions(webhookTestOn("projects_v2_item", map[string]any{
		"types": []any{"edited"},
		"filters": map[string]any{
			"project_id": "PVT_example",
			"field": map[string]any{
				"name": "Status",
				"from": map[string]any{"name": "Todo"},
				"to":   map[string]any{"name": []any{"Ready", "Done"}},
			},
		},
	}), "project-agent")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("testdata/webhook-project-item-edited.json")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(map[string]any)
		match  bool
	}{
		{"matching transition", func(map[string]any) {}, true},
		{"wrong action", func(p map[string]any) { p["action"] = "created" }, false},
		{"missing action", func(p map[string]any) { delete(p, "action") }, false},
		{"different project", func(p map[string]any) { p["projects_v2_item"].(map[string]any)["project_node_id"] = "PVT_other" }, false},
		{"missing project", func(p map[string]any) { delete(p, "projects_v2_item") }, false},
		{"missing change", func(p map[string]any) { delete(p, "changes") }, false},
		{"missing current value", func(p map[string]any) { delete(p["changes"].(map[string]any)["field_value"].(map[string]any), "to") }, false},
		{"cleared field", func(p map[string]any) { p["changes"].(map[string]any)["field_value"].(map[string]any)["to"] = nil }, false},
		{"not a select value", func(p map[string]any) { p["changes"].(map[string]any)["field_value"].(map[string]any)["to"] = "Ready" }, false},
		{"case sensitive", func(p map[string]any) {
			p["changes"].(map[string]any)["field_value"].(map[string]any)["to"].(map[string]any)["name"] = "ready"
		}, false},
		{"OR accepted value", func(p map[string]any) {
			p["changes"].(map[string]any)["field_value"].(map[string]any)["to"].(map[string]any)["name"] = "Done"
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload map[string]any
			if err := json.Unmarshal(fixture, &payload); err != nil {
				t.Fatal(err)
			}
			tt.change(payload)
			if got := matchWebhookContract(subscriptions[0], payload); got != tt.match {
				t.Fatalf("contract match = %v, want %v", got, tt.match)
			}
		})
	}
	_, err = ParseWebhookSubscriptions(webhookTestOn("projects_v2_item", map[string]any{
		"types": []any{"created"},
		"filters": map[string]any{
			"field": map[string]any{"to": map[string]any{"name": "Ready"}},
		},
	}), "project-agent")
	if err == nil || !strings.Contains(err.Error(), "requires one of") {
		t.Fatalf("transition on creation must fail: %v", err)
	}
}

func TestWebhookLabelArrayConformance(t *testing.T) {
	subscriptions, err := ParseWebhookSubscriptions(webhookTestOn("code_scanning_alert", map[string]any{
		"types": []any{"created"},
		"filters": map[string]any{
			"assignee": []any{"octocat", "hubot"},
		},
	}), "dependency-agent")
	if err != nil {
		t.Fatal(err)
	}
	sub := subscriptions[0]
	for _, tt := range []struct {
		payload string
		match   bool
	}{
		{`{"action":"created","alert":{"number":42,"assignees":[{"login":"other"},{"login":"octocat"}]}}`, true},
		{`{"action":"created","alert":{"number":42,"assignees":[{"login":"other"}]}}`, false},
		{`{"action":"created","alert":{"assignees":[]}}`, false},
		{`{"action":"created","alert":{}}`, false},
		{`{"action":"created","alert":{"assignees":null}}`, false},
		{`{"action":"created","alert":{"assignees":{"x":{"login":"octocat"}}}}`, false},
		{`{"action":"created","alert":{"assignees":[{"login":true}]}}`, false},
	} {
		var payload map[string]any
		if err := json.Unmarshal([]byte(tt.payload), &payload); err != nil {
			t.Fatal(err)
		}
		if got := matchWebhookContract(sub, payload); got != tt.match {
			t.Errorf("label contract match for %s = %v, want %v", tt.payload, got, tt.match)
		}
	}
}

func TestWebhookTypedValues(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		spec   WebhookFilter
		failed bool
	}{
		{"boolean", true, WebhookFilter{Type: "boolean"}, false},
		{"boolean string", "true", WebhookFilter{Type: "boolean"}, true},
		{"integer", 42, WebhookFilter{Type: "integer"}, false},
		{"integer string", "42", WebhookFilter{Type: "integer"}, true},
		{"fraction", 1.5, WebhookFilter{Type: "integer"}, true},
		{"unsafe JSON integer", int64(9007199254740992), WebhookFilter{Type: "integer"}, true},
		{"enum valid", "Issue", WebhookFilter{Type: "string", Enum: []any{"Issue", "PullRequest", "DraftIssue"}}, false},
		{"enum invalid", "issue", WebhookFilter{Type: "string", Enum: []any{"Issue", "PullRequest", "DraftIssue"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := webhookFilterValues(tt.value, tt.spec)
			if (err != nil) != tt.failed {
				t.Fatalf("value validation error = %v, expected failure %v", err, tt.failed)
			}
		})
	}
}

func TestWebhookFalseDoesNotMatchMissingOrNull(t *testing.T) {
	subscriptions, err := ParseWebhookSubscriptions(webhookTestOn("secret_scanning_alert", map[string]any{
		"filters": map[string]any{"publicly_leaked": false},
	}), "private-project-agent")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		payload string
		match   bool
	}{
		{`{"action":"created","alert":{"publicly_leaked":false}}`, true},
		{`{"action":"created","alert":{"publicly_leaked":true}}`, false},
		{`{"action":"created","alert":{"publicly_leaked":"false"}}`, false},
		{`{"action":"created","alert":{"publicly_leaked":null}}`, false},
		{`{"action":"created","alert":{}}`, false},
		{`{"action":"created"}`, false},
	} {
		var payload map[string]any
		if err := json.Unmarshal([]byte(tt.payload), &payload); err != nil {
			t.Fatal(err)
		}
		if got := matchWebhookContract(subscriptions[0], payload); got != tt.match {
			t.Errorf("boolean contract match = %v, want %v for %s", got, tt.match, tt.payload)
		}
	}
}

// matchWebhookContract is a test-only reference consumer of the public contract.
// Actual pre-dispatch filtering belongs to CAO, not the gh-aw runtime.
func matchWebhookContract(sub WebhookSubscription, payload map[string]any) bool {
	if len(sub.Types) > 0 {
		action, ok := payload["action"].(string)
		if !ok || !slices.Contains(sub.Types, action) {
			return false
		}
	}
	for _, condition := range sub.Conditions {
		if !matchWebhookPath(payload, condition.Path, condition) {
			return false
		}
	}
	return true
}

func matchWebhookPath(value any, path []string, condition WebhookCondition) bool {
	if value == nil {
		return false
	}
	if len(path) == 0 {
		for _, accepted := range condition.Values {
			if condition.ValueType == "integer" {
				a, aerr := json.Marshal(accepted)
				b, berr := json.Marshal(value)
				if aerr == nil && berr == nil && string(a) == string(b) {
					return true
				}
			} else if value == accepted {
				return true
			}
		}
		return false
	}
	if path[0] == "*" {
		array, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range array {
			if matchWebhookPath(item, path[1:], condition) {
				return true
			}
		}
		return false
	}
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	next, exists := object[path[0]]
	return exists && matchWebhookPath(next, path[1:], condition)
}
