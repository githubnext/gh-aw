//go:build !integration

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNVXGuestCommandAssets(t *testing.T) {
	for _, policy := range []string{"workspace-only", "workspace-and-tool-cache"} {
		t.Run(policy, func(t *testing.T) {
			data := validNVXWorkflowData()
			data.SandboxConfig.Agent.NVX.MountPolicy = policy
			command := GetMCPCLIPathSetup(&WorkflowData{SafeOutputs: &SafeOutputsConfig{NoOp: &NoOpConfig{}}}) +
				" && " + GetNpmBinPathSetup() + " && " + nodeRuntimeResolutionCommand +
				` "${RUNNER_TEMP}/gh-aw/actions/copilot_harness.cjs" "${RUNNER_TEMP}/gh-aw/bin/copilot" --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt`

			guestCommand := rewriteNVXEngineCommand(command)
			assert.NotContains(t, guestCommand, constants.GhAwRootDirShell)
			assert.NotContains(t, guestCommand, "RUNNER_TOOL_CACHE must be set")
			assert.Contains(t, guestCommand, nvxGuestRuntimeDir+"/actions/copilot_harness.cjs")
			assert.Contains(t, guestCommand, nvxGuestRuntimeDir+"/bin/copilot")
			assert.Contains(t, guestCommand, nvxGuestRuntimeDir+"/mcp-cli/bin")
			assert.Contains(t, guestCommand, `${RUNNER_TOOL_CACHE:-}`)
			require.NoError(t, exec.Command("bash", "-n", "-c", guestCommand).Run())

			hostCommand := BuildAWFCommand(AWFCommandConfig{
				EngineName: "copilot", EngineCommand: command, WorkflowData: data,
			})
			assert.Contains(t, hostCommand, `nvx_assets.cjs" stage || exit $?`)
			assert.Contains(t, hostCommand, `if [ ! -x /usr/bin/patchelf ]; then`)
			assert.Contains(t, hostCommand, `sudo apt-get install -y --no-install-recommends patchelf || exit $?`)
			assert.Contains(t, hostCommand, WrapCommandInShell(guestCommand))
			require.NoError(t, exec.Command("bash", "-n", "-c", hostCommand).Run())
		})
	}
}

func TestNVXGuestCommandPreservesShellArguments(t *testing.T) {
	command := `printf '%s\n' 'single quote: '"'"'; $HOME; $(exit 1)'`
	rewritten := rewriteNVXEngineCommand(command)
	assert.Contains(t, rewritten, shellEscapeArg(command))
	require.NoError(t, exec.Command("bash", "-n", "-c", rewritten).Run())
}

func TestNVXDefaultGuestWorkDir(t *testing.T) {
	data := validNVXWorkflowData()
	data.SandboxConfig.Agent.NVX = nil
	command := BuildAWFCommand(AWFCommandConfig{EngineName: "copilot", WorkflowData: data})
	assert.Contains(t, command, "--container-workdir /workspace")
	assert.NotContains(t, command, `--container-workdir "${GITHUB_WORKSPACE}"`)
}

func TestNVXAssetCollectionStep(t *testing.T) {
	data := validNVXWorkflowData()
	step := strings.Join(generateNVXAssetCollectionStep(data), "\n")
	assert.Contains(t, step, "if: always()")
	assert.Contains(t, step, `nvx_assets.cjs" collect`)
	data.UseSamples = true
	assert.Nil(t, generateNVXAssetCollectionStep(data))
	data.UseSamples = false
	data.SandboxConfig.Agent.Runtime = AgentRuntimeDocker
	assert.Nil(t, generateNVXAssetCollectionStep(data))
}

func TestNVXNpmEngineAssets(t *testing.T) {
	for _, engine := range []CodingAgentEngine{NewCodexEngine(), NewGeminiEngine(), NewPiEngine()} {
		t.Run(engine.GetID(), func(t *testing.T) {
			data := validNVXWorkflowData()
			data.EngineConfig.ID = engine.GetID()
			var installation strings.Builder
			for _, step := range engine.GetInstallationSteps(data) {
				installation.WriteString(strings.Join(step, "\n"))
			}
			assert.Contains(t, installation.String(), `--prefix "${RUNNER_TEMP}/gh-aw/engine-cli"`)
			assert.Contains(t, GetMicroVMNpmCLIPathSetup(data), "/gh-aw/engine-cli/bin")
		})
	}
}

func TestDockerEngineCommandIsNotRewrittenForNVX(t *testing.T) {
	data := validNVXWorkflowData()
	data.SandboxConfig.Agent.Runtime = AgentRuntimeDocker
	command := BuildAWFCommand(AWFCommandConfig{
		EngineName: "copilot", EngineCommand: GetNpmBinPathSetup(), WorkflowData: data,
	})
	assert.Contains(t, command, "RUNNER_TOOL_CACHE must be set")
	assert.NotContains(t, command, "nvx_assets.cjs")
	assert.NotContains(t, command, "patchelf")
}

func TestNVXCompiledAssetHandoff(t *testing.T) {
	for _, policy := range []string{"workspace-only", "workspace-and-tool-cache"} {
		t.Run(policy, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "nvx.md")
			require.NoError(t, os.WriteFile(source, []byte(`---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
sandbox:
  agent:
    runtime: nvx
    nvx:
      preview: true
      network-isolation: true
      api-proxy: true
      layer-path: /tmp/nvx/guest-layer
      openvmm-path: /tmp/nvx/openvmm
      kernel-path: /tmp/nvx/vmlinux
      initramfs-path: /tmp/nvx/initramfs.cpio.gz
      artifact-manifest-path: /tmp/nvx/nvx-test-x86_64.manifest.json
      artifact-manifest-bundle-path: /tmp/nvx/nvx-test-x86_64.manifest.sigstore.jsonl
      mount-policy: `+policy+`
safe-outputs:
  noop:
---
Read the workspace and report completion.
`), 0600))
			require.NoError(t, NewCompiler().CompileWorkflow(source))
			lock, err := os.ReadFile(filepath.Join(dir, "nvx.lock.yml"))
			require.NoError(t, err)
			content := string(lock)
			assert.Contains(t, content, nvxGuestRuntimeDir+"/actions/copilot_harness.cjs")
			stageIndex := strings.Index(content, `nvx_assets.cjs" stage`)
			collectIndex := strings.Index(content, "Collect NVX guest outputs")
			cleanupIndex := strings.Index(content, "name: Remove staged NVX artifacts")
			require.NotEqual(t, -1, stageIndex)
			require.Greater(t, collectIndex, stageIndex)
			require.Greater(t, cleanupIndex, collectIndex)
			assert.NotContains(t, content[stageIndex:collectIndex], "RUNNER_TOOL_CACHE must be set")
		})
	}
}
