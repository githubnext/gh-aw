package workflow

import "strings"

func renderStepForRunner(step, runsOn string) string {
	if strings.TrimSpace(runsOn) != "runs-on: windows-latest" || !strings.Contains(step, "run:") {
		return step
	}

	return setBashShell(prefixShellScriptWithBash(step))
}

func prefixShellScriptWithBash(step string) string {
	lines := strings.SplitAfter(step, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		commandLine := trimmed
		if run, ok := strings.CutPrefix(commandLine, "run:"); ok {
			commandLine = strings.TrimSpace(run)
		}
		if commandLine == "" || strings.HasPrefix(commandLine, "bash ") || strings.HasPrefix(commandLine, "sh ") {
			continue
		}

		command := strings.Fields(commandLine)[0]
		if quote := commandLine[0]; quote == '"' || quote == '\'' {
			if end := strings.IndexByte(commandLine[1:], quote); end >= 0 {
				command = commandLine[1 : end+1]
			}
		}
		if strings.HasSuffix(command, ".sh") {
			lines[i] = strings.Replace(line, commandLine, "bash "+commandLine, 1)
		}
	}
	return strings.Join(lines, "")
}

func setBashShell(step string) string {
	lines := strings.SplitAfter(step, "\n")
	var result strings.Builder
	result.Grow(len(step))
	var currentStep strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "      - ") {
			result.WriteString(setBashShellForStep(currentStep.String()))
			currentStep.Reset()
		}
		currentStep.WriteString(line)
	}
	result.WriteString(setBashShellForStep(currentStep.String()))
	return result.String()
}

func setBashShellForStep(step string) string {
	if !strings.Contains(step, "\n        run:") {
		return step
	}
	if strings.Contains(step, "\n        shell:") {
		lines := strings.SplitAfter(step, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "        shell:") {
				lines[i] = "        shell: bash\n"
			}
		}
		return strings.Join(lines, "")
	}
	return strings.Replace(step, "\n        run:", "\n        shell: bash\n        run:", 1)
}
