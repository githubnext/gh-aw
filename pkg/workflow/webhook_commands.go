package workflow

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var gatewaySlashCommandNamePattern = regexp.MustCompile(`^(\*|[^/*][^/*]*\*?)$`)

// WebhookCommandFilter declares receiver-side command matching and consumption.
type WebhookCommandFilter struct {
	Policy      string   `json:"policy"`
	Names       []string `json:"names"`
	Events      []string `json:"events"`
	Strategy    string   `json:"strategy"`
	RemoveLabel *bool    `json:"remove_label,omitempty"`
}

func parseGatewayCommandSubscriptions(config map[string]any, workflowID string, repos, orgs []string) ([]WebhookSubscription, error) {
	var subscriptions []WebhookSubscription
	for _, alias := range []string{"slash-command", "label-command"} {
		if raw, exists := config[alias]; exists {
			subscription, err := parseGatewayCommandSubscription(raw, alias, workflowID)
			if err != nil {
				return nil, err
			}
			subscription.Repos = slices.Clone(repos)
			subscription.Orgs = slices.Clone(orgs)
			subscriptions = append(subscriptions, subscription)
		}
	}
	return subscriptions, nil
}

func parseGatewayCommandSubscription(raw any, alias, workflowID string) (WebhookSubscription, error) {
	config, err := gatewayCommandConfig(raw, alias)
	if err != nil {
		return WebhookSubscription{}, err
	}
	filter, err := parseGatewayCommandFilter(config, alias, workflowID)
	if err != nil {
		return WebhookSubscription{}, err
	}
	var types, nativeEvents []string
	for _, identifier := range filter.Events {
		nativeEvents = append(nativeEvents, GetActualGitHubEventName(identifier))
		if alias == "label-command" {
			types = append(types, "labeled")
		} else {
			types = append(types, GetCommentEventByIdentifier(identifier).Types...)
		}
	}
	slices.Sort(types)
	slices.Sort(nativeEvents)
	subscription := newWebhookSubscription(workflowID, strings.ReplaceAll(alias, "-", "_"), slices.Compact(types), []WebhookCondition{})
	subscription.NativeEvents = slices.Compact(nativeEvents)
	subscription.CommandFilter = filter
	return subscription, nil
}

func gatewayCommandConfig(raw any, alias string) (map[string]any, error) {
	path := "on.gateway-dispatch." + alias
	label := alias == "label-command"
	config := map[string]any{}
	switch value := raw.(type) {
	case nil:
		if label {
			return nil, fmt.Errorf("%s must be a string or command configuration object", path)
		}
	case string:
		config["name"] = value
	case map[string]any:
		config = value
	default:
		return nil, fmt.Errorf("%s must use the same string/object configuration as on.%s", path, strings.ReplaceAll(alias, "-", "_"))
	}
	for _, key := range sortedWebhookKeys(config) {
		allowed := key == "name" || key == "events" || key == "strategy" || (label && (key == "names" || key == "remove_label"))
		if !allowed {
			return nil, fmt.Errorf("unsupported %s property %q", path, key)
		}
	}
	return config, nil
}

func gatewayCommandValues(raw any, path string, limit int) ([]string, error) {
	switch value := raw.(type) {
	case []any:
		if len(value) > limit {
			return nil, fmt.Errorf("%s accepts at most %d values", path, limit)
		}
	case []string:
		if len(value) > limit {
			return nil, fmt.Errorf("%s accepts at most %d values", path, limit)
		}
	}
	return webhookStringValues(raw, path)
}

func gatewayCommandNames(config map[string]any, path, workflowID string, label bool) ([]string, error) {
	var names []string
	for _, key := range []string{"name", "names"} {
		if rawNames, exists := config[key]; exists {
			values, err := gatewayCommandValues(rawNames, path+"."+key, 25)
			if err != nil {
				return nil, err
			}
			names = append(names, values...)
		}
	}
	if len(names) == 0 {
		names = []string{strings.TrimSuffix(filepath.Base(workflowID), ".md")}
	}
	for _, name := range names {
		if strings.Contains(name, "${{") || (!label && !gatewaySlashCommandNamePattern.MatchString(name)) {
			return nil, fmt.Errorf("%s requires literal names without a leading slash; slash wildcards must be '*' or a trailing '*'", path)
		}
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}

func gatewayCommandEvents(config map[string]any, path string, label bool) ([]string, error) {
	allowedEvents := GetCommentEventNames(GetAllCommentEvents())
	limit := 25
	if label {
		allowedEvents = slices.Clone(labelCommandSupportedEvents)
		limit = 3
	}
	events := slices.Clone(allowedEvents)
	if rawEvents, exists := config["events"]; exists {
		values, err := gatewayCommandValues(rawEvents, path+".events", limit)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			if value != "*" && !slices.Contains(allowedEvents, value) {
				return nil, fmt.Errorf("unsupported %s.events value %q; supported events: %s, *", path, value, strings.Join(allowedEvents, ", "))
			}
		}
		if !slices.Contains(values, "*") {
			events = values
		}
	}
	slices.Sort(events)
	return events, nil
}

func parseGatewayCommandFilter(config map[string]any, alias, workflowID string) (*WebhookCommandFilter, error) {
	path := "on.gateway-dispatch." + alias
	label := alias == "label-command"
	names, err := gatewayCommandNames(config, path, workflowID, label)
	if err != nil {
		return nil, err
	}
	events, err := gatewayCommandEvents(config, path, label)
	if err != nil {
		return nil, err
	}
	strategy := "inline"
	if rawStrategy, exists := config["strategy"]; exists {
		value, ok := rawStrategy.(string)
		allowed := value == "inline" || (!label && value == "centralized") || (label && value == "decentralized")
		if !ok || !allowed {
			return nil, fmt.Errorf("%s.strategy must use a supported native command strategy", path)
		}
		strategy = value
	}
	policy := "slash-command/v1"
	if label {
		policy = "label-command/v1"
	}
	filter := &WebhookCommandFilter{Policy: policy, Names: names, Events: events, Strategy: strategy}
	if label {
		remove := true
		if rawRemove, exists := config["remove_label"]; exists {
			var ok bool
			remove, ok = rawRemove.(bool)
			if !ok || !remove {
				return nil, fmt.Errorf("%s.remove_label must be true; gateway label commands do not support disabling label consumption", path)
			}
		}
		filter.RemoveLabel = &remove
	}
	return filter, nil
}
