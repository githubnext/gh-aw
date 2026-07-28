//go:build !integration

package workflow

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArcDindDockerHostDetection exercises the generated shell snippet that probes
// DOCKER_HOST and conditionally sets the --docker-host passthrough value.
// NOTE: --docker-host-path-prefix is no longer emitted (removed for sysroot, gh-aw#34896).
func TestArcDindDockerHostDetection(t *testing.T) {
	tests := []struct {
		name            string
		dockerHost      string
		wantDockerHost  bool
		wantDockerHostV string
	}{
		{"tcp://localhost:2375", "tcp://localhost:2375", true, "tcp://localhost:2375"},
		{"tcp://127.0.0.1:2375", "tcp://127.0.0.1:2375", true, "tcp://127.0.0.1:2375"},
		{"tcp://dind:2375 (K8s service name)", "tcp://dind:2375", true, "tcp://dind:2375"},
		{"tcp://172.30.0.5:2375 (pod IP)", "tcp://172.30.0.5:2375", true, "tcp://172.30.0.5:2375"},
		{"tcp://dind-sidecar.default.svc:2376", "tcp://dind-sidecar.default.svc:2376", true, "tcp://dind-sidecar.default.svc:2376"},
		{"unix socket (not tcp)", "unix:///var/run/docker.sock", false, ""},
		{"bare path", "/var/run/docker.sock", false, ""},
		{"empty (unset)", "", false, ""},
	}

	// Build the shell snippet from the constant (same code the compiler emits).
	scriptTemplate := fmt.Sprintf(`#!/bin/bash
export DOCKER_HOST="%%s"
GH_AW_DOCKER_HOST=""
if [[ "${DOCKER_HOST:-}" =~ %s ]]; then
  GH_AW_DOCKER_HOST="${DOCKER_HOST}"
fi
printf 'docker-host=%%%%s\n' "$GH_AW_DOCKER_HOST"
`, awfArcDindDockerHostRegex)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(scriptTemplate, tt.dockerHost)
			cmd := exec.Command("bash", "-c", script)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "bash script should succeed, output: %s", string(out))

			gotDockerHost := strings.TrimPrefix(strings.TrimSpace(string(out)), "docker-host=")
			if tt.wantDockerHost {
				assert.Equal(t, tt.wantDockerHostV, gotDockerHost,
					"expected docker host passthrough value to be set for DOCKER_HOST=%s", tt.dockerHost)
			} else {
				assert.Empty(t, gotDockerHost,
					"expected docker host passthrough value to NOT be set for DOCKER_HOST=%s", tt.dockerHost)
			}
		})
	}
}

// TestBuildAWFCommand_IncludesChrootInjectScript verifies that BuildAWFCommand
// includes the chroot injection script in the generated run step when the AWF
// version supports it.
func TestBuildAWFCommand_IncludesChrootInjectScript(t *testing.T) {
	t.Run("chroot inject script present when AWF version supports it", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:    "copilot",
			EngineCommand: "copilot --prompt-file /tmp/prompt.txt",
			LogFile:       "/tmp/gh-aw/agent-stdio.log",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
						Version: string(constants.AWFChrootConfigMinVersion),
					},
				},
			},
		}
		command := BuildAWFCommand(config)
		assert.Contains(t, command, awfArcDindChrootBinariesSourcePath,
			"command should include the expected binariesSourcePath constant")
		assert.Contains(t, command, awfArcDindChrootIdentityHome,
			"command should include the expected identity.home constant")
		assert.Contains(t, command, `node "${RUNNER_TEMP}/gh-aw/actions/patch_awf_chroot_config.cjs"`,
			"command should invoke the repository JavaScript helper for chroot config patching")
		assert.NotContains(t, command, "python3 - <<'PY'",
			"command should not inject an inline Python heredoc")
		assert.Contains(t, command, awfArcDindDockerHostRegex,
			"chroot inject script should reuse the DinD Docker host regex")
		// Structural: the chroot injection must appear *after* the DOCKER_HOST guard,
		// confirming it is nested inside the if-block and not emitted at top level.
		dockerhostIdx := strings.Index(command, awfArcDindDockerHostRegex)
		helperIdx := strings.Index(command, "patch_awf_chroot_config.cjs")
		assert.Greater(t, helperIdx, dockerhostIdx,
			"chroot injection must appear after the DOCKER_HOST guard in the generated script")
	})

	t.Run("chroot inject script absent when AWF version too old", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:    "copilot",
			EngineCommand: "copilot --prompt-file /tmp/prompt.txt",
			LogFile:       "/tmp/gh-aw/agent-stdio.log",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
						Version: "v0.27.0",
					},
				},
			},
		}
		command := BuildAWFCommand(config)
		assert.NotContains(t, command, "binariesSourcePath",
			"command should NOT include chroot inject script for old AWF version")
	})
}

