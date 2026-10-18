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
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Document == nil {
		return nil, nil
	}
	sandbox, ok := input.Document.Frontmatter["sandbox"].(map[string]any)
	if !ok {
		return nil, nil
	}
	agent, ok := sandbox["agent"].(map[string]any)
	if !ok {
		return nil, nil
	}
	raw, exists := agent["args"]
	if !exists {
		return nil, nil
	}
	args, err := policyStrings(ctx, raw, "sandbox.agent.args")
	if err != nil {
		return nil, err
	}
	hasURLFilters := false
	sslBumpEnabled := false
	expectURLValue := false
	for _, arg := range args {
		if expectURLValue {
			expectURLValue = false
			if arg != "" && !strings.HasPrefix(arg, "--") {
				hasURLFilters = true
				continue
			}
		}
		switch {
		case arg == "--allow-urls":
			expectURLValue = true
		case strings.HasPrefix(arg, "--allow-urls="):
			hasURLFilters = strings.TrimPrefix(arg, "--allow-urls=") != ""
		case arg == "--ssl-bump":
			sslBumpEnabled = true
		case arg == "--ssl-bump=true":
			sslBumpEnabled = true
		case arg == "--ssl-bump=false":
			sslBumpEnabled = false
		}
	}
	if !hasURLFilters || sslBumpEnabled {
		return nil, nil
	}
	return []Diagnostic{{
		Path:    "sandbox.agent.args",
		Message: "AWF URL filters require SSL inspection to enforce HTTPS path restrictions; enable SSL inspection with --ssl-bump in sandbox.agent.args",
	}}, nil
}
