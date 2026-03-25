// Package workflow provides compiler and runtime support for agentic workflows.
//
// This file contains helpers for parsing full GitHub plugin URLs into marketplace
// URLs and plugin IDs, enabling users to specify a skill/plugin by its GitHub URL
// instead of remembering the separate marketplace URL and plugin identifier.
package workflow

import (
	"net/url"
	"strings"
)

// parseGitHubPluginURL attempts to parse a full GitHub tree URL into a
// (marketplaceURL, pluginID) pair.
//
// Expected URL shape:
//
//	https://github.com/{owner}/{repo}/tree/{branch}/{pluginPath...}
//
// Example:
//
//	https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning
//	→ marketplace: https://github.com/github/copilot-plugins
//	→ pluginID:    plugins/advanced-security/skills/secret-scanning
//
// Returns ("", "", false) when the input is not a recognisable GitHub tree URL,
// so the caller can treat the value as a plain plugin name instead.
func parseGitHubPluginURL(raw string) (marketplaceURL string, pluginID string, ok bool) {
	// Only consider values that look like URLs (have a scheme)
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", "", false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", false
	}

	// Path must match /{owner}/{repo}/tree/{branch}/{pluginPath…}
	// SplitN with -1 to keep all trailing segments.
	trimmed := strings.TrimPrefix(u.Path, "/")
	parts := strings.SplitN(trimmed, "/", -1)

	// Minimum structure: owner / repo / tree / branch / at-least-one-path-segment
	if len(parts) < 5 || parts[2] != "tree" {
		return "", "", false
	}

	owner := parts[0]
	repo := parts[1]
	// parts[3] is the branch name; everything after is the plugin path
	pluginPath := strings.Join(parts[4:], "/")

	marketplace := u.Scheme + "://" + u.Host + "/" + owner + "/" + repo
	return marketplace, pluginPath, true
}

// normalizePlugins processes a list of raw plugin entries (plain names or full
// GitHub tree URLs) and returns:
//
//   - normalizedPlugins: plugin IDs ready to pass to the engine CLI install command
//   - inferredMarketplaces: marketplace URLs inferred from any URL entries; these
//     must be registered before the plugins are installed
//
// Plain plugin names pass through unchanged.  URL entries are split into a
// marketplace URL and a plugin path; the URL itself is replaced by the path.
func normalizePlugins(plugins []string) (normalizedPlugins []string, inferredMarketplaces []string) {
	for _, entry := range plugins {
		if marketplace, pluginID, ok := parseGitHubPluginURL(entry); ok {
			normalizedPlugins = append(normalizedPlugins, pluginID)
			inferredMarketplaces = append(inferredMarketplaces, marketplace)
		} else {
			normalizedPlugins = append(normalizedPlugins, entry)
		}
	}
	return
}
