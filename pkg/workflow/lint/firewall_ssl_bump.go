package lint

import (
	"context"
	"strings"
)

func FirewallSSLBumpRule() Rule {
	return Rule{
		ID: "firewall-ssl-bump", Description: "Require SSL inspection for authored firewall URL filters",
		ThreatRuleID: "CTR-011",
		Severity:     Error, Targets: AWAST, Check: checkFirewallSSLBump,
	}
}

func checkFirewallSSLBump(ctx context.Context, input Input) ([]Diagnostic, error) {
	source, err := authoredPolicyMapping(ctx, input)
	if err != nil || source == nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	if source["network"] != "defaults" {
		network, err := policyMapping(source["network"], "network")
		if err != nil {
			return nil, err
		}
		if _, legacy := network["firewall"]; legacy {
			diagnostics = append(diagnostics, Diagnostic{Path: "network.firewall",
				Message: "network.firewall is unsupported by the compiler; remove this legacy block and use network.allowed for domain policy. Advanced URL filtering uses sandbox.agent.args with --ssl-bump and --allow-urls in non-strict mode",
			})
		}
	}
	if shorthand, ok := source["sandbox"].(string); ok && (shorthand == "awf" || shorthand == "default") {
		return diagnostics, nil
	}
	sandbox, err := policyMapping(source["sandbox"], "sandbox")
	if err != nil {
		return diagnostics, err
	}
	switch sandbox["agent"].(type) {
	case nil, string, bool:
		return diagnostics, nil
	}
	agent, err := policyMapping(sandbox["agent"], "sandbox.agent")
	if err != nil {
		return diagnostics, err
	}
	raw, exists := agent["args"]
	if !exists {
		return diagnostics, nil
	}
	args, err := policyStrings(ctx, raw, "sandbox.agent.args")
	if err != nil {
		return diagnostics, err
	}
	missingInspection, err := firewallURLFiltersNeedInspection(ctx, args)
	if err != nil {
		return diagnostics, err
	}
	if missingInspection {
		diagnostics = append(diagnostics, Diagnostic{Path: "sandbox.agent.args",
			Message: "Firewall URL filters require SSL inspection to enforce HTTPS path restrictions; add --ssl-bump alongside --allow-urls in sandbox.agent.args (internal overrides require non-strict mode)",
		})
	}
	return diagnostics, nil
}

func firewallURLFiltersNeedInspection(ctx context.Context, args []string) (bool, error) {
	var filters, inspection, expectingURL bool
	for _, arg := range args {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if expectingURL {
			expectingURL = false
			if strings.TrimSpace(arg) != "" && !strings.HasPrefix(arg, "--") {
				filters = true
				continue
			}
		}
		switch {
		case arg == "--":
			return filters && !inspection, nil
		case arg == "--ssl-bump", arg == "--ssl-bump=true":
			inspection = true
		case arg == "--ssl-bump=false":
			inspection = false
		case arg == "--allow-urls":
			expectingURL = true
		case strings.HasPrefix(arg, "--allow-urls="):
			filters = filters || strings.TrimSpace(strings.TrimPrefix(arg, "--allow-urls=")) != ""
		}
	}
	return filters && !inspection, nil
}
