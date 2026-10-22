package cli

import (
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getMCPNetworkMigrationCodemod migrates per-server network configuration to workflow configuration.
func getMCPNetworkMigrationCodemod() Codemod {
	return Codemod{
		ID:           "mcp-network-to-top-level-migration",
		Name:         "Migrate MCP network config to top-level",
		Description:  "Moves per-server MCP 'network.allowed' configuration to top-level workflow 'network.allowed'. Per-server network configuration is deprecated.",
		IntroducedIn: "0.6.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			servers, ok := frontmatter["mcp-servers"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			var domains []string
			var migrate []string
			for _, name := range sliceutil.SortedKeys(servers) {
				server, ok := servers[name].(map[string]any)
				if !ok {
					continue
				}
				network, ok := server["network"].(map[string]any)
				if !ok {
					continue
				}
				switch allowed := network["allowed"].(type) {
				case []any:
					if len(allowed) > 0 {
						migrate = append(migrate, name)
						domains = append(domains, networkDomainStrings(allowed)...)
					}
				case []string:
					if len(allowed) > 0 {
						migrate = append(migrate, name)
						domains = append(domains, allowed...)
					}
				}
			}
			if len(migrate) == 0 {
				return content, false, nil
			}
			if network, ok := frontmatter["network"].(map[string]any); ok {
				domains = append(networkDomainStrings(network["allowed"]), domains...)
			}
			domains = sliceutil.Deduplicate(domains)
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, name := range migrate {
					if err := f.Remove("mcp-servers", name, "network"); err != nil {
						return err
					}
				}
				return f.Set([]string{"network", "allowed"}, domains)
			})
		},
	}
}

func networkDomainStrings(value any) []string {
	switch value := value.(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		var domains []string
		for _, domain := range value {
			if domain, ok := domain.(string); ok {
				domains = append(domains, domain)
			}
		}
		return domains
	}
	return nil
}
