//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestThreatDetectionIsolation(t *testing.T) {
	compiler := NewCompiler()

	// Create a temporary directory for the test workflow
	tmpDir := testutil.TempDir(t, "test-*")
	workflowPath := filepath.Join(tmpDir, "test-isolation.md")

	workflowContent := `---
on: push
safe-outputs:
  create-issue:
tools:
  github:
    allowed: ["*"]
---
Test workflow`

	// Write the workflow file
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	yamlStr := string(result)

	// Detection is now a separate detection job - agent job should NOT contain inline detection steps
	agentSection := extractJobSection(yamlStr, "agent")
	if agentSection == "" {
		t.Fatal("Agent job not found in compiled workflow")
	}

	// Test 1: Detection job should exist as a separate job
	detectionSection := extractJobSection(yamlStr, "detection")
	if detectionSection == "" {
		t.Error("Detection job should exist as a separate job")
	}
	if !strings.Contains(detectionSection, "detection_guard") {
		t.Error("Detection job should contain detection_guard step")
	}
	if !strings.Contains(detectionSection, "detection_conclusion") {
		t.Error("Detection job should contain detection_conclusion step")
	}

	// Test 2: Detection engine step should use limited tools (no --allow-all-tools)
	// The detection copilot invocation uses only shell tools for analysis
	if !strings.Contains(detectionSection, "parse_threat_detection_results.cjs") {
		t.Error("Detection job should contain parse_threat_detection_results.cjs for detection")
	}

	// Test 3: Main agent job should still have --allow-tool or --allow-all-tools for the main agent execution
	if !strings.Contains(agentSection, "--allow-tool") && !strings.Contains(agentSection, "--allow-all-tools") {
		t.Error("Main agent job should have --allow-tool or --allow-all-tools arguments")
	}

	// Test 4: Main agent job should have MCP setup
	if !strings.Contains(agentSection, "Start MCP Gateway") {
		t.Error("Main agent job should have MCP setup step")
	}

	// Test 5: A separate detection job should exist
	if !strings.Contains(yamlStr, "  detection:") {
		t.Error("Separate detection job should exist")
	}
}

