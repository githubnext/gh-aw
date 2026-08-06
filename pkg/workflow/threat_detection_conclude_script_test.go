package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestConcludeThreatDetectionScript_MissingBinaryWarnMode verifies that the
// script's sole remaining shell-side special case — threat-detect missing
// from PATH — warns and exits 0 when GH_AW_DETECTION_CONTINUE_ON_ERROR is not
// exactly "false" (warn mode).
func TestConcludeThreatDetectionScript_MissingBinaryWarnMode(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	emptyBinDir := filepath.Join(tmpDir, "empty-bin")

	if err := os.MkdirAll(emptyBinDir, 0755); err != nil {
		t.Fatalf("failed to create empty bin dir: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"GH_AW_DETECTION_CONTINUE_ON_ERROR=true",
		"GITHUB_OUTPUT="+outputFile,
		"PATH="+emptyBinDir+":"+os.Getenv("PATH"),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script should exit 0 in warn mode when threat-detect is missing: %v\nOutput: %s", err, out)
	}

	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read GITHUB_OUTPUT: %v", err)
	}
	outputText := string(outputData)
	if !strings.Contains(outputText, "conclusion=warning") {
		t.Fatalf("expected warning conclusion in GITHUB_OUTPUT, got: %s", outputText)
	}
	if !strings.Contains(outputText, "success=false") {
		t.Fatalf("expected success=false in GITHUB_OUTPUT, got: %s", outputText)
	}
	if !strings.Contains(outputText, "reason=agent_failure") {
		t.Fatalf("expected reason=agent_failure in GITHUB_OUTPUT, got: %s", outputText)
	}
	if !strings.Contains(string(out), "threat-detect binary not found on PATH") {
		t.Fatalf("expected warning message about missing binary, got: %s", out)
	}
}

// TestConcludeThreatDetectionScript_MissingBinaryStrictMode verifies the
// guard hard-fails (exit 1) when GH_AW_DETECTION_CONTINUE_ON_ERROR is
// exactly "false" (strict mode).
func TestConcludeThreatDetectionScript_MissingBinaryStrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	emptyBinDir := filepath.Join(tmpDir, "empty-bin")

	if err := os.MkdirAll(emptyBinDir, 0755); err != nil {
		t.Fatalf("failed to create empty bin dir: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"GH_AW_DETECTION_CONTINUE_ON_ERROR=false",
		"GITHUB_OUTPUT="+outputFile,
		"PATH="+emptyBinDir+":"+os.Getenv("PATH"),
	)

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("script should exit non-zero in strict mode when threat-detect is missing, output: %s", out)
	}
	if !strings.Contains(string(out), "ERR_SYSTEM:") {
		t.Fatalf("expected ERR_SYSTEM message, got: %s", out)
	}
	if !strings.Contains(string(out), "threat-detect binary not found on PATH") {
		t.Fatalf("expected message about missing binary, got: %s", out)
	}
}

// TestConcludeThreatDetectionScript_InvokesThreatDetectConclude verifies the
// script delegates to `threat-detect conclude` with both --result-file and
// an explicit --detection-log, defaulting the log path to
// <result-file-dir>/detection.log.
func TestConcludeThreatDetectionScript_InvokesThreatDetectConclude(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	envFile := filepath.Join(tmpDir, "github_env.txt")
	callLog := filepath.Join(tmpDir, "call.log")
	binDir := filepath.Join(tmpDir, "bin")

	if err := os.WriteFile(resultFile, []byte(`{"conclusion":"success"}`), 0644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	stubPath := filepath.Join(binDir, "threat-detect")
	stub := "#!/usr/bin/env bash\n" +
		"echo \"$*\" >> \"$CALL_LOG\"\n" +
		"echo \"conclusion=success\" >> \"$GITHUB_OUTPUT\"\n"
	if err := os.WriteFile(stubPath, []byte(stub), 0755); err != nil {
		t.Fatalf("failed to write threat-detect stub: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"GITHUB_OUTPUT="+outputFile,
		"GITHUB_ENV="+envFile,
		"CALL_LOG="+callLog,
		"PATH="+binDir+":"+os.Getenv("PATH"),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script failed: %v\nOutput: %s", err, out)
	}

	callData, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatalf("failed to read call log: %v", err)
	}
	expectedLog := filepath.Join(tmpDir, "detection.log")
	if !strings.Contains(string(callData), "conclude --result-file "+resultFile+" --detection-log "+expectedLog) {
		t.Fatalf("expected threat-detect conclude invocation with default detection log, got: %s", callData)
	}
}

// TestConcludeThreatDetectionScript_HonorsDetectionLogOverride verifies that
// an explicit DETECTION_LOG_FILE environment variable overrides the default
// <result-file-dir>/detection.log path passed to --detection-log.
func TestConcludeThreatDetectionScript_HonorsDetectionLogOverride(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	callLog := filepath.Join(tmpDir, "call.log")
	binDir := filepath.Join(tmpDir, "bin")
	customLog := filepath.Join(tmpDir, "custom-detection.log")

	if err := os.WriteFile(resultFile, []byte(`{"conclusion":"success"}`), 0644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	stubPath := filepath.Join(binDir, "threat-detect")
	stub := "#!/usr/bin/env bash\n" +
		"echo \"$*\" >> \"$CALL_LOG\"\n" +
		"echo \"conclusion=success\" >> \"$GITHUB_OUTPUT\"\n"
	if err := os.WriteFile(stubPath, []byte(stub), 0755); err != nil {
		t.Fatalf("failed to write threat-detect stub: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"GITHUB_OUTPUT="+outputFile,
		"CALL_LOG="+callLog,
		"DETECTION_LOG_FILE="+customLog,
		"PATH="+binDir+":"+os.Getenv("PATH"),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script failed: %v\nOutput: %s", err, out)
	}

	callData, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatalf("failed to read call log: %v", err)
	}
	if !strings.Contains(string(callData), "--detection-log "+customLog) {
		t.Fatalf("expected threat-detect conclude invocation with overridden detection log, got: %s", callData)
	}
}
