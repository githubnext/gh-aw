package cli

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getNetworkFirewallCodemod creates a codemod for migrating network.firewall to sandbox.agent
func getNetworkFirewallCodemod() Codemod {
	return Codemod{
		ID:           "network-firewall-migration",
		Name:         "Migrate network.firewall to sandbox.agent",
		Description:  "Removes deprecated 'network.firewall' field (firewall is now always enabled via sandbox.agent: awf default)",
		IntroducedIn: "0.1.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, exists := f.Lookup("network", "firewall")
				if !exists {
					return nil
				}
				_, hasSandbox := f.Lookup("sandbox")
				var agent any
				switch typed := value.(type) {
				case bool:
					if !typed {
						agent = false
					} else if !hasSandbox {
						agent = "awf"
					}
				case string:
					if strings.EqualFold(strings.TrimSpace(typed), "disable") {
						agent = false
					}
				case map[string]any:
					if version, ok := normalizeFirewallVersion(typed["version"]); ok {
						agent = map[string]any{"id": "awf", "version": version}
					} else if !hasSandbox {
						agent = "awf"
					}
				case nil:
					if !hasSandbox {
						agent = "awf"
					}
				}
				if agent != nil {
					if err := f.Set([]string{"sandbox", "agent"}, agent); err != nil {
						return err
					}
				}
				if err := f.Remove("network", "firewall"); err != nil {
					return err
				}
				return f.PruneEmpty("network")
			})
		},
	}
}

func normalizeFirewallVersion(versionValue any) (string, bool) {
	switch value := versionValue.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		return trimmed, trimmed != ""
	case int:
		return strconv.Itoa(value), true
	case int8:
		return strconv.FormatInt(int64(value), 10), true
	case int16:
		return strconv.FormatInt(int64(value), 10), true
	case int32:
		return strconv.FormatInt(int64(value), 10), true
	case int64:
		return strconv.FormatInt(value, 10), true
	case uint:
		return strconv.FormatUint(uint64(value), 10), true
	case uint8:
		return strconv.FormatUint(uint64(value), 10), true
	case uint16:
		return strconv.FormatUint(uint64(value), 10), true
	case uint32:
		return strconv.FormatUint(uint64(value), 10), true
	case uint64:
		return strconv.FormatUint(value, 10), true
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), true
	default:
		return "", false
	}
}
