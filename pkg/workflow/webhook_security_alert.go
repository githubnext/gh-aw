package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

var webhookOrganizationSelectorPattern = regexp.MustCompile(`^[a-z0-9*]([a-z0-9*-]{0,37}[a-z0-9*])?$`)

func parseWebhookOrganizations(config map[string]any, path string) ([]string, error) {
	raw, specified := config["orgs"]
	if !specified {
		return nil, nil
	}
	const message = ".orgs must be a non-empty list of organization names or '*' wildcard patterns using 1-39 letters, digits, hyphens, or '*', without leading or trailing hyphens"
	orgs, err := webhookStringList(raw)
	if err != nil || len(orgs) == 0 {
		return nil, errors.New(path + message)
	}
	for index, org := range orgs {
		org = strings.ToLower(org)
		if !webhookOrganizationSelectorPattern.MatchString(org) {
			return nil, errors.New(path + message)
		}
		orgs[index] = org
	}
	sort.Strings(orgs)
	return slices.Compact(orgs), nil
}

func addSecurityAlertConfigs(on, events map[string]any, paths map[string]string) error {
	for _, alias := range sortedWebhookKeys(securityAlertTriggers) {
		raw, exists := on[alias]
		if !exists {
			continue
		}
		path := "on.gateway-dispatch." + alias
		config, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object with optional types, severity, and filters", path)
		}
		if _, exists := config["repos"]; exists {
			return fmt.Errorf("%s.repos is not supported; use shared on.gateway-dispatch.repos", path)
		}
		config, err := promoteWebhookFilters(config, path, "severity")
		if err != nil {
			return err
		}
		event := securityAlertTriggers[alias]
		events[event] = config
		paths[event] = path
	}
	return nil
}

func parseWebhookRepositories(config map[string]any, path string) ([]string, error) {
	raw, specified := config["repos"]
	if !specified {
		return nil, nil
	}
	const message = ".repos must be a non-empty list of literal owner/repo or owner/* selectors"
	repos, err := webhookStringList(raw)
	if err != nil || len(repos) == 0 {
		return nil, errors.New(path + message)
	}
	for i, repo := range repos {
		repo = strings.ToLower(repo)
		literal := repo
		if strings.HasSuffix(repo, "/*") {
			literal = strings.TrimSuffix(repo, "*") + "repository"
		}
		if !repoSlugPattern.MatchString(literal) || strings.HasSuffix(repo, "/.") || strings.HasSuffix(repo, "/..") {
			return nil, errors.New(path + message)
		}
		repos[i] = repo
	}
	sort.Strings(repos)
	return slices.Compact(repos), nil
}
