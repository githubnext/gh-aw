package workflow

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

func addSecurityAlertConfigs(on, events map[string]any, paths map[string]string, repos []string) error {
	raw, exists := on["security-alerts"]
	if !exists {
		return nil
	}
	const path = "on.security-alerts"
	config, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object with optional sources, types, severity, and filters", path)
	}
	if _, exists := config["repos"]; exists {
		return errors.New("on.security-alerts.repos is not supported; use on.repos")
	}
	if len(repos) > 0 {
		config = maps.Clone(config)
		config["repos"] = repos
	}
	selected, err := securityAlertSourceEvents(config, path)
	if err != nil {
		return err
	}
	config, err = promoteWebhookFilters(config, path, "severity")
	if err != nil {
		return err
	}
	delete(config, "sources")
	for _, event := range selected {
		events[event] = config
		paths[event] = path
	}
	return nil
}

func securityAlertSourceEvents(config map[string]any, path string) ([]string, error) {
	raw, specified := config["sources"]
	if !specified {
		raw = sortedWebhookKeys(securityAlertEvents)
	}
	sources, err := webhookStringValues(raw, path+".sources")
	if err != nil {
		return nil, err
	}
	events := make([]string, 0, len(sources))
	for _, source := range sources {
		event, supported := securityAlertEvents[source]
		if !supported {
			return nil, fmt.Errorf("%s.sources must select only code-scanning or dependabot", path)
		}
		events = append(events, event)
	}
	return events, nil
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
