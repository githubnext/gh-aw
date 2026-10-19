package lint

import (
	"context"
	"errors"
	"strings"
)

func InstallScriptsRule() Rule {
	return Rule{
		ID: "install-scripts", Description: "Keep Node package install hooks disabled",
		ThreatRuleID: "CTR-014",
		Severity:     Warning, Targets: AWAST, Check: checkInstallScripts,
	}
}

func checkInstallScripts(ctx context.Context, input Input) ([]Diagnostic, error) {
	node, err := authoredPolicyMapping(ctx, input, "runtimes", "node")
	if err != nil {
		return nil, err
	}
	value, exists := node["run-install-scripts"]
	if !exists {
		return nil, nil
	}
	enabled, ok := value.(bool)
	if !ok {
		return nil, errors.New("install-scripts requires runtimes.node.run-install-scripts to be a literal boolean")
	}
	if !enabled {
		return nil, nil
	}
	severity, err := sourcePolicySeverity(input, "install-scripts")
	if err != nil {
		return nil, err
	}
	return []Diagnostic{{
		Severity: severity, Path: "runtimes.node.run-install-scripts",
		Message: "Node package install hooks are enabled; remove run-install-scripts or set it to false to prevent dependency lifecycle scripts from executing",
	}}, nil
}

// NetworkScopeRule checks explicit wildcard allowances, not the effective
// firewall, inferred defaults, runtime additions, or MCP sandbox prerequisites.
func NetworkScopeRule() Rule {
	return Rule{
		ID: "network-scope", Description: "Avoid bare wildcard network allowances",
		ThreatRuleID: "CTR-011",
		Severity:     Warning, Targets: AWAST, Check: checkNetworkScope,
	}
}

func checkNetworkScope(ctx context.Context, input Input) ([]Diagnostic, error) {
	source, err := authoredPolicyMapping(ctx, input)
	if err != nil || source == nil {
		return nil, err
	}
	if text, ok := source["network"].(string); ok && text == "defaults" {
		return nil, nil
	}
	network, err := policyMapping(source["network"], "network")
	if err != nil {
		return nil, err
	}
	value, exists := network["allowed"]
	if !exists {
		return nil, nil
	}
	allowed, err := policyStrings(ctx, value, "network.allowed")
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	for _, domain := range allowed {
		if cancelErr := ctx.Err(); cancelErr != nil {
			return nil, errors.Join(err, cancelErr)
		}
		if strings.TrimSpace(domain) != "*" {
			continue
		}
		severity, severityErr := sourcePolicySeverity(input, "network-scope")
		if severityErr != nil {
			return nil, errors.Join(err, severityErr)
		}
		return []Diagnostic{{
			Severity: severity, Path: "network.allowed",
			Message: "Network allowance contains a bare wildcard; replace it with explicit domains or ecosystem identifiers instead of an unrestricted allowance",
		}}, err
	}
	return nil, err
}
