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
// (marketplaceURL, pluginSpec) pair where pluginSpec is in the
// "plugin-name@marketplace-name" format accepted by engine CLIs.
//
// Expected URL shape:
//
//	https://github.com/{owner}/{repo}/tree/{branch}/plugins/{pluginPath...}
//
// Example:
//
//	https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning
//	→ marketplace: https://github.com/github/copilot-plugins
//	→ pluginSpec:  advanced-security/skills/secret-scanning@github/copilot-plugins
//
// The "plugins/" path prefix is stripped from the plugin ID because engine CLIs
// expect the ID relative to the marketplace's plugins directory, not the full
// path from the repo root. The spec uses the "plugin-name@marketplace-name"
// format where marketplace-name is the "owner/repo" of the marketplace.
//
// Returns ("", "", false) when the input is not a recognisable GitHub tree URL,
// so the caller can treat the value as a plain plugin name instead.
func parseGitHubPluginURL(raw string) (marketplaceURL string, pluginSpec string, ok bool) {
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

	// Strip a leading "plugins/" prefix so that the plugin ID is relative to
	// the marketplace's plugins directory (as expected by engine CLIs).
	pluginPath = strings.TrimPrefix(pluginPath, "plugins/")

	marketplace := u.Scheme + "://" + u.Host + "/" + owner + "/" + repo
	// Build the plugin spec in "plugin-name@marketplace-name" format.
	// The marketplace-name uses the "owner/repo" form, which the engine CLI
	// uses to look up the marketplace in its registry.
	spec := pluginPath + "@" + owner + "/" + repo
	return marketplace, spec, true
}

// normalizePlugins processes a list of raw plugin entries (plain names or full
// GitHub tree URLs) and returns:
//
//   - normalizedPlugins: plugin install arguments ready to pass to the engine CLI.
//     Full GitHub tree URLs are converted to "plugin-name@marketplace-name" format
//     (e.g. `advanced-security/skills/secret-scanning@github/copilot-plugins`),
//     which is the format accepted by engine CLIs.
//     Plain plugin names pass through unchanged.
//   - inferredMarketplaces: marketplace URLs inferred from any URL entries; these
//     must be registered before the plugins are installed
func normalizePlugins(plugins []string) (normalizedPlugins []string, inferredMarketplaces []string) {
	for _, entry := range plugins {
		if marketplace, pluginSpec, ok := parseGitHubPluginURL(entry); ok {
			// Use the "plugin-name@marketplace-name" spec format.
			normalizedPlugins = append(normalizedPlugins, pluginSpec)
			inferredMarketplaces = append(inferredMarketplaces, marketplace)
		} else {
			normalizedPlugins = append(normalizedPlugins, entry)
		}
	}
	return
}
