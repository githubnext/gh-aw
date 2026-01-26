package campaign

import (
	"fmt"
	"regexp"
	"strings"
)

type ParsedScope struct {
	Repos []string
	Orgs  []string
}

var (
	repoSelectorPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
	orgNamePattern      = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

func parseScopeSelectors(selectors []string) (ParsedScope, []string) {
	var parsed ParsedScope
	var problems []string

	for _, raw := range selectors {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			problems = append(problems, "scope must not contain empty entries - remove empty strings from the list")
			continue
		}

		// org:<name>
		if strings.HasPrefix(trimmed, "org:") {
			org := strings.TrimSpace(strings.TrimPrefix(trimmed, "org:"))
			if org == "" {
				problems = append(problems, "scope entry 'org:' must include an organization name - example: 'org:github'")
				continue
			}
			if strings.Contains(org, "/") {
				problems = append(problems, fmt.Sprintf("scope entry '%s' must be 'org:<name>' (no slashes) - example: 'org:github'", trimmed))
				continue
			}
			if strings.Contains(org, "*") {
				problems = append(problems, fmt.Sprintf("scope entry '%s' cannot contain wildcards - example: 'org:github'", trimmed))
				continue
			}
			if !orgNamePattern.MatchString(org) {
				problems = append(problems, fmt.Sprintf("scope entry '%s' has an invalid organization name - example: 'org:github'", trimmed))
				continue
			}
			parsed.Orgs = appendUniqueString(parsed.Orgs, org)
			continue
		}

		// Optional sugar: <org>/*
		if strings.HasSuffix(trimmed, "/*") && strings.Count(trimmed, "/") == 1 && !strings.Contains(trimmed, ":") {
			org := strings.TrimSuffix(trimmed, "/*")
			if org == "" {
				problems = append(problems, fmt.Sprintf("scope entry '%s' must include an organization name - example: 'org:github'", trimmed))
				continue
			}
			if strings.Contains(org, "*") {
				problems = append(problems, fmt.Sprintf("scope entry '%s' cannot contain wildcards - example: 'org:github'", trimmed))
				continue
			}
			if !orgNamePattern.MatchString(org) {
				problems = append(problems, fmt.Sprintf("scope entry '%s' has an invalid organization name - example: 'org:github'", trimmed))
				continue
			}
			parsed.Orgs = appendUniqueString(parsed.Orgs, org)
			continue
		}

		if strings.Contains(trimmed, "*") {
			problems = append(problems, fmt.Sprintf("scope entry '%s' cannot contain wildcards - list repositories explicitly or use 'org:<name>' for organization-wide scope", trimmed))
			continue
		}

		if strings.Contains(trimmed, ":") {
			problems = append(problems, fmt.Sprintf("scope entry '%s' is not recognized - valid formats: 'owner/repo' or 'org:<name>'", trimmed))
			continue
		}

		if repoSelectorPattern.MatchString(trimmed) {
			parsed.Repos = appendUniqueString(parsed.Repos, trimmed)
			continue
		}

		problems = append(problems, fmt.Sprintf("scope entry '%s' must be 'owner/repo' or 'org:<name>' - example: 'github/docs' or 'org:github'", trimmed))
	}

	return parsed, problems
}

func appendUniqueString(values []string, value string) []string {
	for _, v := range values {
		if v == value {
			return values
		}
	}
	return append(values, value)
}
