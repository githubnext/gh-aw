package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var genericWebhookTypePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,99}$`)
var genericWebhookFilterKeyPattern = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*|\*)$`)

func parseGenericWebhookSubscriptions(raw any, workflowID string) ([]WebhookSubscription, error) {
	configs, ok := raw.([]any)
	if !ok || len(configs) == 0 {
		return nil, errors.New("on.gateway-dispatch.webhooks must be a non-empty array of objects with type and optional filters")
	}
	subscriptions := make([]WebhookSubscription, 0, len(configs))
	seen := map[string]struct{}{}
	for index, config := range configs {
		subscription, err := parseGenericWebhookSubscription(config, workflowID, fmt.Sprintf("on.gateway-dispatch.webhooks[%d]", index))
		if err != nil {
			return nil, err
		}
		for _, eventType := range subscription.Types {
			if _, exists := seen[eventType]; exists {
				return nil, fmt.Errorf("on.gateway-dispatch.webhooks[%d].type duplicates another event identifier; each type must be unique", index)
			}
			seen[eventType] = struct{}{}
		}
		subscriptions = append(subscriptions, subscription)
	}
	slices.SortFunc(subscriptions, func(a, b WebhookSubscription) int { return slices.Compare(a.Types, b.Types) })
	return subscriptions, nil
}

func parseGenericWebhookSubscription(raw any, workflowID, path string) (WebhookSubscription, error) {
	config, ok := raw.(map[string]any)
	if !ok {
		return WebhookSubscription{}, fmt.Errorf("%s must be an object with a required literal type and optional filters", path)
	}
	for _, key := range sortedWebhookKeys(config) {
		if key != "type" && key != "filters" {
			return WebhookSubscription{}, fmt.Errorf("unsupported %s property %q; use type and declarative filters", path, key)
		}
	}
	eventType, ok := config["type"].(string)
	if !ok || !genericWebhookTypePattern.MatchString(eventType) {
		return WebhookSubscription{}, fmt.Errorf("%s.type must be a literal event identifier of 1-100 letters, digits, '.', '_', ':', '/', or '-', starting with a letter or digit", path)
	}
	definition := WebhookEvent{Filters: map[string]WebhookFilter{}}
	if rawFilters, exists := config["filters"]; exists {
		filters, ok := rawFilters.(map[string]any)
		if !ok || len(filters) == 0 {
			return WebhookSubscription{}, fmt.Errorf("%s.filters must be a non-empty object of declarative equality filters", path)
		}
		flat := map[string]any{}
		if err := flattenWebhookFilters("", filters, flat); err != nil {
			return WebhookSubscription{}, err
		}
		for _, name := range sortedWebhookKeys(flat) {
			filter, err := genericWebhookFilter(name, flat[name])
			if err != nil {
				return WebhookSubscription{}, fmt.Errorf("%s.filters: %w", path, err)
			}
			definition.Filters[name] = filter
		}
	}
	conditions, err := parseWebhookConditions(config, path, nil, definition)
	if err != nil {
		return WebhookSubscription{}, err
	}
	return newWebhookSubscription(workflowID, "webhook", []string{eventType}, conditions), nil
}

func genericWebhookFilter(name string, raw any) (WebhookFilter, error) {
	path := strings.Split(name, ".")
	for _, key := range path {
		if !genericWebhookFilterKeyPattern.MatchString(key) || key == "__proto__" || key == "prototype" || key == "constructor" {
			return WebhookFilter{}, fmt.Errorf("filter %q must use simple property names or '*' array traversal, without prototype properties", name)
		}
	}
	value := raw
	switch values := raw.(type) {
	case []any:
		if len(values) > 0 {
			value = values[0]
		}
	case []string:
		if len(values) > 0 {
			value = values[0]
		}
	}
	filter := WebhookFilter{Path: path}
	switch value.(type) {
	case string:
		filter.Type = "string"
	case bool:
		filter.Type = "boolean"
	case int, int64, uint64, float64:
		filter.Type = "integer"
	default:
		return WebhookFilter{}, fmt.Errorf("filter %q requires a string, boolean, JSON-safe integer, or a non-empty homogeneous list", name)
	}
	return filter, nil
}
