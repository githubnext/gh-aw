package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var networkFirewallCodemodLog = logger.New("cli:codemod_network_firewall")

// getNetworkFirewallCodemod creates a codemod for migrating network.firewall to sandbox.agent
func getNetworkFirewallCodemod() Codemod {
	return newFieldRemovalCodemod(fieldRemovalCodemodConfig{
		ID:           "network-firewall-migration",
		Name:         "Migrate network.firewall to sandbox.agent",
		Description:  "Removes deprecated 'network.firewall' field (firewall is now always enabled via sandbox.agent: awf default)",
		IntroducedIn: "0.1.0",
		ParentKey:    "network",
		FieldKey:     "firewall",
		LogMsg:       "Applied network.firewall migration (firewall now always enabled via sandbox.agent: awf default)",
		Log:          networkFirewallCodemodLog,
		PostTransform: func(lines []string, frontmatter map[string]any, fieldValue any) []string {
			// Note: We no longer set sandbox.agent: false since the firewall is mandatory
			// The firewall is always enabled via the default sandbox.agent: awf

			_, hasSandbox := frontmatter["sandbox"]

			// Add sandbox.agent if not already present AND if firewall was explicitly true
			// (no need to add sandbox.agent: awf if firewall was false, since awf is now the default)
			if !hasSandbox && fieldValue == true {
				// Only add sandbox.agent: awf if firewall was explicitly set to true
				sandboxLines := []string{
					"sandbox:",
					"  agent: awf  # Firewall enabled (migrated from network.firewall)",
				}

				// Try to place it after network block
				insertIndex := -1
				inNet := false
				for i, line := range lines {
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "network:") {
						inNet = true
					} else if inNet && len(trimmed) > 0 {
						// Check if this is a top-level key (no leading whitespace)
						if isTopLevelKey(line) {
							// Found next top-level key
							insertIndex = i
							break
						}
					}
				}

				if insertIndex >= 0 {
					// Insert after network block
					newLines := make([]string, 0, len(lines)+len(sandboxLines))
					newLines = append(newLines, lines[:insertIndex]...)
					newLines = append(newLines, sandboxLines...)
					newLines = append(newLines, lines[insertIndex:]...)
					lines = newLines
				} else {
					// Append at the end
					lines = append(lines, sandboxLines...)
				}

				networkFirewallCodemodLog.Print("Added sandbox.agent: awf (firewall was explicitly enabled)")
			}

			return lines
		},
	})
}
