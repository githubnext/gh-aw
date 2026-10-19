package main

import (
	"errors"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
)

var (
	linearPathPattern    = regexp.MustCompile(`^/([A-Za-z0-9_-]+)/issue/([A-Z][A-Z0-9_-]*-[1-9][0-9]*)(?:/(?:[A-Za-z0-9_-]|%[0-9a-fA-F]{2})+)?$`)
	azureHostPattern     = regexp.MustCompile(`^[a-z0-9-]+\.visualstudio\.com$`)
	azureItemPathPattern = regexp.MustCompile(`^/(?:([^/]+)/)?_apis/wit/work[Ii]tems/([1-9][0-9]*)(?:/(updates|revisions)/([1-9][0-9]*))?$`)
)

func providerProjection(s Source, sub workflow.WebhookSubscription, p map[string]any) (string, []string, string, error) {
	switch s.Provider {
	case "jira":
		return jiraProjection(s, sub, p)
	case "linear":
		return linearProjection(s, sub, p)
	case "azure-devops":
		return azureProjection(s, sub, p)
	}
	return "", nil, "", errors.New("unsupported provider")
}

func jiraProjection(s Source, sub workflow.WebhookSubscription, p map[string]any) (string, []string, string, error) {
	native := text(p["webhookEvent"])
	if !slices.Contains(sub.NativeEvents, native) {
		return "", nil, "", nil
	}
	id := text(at(p, "issue", "id"))
	raw := text(at(p, "issue", "self"))
	if !positiveID.MatchString(id) || text(at(p, "issue", "fields", "project", "key")) != s.Project ||
		(raw != joinURL(s.Origin, "rest/api/2/issue", id) && raw != joinURL(s.Origin, "rest/api/3/issue", id)) {
		return "", nil, "", errors.New("original Jira identity or project is not enrolled")
	}
	if native == "jira:issue_created" {
		return raw, []string{"opened"}, "", nil
	}
	if native == "jira:issue_deleted" {
		return raw, []string{"deleted"}, "", nil
	}
	activities := []string{"edited"}
	changes, changesPresent := at(p, "changelog", "items").([]any)
	if !changesPresent {
		return raw, activities, "", nil
	}
	for _, item := range changes {
		change, ok := item.(map[string]any)
		if !ok || change["fieldtype"] != "jira" {
			continue
		}
		if change["field"] == "resolution" {
			from, fromPresent := change["from"]
			to, toPresent := change["to"]
			if fromPresent && toPresent {
				if from == nil && text(to) != "" {
					activities = append(activities, "closed")
				}
				if text(from) != "" && to == nil {
					activities = append(activities, "reopened")
				}
			}
		}
		if change["field"] == "labels" {
			current, ok := stringArray(at(p, "issue", "fields", "labels"))
			from, fromPresent := change["fromString"]
			to, toPresent := change["toString"]
			if ok && fromPresent && toPresent {
				if from == nil && text(to) != "" && len(current) == 1 && slices.Contains(current, text(to)) {
					activities = append(activities, "labeled")
				}
				if text(from) != "" && to == nil && len(current) == 0 {
					activities = append(activities, "unlabeled")
				}
			}
		}
	}
	return raw, activities, "", nil
}

func linearProjection(s Source, sub workflow.WebhookSubscription, p map[string]any) (string, []string, string, error) {
	native := text(p["action"])
	if p["type"] != "Issue" || !slices.Contains(sub.NativeEvents, native) {
		return "", nil, "", nil
	}
	data, ok := p["data"].(map[string]any)
	raw := text(p["url"])
	dataURL := text(at(data, "url"))
	if raw == "" {
		raw = dataURL
	}
	identifier := text(at(data, "identifier"))
	u, urlErr := url.Parse(raw)
	if urlErr != nil {
		return "", nil, "", errors.New("invalid Linear issue URL")
	}
	match := linearPathPattern.FindStringSubmatch(u.EscapedPath())
	var workspace, issue string
	if len(match) > 2 {
		parts := [3]string(match)
		workspace, issue = parts[1], parts[2]
	} else {
		return "", nil, "", errors.New("invalid Linear workspace-qualified issue URL")
	}
	if !ok || text(data["id"]) == "" || dataURL != "" && dataURL != raw || p["organizationId"] != s.OrganizationID ||
		s.Team != "" && data["teamId"] != s.Team ||
		u.Scheme != "https" || u.Host != "linear.app" || workspace != s.Workspace || issue != identifier {
		return "", nil, "", errors.New("original Linear issue, URL or organization is not enrolled")
	}
	if native == "create" {
		return raw, []string{"opened"}, "", nil
	}
	if native == "remove" {
		return raw, []string{"deleted"}, "", nil
	}
	activities := []string{"edited"}
	before, ok := p["updatedFrom"].(map[string]any)
	if !ok {
		return raw, activities, "", nil
	}
	old := maps.Clone(data)
	maps.Copy(old, before)
	_, completed := before["completedAt"]
	_, canceled := before["canceledAt"]
	if completed || canceled {
		was, knownBefore := terminalStatus(old)
		now, knownAfter := terminalStatus(data)
		if knownBefore && knownAfter && was != now {
			if now {
				activities = append(activities, "closed")
			} else {
				activities = append(activities, "reopened")
			}
		}
	}
	activities = labelActivities(activities, before["labelIds"], data["labelIds"])
	return raw, activities, "", nil
}

