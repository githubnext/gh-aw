package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow"
)

var (
	positiveID          = regexp.MustCompile(`^[1-9][0-9]*$`)
	repositoryName      = regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)
	digestPattern       = regexp.MustCompile(`^[a-f0-9]{64}$`)
	revisionPattern     = regexp.MustCompile(`^[a-f0-9]{40}$`)
	workflowIDPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	repoSelectorPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+/([A-Za-z0-9_.-]+|\*)$`)
	orgSelectorPattern  = regexp.MustCompile(`^[A-Za-z0-9*]([A-Za-z0-9*-]{0,37}[A-Za-z0-9*])?$`)
)

type Source struct {
	Approved       bool               `json:"approved"`
	Provider       string             `json:"provider"`
	SecretEnv      string             `json:"secret_env"`
	Repository     string             `json:"repository,omitempty"`
	RepositoryID   string             `json:"repository_id,omitempty"`
	OwnerID        string             `json:"owner_id,omitempty"`
	Organization   string             `json:"organization,omitempty"`
	OrganizationID string             `json:"organization_id,omitempty"`
	Origin         string             `json:"origin,omitempty"`
	Project        string             `json:"project,omitempty"`
	ProjectPath    string             `json:"project_path,omitempty"`
	Team           string             `json:"team,omitempty"`
	Workspace      string             `json:"workspace,omitempty"`
	CollectionID   string             `json:"collection_id,omitempty"`
	CollectionURL  string             `json:"collection_url,omitempty"`
	BasicUser      string             `json:"basic_user,omitempty"`
	Projects       map[string]Project `json:"projects,omitempty"`
}

type Project struct {
	Owner  string `json:"owner"`
	Kind   string `json:"kind"`
	Number string `json:"number"`
}

type Destination struct {
	Approved       bool   `json:"approved"`
	Repository     string `json:"repository"`
	RepositoryID   string `json:"repository_id"`
	InstallationID string `json:"installation_id"`
	AccountID      string `json:"account_id"`
	Revision       string `json:"revision"`
}

type Route struct {
	Destination    string                         `json:"destination"`
	WorkflowID     string                         `json:"workflow_id"`
	SourceIDs      []string                       `json:"source_ids"`
	ManifestSHA256 string                         `json:"manifest_sha256"`
	Subscriptions  []workflow.WebhookSubscription `json:"subscriptions"`
}

type Config struct {
	APIURL       string                 `json:"api_url"`
	ServerURL    string                 `json:"server_url"`
	AppID        string                 `json:"app_id"`
	BotID        string                 `json:"bot_id"`
	Sources      map[string]Source      `json:"sources"`
	Destinations map[string]Destination `json:"destinations"`
	Routes       map[string]Route       `json:"routes"`
	hash         string
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("cannot read protected configuration")
	}
	var c Config
	if err := parseJSON(data, &c); err != nil {
		return nil, fmt.Errorf("configuration: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	c.hash = digest(data)
	return &c, nil
}

func canonicalBase(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && canonicalResource(raw) && u.Host != "" && u.Host == strings.ToLower(u.Host) &&
		u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" &&
		!strings.HasSuffix(raw, "/") && !strings.Contains(u.Path, "..") &&
		u.RawPath == "" && u.String() == raw
}

func (c *Config) validate() error {
	if !canonicalBase(c.APIURL) || !canonicalBase(c.ServerURL) ||
		!positiveID.MatchString(c.AppID) || !positiveID.MatchString(c.BotID) ||
		c.AppID != os.Getenv("GH_AW_GATEWAY_APP_ID") ||
		c.BotID != os.Getenv("GH_AW_GATEWAY_APP_BOT_ID") {
		return errors.New("invalid central GitHub host or App/bot binding")
	}
	for _, s := range c.Sources {
		if err := validateSource(s); err != nil {
			return err
		}
	}
	for _, d := range c.Destinations {
		if !repositoryName.MatchString(d.Repository) || !positiveID.MatchString(d.RepositoryID) ||
			!positiveID.MatchString(d.InstallationID) || !positiveID.MatchString(d.AccountID) ||
			!revisionPattern.MatchString(d.Revision) {
			return errors.New("invalid destination enrollment or approved default-branch revision")
		}
	}
	catalog, err := workflow.WebhookCatalog()
	if err != nil {
		return errors.New("cannot load compiler filter catalog")
	}
	targets := map[string]struct{}{}
	for _, route := range c.Routes {
		target := route.Destination + "\x00" + route.WorkflowID
		if _, exists := targets[target]; exists {
			return errors.New("duplicate destination workflow route")
		}
		targets[target] = struct{}{}
		if err := c.validateRoute(route, catalog); err != nil {
			return err
		}
	}
	return nil
}

func validateSource(s Source) error {
	if s.SecretEnv == "" || os.Getenv(s.SecretEnv) == "" {
		return errors.New("source authentication secret is missing")
	}
	switch s.Provider {
	case "github":
		if !positiveID.MatchString(s.OwnerID) || s.Repository == "" && len(s.Projects) == 0 {
			return errors.New("GitHub source needs immutable owner and repository or project enrollment")
		}
		if s.Repository != "" && (!repositoryName.MatchString(s.Repository) || !positiveID.MatchString(s.RepositoryID) || !positiveID.MatchString(s.OwnerID)) {
			return errors.New("invalid GitHub source repository binding")
		}
		if s.Organization != "" && (!positiveID.MatchString(s.OrganizationID) || s.OwnerID != s.OrganizationID) {
			return errors.New("invalid independently approved organization binding")
		}
		for node, p := range s.Projects {
			if node == "" || !positiveID.MatchString(p.Number) || p.Owner == "" || (p.Kind != "orgs" && p.Kind != "users") ||
				(p.Kind == "orgs" && !strings.EqualFold(p.Owner, s.Organization)) ||
				(p.Kind == "users" && s.Organization != "") {
				return errors.New("invalid project enrollment")
			}
		}
	case "jira":
		if !canonicalBase(s.Origin) || s.Project == "" {
			return errors.New("invalid Jira source binding")
		}
	case "linear":
		if s.OrganizationID == "" || s.Workspace == "" {
			return errors.New("invalid Linear organization/workspace binding")
		}
	case "azure-devops":
		if s.CollectionID == "" || s.Project == "" || s.BasicUser == "" || !canonicalBase(s.CollectionURL) {
			return errors.New("invalid Azure collection/project binding")
		}
	default:
		return errors.New("unsupported source authentication provider")
	}
	return nil
}

func (c *Config) validateRoute(route Route, catalog map[string]workflow.WebhookEvent) error {
	if _, ok := c.Destinations[route.Destination]; !ok || len(route.SourceIDs) == 0 || len(route.Subscriptions) == 0 {
		return errors.New("route requires independent source and destination bindings")
	}
	for _, id := range route.SourceIDs {
		if _, ok := c.Sources[id]; !ok {
			return errors.New("route references unknown source")
		}
	}
	manifest, err := json.Marshal(route.Subscriptions)
	if err != nil || !digestPattern.MatchString(route.ManifestSHA256) || digest(manifest) != route.ManifestSHA256 {
		return errors.New("compiled subscription artifact digest is not approved")
	}
	seen := map[string]struct{}{}
	for _, sub := range route.Subscriptions {
		if _, exists := seen[sub.Event]; exists {
			return errors.New("ambiguous subscription family on one route")
		}
		seen[sub.Event] = struct{}{}
		if err := validateSubscription(route.WorkflowID, sub, catalog); err != nil {
			return err
		}
	}
	return nil
}

func validateSubscription(workflowID string, s workflow.WebhookSubscription, catalog map[string]workflow.WebhookEvent) error {
	if !workflowIDPattern.MatchString(workflowID) {
		return errors.New("unsupported workflow identifier")
	}
	d := s.Dispatch
	expectedContext := workflow.WebhookContext{
		Envelope: "github.event.client_payload", AWContext: "github.event.client_payload.aw_context",
		Event:      "github.event.client_payload.aw_context.webhook.event",
		Deliveries: "github.event.client_payload.aw_context.webhook.deliveries", Items: "aw_context.items",
	}
	if s.SchemaVersion != 2 || s.Conditions == nil || len(s.Types) == 0 ||
		d.EventType != workflow.WebhookDispatchEventType(workflowID) ||
		d.Repository != "${{ github.repository }}" || d.Method != "repository_dispatch" ||
		d.Workflow != constants.WorkflowsDirSlash+workflowID+".lock.yml" ||
		d.Context != expectedContext || d.Batch != (workflow.WebhookBatch{Mode: "same-event", DeduplicateBy: "delivery_id"}) ||
		d.MaxPayloadBytes != 65535 || d.MaxRequestBytes != 65535 ||
		d.PayloadProjection == nil || d.PayloadProjection.Policy != workflow.WebhookPayloadProjectionPolicy {
		return errors.New("unsupported compiled transport declaration")
	}
	if s.Event == "label_command" {
		return errors.New("label_command is nonactivating: source-write fencing is not implemented")
	}
	if s.Event == "slash_command" {
		return validateSlash(s)
	}
	if err := validateAlternatives(s); err != nil {
		return err
	}
	def, ok := catalog[s.Event]
	_, notification := notificationFamilies[s.Event]
	if !ok || (!notification && !slices.Contains([]string{"projects_v2_item", "code_scanning_alert", "dependabot_alert", "jira_issue", "linear_issue", "azure_devops_issue"}, s.Event)) {
		return errors.New("unsupported gateway event family")
	}
	if s.CommandFilter != nil || s.Provider != def.Provider || (s.AnyOf != nil && s.Event != "projects_v2_item") {
		return errors.New("invalid event-specific subscription metadata")
	}
	if err := validateActivities(s, def, notification); err != nil {
		return err
	}
	return validateConditions(s, def)
}

func validateAlternatives(s workflow.WebhookSubscription) error {
	if s.AnyOf != nil {
		if len(s.AnyOf) == 0 {
			return errors.New("empty condition alternatives")
		}
		for _, group := range s.AnyOf {
			if len(group) != 2 ||
				!slices.ContainsFunc(group, func(c workflow.WebhookCondition) bool { return c.Filter == "field.from.name" }) ||
				!slices.ContainsFunc(group, func(c workflow.WebhookCondition) bool { return c.Filter == "field.to.name" }) {
				return errors.New("project alternatives require complete Status transition pairs")
			}
		}
		for name, value := range map[string]string{"field.name": "Status", "field.type": "single_select"} {
			if !slices.ContainsFunc(s.Conditions, func(c workflow.WebhookCondition) bool {
				return c.Filter == name && len(c.Values) == 1 && slices.Contains(c.Values, any(value))
			}) {
				return errors.New("project Status transition evidence is incomplete")
			}
		}
	}
	return nil
}

func validateActivities(s workflow.WebhookSubscription, def workflow.WebhookEvent, notification bool) error {
	for _, typ := range s.Types {
		if !slices.Contains(def.Actions, typ) {
			return errors.New("unsupported subscription activity")
		}
	}
	if len(s.Types) != len(slices.Compact(slices.Sorted(slices.Values(s.Types)))) {
		return errors.New("duplicate subscription activity")
	}
	if notification {
		if s.ActivityNormalization != nil || !slices.Equal(s.NativeEvents, []string{notificationFamilies[s.Event].Native}) {
			return errors.New("invalid notification native registration")
		}
	} else if def.Provider != "" {
		if s.ActivityNormalization == nil || s.ActivityNormalization.Policy != def.Provider+"-issue/v1" {
			return errors.New("unsupported provider normalization policy")
		}
		if s.Event != "azure_devops_issue" && s.ActivityNormalization.ClosedStates != nil {
			return errors.New("terminal states are Azure-only")
		}
		expected := []string{}
		for _, typ := range s.Types {
			expected = append(expected, def.NativeEvents[typ])
		}
		slices.Sort(expected)
		if !slices.Equal(s.NativeEvents, slices.Compact(expected)) {
			return errors.New("incomplete provider native registration")
		}
		if s.Event == "azure_devops_issue" && (slices.Contains(s.Types, "closed") || slices.Contains(s.Types, "reopened")) && len(s.ActivityNormalization.ClosedStates) == 0 {
			return errors.New("azure lifecycle requires explicit terminal states")
		}
	} else if s.ActivityNormalization != nil || len(s.NativeEvents) > 0 {
		return errors.New("invalid GitHub resource metadata")
	}
	return nil
}

func validateConditions(s workflow.WebhookSubscription, def workflow.WebhookEvent) error {
	groups := append([][]workflow.WebhookCondition{s.Conditions}, s.AnyOf...)
	for _, group := range groups {
		for _, condition := range group {
			filter, ok := def.Filters[condition.Filter]
			if !ok || !slices.Equal(condition.Path, filter.Path) || condition.ValueType != filter.Type ||
				condition.Operator != "equals" || len(condition.Values) == 0 {
				return errors.New("condition does not match normative compiler catalog")
			}
			for _, typ := range s.Types {
				if len(filter.Actions) > 0 && !slices.Contains(filter.Actions, typ) {
					return errors.New("condition unavailable for selected activity")
				}
			}
			for _, value := range condition.Values {
				if !validScalar(value, filter.Type) || (len(filter.Enum) > 0 && !slices.ContainsFunc(filter.Enum, func(v any) bool { return equal(value, v) })) {
					return errors.New("invalid typed condition literal")
				}
			}
		}
	}
	required := map[string]string{"projects_v2_item": "project_id", "jira_issue": "project", "linear_issue": "organization-id", "azure_devops_issue": "project_id"}[s.Event]
	if required != "" && !slices.ContainsFunc(s.Conditions, func(c workflow.WebhookCondition) bool { return c.Filter == required }) {
		return errors.New("required source selector missing")
	}
	if def.Provider != "" && (s.Repos != nil || s.Orgs != nil) || s.Event == "projects_v2_item" && s.Repos != nil {
		return errors.New("unsupported shared source selector")
	}
	return validateSelectors(s)
}

func validateSelectors(s workflow.WebhookSubscription) error {
	for _, repo := range s.Repos {
		if !repoSelectorPattern.MatchString(repo) {
			return errors.New("invalid repository selector")
		}
	}
	for _, org := range s.Orgs {
		if !orgSelectorPattern.MatchString(org) {
			return errors.New("invalid organization selector")
		}
	}
	return nil
}

func validScalar(v any, typ string) bool {
	switch typ {
	case "string":
		s, ok := v.(string)
		return ok && !strings.Contains(s, "${{")
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "integer":
		n, ok := v.(json.Number)
		if !ok {
			return false
		}
		i, err := n.Int64()
		return err == nil && i >= -9007199254740991 && i <= 9007199254740991
	}
	return false
}
