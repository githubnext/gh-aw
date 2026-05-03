// This file provides AWF (Agentic Workflow Firewall) configuration file generation.
//
// AWF supports loading configuration from a JSON/YAML file via the --config flag.
// Generating a config file rather than a long list of CLI flags improves:
//   - Readability: structured JSON is easier to audit than a one-liner flag list
//   - Correctness: complex values (JSON objects) avoid shell escaping issues
//   - Composability: config files can be layered and merged
//   - Extensibility: new features add JSON fields, not more argv flags
//
// # Config File Schema
//
// The generated config file follows the AWF config file format:
//
//	{
//	  "$schema": "https://github.com/github/gh-aw-firewall/releases/download/vX.Y.Z/awf-config.schema.json",
//	  "network": {
//	    "allowDomains": ["github.com", "api.github.com"],
//	    "blockDomains": ["ads.example.com"]
//	  },
//	  "apiProxy": {
//	    "enabled": true,
//	    "targets": {
//	      "openai":    { "host": "api.openai.com" },
//	      "anthropic": { "host": "api.anthropic.com" },
//	      "copilot":   { "host": "api.githubcopilot.com" },
//	      "gemini":    { "host": "generativelanguage.googleapis.com" }
//	    }
//	  },
//	  "container": {
//	    "imageTag": "0.25.29,squid=sha256:..."
//	  }
//	}
//
// # Runtime Usage
//
// The config file is written to ${RUNNER_TEMP}/gh-aw/awf-config.json before the
// AWF invocation, and referenced via: awf --config "${RUNNER_TEMP}/gh-aw/awf-config.json"
//
// Flags not yet represented in the config schema (--env-all, --exclude-env, --mount,
// --container-workdir, --log-level, --proxy-logs-dir, --audit-dir, --enable-host-access,
// --allow-host-ports, --skip-pull, --tty, --difc-proxy-host, --difc-proxy-ca-cert,
// --ssl-bump, --memory-limit, --diagnostic-logs) remain as CLI flags.

package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var awfConfigLog = logger.New("workflow:awf_config")

// AWFConfigFile represents the AWF configuration file schema.
// This is the top-level structure written to awf-config.json.
type AWFConfigFile struct {
	// Schema is the JSON schema reference for IDE auto-complete support.
	Schema string `json:"$schema,omitempty"`

	// Network contains network egress control configuration.
	Network *AWFNetworkConfig `json:"network,omitempty"`

	// APIProxy contains API proxy (LLM gateway) configuration.
	APIProxy *AWFAPIProxyConfig `json:"apiProxy,omitempty"`

	// Container contains container execution configuration.
	Container *AWFContainerConfig `json:"container,omitempty"`
}

// AWFNetworkConfig is the "network" section of the AWF config file.
// It maps to the --allow-domains and --block-domains CLI flags.
type AWFNetworkConfig struct {
	// AllowDomains is the list of allowed egress domains.
	// Supports wildcards (e.g. "*.github.com") and exact matches.
	// Maps to: --allow-domains <comma-separated>
	AllowDomains []string `json:"allowDomains,omitempty"`

	// BlockDomains is the list of explicitly blocked egress domains.
	// Maps to: --block-domains <comma-separated>
	BlockDomains []string `json:"blockDomains,omitempty"`
}

// AWFAPIProxyConfig is the "apiProxy" section of the AWF config file.
// It maps to the --enable-api-proxy and --*-api-target CLI flags.
type AWFAPIProxyConfig struct {
	// Enabled enables the API proxy sidecar for LLM gateway credential isolation.
	// Maps to: --enable-api-proxy
	Enabled bool `json:"enabled"`

	// Targets holds per-provider API target overrides.
	// Supported keys: "openai", "anthropic", "copilot", "gemini"
	Targets map[string]*AWFAPITargetConfig `json:"targets,omitempty"`
}

// AWFAPITargetConfig is a single API proxy target entry.
// Maps to: --<provider>-api-target <host>
type AWFAPITargetConfig struct {
	// Host is the hostname (and optional port) of the API endpoint.
	Host string `json:"host"`
}

// AWFContainerConfig is the "container" section of the AWF config file.
// It maps to the --image-tag CLI flag.
type AWFContainerConfig struct {
	// ImageTag is the pinned AWF Docker image tag, with optional digest metadata.
	// Format: "<tag>" or "<tag>,squid=sha256:...,agent=sha256:..."
	// Maps to: --image-tag <value>
	ImageTag string `json:"imageTag,omitempty"`
}