// TestExternalDetectorPath verifies that when features: gh-aw-detection: true is set,
// the compiler emits the external threat-detect binary path instead of the inline engine path.
func TestExternalDetectorPath(t *testing.T) {
	compiler := NewCompiler()

	tmpDir := testutil.TempDir(t, "test-external-detector-*")
	workflowPath := filepath.Join(tmpDir, "test-external-detector.md")

	workflowContent := `---
on: push
engine: copilot
safe-outputs:
  create-issue:
features:
  gh-aw-detection: true
tools:
  github:
    allowed: ["*"]
---
Test workflow`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	yamlStr := string(result)
	detectionSection := extractJobSection(yamlStr, "detection")
	if detectionSection == "" {
		t.Fatal("Detection job not found in compiled workflow")
	}

	// The external detector path must emit threat-detect conclude, not the .cjs module
	if strings.Contains(detectionSection, "parse_threat_detection_results.cjs") {
		t.Error("External detector path must NOT emit parse_threat_detection_results.cjs")
	}
	if !strings.Contains(detectionSection, "conclude_threat_detection.sh") {
		t.Error("External detector path must invoke conclude_threat_detection.sh for the conclude step")
	}
	if !strings.Contains(detectionSection, "GH_AW_DETECTION_CONTINUE_ON_ERROR") {
		t.Error("External detector path must pass GH_AW_DETECTION_CONTINUE_ON_ERROR to conclude_threat_detection.sh")
	}

	// The install step must reference the pinned version
	if !strings.Contains(detectionSection, "install_awf_binary.sh") {
		t.Error("External detector path must emit 'install_awf_binary.sh' install step")
	}
	if !strings.Contains(detectionSection, "install_threat_detect_binary.sh") {
		t.Error("External detector path must emit 'install_threat_detect_binary.sh' install step")
	}
	if !strings.Contains(detectionSection, "install_copilot_cli.sh") {
		t.Error("External detector path must emit engine installation step for copilot")
	}
	// The install step must pass the pinned DefaultThreatDetectVersion to the script
	if !strings.Contains(detectionSection, string(constants.DefaultThreatDetectVersion)) {
		t.Errorf("External detector path must use pinned version %q from DefaultThreatDetectVersion", constants.DefaultThreatDetectVersion)
	}

	// The AWF execution step must use threat-detect as the command
	if !strings.Contains(detectionSection, "threat-detect --engine") {
		t.Error("External detector path must invoke 'threat-detect --engine' inside AWF")
	}

	// The AWF execution step must prepend npm PATH setup so npm-installed engine CLIs
	// (e.g. claude, codex) are found by threat-detect inside the AWF chroot.
	if !strings.Contains(detectionSection, "RUNNER_TOOL_CACHE") {
		t.Error("External detector AWF step must prepend npm PATH setup (RUNNER_TOOL_CACHE) so engine CLIs are on PATH")
	}

	// The upload step must include detection_result.json
	if !strings.Contains(detectionSection, "detection_result.json") {
		t.Error("External detector path must upload detection_result.json")
	}

	// The detection guard and detection_conclusion step must still exist (gate contract preserved)
	if !strings.Contains(detectionSection, "detection_guard") {
		t.Error("External detector path must contain detection_guard step")
	}
	if !strings.Contains(detectionSection, "detection_conclusion") {
		t.Error("External detector path must contain detection_conclusion step")
	}
	if !strings.Contains(detectionSection, "id: parse_detection_token_usage") {
		t.Error("External detector path must contain parse_detection_token_usage step so detection AIC is exported")
	}
	parseIdx := strings.Index(detectionSection, "id: parse_detection_token_usage")
	concludeIdx := strings.Index(detectionSection, "id: detection_conclusion")
	if concludeIdx == -1 || parseIdx >= concludeIdx {
		t.Error("External detector path must emit parse_detection_token_usage before detection_conclusion so detection AIC is exported")
	}

	// The rw mount for the threat-detection directory must be present
	if !strings.Contains(detectionSection, "/tmp/gh-aw/threat-detection:/tmp/gh-aw/threat-detection:rw") {
		t.Error("External detector path must include read-write mount for /tmp/gh-aw/threat-detection")
	}

	// The detector invocation must pass the artifacts directory positionally and write a structured result file.
	invocationNeedle := "threat-detect --engine "
	invocationIndex := strings.Index(detectionSection, invocationNeedle)
	if invocationIndex == -1 {
		t.Error("External detector path must invoke threat-detect with --engine")
	} else {
		invocationLineEnd := strings.Index(detectionSection[invocationIndex:], "\n")
		if invocationLineEnd == -1 {
			invocationLineEnd = len(detectionSection) - invocationIndex
		}
		invocationLine := detectionSection[invocationIndex : invocationIndex+invocationLineEnd]
		if !strings.Contains(invocationLine, " /tmp/gh-aw/threat-detection") {
			t.Error("External detector path must pass /tmp/gh-aw/threat-detection as the positional artifacts directory")
		}
	}
	if !strings.Contains(detectionSection, "--output /tmp/gh-aw/threat-detection/detection_result.json") {
		t.Error("External detector path must pass --output /tmp/gh-aw/threat-detection/detection_result.json to threat-detect")
	}

	// The AWF execution pipeline must preserve non-zero threat-detect exits.
	if !strings.Contains(detectionSection, "set -o pipefail") {
		t.Error("External detector AWF step must use set -o pipefail so non-zero threat-detect exits fail the step")
	}

	// The external detector run must inherit engine runtime env config (auth/model/etc).
	if !strings.Contains(detectionSection, "COPILOT_GITHUB_TOKEN:") {
		t.Error("External detector path must configure engine auth env like the agent job")
	}

}

