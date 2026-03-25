// Package workflow provides compiler and runtime support for agentic workflows.
//
// This file contains helpers for parsing full GitHub plugin URLs into
// OWNER/REPO:PATH/TO/PLUGIN plugin specs, enabling users to specify a
// skill/plugin by its GitHub URL instead of remembering the separate repo
// and path components.
package workflow

import (
	"net/url"
	"strings"
)

// parseGitHubPluginURL attempts to parse a full GitHub tree URL into an
// OWNER/REPO:PATH/TO/PLUGIN plugin spec as required by the Copilot CLI
// plugin specification.
//
// Expected URL shape:
//
//	https://github.com/{owner}/{repo}/tree/{branch}/{pluginPath...}
//
// Example:
//
//	https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning
//	→ pluginSpec: github/copilot-plugins:plugins/advanced-security/skills/secret-scanning
//
// The OWNER/REPO:PATH format ("subdirectory in a repository") is a direct
// GitHub reference that does not require marketplace registration — the
// Copilot CLI resolves it straight from the repository.
//
// Returns ("", false) when the input is not a recognisable GitHub tree URL,
// so the caller can treat the value as a plain plugin name instead.
func parseGitHubPluginURL(raw string) (pluginSpec string, ok bool) {
	// Only consider values that look like URLs (have a scheme)
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", false
	}

	// Path must match /{owner}/{repo}/tree/{branch}/{pluginPath…}
	// SplitN with -1 to keep all trailing segments.
	trimmed := strings.TrimPrefix(u.Path, "/")
	parts := strings.SplitN(trimmed, "/", -1)

	// Minimum structure: owner / repo / tree / branch / at-least-one-path-segment
	if len(parts) < 5 || parts[2] != "tree" {
		return "", false
	}

	owner := parts[0]
	repo := parts[1]
	// parts[3] is the branch name; everything after is the plugin path
	pluginPath := strings.Join(parts[4:], "/")

	// Build the OWNER/REPO:PATH spec — the "subdirectory in a repository"
	// format accepted by the Copilot CLI.  The full path from the repository
	// root (including any "plugins/" prefix) is preserved so the CLI can
	// locate the plugin's manifest file.
	spec := owner + "/" + repo + ":" + pluginPath
	return spec, true
}

// normalizePlugins processes a list of raw plugin entries (plain names or full
// GitHub tree URLs) and returns normalizedPlugins: plugin install arguments
// ready to pass to the engine CLI.
//
// Full GitHub tree URLs are converted to OWNER/REPO:PATH/TO/PLUGIN format
// (e.g. `github/copilot-plugins:plugins/advanced-security/skills/secret-scanning`),
// which is the "subdirectory in a repository" form accepted by the Copilot CLI.
// This format is a direct GitHub reference and does NOT require a prior
// marketplace registration step.
//
// Plain plugin names (e.g. `my-plugin`, `my-plugin@marketplace`) pass through
// unchanged.
func normalizePlugins(plugins []string) []string {
	normalized := make([]string, 0, len(plugins))
	for _, entry := range plugins {
		if spec, ok := parseGitHubPluginURL(entry); ok {
			normalized = append(normalized, spec)
		} else {
			normalized = append(normalized, entry)
		}
	}
	return normalized
}
