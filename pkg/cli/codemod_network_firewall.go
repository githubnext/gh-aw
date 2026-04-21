package cli

import (
	"strconv"
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
				return lines
			}

			lines, merged := mergeFirewallIntoExistingSandbox(lines, fieldValue)
			if merged {
				networkFirewallCodemodLog.Print("Merged deprecated network.firewall into existing sandbox.agent")
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
			if version, ok := normalizeFirewallVersion(versionValue); ok {
				return []string{
					"sandbox:",
					"  agent:",
					"    id: awf  # Migrated from deprecated network setting",
					"    version: " + formatSandboxVersionYAML(version),
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

func formatSandboxVersionYAML(version string) string {
	// Always quote because sandbox.agent.version is a string field, and this prevents
	// YAML from interpreting numeric-like versions as numbers.
	return strconv.Quote(version)
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

func mergeFirewallIntoExistingSandbox(lines []string, fieldValue any) ([]string, bool) {
	agentLines := sandboxAgentLinesForExistingSandbox(fieldValue)
	if len(agentLines) == 0 {
		return lines, false
	}

	sandboxIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "sandbox:") {
			sandboxIdx = i
			break
		}
	}
	if sandboxIdx == -1 {
		return lines, false
	}

	sandboxIndent := getIndentation(lines[sandboxIdx])
	agentIndent := sandboxIndent + "  "
	sandboxEnd := len(lines)
	for i := sandboxIdx + 1; i < len(lines); i++ {
		if isTopLevelKey(lines[i]) {
			sandboxEnd = i
			break
		}
	}

	agentStart := -1
	for i := sandboxIdx + 1; i < sandboxEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "agent:") && getIndentation(lines[i]) == agentIndent {
			agentStart = i
			break
		}
	}

	indentedAgentLines := indentLines(agentLines, agentIndent)
	if agentStart == -1 {
		newLines := make([]string, 0, len(lines)+len(indentedAgentLines))
		newLines = append(newLines, lines[:sandboxIdx+1]...)
		newLines = append(newLines, indentedAgentLines...)
		newLines = append(newLines, lines[sandboxIdx+1:]...)
		return newLines, true
	}

	agentEnd := agentStart + 1
	agentFieldIndent := getIndentation(lines[agentStart])
	for agentEnd < sandboxEnd {
		trimmed := strings.TrimSpace(lines[agentEnd])
		if trimmed == "" {
			agentEnd++
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if len(getIndentation(lines[agentEnd])) > len(agentFieldIndent) {
				agentEnd++
				continue
			}
			break
		}
		if len(getIndentation(lines[agentEnd])) > len(agentFieldIndent) {
			agentEnd++
			continue
		}
		break
	}

	newLines := make([]string, 0, len(lines)-((agentEnd-agentStart)-len(indentedAgentLines)))
	newLines = append(newLines, lines[:agentStart]...)
	newLines = append(newLines, indentedAgentLines...)
	newLines = append(newLines, lines[agentEnd:]...)
	return newLines, true
}

func sandboxAgentLinesForExistingSandbox(fieldValue any) []string {
	switch value := fieldValue.(type) {
	case bool:
		if !value {
			return []string{"agent: false  # Migrated from deprecated network setting"}
		}
	case string:
		if strings.EqualFold(strings.TrimSpace(value), "disable") {
			return []string{"agent: false  # Migrated from deprecated network setting"}
		}
	case map[string]any:
		versionValue, hasVersion := value["version"]
		if hasVersion {
			if version, ok := normalizeFirewallVersion(versionValue); ok {
				return []string{
					"agent:",
					"  id: awf  # Migrated from deprecated network setting",
					"  version: " + formatSandboxVersionYAML(version),
				}
			}
		}
	}

	return nil
}

func indentLines(lines []string, indent string) []string {
	indented := make([]string, 0, len(lines))
	for _, line := range lines {
		indented = append(indented, indent+line)
	}
	return indented
}
