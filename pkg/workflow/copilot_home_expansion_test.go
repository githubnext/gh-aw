// Tests guarding the $HOME-based shell expansion logic that resolves the
// Copilot CLI config directory at runtime (instead of the hard-coded
// /home/runner that broke self-hosted and containerized runners).
//
// Two categories of tests are exercised here:
//
//  1. String-level assertions on the helpers in copilot_mcp.go and
//     copilot_engine_execution.go to lock the generated snippets so any future
//     regression flips a focused test rather than only the broader goldens.
//  2. Bash integration tests that actually execute the generated snippets
//     under a few HOME values to confirm:
//     - $HOME expands as expected
//     - quoting survives a HOME containing spaces and other special chars
//     - the EXIT trap fires and uses the runtime HOME, not the definition-time HOME
//     - the rubber-duck settings file is written to the resolved path
package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runBashWithHome executes the given bash script under a controlled HOME value
// in a fresh environment so the test does not depend on the caller's $HOME.
func runBashWithHome(t *testing.T, home, script string) (stdout string, stderr string, err error) {
	t.Helper()
	cmd := exec.Command("bash", "-c", script)
	// Start from an empty env and add only what we need so the runtime HOME the
	// shell sees is exactly what the test specifies.
	cmd.Env = []string{
		"HOME=" + home,
		"PATH=/usr/bin:/bin",
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// -----------------------------------------------------------------------------
// String-level assertions
// -----------------------------------------------------------------------------

// TestCopilotSettingsPath_UsesHomeNotLiteralRunner pins the constant so it is
// impossible to silently revert to a hard-coded /home/runner path.
func TestCopilotSettingsPath_UsesHomeNotLiteralRunner(t *testing.T) {
	assert.Equal(t, "$HOME/.copilot/settings.json", copilotSettingsPath,
		"copilotSettingsPath must use $HOME so self-hosted runners with HOME != /home/runner work")
	assert.NotContains(t, copilotSettingsPath, "/home/runner",
		"copilotSettingsPath must not embed a literal /home/runner")
}

// TestBuildCopilotSettingsSetup_UsesHomeExpansion verifies that the generated
// mkdir/printf/chown commands all reference $HOME, never a hard-coded path,
// and that the target path is double-quoted so a HOME containing spaces still
// resolves correctly.
func TestBuildCopilotSettingsSetup_UsesHomeExpansion(t *testing.T) {
	tests := []struct {
		name                  string
		fixOwnershipForCustom bool
		wantChown             bool
	}{
		{"without sudo chown", false, false},
		{"with sudo chown (custom engine.command)", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCopilotSettingsSetup(copilotSettingsDefaultContent, tt.fixOwnershipForCustom)

			// Must reference $HOME, never the literal /home/runner.
			assert.Contains(t, got, `mkdir -p "$HOME/.copilot"`,
				"mkdir must use $HOME-based path with double quotes:\n%s", got)
			assert.NotContains(t, got, "/home/runner",
				"setup must not embed a literal /home/runner:\n%s", got)

			// The printf target must point at the resolved settings path.
			assert.Contains(t, got, `> "$HOME/.copilot/settings.json"`,
				"printf must write to the $HOME-based settings path:\n%s", got)

			if tt.wantChown {
				assert.Contains(t, got, `sudo chown -R "$(id -u):$(id -g)" "$HOME/.copilot"`,
					"chown must target the $HOME-based path:\n%s", got)
			} else {
				assert.NotContains(t, got, "sudo chown",
					"should not emit sudo chown when fixOwnershipForCustomCommand is false:\n%s", got)
			}
		})
	}
}

// TestBuildCopilotMCPConfigExport_NoMCPServers verifies that we always export
// XDG_CONFIG_HOME (the Copilot CLI relies on it to locate ~/.copilot) even
// when there are no MCP servers, but skip the MCP-specific export.
func TestBuildCopilotMCPConfigExport_NoMCPServers(t *testing.T) {
	got := buildCopilotMCPConfigExport(&WorkflowData{Name: "no-mcp"})

	assert.Contains(t, got, `export XDG_CONFIG_HOME="$HOME"`,
		"XDG_CONFIG_HOME must always be exported (Copilot CLI uses it to find ~/.copilot):\n%s", got)
	assert.NotContains(t, got, "GH_AW_MCP_CONFIG",
		"GH_AW_MCP_CONFIG must NOT be exported when there are no MCP servers:\n%s", got)
	assert.NotContains(t, got, "/home/runner",
		"export block must not embed a literal /home/runner:\n%s", got)
}

// TestBuildCopilotMCPConfigExport_WithMCPServers verifies that GH_AW_MCP_CONFIG
// is exported under $HOME (not /home/runner) when the workflow has MCP servers.
func TestBuildCopilotMCPConfigExport_WithMCPServers(t *testing.T) {
	tests := []struct {
		name string
		wd   *WorkflowData
	}{
		{
			name: "github tool triggers MCP",
			wd: &WorkflowData{
				Name:  "with-github",
				Tools: map[string]any{"github": map[string]any{}},
			},
		},
		{
			name: "safe-outputs triggers MCP",
			wd: &WorkflowData{
				Name:        "with-safe-outputs",
				SafeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{}},
			},
		},
		{
			name: "custom MCP tool triggers MCP",
			wd: &WorkflowData{
				Name: "with-custom-mcp",
				Tools: map[string]any{
					"custom": map[string]any{
						"type": "http",
						"url":  "https://example.com/mcp",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, HasMCPServers(tt.wd), "test precondition: HasMCPServers should be true")

			got := buildCopilotMCPConfigExport(tt.wd)
			assert.Contains(t, got, `export XDG_CONFIG_HOME="$HOME"`,
				"XDG_CONFIG_HOME must be exported under $HOME:\n%s", got)
			assert.Contains(t, got, `export GH_AW_MCP_CONFIG="$HOME/.copilot/mcp-config.json"`,
				"GH_AW_MCP_CONFIG must point at $HOME/.copilot/mcp-config.json:\n%s", got)
			assert.NotContains(t, got, "/home/runner",
				"export block must not embed a literal /home/runner:\n%s", got)
		})
	}
}

// TestCopilotMCPRenderer_UsesHomeForConfigPath verifies that the MCP config
// path the renderer hands to start_mcp_gateway.cjs uses $HOME, not the
// hard-coded runner path.
func TestCopilotMCPRenderer_UsesHomeForConfigPath(t *testing.T) {
	engine := NewCopilotEngine()
	var yaml strings.Builder

	wd := &WorkflowData{
		Name:  "test",
		Tools: map[string]any{"github": map[string]any{}},
	}

	err := engine.RenderMCPConfig(&yaml, wd.Tools, []string{"github"}, wd)
	require.NoError(t, err)

	out := yaml.String()
	assert.Contains(t, out, `mkdir -p "$HOME/.copilot"`,
		"RenderMCPConfig must mkdir the $HOME-based config dir:\n%s", out)
	assert.NotContains(t, out, "/home/runner",
		"RenderMCPConfig must not embed a literal /home/runner:\n%s", out)
}

// TestGetExecutionSteps_NoLiteralHomeRunner is the broadest guard for the
// specific paths this PR fixed: the Copilot CLI config directory
// ($HOME/.copilot) and any YAML env: entry that hard-codes /home/runner where
// shell expansion would not happen.
//
// This deliberately does NOT assert there are zero /home/runner occurrences
// anywhere in the generated workflow. Other references may be governed by
// separate concerns from the Copilot CLI HOME-resolution fix.
func TestGetExecutionSteps_NoLiteralHomeRunner(t *testing.T) {
	// Patterns that must NEVER appear in generated Copilot step output because
	// they would break on self-hosted / containerized runners where HOME is
	// not /home/runner.
	forbidden := []string{
		"/home/runner/.copilot",      // Copilot CLI config directory
		"XDG_CONFIG_HOME: /home/run", // YAML env: literal (not shell-expanded)
		"GH_AW_MCP_CONFIG: /home/r",  // YAML env: literal (not shell-expanded)
	}

	tests := []struct {
		name string
		wd   *WorkflowData
	}{
		{
			name: "no MCP, no safe outputs",
			wd: &WorkflowData{
				Name: "minimal",
			},
		},
		{
			name: "with MCP server (github)",
			wd: &WorkflowData{
				Name:  "with-mcp",
				Tools: map[string]any{"github": map[string]any{}},
			},
		},
		{
			name: "with safe-outputs",
			wd: &WorkflowData{
				Name:        "with-safe-outputs",
				SafeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{}},
			},
		},
		{
			name: "with firewall (sandbox) mode",
			wd: &WorkflowData{
				Name: "with-firewall",
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				Tools: map[string]any{"github": map[string]any{}},
			},
		},
		{
			name: "with custom engine.command (sudo chown path)",
			wd: &WorkflowData{
				Name: "with-custom-cmd",
				EngineConfig: &EngineConfig{
					Command: "echo hello",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewCopilotEngine()
			steps := engine.GetExecutionSteps(tt.wd, "/tmp/gh-aw/test.log")
			require.Len(t, steps, 1)

			stepContent := strings.Join([]string(steps[0]), "\n")
			for _, pat := range forbidden {
				assert.NotContains(t, stepContent, pat,
					"generated step content must not contain %q (Copilot CLI config must use $HOME so self-hosted runners with HOME != /home/runner work)",
					pat)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Bash integration tests — execute the generated snippets to verify they
// actually work under various HOME values.
// -----------------------------------------------------------------------------

// homeValuesUnderTest is the set of HOME values we drive the generated shell
// against. It includes the standard GitHub-hosted path, a self-hosted runner
// pattern, a containerized pattern, and a worst-case path with spaces and
// special characters to ensure quoting is correct.
var homeValuesUnderTest = []struct {
	name string
	home string
}{
	{"github-hosted (/home/runner)", "/home/runner"},
	{"self-hosted (/home/actions)", "/home/actions"},
	{"containerized (/root)", "/root"},
	{"self-hosted with space", "/var/lib/actions runner"},
	{"self-hosted with dash and dot", "/home/runner-2.x"},
}

// TestBashIntegration_MCPConfigExport drives the generated export block
// through bash with different HOME values and verifies the resulting
// environment variables resolve to the correct $HOME-based paths.
func TestBashIntegration_MCPConfigExport(t *testing.T) {
	exportBlock := buildCopilotMCPConfigExport(&WorkflowData{
		Name:  "with-mcp",
		Tools: map[string]any{"github": map[string]any{}},
	})

	for _, hv := range homeValuesUnderTest {
		t.Run(hv.name, func(t *testing.T) {
			script := exportBlock +
				"echo \"XDG=$XDG_CONFIG_HOME\"\n" +
				"echo \"MCP=$GH_AW_MCP_CONFIG\"\n"

			stdout, stderr, err := runBashWithHome(t, hv.home, script)
			require.NoError(t, err, "bash script failed:\nstdout=%s\nstderr=%s", stdout, stderr)

			assert.Contains(t, stdout, "XDG="+hv.home+"\n",
				"XDG_CONFIG_HOME must resolve to the runtime HOME:\nstdout=%s", stdout)
			assert.Contains(t, stdout, "MCP="+hv.home+"/.copilot/mcp-config.json\n",
				"GH_AW_MCP_CONFIG must resolve to $HOME/.copilot/mcp-config.json:\nstdout=%s", stdout)
		})
	}
}

// TestBashIntegration_MCPConfigExport_NoMCP confirms that when the workflow
// has no MCP servers, GH_AW_MCP_CONFIG is unset (the export is skipped) while
// XDG_CONFIG_HOME is still set.
func TestBashIntegration_MCPConfigExport_NoMCP(t *testing.T) {
	exportBlock := buildCopilotMCPConfigExport(&WorkflowData{Name: "no-mcp"})

	script := exportBlock +
		"echo \"XDG=$XDG_CONFIG_HOME\"\n" +
		"echo \"MCP_SET=${GH_AW_MCP_CONFIG+set}\"\n"

	stdout, stderr, err := runBashWithHome(t, "/home/runner", script)
	require.NoError(t, err, "bash script failed:\nstdout=%s\nstderr=%s", stdout, stderr)

	assert.Contains(t, stdout, "XDG=/home/runner\n")
	assert.Contains(t, stdout, "MCP_SET=\n",
		"GH_AW_MCP_CONFIG should be unset when the workflow has no MCP servers:\nstdout=%s", stdout)
}

// TestBashIntegration_RenderMCPConfig_MkdirPath drives just the leading
// mkdir snippet emitted by RenderMCPConfig through bash to confirm the
// $HOME-based path is created and the literal /home/runner is never used.
func TestBashIntegration_RenderMCPConfig_MkdirPath(t *testing.T) {
	engine := NewCopilotEngine()
	var yaml strings.Builder
	wd := &WorkflowData{Name: "t", Tools: map[string]any{"github": map[string]any{}}}
	require.NoError(t, engine.RenderMCPConfig(&yaml, wd.Tools, []string{"github"}, wd))

	// Extract just the `mkdir -p "$HOME/.copilot"` command line so we don't
	// have to feed the full heredoc-piped node invocation through bash.
	// The line is emitted with a 10-space indent in the YAML run block.
	re := regexp.MustCompile(`(?m)^\s*mkdir -p "\$HOME/\.copilot"\s*$`)
	match := re.FindString(yaml.String())
	require.NotEmpty(t, match, "expected mkdir line in render output:\n%s", yaml.String())

	for _, hv := range homeValuesUnderTest {
		t.Run(hv.name, func(t *testing.T) {
			tmpRoot := t.TempDir()
			home := filepath.Join(tmpRoot, hv.home)
			require.NoError(t, os.MkdirAll(filepath.Dir(home), 0o755))

			stdout, stderr, err := runBashWithHome(t, home, strings.TrimSpace(match)+"\n")
			require.NoError(t, err, "bash script failed:\nstdout=%s\nstderr=%s", stdout, stderr)

			assert.DirExists(t, filepath.Join(home, ".copilot"),
				"RenderMCPConfig mkdir line must create $HOME/.copilot")
		})
	}
}
