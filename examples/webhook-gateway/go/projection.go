package main

import (
	"errors"
	"net"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/github/gh-aw/pkg/workflow"
)

type notificationFamily struct {
	Native, Parent, Path, Anchor, Resource string
}

var notificationFamilies = map[string]notificationFamily{
	"issues":                      {"issues", "issue", "issues", "", "issue"},
	"pull_request":                {"pull_request", "pull_request", "pull", "", "pull_request"},
	"discussion":                  {"discussion", "discussion", "discussions", "", "discussion"},
	"issue_comment":               {"issue_comment", "issue", "issues", "issuecomment-", "comment"},
	"pull_request_comment":        {"issue_comment", "issue", "pull", "issuecomment-", "comment"},
	"pull_request_review":         {"pull_request_review", "pull_request", "pull", "pullrequestreview-", "review"},
	"pull_request_review_comment": {"pull_request_review_comment", "pull_request", "pull", "discussion_r", "comment"},
	"pull_request_review_thread":  {"pull_request_review_thread", "pull_request", "pull", "discussion_r", "thread"},
	"discussion_comment":          {"discussion_comment", "discussion", "discussions", "discussioncomment-", "comment"},
	"commit_comment":              {"commit_comment", "", "commit", "commitcomment-", "comment"},
}

var (
	slashNamePattern   = regexp.MustCompile(`^(\*|[A-Za-z0-9][A-Za-z0-9._-]*\*?)$`)
	slashTokenPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	commitPattern      = regexp.MustCompile(`^[a-f0-9]{40}$`)
	asciiHostPattern   = regexp.MustCompile(`^[a-z0-9.-]+$`)
	numericHostPattern = regexp.MustCompile(`^(0x[0-9a-f]+|[0-9]+)$`)
)

func joinURL(base string, segments ...string) string {
	return base + path.Join(append([]string{"/"}, segments...)...)
}

var slashActions = map[string][]string{
	"issues": {"opened", "edited", "reopened"}, "issue_comment": {"created", "edited"},
	"pull_request_comment": {"created", "edited"}, "pull_request": {"opened", "edited", "reopened"},
	"pull_request_review_comment": {"created", "edited"}, "discussion": {"created", "edited"},
	"discussion_comment": {"created", "edited"},
}

type Payload struct {
	URL     string           `json:"url"`
	Type    []string         `json:"type"`
	Filters map[string][]any `json:"filters,omitempty"`
}

func validateSlash(s workflow.WebhookSubscription) error {
	f := s.CommandFilter
	if f == nil || f.Policy != "slash-command/v1" || f.RemoveLabel != nil || len(f.Names) == 0 ||
		len(f.Events) == 0 || len(s.Conditions) != 0 || s.AnyOf != nil || s.Provider != "" ||
		s.ActivityNormalization != nil || !slices.Contains([]string{"inline", "centralized"}, f.Strategy) {
		return errors.New("invalid slash-command declaration")
	}
	if err := validateSelectors(s); err != nil {
		return err
	}
	native, types := []string{}, []string{}
	for _, event := range f.Events {
		actions, ok := slashActions[event]
		if !ok {
			return errors.New("unsupported slash-command source")
		}
		native = append(native, notificationFamilies[event].Native)
		types = append(types, actions...)
	}
	for _, name := range f.Names {
		if !slashNamePattern.MatchString(name) {
			return errors.New("unsupported slash-command name")
		}
	}
	slices.Sort(native)
	slices.Sort(types)
	if !slices.Equal(s.NativeEvents, slices.Compact(native)) || !slices.Equal(s.Types, slices.Compact(types)) {
		return errors.New("invalid slash-command native registration")
	}
	return nil
}