func TestExternalDetectorPathUsesCopilotForPiWorkflows(t *testing.T) {
	compiler := NewCompiler()

	tmpDir := testutil.TempDir(t, "test-external-detector-pi-*")
	workflowPath := filepath.Join(tmpDir, "test-external-detector-pi.md")

	workflowContent := `---
on: push
engine:
  id: pi
  model: copilot/gpt-5.4
safe-outputs:
  create-issue:
features:
  gh-aw-detection: true
tools:
  github:
    mode: gh-proxy
  cli-proxy: true
---
Test workflow`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	detectionSection := extractJobSection(string(result), "detection")
	if detectionSection == "" {
		t.Fatal("Detection job not found in compiled workflow")
	}

	if !strings.Contains(detectionSection, "install_copilot_cli.sh") {
		t.Error("Pi external detector path must install the Copilot engine")
	}
	if strings.Contains(detectionSection, "@earendil-works/pi-coding-agent") {
		t.Error("Pi external detector path must not install the Pi engine")
	}
	if !strings.Contains(detectionSection, "threat-detect --engine copilot") {
		t.Error("Pi external detector path must invoke threat-detect with the copilot engine")
	}
	if strings.Contains(detectionSection, "threat-detect --engine pi") {
		t.Error("Pi external detector path must not invoke threat-detect with the pi engine")
	}
	if !strings.Contains(detectionSection, "COPILOT_GITHUB_TOKEN:") {
		t.Error("Pi external detector path must inherit Copilot auth env")
	}
}

func TestInlineDetectionUsesCopilotForPiWorkflows(t *testing.T) {
	compiler := NewCompiler()

	tmpDir := testutil.TempDir(t, "test-inline-detector-pi-*")
	workflowPath := filepath.Join(tmpDir, "test-inline-detector-pi.md")

	workflowContent := `---
on: push
engine:
  id: pi
  model: copilot/gpt-5.4
safe-outputs:
  create-issue:
tools:
  github:
    mode: gh-proxy
  cli-proxy: true
---
Test workflow`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	detectionSection := extractJobSection(string(result), "detection")
	if detectionSection == "" {
		t.Fatal("Detection job not found in compiled workflow")
	}

	if !strings.Contains(detectionSection, "install_copilot_cli.sh") {
		t.Error("Pi inline detection path must install the Copilot engine")
	}
	if strings.Contains(detectionSection, "@earendil-works/pi-coding-agent") {
		t.Error("Pi inline detection path must not install the Pi engine")
	}
	if !strings.Contains(detectionSection, "COPILOT_GITHUB_TOKEN:") {
		t.Error("Pi inline detection path must inherit Copilot auth env")
	}
}

func TestExternalDetectorPathPreparesCodexConfig(t *testing.T) {
	compiler := NewCompiler()

	tmpDir := testutil.TempDir(t, "test-external-detector-codex-*")
	workflowPath := filepath.Join(tmpDir, "test-external-detector-codex.md")

	workflowContent := `---
on: push
engine: codex
safe-outputs:
  create-issue:
features:
  gh-aw-detection: true
tools:
  github:
    mode: gh-proxy
---
Test workflow`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	detectionSection := extractJobSection(string(result), "detection")
	if detectionSection == "" {
		t.Fatal("Detection job not found in compiled workflow")
	}

	if !strings.Contains(detectionSection, "Prepare Codex config for threat-detect") {
		t.Error("Codex external detector path must prepare Codex config files before execution")
	}
	if !strings.Contains(detectionSection, string(constants.ShellMcpServersJsonPath)) {
		t.Error("Codex external detector path must create an empty mcp-servers.json for Codex")
	}
	if !strings.Contains(detectionSection, string(constants.TmpMcpConfigDir)+"/config.toml") {
		t.Error("Codex external detector path must create a writable CODEX_HOME config.toml")
	}
	if !strings.Contains(detectionSection, "model_provider = \"openai-proxy\"") {
		t.Error("Codex external detector path must route Codex through the AWF OpenAI proxy")
	}
	if !strings.Contains(detectionSection, "api_base = \"http://") {
		t.Error("Codex external detector path must set api_base in config.toml")
	}
	if !strings.Contains(detectionSection, "wss_base = \"ws://") {
		t.Error("Codex external detector path must set wss_base in config.toml")
	}
	if !strings.Contains(detectionSection, "supports_websockets = false") {
		t.Error("Codex external detector path must disable websocket startup for the proxy config")
	}
}

