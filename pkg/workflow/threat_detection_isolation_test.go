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
	if !strings.Contains(detectionSection, "threat-detect conclude") {
		t.Error("External detector path must emit 'threat-detect conclude' as the conclude step")
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