func terminalStatus(data map[string]any) (bool, bool) {
	allNull := true
	for _, key := range []string{"completedAt", "canceledAt"} {
		value, exists := data[key]
		if !exists {
			allNull = false
			continue
		}
		if value != nil {
			allNull = false
			if _, err := time.Parse(time.RFC3339Nano, text(value)); err == nil {
				return true, true
			}
		}
	}
	return false, allNull
}

func azureProjection(s Source, sub workflow.WebhookSubscription, p map[string]any) (string, []string, string, error) {
	native := text(p["eventType"])
	if !slices.Contains(sub.NativeEvents, native) {
		return "", nil, "", nil
	}
	r, ok := p["resource"].(map[string]any)
	if !ok || at(p, "resourceContainers", "collection", "id") != s.CollectionID ||
		at(p, "resourceContainers", "project", "id") != s.Project {
		return "", nil, "", errors.New("original Azure collection/project is not enrolled")
	}
	id := decimal(r["workItemId"])
	_, revision := r["revision"]
	if id == "" && (native != "workitem.updated" || p["resourceVersion"] == "1.0" && !revision) {
		id = decimal(r["id"])
	}
	if revisionObject, ok := r["revision"].(map[string]any); ok {
		if originalID, present := revisionObject["id"]; present && decimal(originalID) != id {
			return "", nil, "", errors.New("original Azure revision and work item identities conflict")
		}
	}
	raw, err := azureResourceURL(s, id, p)
	if err != nil {
		return "", nil, "", err
	}
	switch native {
	case "workitem.created":
		return raw, []string{"opened"}, "", nil
	case "workitem.deleted":
		return raw, []string{"deleted"}, "", nil
	case "workitem.restored":
		return raw, []string{"restored"}, "", nil
	}
	activities, terminal := azureUpdateActivities(r, sub.ActivityNormalization.ClosedStates)
	return raw, activities, terminal, nil
}

func azureResourceURL(s Source, id string, p map[string]any) (string, error) {
	if id == "" {
		return "", errors.New("the Azure canonical work item URL is unavailable")
	}
	raw := joinURL(s.CollectionURL, "_apis/wit/workItems", id)
	base, err := url.Parse(s.CollectionURL)
	if err != nil || base.Host != "dev.azure.com" && !azureHostPattern.MatchString(base.Host) {
		return "", errors.New("invalid enrolled Azure collection URL")
	}
	for _, field := range [][]string{{"resource", "url"}, {"resource", "revision", "url"}, {"resource", "_links", "workItem", "href"}} {
		value := at(p, field...)
		if value == nil {
			continue
		}
		link, err := url.Parse(text(value))
		if err != nil || !canonicalResource(text(value)) || link.Scheme != base.Scheme || link.Host != base.Host {
			return "", errors.New("the Azure delivered resource link conflicts with enrollment")
		}
		relative, ok := strings.CutPrefix(link.EscapedPath(), base.EscapedPath())
		if !ok {
			return "", errors.New("the Azure delivered collection path conflicts")
		}
		match := azureItemPathPattern.FindStringSubmatch(relative)
		if len(match) > 4 {
			parts := [5]string(match)
			if parts[2] != id || parts[1] != "" && parts[1] != s.Project && parts[1] != s.ProjectPath {
				return "", errors.New("the Azure delivered item identity conflicts")
			}
			if parts[3] == "" {
				raw = text(value)
			}
		} else {
			return "", errors.New("the Azure canonical work item URL is unavailable")
		}
	}
	return raw, nil
}

func azureUpdateActivities(r map[string]any, closed []string) ([]string, string) {
	activities := []string{"edited"}
	old := text(at(r, "fields", "System.State", "oldValue"))
	newState := text(at(r, "fields", "System.State", "newValue"))
	terminal := ""
	if old != "" && newState != "" {
		if !slices.Contains(closed, old) && slices.Contains(closed, newState) {
			activities, terminal = append(activities, "closed"), newState
		}
		if slices.Contains(closed, old) && !slices.Contains(closed, newState) {
			activities, terminal = append(activities, "reopened"), old
		}
	}
	oldTags, beforeOK := at(r, "fields", "System.Tags", "oldValue").(string)
	newTags, afterOK := at(r, "fields", "System.Tags", "newValue").(string)
	if beforeOK && afterOK {
		activities = labelActivities(activities, splitTags(oldTags), splitTags(newTags))
	}
	return activities, terminal
}

func stringArray(value any) ([]string, bool) {
	list, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := []string{}
	for _, item := range list {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, false
		}
		result = append(result, s)
	}
	return result, true
}

func labelActivities(activities []string, before, after any) []string {
	old, oldOK := stringArray(before)
	current, currentOK := stringArray(after)
	if oldOK && currentOK {
		if slices.ContainsFunc(current, func(s string) bool { return !slices.Contains(old, s) }) {
			activities = append(activities, "labeled")
		}
		if slices.ContainsFunc(old, func(s string) bool { return !slices.Contains(current, s) }) {
			activities = append(activities, "unlabeled")
		}
	}
	return activities
}

func splitTags(raw string) []any {
	result := []any{}
	for tag := range strings.SplitSeq(raw, ";") {
		if tag = strings.TrimSpace(tag); tag != "" {
			result = append(result, tag)
		}
	}
	return result
}
