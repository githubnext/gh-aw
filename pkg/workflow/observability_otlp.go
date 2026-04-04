package workflow

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var otlpLog = logger.New("workflow:observability_otlp")

// extractOTLPEndpointDomain parses an OTLP endpoint URL and returns its hostname.
// Returns an empty string when the endpoint is a GitHub Actions expression (which
// cannot be resolved at compile time) or when the URL is otherwise invalid.
func extractOTLPEndpointDomain(endpoint string) string {
	if endpoint == "" {
		return ""
	}

	// GitHub Actions expressions (e.g. ${{ secrets.OTLP_ENDPOINT }}) cannot be
	// resolved at compile time, so skip domain extraction for them.
	if strings.Contains(endpoint, "${{") {
		otlpLog.Printf("OTLP endpoint is a GitHub Actions expression, skipping domain extraction: %s", endpoint)
		return ""
	}

	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		otlpLog.Printf("Failed to extract domain from OTLP endpoint %q: %v", endpoint, err)
		return ""
	}

	// Strip the port from the host so the AWF domain allowlist entry matches all ports
	// (e.g. "traces.example.com:4317" → "traces.example.com").
	host := parsed.Hostname()
	otlpLog.Printf("Extracted OTLP domain: %s", host)
	return host
}

// getOTLPEndpointEnvValue returns the raw endpoint value suitable for injecting as an
// environment variable in the generated GitHub Actions workflow YAML.
// Returns an empty string when no OTLP endpoint is configured.
func getOTLPEndpointEnvValue(config *FrontmatterConfig) string {
	if config == nil || config.Observability == nil || config.Observability.OTLP == nil {
		return ""
	}
	return config.Observability.OTLP.Endpoint
}

// extractOTLPConfigFromRaw reads OTLP endpoint and headers directly from the raw
// frontmatter map[string]any.  This avoids dependence on ParseFrontmatterConfig
// succeeding -- that function may fail for workflows with complex tool configurations
// (e.g. engine objects, array-style bash configs), which would leave ParsedFrontmatter
// nil and prevent OTLP injection.
func extractOTLPConfigFromRaw(frontmatter map[string]any) (endpoint, headers string) {
	obs, ok := frontmatter["observability"]
	if !ok {
		return
	}
	obsMap, ok := obs.(map[string]any)
	if !ok {
		return
	}
	otlp, ok := obsMap["otlp"]
	if !ok {
		return
	}
	otlpMap, ok := otlp.(map[string]any)
	if !ok {
		return
	}
	if ep, ok := otlpMap["endpoint"].(string); ok {
		endpoint = ep
	}
	if h, ok := otlpMap["headers"].(string); ok {
		headers = h
	}
	return
}

//  1. When the endpoint is a static URL, its hostname is appended to
//     NetworkPermissions.Allowed so the AWF firewall allows outbound traffic to it.
//
//  2. OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_SERVICE_NAME are appended to the
//     workflow-level env: YAML block (workflowData.Env) so they are available to
//     every step in the generated GitHub Actions workflow.
//
//  3. When headers are configured, OTEL_EXPORTER_OTLP_HEADERS is also appended
//     to the workflow-level env: block.
//
// When no OTLP endpoint is configured the function is a no-op.
func (c *Compiler) injectOTLPConfig(workflowData *WorkflowData) {
	// Read OTLP config from the raw frontmatter map so that injection works even
	// when ParseFrontmatterConfig failed (e.g. due to complex tool configs).
	endpoint, headers := extractOTLPConfigFromRaw(workflowData.RawFrontmatter)

	// Fall back to ParsedFrontmatter when the raw map didn't yield an endpoint.
	if endpoint == "" {
		endpoint = getOTLPEndpointEnvValue(workflowData.ParsedFrontmatter)
	}

	if endpoint == "" {
		return
	}

	otlpLog.Printf("Injecting OTLP configuration: endpoint=%s", endpoint)

	// 1. Add OTLP endpoint domain to the firewall allowlist.
	if domain := extractOTLPEndpointDomain(endpoint); domain != "" {
		if workflowData.NetworkPermissions == nil {
			workflowData.NetworkPermissions = &NetworkPermissions{}
		}
		workflowData.NetworkPermissions.Allowed = append(workflowData.NetworkPermissions.Allowed, domain)
		otlpLog.Printf("Added OTLP domain to network allowlist: %s", domain)
	}

	// 2. Inject OTEL env vars into the workflow-level env: block.
	otlpEnvLines := fmt.Sprintf("  OTEL_EXPORTER_OTLP_ENDPOINT: %s\n  OTEL_SERVICE_NAME: gh-aw", endpoint)

	// 3. Inject OTEL_EXPORTER_OTLP_HEADERS when configured.
	// Prefer raw frontmatter value (already read above); fall back to ParsedFrontmatter.
	if headers == "" && workflowData.ParsedFrontmatter != nil &&
		workflowData.ParsedFrontmatter.Observability != nil &&
		workflowData.ParsedFrontmatter.Observability.OTLP != nil {
		headers = workflowData.ParsedFrontmatter.Observability.OTLP.Headers
	}
	if headers != "" {
		otlpEnvLines += "\n  OTEL_EXPORTER_OTLP_HEADERS: " + headers
		otlpLog.Printf("Injected OTEL_EXPORTER_OTLP_HEADERS env var")
	}

	if workflowData.Env == "" {
		workflowData.Env = "env:\n" + otlpEnvLines
	} else {
		workflowData.Env = workflowData.Env + "\n" + otlpEnvLines
	}
	otlpLog.Printf("Injected OTEL env vars into workflow env block")
}
