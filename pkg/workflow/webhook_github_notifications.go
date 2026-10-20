package workflow

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/github/gh-aw/actions/setup"
)

var githubNotificationTriggers = map[string]string{
	"issue":                       "issues",
	"pull-request":                "pull_request",
	"discussion":                  "discussion",
	"issue-comment":               "issue_comment",
	"pull-request-comment":        "pull_request_comment",
	"pull-request-review":         "pull_request_review",
	"pull-request-review-comment": "pull_request_review_comment",
	"pull-request-review-thread":  "pull_request_review_thread",
	"discussion-comment":          "discussion_comment",
	"commit-comment":              "commit_comment",
}

func addGitHubNotificationCatalog(catalog map[string]WebhookEvent) error {
	var definitions struct {
		CommonFilters map[string]WebhookFilter `json:"common_filters"`
		Events        map[string]struct {
			WebhookEvent
			NativeEvent string `json:"native_event"`
		} `json:"events"`
	}
	if err := json.Unmarshal(setup.GitHubNotificationWebhooks, &definitions); err != nil {
		return fmt.Errorf("invalid embedded GitHub notification catalog: %w", err)
	}
	for name, definition := range definitions.Events {
		event := definition.WebhookEvent
		event.Filters = maps.Clone(definition.Filters)
		maps.Copy(event.Filters, definitions.CommonFilters)
		event.NativeEvents = map[string]string{}
		for _, action := range event.Actions {
			event.NativeEvents[action] = definition.NativeEvent
		}
		event.Source = "https://docs.github.com/en/webhooks/webhook-events-and-payloads#" + definition.NativeEvent
		event.Availability = []string{"repository", "organization", "app"}
		catalog[name] = event
	}
	return nil
}

func addGitHubNotificationConfigs(on, events map[string]any, paths map[string]string) {
	for trigger, event := range githubNotificationTriggers {
		if raw, exists := on[trigger]; exists {
			if raw == nil {
				raw = map[string]any{}
			}
			events[event] = raw
			paths[event] = "on.gateway-dispatch." + trigger
		}
	}
}

func githubNotificationPermission(event string) PermissionScope {
	switch event {
	case "issues", "issue_comment":
		return PermissionIssues
	case "pull_request", "pull_request_comment", "pull_request_review", "pull_request_review_comment", "pull_request_review_thread":
		return PermissionPullRequests
	case "discussion", "discussion_comment":
		return PermissionDiscussions
	case "commit_comment":
		return PermissionContents
	default:
		return ""
	}
}

func addGatewaySubscriptionPermissions(permissions map[string]string, subscription WebhookSubscription) {
	add := func(event, level string) {
		if scope := githubNotificationPermission(event); scope != "" {
			key, supported := GitHubAppManifestPermissionKey(scope)
			if supported && permissions[key] != "write" {
				permissions[key] = level
			}
		}
	}
	if subscription.CommandFilter == nil {
		add(subscription.Event, "read")
		return
	}
	level := "read"
	if subscription.CommandFilter.RemoveLabel != nil && *subscription.CommandFilter.RemoveLabel {
		level = "write"
	}
	for _, event := range subscription.CommandFilter.Events {
		add(event, level)
	}
}
