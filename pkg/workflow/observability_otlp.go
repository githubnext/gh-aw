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

// generateOTLPConclusionSpanStep generates a GitHub Actions step that sends an OTLP
// conclusion span from a downstream job (safe_outputs or conclusion).
//
// The step is a no-op when OTEL_EXPORTER_OTLP_ENDPOINT is not set, so it is safe to
// emit unconditionally.  It runs with if: always() and continue-on-error: true so OTLP
// failures can never block the job.
//
// Parameters:
//   - spanName: the OTLP span name, e.g. "gh-aw.job.safe-outputs"
func generateOTLPConclusionSpanStep(spanName string) string {
	var sb strings.Builder
	sb.WriteString("      - name: Send OTLP job span\n")
	sb.WriteString("        if: always()\n")
	sb.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/github-script"))
	sb.WriteString("        with:\n")
	sb.WriteString("          script: |\n")
	fmt.Fprintf(&sb, "            const { sendJobConclusionSpan } = require('%s/send_otlp_span.cjs');\n", SetupActionDestination)
	fmt.Fprintf(&sb, "            await sendJobConclusionSpan(%q);\n", spanName)
	return sb.String()
}

//  1. When the endpoint is a static URL, its hostname is appended to
//     NetworkPermissions.Allowed so the AWF firewall allows outbound traffic to it.
//
//  2. OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_SERVICE_NAME are appended to the
//     workflow-level env: YAML block (workflowData.Env) so they are available to
//     every step in the generated GitHub Actions workflow.
//
// When no OTLP endpoint is configured the function is a no-op.
func (c *Compiler) injectOTLPConfig(workflowData *WorkflowData) {
	endpoint := getOTLPEndpointEnvValue(workflowData.ParsedFrontmatter)
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
	if workflowData.Env == "" {
		workflowData.Env = "env:\n" + otlpEnvLines
	} else {
		workflowData.Env = workflowData.Env + "\n" + otlpEnvLines
	}
	otlpLog.Printf("Injected OTEL env vars into workflow env block")
}