func canonicalResource(raw string) bool {
	for _, ch := range raw {
		if ch > 127 || ch < 33 {
			return false
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	for segment := range strings.SplitSeq(u.Path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port > 65535 || port == 443 || strconv.Itoa(port) != u.Port() {
			return false
		}
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if ip.String() != host {
			return false
		}
	} else {
		last := host
		if i := strings.LastIndexByte(host, '.'); i >= 0 {
			last = host[i+1:]
		}
		if !asciiHostPattern.MatchString(host) || numericHostPattern.MatchString(last) {
			return false
		}
	}
	return err == nil && u.Scheme == "https" && u.Host != "" &&
		u.Host == strings.ToLower(u.Host) && !strings.HasSuffix(u.Host, ":443") &&
		u.User == nil && u.RawQuery == "" && !u.ForceQuery &&
		!strings.ContainsAny(raw, " \r\n\t\\") && u.String() == raw &&
		!strings.Contains(u.Path, "/../") && !strings.Contains(u.Path, "/./") &&
		!strings.Contains(strings.ToLower(u.RawPath), "%2f")
}

func githubSource(s Source, p map[string]any, projects bool) bool {
	if projects {
		if s.Organization != "" {
			return decimal(at(p, "organization", "id")) == s.OrganizationID &&
				strings.EqualFold(text(at(p, "organization", "login")), s.Organization)
		}
		return s.OwnerID != ""
	}
	if s.Repository == "" || decimal(at(p, "repository", "id")) != s.RepositoryID ||
		!strings.EqualFold(text(at(p, "repository", "full_name")), s.Repository) ||
		decimal(at(p, "repository", "owner", "id")) != s.OwnerID {
		return false
	}
	if s.Organization != "" {
		return text(at(p, "repository", "owner", "type")) == "Organization" &&
			decimal(at(p, "organization", "id")) == s.OrganizationID &&
			strings.EqualFold(text(at(p, "organization", "login")), s.Organization)
	}
	return text(at(p, "repository", "owner", "type")) == "User"
}

func selectedSource(s Source, sub workflow.WebhookSubscription) bool {
	if sub.Repos != nil && !slices.ContainsFunc(sub.Repos, func(pattern string) bool {
		pattern = strings.ToLower(pattern)
		repo := strings.ToLower(s.Repository)
		return pattern == repo || strings.HasSuffix(pattern, "/*") && strings.HasPrefix(repo, strings.TrimSuffix(pattern, "*"))
	}) {
		return false
	}
	if sub.Orgs != nil && (s.Organization == "" || !slices.ContainsFunc(sub.Orgs, func(pattern string) bool {
		return wildcardMatch(strings.ToLower(pattern), strings.ToLower(s.Organization))
	})) {
		return false
	}
	return true
}

func wildcardMatch(pattern, value string) bool {
	prefix, rest, wildcard := strings.Cut(pattern, "*")
	if !wildcard {
		return pattern == value
	}
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	value = strings.TrimPrefix(value, prefix)
	for {
		part, remaining, found := strings.Cut(rest, "*")
		if !found {
			return strings.HasSuffix(value, part)
		}
		i := strings.Index(value, part)
		if i < 0 {
			return false
		}
		value, rest = value[i+len(part):], remaining
	}
}

func notificationURL(c *Config, family string, p map[string]any) (string, error) {
	def := notificationFamilies[family]
	parent := at(p, def.Parent)
	if def.Parent != "" {
		if _, ok := parent.(map[string]any); !ok {
			return "", errors.New("missing original parent identity")
		}
	}
	if def.Parent == "issue" {
		issue, ok := parent.(map[string]any)
		if !ok {
			return "", errors.New("missing original issue object")
		}
		evidence, exists := issue["pull_request"]
		if family == "pull_request_comment" {
			if _, ok := evidence.(map[string]any); !exists || !ok {
				return "", errors.New("missing original PR evidence")
			}
		} else if exists {
			return "", errors.New("issue and PR source evidence conflict")
		}
	}
	number := decimal(at(parent, "number"))
	resource := at(p, def.Resource)
	if family == "pull_request_review_thread" {
		resource = at(p, "thread", "comments", "0")
	}
	if family == "commit_comment" {
		number = text(at(resource, "commit_id"))
		if !commitPattern.MatchString(number) {
			return "", errors.New("invalid original commit identity")
		}
	} else if number == "" {
		return "", errors.New("invalid original parent number")
	}
	repo := text(at(p, "repository", "full_name"))
	if !repositoryName.MatchString(repo) || strings.HasSuffix(repo, "/.") || strings.HasSuffix(repo, "/..") {
		return "", errors.New("invalid original repository")
	}
	expected := joinURL(c.ServerURL, repo, def.Path, number)
	if def.Anchor != "" {
		id := decimal(at(resource, "id"))
		if id == "" {
			return "", errors.New("missing original comment or review identity")
		}
		expected += "#" + def.Anchor + id
	}
	raw := text(at(resource, "html_url"))
	if !canonicalResource(raw) || raw != expected {
		return "", errors.New("canonical URL is not bound to original source identities")
	}
	return raw, nil
}

func matchedFilters(sub workflow.WebhookSubscription, p map[string]any) (map[string][]any, bool) {
	groups := sub.AnyOf
	if groups == nil {
		groups = [][]workflow.WebhookCondition{{}}
	}
	for _, group := range groups {
		filters := map[string][]any{}
		matches := true
		for _, condition := range append(slices.Clone(sub.Conditions), group...) {
			var actual []any
			for _, value := range valuesAt(p, condition.Path) {
				if validScalar(value, condition.ValueType) && slices.ContainsFunc(condition.Values, func(v any) bool { return equal(value, v) }) {
					actual = append(actual, value)
				}
			}
			actual = distinct(actual)
			if prior, ok := filters[condition.Filter]; ok {
				actual = slices.DeleteFunc(actual, func(v any) bool { return !slices.ContainsFunc(prior, func(w any) bool { return equal(v, w) }) })
			}
			if len(actual) == 0 {
				matches = false
				break
			}
			filters[condition.Filter] = actual
		}
		if matches {
			return filters, true
		}
	}
	return nil, false
}

func project(c *Config, s Source, sub workflow.WebhookSubscription, native string, p map[string]any) (*Payload, error) {
	if !selectedSource(s, sub) {
		return nil, nil
	}
	if sub.Event == "slash_command" {
		return projectSlash(c, s, sub, native, p)
	}
	var raw string
	var activities []string
	var terminal string
	var err error
	if s.Provider == "github" {
		raw, activities, err = githubProjection(c, s, sub, native, p)
	} else {
		if s.Provider != sub.Provider {
			return nil, nil
		}
		raw, activities, terminal, err = providerProjection(s, sub, p)
	}
	if err != nil {
		return nil, err
	}
	activities = slices.DeleteFunc(activities, func(v string) bool { return !slices.Contains(sub.Types, v) })
	slices.Sort(activities)
	activities = slices.Compact(activities)
	if len(activities) == 0 {
		return nil, nil
	}
	if slices.Contains(activities, "closed") && slices.Contains(activities, "reopened") {
		return nil, errors.New("contradictory lifecycle evidence")
	}
	if !canonicalResource(raw) {
		return nil, errors.New("noncanonical resource URL")
	}
	filters, ok := matchedFilters(sub, p)
	if !ok {
		return nil, nil
	}
	if terminal != "" && (slices.Contains(activities, "closed") || slices.Contains(activities, "reopened")) {
		filters["closed-states"] = []any{terminal}
	}
	return &Payload{URL: raw, Type: activities, Filters: filters}, nil
}

func githubProjection(c *Config, s Source, sub workflow.WebhookSubscription, native string, p map[string]any) (string, []string, error) {
	var raw string
	var err error
	if !githubSource(s, p, sub.Event == "projects_v2_item") {
		return "", nil, errors.New("authenticated GitHub source is not enrolled")
	}
	if def, ok := notificationFamilies[sub.Event]; ok {
		if native != def.Native {
			return "", nil, nil
		}
		raw, err = notificationURL(c, sub.Event, p)
	} else if native != sub.Event {
		return "", nil, nil
	} else if sub.Event == "projects_v2_item" {
		pj, ok := s.Projects[text(at(p, "projects_v2_item", "project_node_id"))]
		id := decimal(at(p, "projects_v2_item", "id"))
		if !ok || id == "" {
			return "", nil, errors.New("project URL binding is not enrolled")
		}
		raw = joinURL(c.APIURL, pj.Kind, pj.Owner, "projectsV2", pj.Number, "items", id)
	} else {
		id := decimal(at(p, "alert", "number"))
		path := "dependabot"
		if sub.Event == "code_scanning_alert" {
			path = "code-scanning"
		}
		raw = text(at(p, "alert", "url"))
		if id == "" || raw != joinURL(c.APIURL, "repos", text(at(p, "repository", "full_name")), path, "alerts", id) {
			return "", nil, errors.New("security alert URL is not bound to original identity")
		}
	}
	return raw, []string{text(p["action"])}, err
}

func projectSlash(c *Config, s Source, sub workflow.WebhookSubscription, native string, p map[string]any) (*Payload, error) {
	if s.Provider != "github" || !githubSource(s, p, false) {
		return nil, errors.New("command source is not enrolled")
	}
	for _, family := range sub.CommandFilter.Events {
		if notificationFamilies[family].Native != native || !slices.Contains(slashActions[family], text(p["action"])) {
			continue
		}
		// A native issue_comment has exactly one semantic family.
		if native == "issue_comment" {
			_, isPR := at(p, "issue", "pull_request").(map[string]any)
			if isPR != (family == "pull_request_comment") {
				continue
			}
		}
		raw, err := notificationURL(c, family, p)
		if err != nil {
			return nil, err
		}
		body := text(at(p, notificationFamilies[family].Resource, "body"))
		token := slashToken(body)
		if token == "" || !slices.ContainsFunc(sub.CommandFilter.Names, func(name string) bool {
			return name == token || strings.HasSuffix(name, "*") && strings.HasPrefix(token, strings.TrimSuffix(name, "*"))
		}) {
			continue
		}
		return &Payload{URL: raw, Type: []string{text(p["action"])}, Filters: map[string][]any{"event": {family}, "command": {token}}}, nil
	}
	return nil, nil
}

func slashToken(body string) string {
	if !strings.HasPrefix(body, "/") {
		return ""
	}
	end := strings.IndexFunc(body, unicode.IsSpace)
	if end < 0 {
		end = len(body)
	}
	token := body[1:end]
	if !slashTokenPattern.MatchString(token) {
		return ""
	}
	return token
}
