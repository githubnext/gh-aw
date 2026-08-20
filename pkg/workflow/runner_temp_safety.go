package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	unsafeRunnerTempActionsPath = "${{ runner.temp }}/gh-aw/actions"
	safeActionsDirLine          = "const actionsDir = path.join(process.env.RUNNER_TEMP, 'gh-aw', 'actions');"
)

var (
	githubScriptActionRequireRE = regexp.MustCompile(`^(\s*)(.*\brequire\()'\$\{\{\s*runner\.temp\s*\}\}/gh-aw/actions/([^']+)'(\).*)$`)
	shellActionCommandRE        = regexp.MustCompile(`\b(node|bash|sh) \$\{\{\s*runner\.temp\s*\}\}/gh-aw/actions/([^ \t\r\n]+)`)
	blockScalarHeaderRE         = regexp.MustCompile(`^(\s*)(?:-\s+)?(script|run):\s*[|>]`)
	singleLineRunRE             = regexp.MustCompile(`^\s*(?:-\s+)?run:\s+`)
	singleLineExecutableRE      = regexp.MustCompile(`^\s*(?:-\s+)?(script|run):\s+`)
)

func rewriteRunnerTempInExecutableBodies(yamlContent string) string {
	lines := strings.SplitAfter(yamlContent, "\n")
	var out strings.Builder
	out.Grow(len(yamlContent))

	inScriptBlock := false
	inRunBlock := false
	blockIndent := -1
	scriptBlockHasActionsDir := false

	for _, line := range lines {
		lineWithoutNewline := strings.TrimSuffix(line, "\n")
		newline := ""
		if line != lineWithoutNewline {
			newline = "\n"
		}

		if inScriptBlock || inRunBlock {
			trimmed := strings.TrimSpace(lineWithoutNewline)
			currentIndent := leadingSpaces(lineWithoutNewline)
			if trimmed != "" && currentIndent <= blockIndent {
				inScriptBlock = false
				inRunBlock = false
				blockIndent = -1
				scriptBlockHasActionsDir = false
			}
		}

		if matches := blockScalarHeaderRE.FindStringSubmatch(lineWithoutNewline); matches != nil {
			inScriptBlock = matches[2] == "script"
			inRunBlock = matches[2] == "run"
			blockIndent = len(matches[1])
			if inScriptBlock {
				scriptBlockHasActionsDir = false
			}
		}

		rewrittenLine := lineWithoutNewline
		if inRunBlock || singleLineRunRE.MatchString(lineWithoutNewline) {
			rewrittenLine = shellActionCommandRE.ReplaceAllString(lineWithoutNewline, `${1} "$${RUNNER_TEMP}/gh-aw/actions/${2}"`)
		}
		if inScriptBlock {
			if strings.Contains(rewrittenLine, safeActionsDirLine) {
				scriptBlockHasActionsDir = true
			}
			if matches := githubScriptActionRequireRE.FindStringSubmatch(rewrittenLine); matches != nil {
				indent := matches[1]
				if !scriptBlockHasActionsDir {
					out.WriteString(indent)
					out.WriteString("const path = require('path');\n")
					out.WriteString(indent)
					out.WriteString(safeActionsDirLine)
					out.WriteString("\n")
					scriptBlockHasActionsDir = true
				}
				rewrittenLine = matches[1] + matches[2] + "path.join(actionsDir, '" + matches[3] + "')" + matches[4]
			}
		}

		out.WriteString(rewrittenLine)
		out.WriteString(newline)
	}

	return out.String()
}

func finalizeRunnerTempSafety(yamlContent string) (string, error) {
	yamlContent = rewriteRunnerTempInExecutableBodies(yamlContent)
	if err := validateNoRunnerTempInExecutableBodies(yamlContent); err != nil {
		return "", err
	}
	return yamlContent, nil
}

func validateNoRunnerTempInExecutableBodies(yamlContent string) error {
	lines := strings.Split(yamlContent, "\n")
	inExecutableBlock := false
	executableBlockIndent := -1

	for i, line := range lines {
		if inExecutableBlock {
			trimmed := strings.TrimSpace(line)
			currentIndent := leadingSpaces(line)
			if trimmed != "" && currentIndent <= executableBlockIndent {
				inExecutableBlock = false
				executableBlockIndent = -1
			}
		}

		if matches := blockScalarHeaderRE.FindStringSubmatch(line); matches != nil {
			inExecutableBlock = true
			executableBlockIndent = len(matches[1])
			continue
		}

		if strings.Contains(line, unsafeRunnerTempActionsPath) &&
			(inExecutableBlock || singleLineExecutableRE.MatchString(line)) {
			return fmt.Errorf("generated executable step contains unsafe %s expression on line %d", unsafeRunnerTempActionsPath, i+1)
		}
	}

	return nil
}

func leadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}