// TestExternalDetectorCodexConfigModelProviderAtRoot verifies that the top-level
// model_provider selector is emitted before any TOML table header ([history],
// [model_providers.*]). If it appears after [history], TOML parses it as
// history.model_provider, which Codex ignores, causing it to fall back to the
// default openai provider and bypass the AWF api-proxy sidecar (401 Unauthorized).
func TestExternalDetectorCodexConfigModelProviderAtRoot(t *testing.T) {
	config := buildExternalDetectorCodexConfig("http://172.30.0.30:10000", "ws://172.30.0.30:10000")

	// Strip any leading whitespace from each line so the config reads as plain
	// TOML regardless of the exact indentation used in the source template.
	var trimmedLines []string
	for line := range strings.SplitSeq(config, "\n") {
		trimmedLines = append(trimmedLines, strings.TrimLeft(line, " \t"))
	}
	toml := strings.Join(trimmedLines, "\n")

	expected := "model_provider = \"" + codexOpenAIProxyProviderID + "\""
	modelProviderIndex := strings.Index(toml, expected)
	if modelProviderIndex == -1 {
		t.Fatalf("Expected model_provider selector in detection config, got:\n%s", toml)
	}

	// Find the first real TOML table header by scanning line-by-line; this
	// avoids matching `[` that appears inside quoted string values or comments.
	firstTableIndex := -1
	lineStart := 0
	for line := range strings.SplitSeq(toml, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstTableIndex = lineStart
			break
		}
		lineStart += len(line) + 1 // +1 for the newline
	}
	if firstTableIndex == -1 {
		t.Fatalf("Expected at least one TOML table header in detection config, got:\n%s", toml)
	}
	// The top-level model_provider must precede any table header so TOML assigns
	// it to the document root rather than the most recent table.
	if modelProviderIndex > firstTableIndex {
		t.Errorf("model_provider selector must appear before any table header (e.g. [history]) so TOML assigns it to the document root, got:\n%s", toml)
	}

	// Guard against regression: the [history] table section must not contain a
	// model_provider key. Extract the [history] body line-by-line so that `[`
	// characters inside quoted values or arrays do not prematurely truncate it.
	var historyLines []string
	inHistory := false
	for line := range strings.SplitSeq(toml, "\n") {
		if line == "[history]" {
			inHistory = true
			continue
		}
		if inHistory {
			if strings.HasPrefix(strings.TrimSpace(line), "[") {
				break
			}
			historyLines = append(historyLines, line)
		}
	}
	if !inHistory {
		t.Fatalf("Expected [history] table in detection config, got:\n%s", toml)
	}
	historyBody := strings.Join(historyLines, "\n")
	if strings.Contains(historyBody, "model_provider") {
		t.Errorf("[history] table must NOT contain a model_provider key, got:\n%s", historyBody)
	}
}

// TestExternalDetectorCodexFirewallDomains verifies that the Codex external detection
// path includes the required API domains in the AWF firewall allowlist.
// Without the correct allowed domains the firewall blocks api.openai.com, chatgpt.com
// and api.github.com and the detection job exits with code 1/2 (engine_error).
func TestExternalDetectorCodexFirewallDomains(t *testing.T) {
	compiler := NewCompiler()

	tmpDir := testutil.TempDir(t, "test-external-detector-codex-fw-*")
	workflowPath := filepath.Join(tmpDir, "test-codex-fw.md")

	workflowContent := `---
on: push
engine: codex
safe-outputs:
  create-issue:
features:
  gh-aw-detection: true
tools:
  github:
    mode: gh-proxy
---
Test workflow`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	detectionSection := extractJobSection(string(result), "detection")
	if detectionSection == "" {
		t.Fatal("Detection job not found in compiled workflow")
	}

	allowDomainsLine := ""
	for line := range strings.SplitSeq(detectionSection, "\n") {
		if strings.Contains(line, "allowDomains") {
			allowDomainsLine = line
			break
		}
	}
	if allowDomainsLine == "" {
		t.Fatal("Detection job AWF config must include network.allowDomains")
	}

	// Assert against the specific AWF allowDomains JSON line to avoid false positives
	// from unrelated occurrences elsewhere in the rendered detection job.
	for _, domain := range []string{"api.openai.com", "chatgpt.com", "openai.com", "github.com", "api.github.com"} {
		if !strings.Contains(allowDomainsLine, domain) {
			t.Errorf("Codex external detector AWF allowDomains must include %q", domain)
		}
	}
}