func TestBuildModelsJSONPathExportScript(t *testing.T) {
	t.Run("uses tmp path by default", func(t *testing.T) {
		assert.Equal(t, `export GH_AW_MODELS_JSON_PATH="/tmp/gh-aw/models.json"`, buildModelsJSONPathExportScript(false))
	})

	t.Run("uses runner temp path for arc-dind", func(t *testing.T) {
		assert.Equal(t, `export GH_AW_MODELS_JSON_PATH="${RUNNER_TEMP}/gh-aw/models.json"`, buildModelsJSONPathExportScript(true))
	})
}

func TestRewriteArcDindPath(t *testing.T) {
	t.Run("rewrites tmp gh-aw prefix", func(t *testing.T) {
		assert.Equal(t, "${RUNNER_TEMP}/gh-aw/aw-prompts/prompt.txt", rewriteArcDindPath("/tmp/gh-aw/aw-prompts/prompt.txt"))
	})

	t.Run("rewrites multiple occurrences", func(t *testing.T) {
		input := "/tmp/gh-aw/a /tmp/gh-aw/b"
		expected := "${RUNNER_TEMP}/gh-aw/a ${RUNNER_TEMP}/gh-aw/b"
		assert.Equal(t, expected, rewriteArcDindPath(input))
	})

	t.Run("leaves unrelated paths unchanged", func(t *testing.T) {
		assert.Equal(t, "/tmp/not-gh-aw/file.txt", rewriteArcDindPath("/tmp/not-gh-aw/file.txt"))
	})
}

func TestRewriteArcDindEngineCommand(t *testing.T) {
	command := "copilot --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt"
	rewritten := rewriteArcDindEngineCommand(command)

	assert.Contains(t, rewritten, "export HOME=${RUNNER_TEMP}/gh-aw/home")
	assert.Contains(t, rewritten, "copilot --prompt-file ${RUNNER_TEMP}/gh-aw/aw-prompts/prompt.txt")
}

func TestBuildAWFCommand_ArcDindPreCreatesMountDirs(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:    "copilot",
		EngineCommand: "copilot run",
		LogFile:       "/tmp/log.txt",
		PathSetup:     "export PATH=/usr/bin:$PATH",
		WorkflowData: &WorkflowData{
			Name:            "Test",
			AI:              "copilot",
			MarkdownContent: "test",
			RunnerConfig:    &RunnerConfig{Topology: RunnerTopologyArcDind},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{ID: "awf"},
			},
		},
	}

	command := BuildAWFCommand(config)

	// Verify mount source directories are pre-created before AWF invocation
	assert.Contains(t, command, `mkdir -p "${RUNNER_TEMP}/gh-aw/home" "${RUNNER_TEMP}/gh-aw/sandbox/agent"`,
		"should pre-create rw mount source directories for arc-dind")

	// Verify the mounts themselves are present
	assert.Contains(t, command, `--mount "${RUNNER_TEMP}/gh-aw/home:${RUNNER_TEMP}/gh-aw/home:rw"`)
	assert.Contains(t, command, `--mount "${RUNNER_TEMP}/gh-aw/sandbox/agent:${RUNNER_TEMP}/gh-aw/sandbox/agent:rw"`)
}
