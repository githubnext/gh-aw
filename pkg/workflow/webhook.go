package workflow

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// webhookCatalogJSON is the compiler's authoritative, payload-only filter catalog.
//
//go:embed data/webhook_events.json
var webhookCatalogJSON []byte

// webhookTriggerAliases maps first-class frontmatter triggers to wire event names.
// Projects aliases always refer to Projects v2, never Projects (classic).
var webhookTriggerAliases = map[string]string{
	"projects-item": "projects_v2_item",
}

var securityAlertEvents = map[string]string{
	"code-scanning":   "code_scanning_alert",
	"dependabot":      "dependabot_alert",
	"secret-scanning": "secret_scanning_alert",
}

// WebhookFilter describes an allowlisted path in the original webhook payload.
// A "*" path segment traverses array elements, never object keys.
type WebhookFilter struct {
	Path        []string `json:"path"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Actions     []string `json:"actions,omitempty"`
	Enum        []any    `json:"enum,omitempty"`
}

// WebhookEvent describes a webhook not available as a native Actions trigger.
type WebhookEvent struct {
	Actions      []string                 `json:"actions"`
	Description  string                   `json:"description"`
	Source       string                   `json:"source"`
	Availability []string                 `json:"availability"`
	Filters      map[string]WebhookFilter `json:"filters"`
}

// WebhookCondition uses strict typed equality: AND between conditions, OR between
// values and array elements. Missing and null payload values never match.
type WebhookCondition struct {
	Filter    string   `json:"filter"`
	Path      []string `json:"path"`
	Operator  string   `json:"operator"`
	ValueType string   `json:"value_type"`
	Values    []any    `json:"values"`
}

// WebhookContext maps the inline envelope into the standard GitHub context.
type WebhookContext struct {
	Envelope   string `json:"envelope"`
	Event      string `json:"event"`
	Payload    string `json:"payload"`
	DeliveryID string `json:"delivery_id"`
}

// WebhookDispatch is the complete routing and trust contract for a subscription.
// Repository is resolved against the repository containing the compiled workflow,
// not the repository in the incoming webhook payload.
type WebhookDispatch struct {
	Repository      string         `json:"repository"`
	Method          string         `json:"method"`
	EventType       string         `json:"event_type"`
	Workflow        string         `json:"workflow"`
	Context         WebhookContext `json:"context"`
	MaxPayloadBytes int            `json:"max_payload_bytes"`
	MaxRequestBytes int            `json:"max_request_bytes"`
	TrustedActors   []string       `json:"trusted_actors"`
}

// WebhookSubscription is a versioned, self-contained CAO subscription.
type WebhookSubscription struct {
	SchemaVersion int                `json:"schema_version"`
	Event         string             `json:"event"`
	Types         []string           `json:"types"`
	Conditions    []WebhookCondition `json:"conditions"`
	Dispatch      WebhookDispatch    `json:"dispatch"`
}

// WebhookCatalog returns an independent copy of the embedded event catalog.
func WebhookCatalog() (map[string]WebhookEvent, error) {
	var catalog map[string]WebhookEvent
	if err := json.Unmarshal(webhookCatalogJSON, &catalog); err != nil {
		return nil, fmt.Errorf("invalid embedded webhook catalog: %w", err)
	}
	return catalog, nil
}

// WebhookDispatchEventType scopes repository_dispatch to exactly one workflow.
func WebhookDispatchEventType(workflowID string) string {
	digest := sha256.Sum256([]byte(workflowID))
	return fmt.Sprintf("gh-aw-webhook-%x", digest[:12])
}

// ParseWebhookSubscriptions validates and normalizes webhook triggers without
// fetching or hydrating payload fields. Catalog paths are emitted for CAO.
func ParseWebhookSubscriptions(on any, workflowID string) ([]WebhookSubscription, error) {
	onMap, ok := on.(map[string]any)
	if !ok {
		for _, name := range rawOnSectionTriggerNames(on) {
			if _, alias := webhookTriggerAliases[name]; alias || name == "security-alert" {
				return nil, fmt.Errorf("on.%s requires mapping syntax and on.webhook-auth: {actors: [trusted-gateway-login]}", name)
			}
		}
		return nil, nil
	}
	events, paths, err := webhookTriggerConfigs(onMap)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		if _, hasAuth := onMap["webhook-auth"]; hasAuth {
			return nil, errors.New("on.webhook-auth requires on.webhook or a first-class webhook trigger")
		}
		return nil, nil
	}
	actors, err := parseWebhookActors(onMap["webhook-auth"])
	if err != nil {
		return nil, err
	}
	catalog, err := WebhookCatalog()
	if err != nil {
		return nil, err
	}
	subscriptions := make([]WebhookSubscription, 0, len(events))
	for _, name := range sortedWebhookKeys(events) {
		path := paths[name]
		definition, known := catalog[name]
		if !known {
			return nil, fmt.Errorf("unsupported on.webhook event %q; use a native Actions trigger where available; supported webhook events: %s", name, strings.Join(sortedWebhookKeys(catalog), ", "))
		}
		config, ok := events[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be an object with optional types and filters", path)
		}
		for _, key := range sortedWebhookKeys(config) {
			if key != "types" && key != "filters" {
				return nil, fmt.Errorf("unsupported %s property %q; use types or filters", path, key)
			}
		}
		types, err := parseWebhookTypes(config, path, definition.Actions)
		if err != nil {
			return nil, err
		}
		conditions, err := parseWebhookConditions(config, path, types, definition)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, newWebhookSubscription(workflowID, name, types, actors, conditions))
	}
	return subscriptions, nil
}

func webhookTriggerConfigs(onMap map[string]any) (map[string]any, map[string]string, error) {
	events := map[string]any{}
	paths := map[string]string{}
	if raw, exists := onMap["webhook"]; exists {
		configured, ok := raw.(map[string]any)
		if !ok || len(configured) == 0 {
			return nil, nil, errors.New("on.webhook must be a non-empty mapping of webhook event names to subscriptions")
		}
		events = maps.Clone(configured)
		for event := range events {
			paths[event] = "on.webhook." + event
		}
	}
	for _, alias := range sortedWebhookKeys(webhookTriggerAliases) {
		raw, exists := onMap[alias]
		if !exists {
			continue
		}
		event := webhookTriggerAliases[alias]
		if _, duplicate := events[event]; duplicate {
			return nil, nil, fmt.Errorf("on.%s and on.webhook.%s subscribe to the same event; use only one spelling", alias, event)
		}
		if raw == nil {
			raw = map[string]any{}
		}
		events[event] = raw
		paths[event] = "on." + alias
	}
	if raw, exists := onMap["security-alert"]; exists {
		config, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, errors.New("on.security-alert must be an object with a required type and optional types and filters")
		}
		event, err := securityAlertEvent(config)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := events[event]; duplicate {
			return nil, nil, fmt.Errorf("on.security-alert and on.webhook.%s subscribe to the same event; use only one spelling", event)
		}
		config = maps.Clone(config)
		delete(config, "type")
		events[event] = config
		paths[event] = "on.security-alert"
	}
	return events, paths, nil
}

func securityAlertEvent(config map[string]any) (string, error) {
	kind, ok := config["type"].(string)
	if ok {
		if event, supported := securityAlertEvents[kind]; supported {
			return event, nil
		}
	}
	return "", errors.New("on.security-alert.type must be one of: code-scanning, dependabot, secret-scanning")
}

func newWebhookSubscription(workflowID, event string, types, actors []string, conditions []WebhookCondition) WebhookSubscription {
	return WebhookSubscription{
		SchemaVersion: 1, Event: event, Types: types, Conditions: conditions,
		Dispatch: WebhookDispatch{
			Repository: "${{ github.repository }}", Method: "repository_dispatch",
			EventType: WebhookDispatchEventType(workflowID),
			Workflow:  constants.WorkflowsDirSlash + workflowID + ".lock.yml",
			Context: WebhookContext{
				Envelope: "github.event.client_payload", Event: "github.event.client_payload.event",
				Payload: "github.event.client_payload.payload", DeliveryID: "github.event.client_payload.delivery_id",
			},
			MaxPayloadBytes: 65535, MaxRequestBytes: 65535, TrustedActors: slices.Clone(actors),
		},
	}
}

func parseWebhookActors(raw any) ([]string, error) {
	auth, ok := raw.(map[string]any)
	if !ok || len(auth) != 1 {
		return nil, errors.New("webhook triggers require on.webhook-auth: {actors: [trusted-gateway-login]}; dispatch payloads do not authenticate their sender")
	}
	value, ok := auth["actors"]
	if !ok {
		return nil, errors.New("on.webhook-auth only supports actors")
	}
	actors, err := webhookStringList(value)
	if err != nil || len(actors) == 0 {
		return nil, errors.New("on.webhook-auth.actors must be a non-empty list of exact GitHub actor logins")
	}
	for _, actor := range actors {
		for _, char := range actor {
			allowed := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '[' || char == ']'
			if !allowed {
				return nil, fmt.Errorf("invalid on.webhook-auth actor %q; use an exact GitHub login, not a pattern or expression", actor)
			}
		}
		if actor == "" || len(actor) > 100 {
			return nil, errors.New("invalid on.webhook-auth actor: logins must contain 1 to 100 characters")
		}
	}
	sort.Strings(actors)
	return slices.Compact(actors), nil
}

func parseWebhookTypes(config map[string]any, path string, supported []string) ([]string, error) {
	value, exists := config["types"]
	if !exists {
		result := slices.Clone(supported)
		sort.Strings(result)
		if result == nil {
			result = []string{}
		}
		return result, nil
	}
	types, err := webhookStringList(value)
	if err != nil || len(types) == 0 {
		return nil, fmt.Errorf("%s.types must be a non-empty list of supported activity types", path)
	}
	for _, activity := range types {
		if !slices.Contains(supported, activity) {
			return nil, fmt.Errorf("unsupported activity type %q for %s; supported types: %s", activity, path, strings.Join(supported, ", "))
		}
	}
	sort.Strings(types)
	return slices.Compact(types), nil
}

func parseWebhookConditions(config map[string]any, path string, types []string, definition WebhookEvent) ([]WebhookCondition, error) {
	conditions := []WebhookCondition{}
	raw, exists := config["filters"]
	if !exists {
		return conditions, nil
	}
	filters, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s.filters must be an object", path)
	}
	flat := map[string]any{}
	if err := flattenWebhookFilters("", filters, flat); err != nil {
		return nil, err
	}
	for _, name := range sortedWebhookKeys(flat) {
		spec, supported := definition.Filters[name]
		if !supported {
			return nil, fmt.Errorf("unsupported filter %q for %s; only fields present in the delivered payload can be filtered (labels and item status may require unavailable hydration); supported filters: %s", name, path, strings.Join(sortedWebhookKeys(definition.Filters), ", "))
		}
		if len(spec.Actions) > 0 && !slices.ContainsFunc(types, func(activity string) bool { return slices.Contains(spec.Actions, activity) }) {
			return nil, fmt.Errorf("filter %q for %s requires one of these activity types: %s", name, path, strings.Join(spec.Actions, ", "))
		}
		values, err := webhookFilterValues(flat[name], spec)
		if err != nil {
			return nil, fmt.Errorf("invalid filter %q for %s: %w", name, path, err)
		}
		conditions = append(conditions, WebhookCondition{
			Filter: name, Path: slices.Clone(spec.Path), Operator: "equals", ValueType: spec.Type, Values: values,
		})
	}
	return conditions, nil
}

func flattenWebhookFilters(prefix string, input map[string]any, output map[string]any) error {
	for _, key := range sortedWebhookKeys(input) {
		if key == "" || strings.Contains(key, ".") {
			return errors.New("webhook filter keys must be non-empty names; express nested filters as YAML objects, not dotted keys")
		}
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		if nested, ok := input[key].(map[string]any); ok {
			if len(nested) == 0 {
				return fmt.Errorf("webhook filter %q must not be an empty object", name)
			}
			if err := flattenWebhookFilters(name, nested, output); err != nil {
				return err
			}
		} else {
			output[name] = input[key]
		}
	}
	return nil
}

func webhookFilterValues(raw any, spec WebhookFilter) ([]any, error) {
	var values []any
	switch value := raw.(type) {
	case []any:
		values = slices.Clone(value)
	case []string:
		for _, item := range value {
			values = append(values, item)
		}
	default:
		values = []any{value}
	}
	if len(values) == 0 {
		return nil, errors.New("accepted values must not be empty")
	}
	for i, value := range values {
		valid := false
		switch spec.Type {
		case "string":
			text, isString := value.(string)
			valid = isString
			if isString && strings.Contains(text, "${{") {
				return nil, errors.New("filter values must be literals, not GitHub Actions expressions")
			}
		case "boolean":
			_, valid = value.(bool)
		case "integer":
			switch number := value.(type) {
			case int:
				valid = number >= -9007199254740991 && number <= 9007199254740991
			case int64:
				valid = number >= -9007199254740991 && number <= 9007199254740991
			case uint64:
				valid = number <= 9007199254740991
			case float64:
				valid = !math.IsNaN(number) && !math.IsInf(number, 0) && math.Trunc(number) == number && math.Abs(number) <= 9007199254740991
			}
		}
		if !valid {
			return nil, fmt.Errorf("expected %s or a non-empty list of %s values (integers must be JSON-safe)", spec.Type, spec.Type)
		}
		if len(spec.Enum) > 0 && !slices.ContainsFunc(spec.Enum, func(allowed any) bool { return fmt.Sprint(allowed) == fmt.Sprint(value) }) {
			return nil, fmt.Errorf("value %v is not one of %v", value, spec.Enum)
		}
		values[i] = value
	}
	// Every value has the same validated scalar type.
	slices.SortFunc(values, func(a, b any) int {
		return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
	})
	return slices.CompactFunc(values, func(a, b any) bool {
		return fmt.Sprint(a) == fmt.Sprint(b)
	}), nil
}

func webhookStringList(raw any) ([]string, error) {
	var result []string
	switch value := raw.(type) {
	case []string:
		return slices.Clone(value), nil
	case []any:
		for _, item := range value {
			text, ok := item.(string)
			if !ok || text == "" {
				return nil, errors.New("expected non-empty strings")
			}
			result = append(result, text)
		}
	default:
		return nil, errors.New("expected a list of strings")
	}
	return result, nil
}

func sortedWebhookKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
