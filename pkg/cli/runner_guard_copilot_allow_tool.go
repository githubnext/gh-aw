package cli

import "strings"

const copilotExecutionStepNameMarker = "Execute GitHub Copilot CLI"

// filterCopilotLocalAllowToolFindings drops RGS-012 findings that point at Copilot CLI
// allow-tool declarations for loopback curl targets. These declarations document the agent's
// permitted tools; they are not executable curl calls and cannot exfiltrate secrets.
func filterCopilotLocalAllowToolFindings(findings []runnerGuardFinding, gitRoot string) []runnerGuardFinding {
	filtered := make([]runnerGuardFinding, 0, len(findings))
	fileLinesByPath := make(map[string][]string)

	for _, finding := range findings {
		if finding.RuleID != runnerGuardSecretExfiltrationRule {
			filtered = append(filtered, finding)
			continue
		}

		resolvedPath := resolveRunnerGuardFilePath(gitRoot, finding.File)
		lines, ok := fileLinesByPath[resolvedPath]
		if !ok {
			lines = readWorkflowLines(resolvedPath)
			fileLinesByPath[resolvedPath] = lines
		}

		if findingInCopilotLocalCurlAllowTool(lines, finding.Line) {
			runnerGuardLog.Printf("Suppressing %s finding for Copilot local curl allow-tool declaration in %s", finding.RuleID, finding.File)
			continue
		}
		filtered = append(filtered, finding)
	}

	return filtered
}

func findingInCopilotLocalCurlAllowTool(lines []string, lineNum int) bool {
	if len(lines) == 0 || lineNum <= 0 || lineNum > len(lines) {
		return false
	}

	lineIndex := lineNum - 1
	if isLocalCurlAllowToolComment(lines[lineIndex]) || isLocalCurlAllowToolArgumentLine(lines[lineIndex]) {
		return true
	}

	stepStart := -1
	for i := lineIndex; i >= 0; i-- {
		if isStepBoundaryLine(lines[i]) {
			stepStart = i
			break
		}
	}
	if stepStart == -1 || !isStepNameLine(lines[stepStart], copilotExecutionStepNameMarker) {
		return false
	}

	stepEnd := len(lines)
	for i := stepStart + 1; i < len(lines); i++ {
		if isStepBoundaryLine(lines[i]) {
			stepEnd = i
			break
		}
	}

	runIndex := -1
	for i := stepStart + 1; i < stepEnd; i++ {
		if isRunLine(lines[i]) {
			runIndex = i
			break
		}
	}
	if runIndex == -1 || lineIndex >= runIndex {
		return false
	}

	hasToolCommentHeader := false
	hasLocalCurlAllowTool := false
	hasNonLocalCurlAllowTool := false
	for i := stepStart; i < runIndex; i++ {
		line := lines[i]
		if strings.Contains(line, "# Copilot CLI tool arguments") {
			hasToolCommentHeader = true
		}
		host, ok := curlAllowToolCommentHost(line)
		if !ok {
			continue
		}
		if isLocalCurlAllowToolHost(host) {
			hasLocalCurlAllowTool = true
		} else {
			hasNonLocalCurlAllowTool = true
		}
	}

	return hasToolCommentHeader && hasLocalCurlAllowTool && !hasNonLocalCurlAllowTool
}

func isRunLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "run:" || strings.HasPrefix(trimmed, "run: ")
}

func isLocalCurlAllowToolComment(line string) bool {
	host, ok := curlAllowToolCommentHost(line)
	return ok && isLocalCurlAllowToolHost(host)
}

func isLocalCurlAllowToolArgumentLine(line string) bool {
	if !strings.Contains(line, "--allow-tool") || !strings.Contains(line, "copilot") {
		return false
	}

	hosts := curlAllowToolHosts(line)
	if len(hosts) == 0 {
		return false
	}
	for _, host := range hosts {
		if !isLocalCurlAllowToolHost(host) {
			return false
		}
	}
	return true
}

func curlAllowToolCommentHost(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "#")
	trimmed = strings.TrimSpace(trimmed)
	const prefix = "--allow-tool shell(curl "
	if !strings.HasPrefix(trimmed, prefix) || !strings.HasSuffix(trimmed, ")") {
		return "", false
	}

	return curlTargetHost(strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), ")"))
}

func curlAllowToolHosts(line string) []string {
	const prefix = "shell(curl "
	var hosts []string
	remaining := line
	for {
		index := strings.Index(remaining, prefix)
		if index < 0 {
			return hosts
		}
		remaining = remaining[index+len(prefix):]
		end := strings.Index(remaining, ")")
		if end < 0 {
			return hosts
		}
		if host, ok := curlTargetHost(remaining[:end]); ok {
			hosts = append(hosts, host)
		}
		remaining = remaining[end+1:]
	}
}

func curlTargetHost(target string) (string, bool) {
	target = strings.TrimSpace(target)
	if fields := strings.Fields(target); len(fields) > 0 {
		target = fields[0]
	}
	target = strings.Trim(target, `"'`)

	if schemeEnd := strings.Index(target, "://"); schemeEnd >= 0 {
		target = target[schemeEnd+len("://"):]
	}
	if slash := strings.Index(target, "/"); slash >= 0 {
		target = target[:slash]
	}
	target = strings.Trim(target, "[]")
	if colon := strings.LastIndex(target, ":"); colon >= 0 {
		port := target[colon+1:]
		if port == "*" || allDigits(port) {
			target = target[:colon]
		}
	}

	if target == "" {
		return "", false
	}
	return strings.ToLower(strings.Trim(target, "[]")), true
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isLocalCurlAllowToolHost(host string) bool {
	switch strings.ToLower(strings.Trim(host, "[]")) {
	case "localhost", "127.0.0.1", "::1", "host.docker.internal":
		return true
	default:
		return false
	}
}
