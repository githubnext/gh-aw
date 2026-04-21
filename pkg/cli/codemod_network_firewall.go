package cli

import (
	"fmt"
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
			_, hasSandbox := frontmatter["sandbox"]

			if !hasSandbox {
				sandboxLines := sandboxAgentLinesFromFirewall(fieldValue)
				if len(sandboxLines) > 0 {
					lines = insertSandboxAfterNetworkBlock(lines, sandboxLines)
					networkFirewallCodemodLog.Print("Converted deprecated network.firewall to sandbox.agent")
				}
			}

			return lines
		},
	})
}

func sandboxAgentLinesFromFirewall(fieldValue any) []string {
	switch value := fieldValue.(type) {
	case bool:
		if value {
			return []string{
				"sandbox:",
				"  agent: awf  # Migrated from deprecated network setting",
			}
		}
		return []string{
			"sandbox:",
			"  agent: false  # Migrated from deprecated network setting",
		}
	case string:
		if strings.EqualFold(strings.TrimSpace(value), "disable") {
			return []string{
				"sandbox:",
				"  agent: false  # Migrated from deprecated network setting",
			}
		}
	case map[string]any:
		versionValue, hasVersion := value["version"]
		if hasVersion {
			version := strings.TrimSpace(fmt.Sprintf("%v", versionValue))
			if version != "" {
				return []string{
					"sandbox:",
					"  agent:",
					"    id: awf  # Migrated from deprecated network setting",
					fmt.Sprintf("    version: %q", version),
				}
			}
		}
		return []string{
			"sandbox:",
			"  agent: awf  # Migrated from deprecated network setting",
		}
	case nil:
		return []string{
			"sandbox:",
			"  agent: awf  # Migrated from deprecated network setting",
		}
	}
	return nil
}

func insertSandboxAfterNetworkBlock(lines []string, sandboxLines []string) []string {
	insertIndex := -1
	inNetworkBlock := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "network:") {
			inNetworkBlock = true
			continue
		}
		if inNetworkBlock && len(trimmed) > 0 && isTopLevelKey(line) {
			insertIndex = i
			break
		}
	}

	if insertIndex >= 0 {
		newLines := make([]string, 0, len(lines)+len(sandboxLines))
		newLines = append(newLines, lines[:insertIndex]...)
		newLines = append(newLines, sandboxLines...)
		newLines = append(newLines, lines[insertIndex:]...)
		return newLines
	}

	return append(lines, sandboxLines...)
}
