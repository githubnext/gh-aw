package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func isolatedPath(t *testing.T, binDir string) string {
	t.Helper()

	dirnamePath, err := exec.LookPath("dirname")
	if err != nil {
		t.Fatalf("failed to locate dirname: %v", err)
	}
	if err := os.Symlink(dirnamePath, filepath.Join(binDir, "dirname")); err != nil {
		t.Fatalf("failed to make dirname available in test PATH: %v", err)
	}
	return binDir
}

// TestConcludeThreatDetectionScript_MissingBinaryWarnMode verifies that the
// script's sole remaining shell-side special case — threat-detect missing
// from PATH — warns and exits 0 when GH_AW_DETECTION_CONTINUE_ON_ERROR is not
// "false", ignoring case (warn mode).
func TestConcludeThreatDetectionScript_MissingBinaryWarnMode(t *testing.T) {
	for _, continueOnError := range []string{"true", "True", "TRUE", "yes", "unset"} {
		t.Run(continueOnError, func(t *testing.T) {
			tmpDir := t.TempDir()
			scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
			outputFile := filepath.Join(tmpDir, "github_output.txt")
			resultFile := filepath.Join(tmpDir, "detection_result.json")
			binDir := filepath.Join(tmpDir, "bin")
			if err := os.MkdirAll(binDir, 0755); err != nil {
				t.Fatalf("failed to create test bin directory: %v", err)
			}

			cmd := exec.Command("bash", scriptPath, resultFile)
			for _, env := range os.Environ() {
				if !strings.HasPrefix(env, "GH_AW_DETECTION_CONTINUE_ON_ERROR=") {
					cmd.Env = append(cmd.Env, env)
				}
			}
			if continueOnError != "unset" {
				cmd.Env = append(cmd.Env, "GH_AW_DETECTION_CONTINUE_ON_ERROR="+continueOnError)
			}
			cmd.Env = append(cmd.Env,
				"RUN_DETECTION=true",
				"GITHUB_OUTPUT="+outputFile,
				"PATH="+isolatedPath(t, binDir),
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
			if !strings.Contains(outputText, "conclusion=warning") || !strings.Contains(outputText, "success=false") || !strings.Contains(outputText, "reason=agent_failure") {
				t.Fatalf("expected warning failure outputs in GITHUB_OUTPUT, got: %s", outputText)
			}
			if !strings.Contains(string(out), "threat-detect binary not found on PATH") {
				t.Fatalf("expected warning message about missing binary, got: %s", out)
			}
		})
	}
}

// TestConcludeThreatDetectionScript_MissingBinaryStrictMode verifies the
// guard hard-fails (exit 1) when GH_AW_DETECTION_CONTINUE_ON_ERROR is
// "false", ignoring case (strict mode).
func TestConcludeThreatDetectionScript_MissingBinaryStrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	binDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create test bin directory: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"GH_AW_DETECTION_CONTINUE_ON_ERROR=false",
		"GITHUB_OUTPUT="+outputFile,
		"PATH="+isolatedPath(t, binDir),
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
	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read GITHUB_OUTPUT: %v", err)
	}
	outputText := string(outputData)
	if !strings.Contains(outputText, "conclusion=failure") || !strings.Contains(outputText, "success=false") || !strings.Contains(outputText, "reason=agent_failure") {
		t.Fatalf("expected strict failure outputs in GITHUB_OUTPUT, got: %s", outputText)
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

func TestConcludeThreatDetectionScript_PropagatesThreatDetectFailure(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "threat-detect"), []byte("#!/bin/bash\nexit 1\n"), 0755); err != nil {
		t.Fatalf("failed to write threat-detect stub: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, filepath.Join(tmpDir, "detection_result.json"))
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("script should propagate threat-detect failure, output: %s", out)
	}
}

