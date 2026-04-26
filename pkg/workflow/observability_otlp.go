package workflow

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var otlpLog = logger.New("workflow:observability_otlp")

// normalizeOTLPHeaders converts the headers field value (which may be a string or a map)
// into the comma-separated key=value format required by OTEL_EXPORTER_OTLP_HEADERS.
//
// The second return value is true when the deprecated string form was used, so callers
// can emit a deprecation warning.
//
// String form (deprecated): "Authorization=Bearer tok,X-Tenant=acme"
// Map form (preferred):     map[string]any{"Authorization": "Bearer tok", "X-Tenant": "acme"}
func normalizeOTLPHeaders(raw any) (string, bool) {
	if raw == nil {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return "", false
		}
		return v, true // string form is deprecated
	case map[string]any:
		if len(v) == 0 {
			return "", false
		}
		// Sort keys for deterministic output
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			val, ok := v[k].(string)
			if !ok {
				otlpLog.Printf("OTLP headers map: value for key %q is not a string (got %T), skipping", k, v[k])
				continue
			}
			parts = append(parts, k+"="+val)
		}
		return strings.Join(parts, ","), false
	default:
		otlpLog.Printf("Unexpected type for OTLP headers: %T", raw)
		return "", false
	}
}

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

// isOTLPHeadersPresent returns true when OTEL_EXPORTER_OTLP_HEADERS has been injected
// into the workflow-level env block. This indicates that header masking is needed so
// that authentication tokens in the header value do not leak into GitHub Actions runner
// logs (including debug/step-debug logs).
func isOTLPHeadersPresent(data *WorkflowData) bool {
	if data == nil {
		return false
	}
	return strings.Contains(data.Env, "OTEL_EXPORTER_OTLP_HEADERS")
}

// generateOTLPHeadersMaskStep returns a GitHub Actions step that runs
// mask_otlp_headers.sh to issue the ::add-mask:: workflow command for the
// OTEL_EXPORTER_OTLP_HEADERS environment variable. Masking the value causes the
// GitHub Actions runner to replace any subsequent occurrence of it in the job
// logs with "***", preventing authentication tokens from leaking even when runner
// debug logging is enabled.
//
// The script performs three levels of masking:
//  1. The entire OTEL_EXPORTER_OTLP_HEADERS value (comma-separated header pairs).
//  2. Each individual header value extracted from the pairs, so that a token
//     appearing without its header name prefix is also redacted.
//  3. For Authorization-style "Bearer <token>" credentials, the raw token after
//     stripping the "Bearer " scheme prefix, so it is masked even when it appears
//     without the scheme (e.g. in downstream tool logs).
func generateOTLPHeadersMaskStep() string {
	var sb strings.Builder
	sb.WriteString("      - name: Mask OTLP telemetry headers\n")
	sb.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/mask_otlp_headers.sh\"\n")
	return sb.String()
}

// extractOTLPConfigFromRaw reads OTLP endpoint and headers directly from the raw
// frontmatter map[string]any.  This avoids dependence on ParseFrontmatterConfig
// succeeding -- that function may fail for workflows with complex tool configurations
// (e.g. engine objects, array-style bash configs), which would leave ParsedFrontmatter
// nil and prevent OTLP injection.
//
// The third return value is true when the deprecated string form was used for headers,
// so the caller can emit a deprecation warning.
func extractOTLPConfigFromRaw(frontmatter map[string]any) (endpoint, headers string, deprecated bool) {
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
	if raw, ok := otlpMap["headers"]; ok {
		headers, deprecated = normalizeOTLPHeaders(raw)
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
	endpoint, headers, deprecated := extractOTLPConfigFromRaw(workflowData.RawFrontmatter)

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
		var parsedDeprecated bool
		headers, parsedDeprecated = normalizeOTLPHeaders(workflowData.ParsedFrontmatter.Observability.OTLP.Headers)
		if parsedDeprecated {
			deprecated = true
		}
	}

	// Emit the deprecation warning once after resolving headers from all sources.
	if deprecated {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			"observability.otlp.headers: string form is deprecated. Use the map form instead (e.g. headers: {Authorization: \"Bearer ${{ secrets.TOKEN }}\"})",
		))
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

	// Store the resolved endpoint and headers so downstream code (mcp_gateway_config,
	// mcp_setup_generator) can use workflowData.OTLPEndpoint / OTLPHeaders as the single
	// source of truth instead of re-reading raw frontmatter independently.
	workflowData.OTLPEndpoint = endpoint
	workflowData.OTLPHeaders = headers
}
