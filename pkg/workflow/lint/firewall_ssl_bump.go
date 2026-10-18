package lint

import (
	"context"
	"errors"
)

func FirewallSSLBumpRule() Rule {
	return Rule{
		ID: "firewall-ssl-bump", Description: "Require SSL inspection for authored firewall URL filters",
		Severity: Error, Targets: AWAST, Check: checkFirewallSSLBump,
	}
}

func checkFirewallSSLBump(ctx context.Context, input Input) ([]Diagnostic, error) {
	source, err := authoredPolicyMapping(ctx, input)
	if err != nil || source == nil || source["network"] == "defaults" {
		return nil, err
	}
	network, err := policyMapping(source["network"], "network")
	if err != nil {
		return nil, err
	}
	firewall, err := policyMapping(network["firewall"], "network.firewall")
	if err != nil {
		return nil, err
	}
	raw, exists := firewall["allow-urls"]
	if !exists {
		return nil, nil
	}
	urls, err := policyStrings(ctx, raw, "network.firewall.allow-urls")
	if len(urls) == 0 {
		return nil, err
	}
	enabled, specified := firewall["ssl-bump"]
	if specified {
		if _, ok := enabled.(bool); !ok {
			return nil, errors.Join(err, errors.New("firewall-ssl-bump requires a literal boolean at network.firewall.ssl-bump"))
		}
		if enabled == true {
			return nil, err
		}
	} else {
		pending, importErr := authoredImportsPending(input)
		err = errors.Join(err, importErr)
		if importErr != nil || pending {
			return nil, err
		}
	}
	return []Diagnostic{{Path: "network.firewall.allow-urls",
		Message: "Firewall URL filters require SSL inspection to enforce HTTPS path restrictions; set network.firewall.ssl-bump to true",
	}}, err
}