func TestConcludeThreatDetectionScript_ExecutionOutcomeModeCase(t *testing.T) {
	for _, mode := range []struct {
		name   string
		value  string
		set    bool
		strict bool
	}{
		{name: "false", value: "false", set: true, strict: true},
		{name: "False", value: "False", set: true, strict: true},
		{name: "FALSE", value: "FALSE", set: true, strict: true},
		{name: "unset"},
		{name: "empty", set: true},
		{name: "true", value: "true", set: true},
		{name: "True", value: "True", set: true},
		{name: "TRUE", value: "TRUE", set: true},
	} {
		for _, outcome := range []string{"failure", "cancelled", "success"} {
			for _, runDetection := range []string{"true", "false"} {
				t.Run(mode.name+"/"+outcome+"/run="+runDetection, func(t *testing.T) {
					dir := t.TempDir()
					binDir := filepath.Join(dir, "bin")
					if err := os.MkdirAll(binDir, 0755); err != nil {
						t.Fatal(err)
					}
					resultFile := filepath.Join(dir, "detection_result.json")
					if err := os.WriteFile(resultFile, []byte(`{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}`), 0644); err != nil {
						t.Fatal(err)
					}
					callLog := filepath.Join(dir, "call.log")
					outputFile := filepath.Join(dir, "github_output.txt")
					stub := "#!/usr/bin/env bash\n" +
						"echo \"$*\" >> \"$CALL_LOG\"\n" +
						"if [[ \"$RUN_DETECTION\" == false ]]; then\n" +
						"  echo 'conclusion=skipped' >> \"$GITHUB_OUTPUT\"\n" +
						"else\n" +
						"  echo 'conclusion=success' >> \"$GITHUB_OUTPUT\"\n" +
						"fi\n" +
						"echo 'success=true' >> \"$GITHUB_OUTPUT\"\n" +
						"echo 'reason=' >> \"$GITHUB_OUTPUT\"\n"
					if err := os.WriteFile(filepath.Join(binDir, "threat-detect"), []byte(stub), 0755); err != nil {
						t.Fatal(err)
					}
					cmd := exec.Command("bash", filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh"), resultFile)
					for _, env := range os.Environ() {
						if !strings.HasPrefix(env, "GH_AW_DETECTION_CONTINUE_ON_ERROR=") {
							cmd.Env = append(cmd.Env, env)
						}
					}
					if mode.set {
						cmd.Env = append(cmd.Env, "GH_AW_DETECTION_CONTINUE_ON_ERROR="+mode.value)
					}
					cmd.Env = append(cmd.Env,
						"RUN_DETECTION="+runDetection,
						"THREAT_DETECT_INSTALL_OUTCOME=success",
						"DETECTION_AGENTIC_EXECUTION_OUTCOME="+outcome,
						"GITHUB_OUTPUT="+outputFile,
						"CALL_LOG="+callLog,
						"PATH="+binDir+":"+os.Getenv("PATH"),
					)
					out, err := cmd.CombinedOutput()
					wantBlocked := mode.strict && runDetection == "true" && outcome != "success"
					if (err != nil) != wantBlocked {
						t.Fatalf("blocked=%t, got error %v\nOutput: %s", wantBlocked, err, out)
					}
					output, err := os.ReadFile(outputFile)
					if err != nil {
						t.Fatal(err)
					}
					wantConclusion, wantSuccess, wantReason := "success", "true", ""
					if wantBlocked {
						wantConclusion, wantSuccess, wantReason = "failure", "false", "agent_failure"
					} else if runDetection == "false" {
						wantConclusion = "skipped"
					}
					for _, expected := range []string{"conclusion=" + wantConclusion + "\n", "success=" + wantSuccess + "\n", "reason=" + wantReason + "\n"} {
						if !strings.Contains(string(output), expected) {
							t.Fatalf("expected %q, got: %s", expected, output)
						}
					}
					if _, err := os.Stat(callLog); os.IsNotExist(err) != wantBlocked {
						t.Fatalf("detector invocation should be blocked=%t, got stat error %v", wantBlocked, err)
					}
				})
			}
		}
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

// TestConcludeThreatDetectionScript_FailedInstallDoesNotInvokeDetector verifies
// that a non-success threat-detect install outcome reports the agent_failure
// conclusion directly, without executing any binary found on PATH. A binary
// present after a failed install is either unverified or the install script's
// fail-closed placeholder, and the placeholder cannot report a conclusion itself.
func TestConcludeThreatDetectionScript_FailedInstallDoesNotInvokeDetector(t *testing.T) {
	tests := []struct {
		name            string
		continueOnError string
		installOutcome  string
		wantErr         bool
		wantConclusion  string
	}{
		{name: "warn mode failure", continueOnError: "true", installOutcome: "failure", wantErr: false, wantConclusion: "conclusion=warning"},
		{name: "warn mode cancelled", continueOnError: "true", installOutcome: "cancelled", wantErr: false, wantConclusion: "conclusion=warning"},
		{name: "strict mode failure", continueOnError: "false", installOutcome: "failure", wantErr: true, wantConclusion: "conclusion=failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
			resultFile := filepath.Join(tmpDir, "detection_result.json")
			outputFile := filepath.Join(tmpDir, "github_output.txt")
			callLog := filepath.Join(tmpDir, "call.log")
			binDir := filepath.Join(tmpDir, "bin")

			if err := os.MkdirAll(binDir, 0755); err != nil {
				t.Fatalf("failed to create bin dir: %v", err)
			}
			// Stand in for an unverified binary left on PATH (preinstalled copy or
			// the install script's fail-closed placeholder).
			stub := "#!/usr/bin/env bash\necho \"$*\" >> \"$CALL_LOG\"\n"
			if err := os.WriteFile(filepath.Join(binDir, "threat-detect"), []byte(stub), 0755); err != nil {
				t.Fatalf("failed to write threat-detect stub: %v", err)
			}

			cmd := exec.Command("bash", scriptPath, resultFile)
			cmd.Env = append(os.Environ(),
				"RUN_DETECTION=true",
				"THREAT_DETECT_INSTALL_OUTCOME="+tt.installOutcome,
				"GH_AW_DETECTION_CONTINUE_ON_ERROR="+tt.continueOnError,
				"GITHUB_OUTPUT="+outputFile,
				"CALL_LOG="+callLog,
				"PATH="+binDir+":"+os.Getenv("PATH"),
			)

			out, err := cmd.CombinedOutput()
			if tt.wantErr && err == nil {
				t.Fatalf("script should exit non-zero in strict mode, output: %s", out)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("script should exit 0 in warn mode: %v\nOutput: %s", err, out)
			}

			if _, statErr := os.Stat(callLog); statErr == nil {
				t.Fatal("script must not invoke threat-detect when installation did not succeed")
			}

			outputData, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read GITHUB_OUTPUT: %v", err)
			}
			outputText := string(outputData)
			if !strings.Contains(outputText, tt.wantConclusion) || !strings.Contains(outputText, "success=false") || !strings.Contains(outputText, "reason=agent_failure") {
				t.Fatalf("expected %q with agent_failure outputs, got: %s", tt.wantConclusion, outputText)
			}
			if !strings.Contains(string(out), "threat-detect installation did not complete verification") {
				t.Fatalf("expected message about incomplete installation, got: %s", out)
			}
		})
	}
}

// TestConcludeThreatDetectionScript_SuccessfulInstallDelegates verifies that a
// successful install outcome leaves the normal delegation path intact.
func TestConcludeThreatDetectionScript_SuccessfulInstallDelegates(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	callLog := filepath.Join(tmpDir, "call.log")
	binDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	stub := "#!/usr/bin/env bash\n" +
		"echo \"$*\" >> \"$CALL_LOG\"\n" +
		"echo \"conclusion=success\" >> \"$GITHUB_OUTPUT\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "threat-detect"), []byte(stub), 0755); err != nil {
		t.Fatalf("failed to write threat-detect stub: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=true",
		"THREAT_DETECT_INSTALL_OUTCOME=success",
		"GITHUB_OUTPUT="+outputFile,
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
		t.Fatalf("expected threat-detect conclude invocation after a successful install, got: %s", callData)
	}
}

// TestConcludeThreatDetectionScript_SkippedDetectionIgnoresInstallOutcome verifies
// that a skipped install does not prevent delegation when the binary is available.
func TestConcludeThreatDetectionScript_SkippedDetectionIgnoresInstallOutcome(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh")
	resultFile := filepath.Join(tmpDir, "detection_result.json")
	outputFile := filepath.Join(tmpDir, "github_output.txt")
	callLog := filepath.Join(tmpDir, "call.log")
	binDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	stub := "#!/usr/bin/env bash\n" +
		"echo \"$*\" >> \"$CALL_LOG\"\n" +
		"echo \"conclusion=skipped\" >> \"$GITHUB_OUTPUT\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "threat-detect"), []byte(stub), 0755); err != nil {
		t.Fatalf("failed to write threat-detect stub: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, resultFile)
	cmd.Env = append(os.Environ(),
		"RUN_DETECTION=false",
		"THREAT_DETECT_INSTALL_OUTCOME=skipped",
		"GITHUB_OUTPUT="+outputFile,
		"CALL_LOG="+callLog,
		"PATH="+binDir+":"+os.Getenv("PATH"),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script failed: %v\nOutput: %s", err, out)
	}
	if _, statErr := os.Stat(callLog); statErr != nil {
		t.Fatalf("expected threat-detect conclude to run for a skipped detection: %v\nOutput: %s", statErr, out)
	}
}

func TestConcludeThreatDetectionScript_SkippedDetectionWithoutBinary(t *testing.T) {
	for _, tt := range []struct {
		name            string
		runDetection    string
		setRunDetection bool
		installOutcome  string
		continueOnError string
		wantErr         bool
		wantConclusion  string
		wantSuccess     string
	}{
		{name: "strict mode explicit skip", runDetection: "false", setRunDetection: true, installOutcome: "skipped", continueOnError: "false", wantConclusion: "conclusion=skipped", wantSuccess: "success=true"},
		{name: "uppercase strict mode explicit skip", runDetection: "false", setRunDetection: true, installOutcome: "skipped", continueOnError: "FALSE", wantConclusion: "conclusion=skipped", wantSuccess: "success=true"},
		{name: "warn mode explicit skip", runDetection: "false", setRunDetection: true, installOutcome: "skipped", continueOnError: "true", wantConclusion: "conclusion=skipped", wantSuccess: "success=true"},
		{name: "unset guard output", installOutcome: "skipped", continueOnError: "false", wantErr: true, wantConclusion: "conclusion=failure", wantSuccess: "success=false"},
		{name: "empty guard output", runDetection: "", setRunDetection: true, installOutcome: "skipped", continueOnError: "false", wantErr: true, wantConclusion: "conclusion=failure", wantSuccess: "success=false"},
		{name: "unexpected guard output", runDetection: "unexpected", setRunDetection: true, installOutcome: "skipped", continueOnError: "false", wantErr: true, wantConclusion: "conclusion=failure", wantSuccess: "success=false"},
		{name: "empty guard output warn mode", runDetection: "", setRunDetection: true, installOutcome: "skipped", continueOnError: "true", wantConclusion: "conclusion=warning", wantSuccess: "success=false"},
		{name: "explicit skip without skipped install", runDetection: "false", setRunDetection: true, continueOnError: "false", wantErr: true, wantConclusion: "conclusion=failure", wantSuccess: "success=false"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			binDir := filepath.Join(tmpDir, "bin")
			if err := os.MkdirAll(binDir, 0755); err != nil {
				t.Fatalf("failed to create bin directory: %v", err)
			}
			outputFile := filepath.Join(tmpDir, "github_output.txt")
			cmd := exec.Command("bash", filepath.Join("..", "..", "actions", "setup", "sh", "conclude_threat_detection.sh"), filepath.Join(tmpDir, "detection_result.json"))
			cmd.Env = make([]string, 0, len(os.Environ())+5)
			for _, env := range os.Environ() {
				if !strings.HasPrefix(env, "RUN_DETECTION=") {
					cmd.Env = append(cmd.Env, env)
				}
			}
			if tt.setRunDetection {
				cmd.Env = append(cmd.Env, "RUN_DETECTION="+tt.runDetection)
			}
			cmd.Env = append(cmd.Env,
				"THREAT_DETECT_INSTALL_OUTCOME="+tt.installOutcome,
				"DETECTION_AGENTIC_EXECUTION_OUTCOME=skipped",
				"GH_AW_DETECTION_CONTINUE_ON_ERROR="+tt.continueOnError,
				"GITHUB_OUTPUT="+outputFile,
				"PATH="+isolatedPath(t, binDir),
			)
			out, err := cmd.CombinedOutput()
			if tt.wantErr && err == nil {
				t.Fatalf("script should exit non-zero for an unavailable detector: %s", out)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("script should exit 0: %v\nOutput: %s", err, out)
			}
			outputData, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read GITHUB_OUTPUT: %v", err)
			}
			outputText := string(outputData)
			if !strings.Contains(outputText, tt.wantConclusion) || !strings.Contains(outputText, tt.wantSuccess) {
				t.Fatalf("unexpected detector conclusion outputs: %s", outputData)
			}
			if tt.wantSuccess == "success=true" && !strings.Contains(outputText, "reason=\n") {
				t.Fatalf("expected an empty reason for skipped detection, got: %s", outputData)
			}
			if tt.wantSuccess == "success=false" && !strings.Contains(outputText, "reason=agent_failure") {
				t.Fatalf("expected unavailable detector reason, got: %s", outputData)
			}
			if tt.wantSuccess == "success=true" && strings.Contains(string(out), "threat-detect binary not found") {
				t.Fatalf("skipped detection should not report a missing detector: %s", out)
			}
		})
	}
}