// buildAWFConfigSchemaURL returns the release-pinned JSON schema URL for the AWF config file.
// The URL is versioned so that schema validation tools always reference the exact schema
// that matches the AWF binary being used. When DefaultFirewallVersion is bumped the URL
// automatically tracks the new release.
//
// If firewallConfig carries an explicit version (e.g. sandbox.agent.version) that version
// is used; otherwise DefaultFirewallVersion is used.
func buildAWFConfigSchemaURL(firewallConfig *FirewallConfig) string {
	version := string(constants.DefaultFirewallVersion)
	if firewallConfig != nil && firewallConfig.Version != "" {
		version = firewallConfig.Version
	}
	// Special-case "latest": the GitHub Releases /latest/download/ shortcut serves
	// assets from the most recent release without requiring a tag in the path.
	if strings.EqualFold(version, "latest") {
		return "https://github.com/github/gh-aw-firewall/releases/latest/download/awf-config.schema.json"
	}
	// Ensure version has the 'v' prefix required by GitHub release tag URLs.
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return fmt.Sprintf("https://github.com/github/gh-aw-firewall/releases/download/%s/awf-config.schema.json", version)
}

// BuildAWFConfigJSON generates a compact JSON config file for AWF from the provided
// command configuration. The JSON is single-line (no indentation) for safe embedding
// in a shell printf command.
//
// The caller is responsible for writing the returned JSON to disk at the path expected
// by the AWF --config flag. See BuildAWFCommand for how this is wired together.
func BuildAWFConfigJSON(config AWFCommandConfig) (string, error) {
	awfConfigLog.Printf("Building AWF config JSON: engine=%s, allowed_domains=%q", config.EngineName, config.AllowedDomains)

	// Resolve firewall config once — used for both the schema URL and the container image tag.
	firewallConfig := getFirewallConfig(config.WorkflowData)

	awfConfig := AWFConfigFile{
		Schema: buildAWFConfigSchemaURL(firewallConfig),
	}

	// ── Network section ──────────────────────────────────────────────────────
	if config.AllowedDomains != "" {
		allowList := splitDomainList(config.AllowedDomains)
		awfConfig.Network = &AWFNetworkConfig{
			AllowDomains: allowList,
		}
		awfConfigLog.Printf("Network section: %d allowed domains", len(allowList))

		// Blocked domains (if configured in the workflow)
		if config.WorkflowData != nil {
			blockedDomainsStr := formatBlockedDomains(config.WorkflowData.NetworkPermissions)
			if blockedDomainsStr != "" {
				blockList := splitDomainList(blockedDomainsStr)
				awfConfig.Network.BlockDomains = blockList
				awfConfigLog.Printf("Network section: %d blocked domains", len(blockList))
			}
		}
	}

	// ── API proxy section ─────────────────────────────────────────────────────
	apiProxy := &AWFAPIProxyConfig{
		Enabled: true,
	}

	targets := map[string]*AWFAPITargetConfig{}

	if openaiTarget := extractAPITargetHost(config.WorkflowData, "OPENAI_BASE_URL"); openaiTarget != "" {
		targets["openai"] = &AWFAPITargetConfig{Host: openaiTarget}
		awfConfigLog.Printf("API proxy: custom openai target=%s", openaiTarget)
	}
	if anthropicTarget := extractAPITargetHost(config.WorkflowData, "ANTHROPIC_BASE_URL"); anthropicTarget != "" {
		targets["anthropic"] = &AWFAPITargetConfig{Host: anthropicTarget}
		awfConfigLog.Printf("API proxy: custom anthropic target=%s", anthropicTarget)
	}
	if copilotTarget := GetCopilotAPITarget(config.WorkflowData); copilotTarget != "" {
		targets["copilot"] = &AWFAPITargetConfig{Host: copilotTarget}
		awfConfigLog.Printf("API proxy: custom copilot target=%s", copilotTarget)
	}
	if geminiTarget := GetGeminiAPITarget(config.WorkflowData, config.EngineName); geminiTarget != "" {
		targets["gemini"] = &AWFAPITargetConfig{Host: geminiTarget}
		awfConfigLog.Printf("API proxy: custom gemini target=%s", geminiTarget)
	}

	if len(targets) > 0 {
		apiProxy.Targets = targets
		awfConfigLog.Printf("API proxy: %d custom targets configured", len(targets))
	}
	awfConfig.APIProxy = apiProxy

	// ── Container section ─────────────────────────────────────────────────────
	awfImageTag := buildAWFImageTagWithDigests(getAWFImageTag(firewallConfig), config.WorkflowData)
	if awfImageTag != "" {
		awfConfig.Container = &AWFContainerConfig{
			ImageTag: awfImageTag,
		}
		awfConfigLog.Printf("Container section: image_tag=%s", awfImageTag)
	}

	jsonBytes, err := json.Marshal(awfConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AWF config to JSON: %w", err)
	}
	awfConfigLog.Printf("AWF config JSON generated: %d bytes", len(jsonBytes))
	return string(jsonBytes), nil
}

// splitDomainList splits a comma-separated domain string into a deduplicated
// slice. Empty entries are ignored. The order of the original list is preserved for
// non-duplicate entries; this keeps the allow-list deterministic.
func splitDomainList(domains string) []string {
	var result []string
	seen := make(map[string]bool)
	for d := range strings.SplitSeq(domains, ",") {
		d = strings.TrimSpace(d)
		if d != "" && !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}
	return result
}
