package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const detectorStatusLine = "THREAT_DETECTION_STATUS: reason=engine_error exit=2"

func TestConcludeThreatDetectionScript_MissingResultContinueOnError(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	outputFile := filepath.Join(t.TempDir(), "github_output.txt")
	envFile := filepath.Join(t.TempDir(), "github_env.txt")
	missingResult := filepath.Join(t.TempDir(), "missing_detection_result.json")

	cmd := exec.Command("bash", scriptPath, missingResult)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"DETECTION_AGENTIC_EXECUTION_OUTCOME=failure",
		"GH_AW_DETECTION_CONTINUE_ON_ERROR=TRUE",
		"GITHUB_OUTPUT="+outputFile,
		"GITHUB_ENV="+envFile,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script should continue on missing result when continue-on-error is true: %v\nOutput: %s", err, out)
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

	envData, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("failed to read GITHUB_ENV: %v", err)
	}
	envText := string(envData)
	if !strings.Contains(envText, "GH_AW_DETECTION_CONCLUSION=warning") {
		t.Fatalf("expected warning conclusion in GITHUB_ENV, got: %s", envText)
	}
	if !strings.Contains(envText, "GH_AW_DETECTION_REASON=agent_failure") {
		t.Fatalf("expected agent_failure reason in GITHUB_ENV, got: %s", envText)
	}
	if !strings.Contains(string(out), "continuing because GH_AW_DETECTION_CONTINUE_ON_ERROR=true") {
		t.Fatalf("expected warning message about continue-on-error, got: %s", out)
	}
}

func TestConcludeThreatDetectionScript_MissingResultSurfacesDetectorStatus(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	outputFile := filepath.Join(tempDir, "github_output.txt")
	envFile := filepath.Join(tempDir, "github_env.txt")
	missingResult := filepath.Join(tempDir, "missing_detection_result.json")
	detectionLog := filepath.Join(tempDir, "detection.log")

	if err := os.WriteFile(detectionLog, []byte(detectorStatusLine+"\n"), 0600); err != nil {
		t.Fatalf("failed to write detection log: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, missingResult)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"DETECTION_AGENTIC_EXECUTION_OUTCOME=failure",
		"GH_AW_DETECTION_CONTINUE_ON_ERROR=true",
		"DETECTION_LOG_FILE="+detectionLog,
		"GITHUB_OUTPUT="+outputFile,
		"GITHUB_ENV="+envFile,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script should continue when detector status is available: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(string(out), detectorStatusLine) {
		t.Fatalf("expected warning output to surface detector status, got: %s", out)
	}
	if !strings.Contains(string(out), "execution outcome: failure") {
		t.Fatalf("expected warning output to include execution outcome, got: %s", out)
	}

	envData, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("failed to read GITHUB_ENV: %v", err)
	}
	envText := string(envData)
	if !strings.Contains(envText, "GH_AW_DETECTION_CONCLUSION=warning") {
		t.Fatalf("expected warning conclusion in GITHUB_ENV, got: %s", envText)
	}
	if !strings.Contains(envText, "GH_AW_DETECTION_REASON=agent_failure") {
		t.Fatalf("expected agent_failure reason in GITHUB_ENV, got: %s", envText)
	}
}

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
	if !strings.Contains(string(callData), "conclude --result-file "+resultFile) {
		t.Fatalf("expected threat-detect conclude invocation, got: %s", callData)
	}
}
