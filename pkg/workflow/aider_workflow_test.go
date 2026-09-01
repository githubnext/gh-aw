//go:build !integration && !windows

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAiderHarnessPreservesProcessFailureDetails(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/shared/aider.md")
	if err != nil {
		t.Fatalf("failed to read shared Aider workflow: %v", err)
	}

	var frontmatter struct {
		Engine EngineDefinition `yaml:"engine"`
	}
	parts := strings.SplitN(string(sourceContent), "---", 3)
	if len(parts) != 3 {
		t.Fatal("shared Aider workflow has invalid frontmatter")
	}
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		t.Fatalf("failed to parse shared Aider workflow: %v", err)
	}
	if frontmatter.Engine.Behaviors == nil {
		t.Fatal("shared Aider workflow has no engine behaviors")
	}

	tempDir := t.TempDir()
	harnessPath := filepath.Join(tempDir, "aider_harness.cjs")
	if err := os.WriteFile(harnessPath, []byte(frontmatter.Engine.Behaviors.HarnessScript), 0o600); err != nil {
		t.Fatalf("failed to write Aider harness: %v", err)
	}
	promptPath := filepath.Join(tempDir, "prompt.md")
	if err := os.WriteFile(promptPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}

	runHarness := func(t *testing.T, command string) string {
		t.Helper()
		cmd := exec.Command("node", harnessPath, command)
		cmd.Env = append(os.Environ(), "GH_AW_PROMPT="+promptPath)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected Aider harness to fail")
		}
		return string(output)
	}

	t.Run("spawn error", func(t *testing.T) {
		output := runHarness(t, filepath.Join(tempDir, "missing-aider"))
		if !strings.Contains(output, "ENOENT") {
			t.Fatalf("expected spawn error detail, got:\n%s", output)
		}
	})

	exitPath := filepath.Join(tempDir, "exit-aider")
	if err := os.WriteFile(exitPath, []byte("#!/bin/sh\nexit 7\n"), 0o700); err != nil {
		t.Fatalf("failed to write exit fixture: %v", err)
	}
	t.Run("exit code", func(t *testing.T) {
		output := runHarness(t, exitPath)
		if !strings.Contains(output, "Aider execution failed with exit code 7") {
			t.Fatalf("expected exit code detail, got:\n%s", output)
		}
	})

	signalPath := filepath.Join(tempDir, "signal-aider")
	if err := os.WriteFile(signalPath, []byte("#!/bin/sh\nkill -TERM $$\n"), 0o700); err != nil {
		t.Fatalf("failed to write signal fixture: %v", err)
	}
	t.Run("signal", func(t *testing.T) {
		output := runHarness(t, signalPath)
		if !strings.Contains(output, "Aider execution failed with signal SIGTERM") {
			t.Fatalf("expected signal detail, got:\n%s", output)
		}
	})

	liteLLMPath := filepath.Join(tempDir, "litellm-aider")
	if err := os.WriteFile(liteLLMPath, []byte("#!/bin/sh\necho 'litellm.APIError: unavailable' >&2\n"), 0o700); err != nil {
		t.Fatalf("failed to write LiteLLM fixture: %v", err)
	}
	t.Run("LiteLLM error", func(t *testing.T) {
		output := runHarness(t, liteLLMPath)
		if !strings.Contains(output, "Aider execution reported a LiteLLM error") {
			t.Fatalf("expected LiteLLM error detail, got:\n%s", output)
		}
	})

	for _, path := range []string{
		"../../.github/workflows/shared/aider.md",
		"../../.github/workflows/smoke-aider.lock.yml",
		"../../.github/workflows/daily-code-debt-aider.lock.yml",
		"../../.github/workflows/daily-go-test-stubs-aider.lock.yml",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read Aider workflow file %s: %v", path, err)
		}
		config := string(content)

		for _, expected := range []string{
			"if (result.error) throw result.error;",
			"`signal ${result.signal}`",
			"`exit code ${result.status ?? \"unknown\"}`",
			"`${action} reported a LiteLLM error`",
		} {
			if !strings.Contains(config, expected) {
				t.Errorf("expected %s to preserve Aider failure detail %q", path, expected)
			}
		}
	}
}

func TestAiderWorkflowsUseSafeoutputsCLI(t *testing.T) {
	for _, path := range []string{
		"../../.github/workflows/daily-code-debt-aider.md",
		"../../.github/workflows/daily-go-test-stubs-aider.md",
		"../../.github/workflows/smoke-aider.md",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read Aider workflow %s: %v", path, err)
		}
		workflow := string(content)
		if !strings.Contains(workflow, "safeoutputs ") {
			t.Errorf("expected workflow %s to use the safeoutputs CLI", path)
		}
		if strings.Contains(workflow, "GH_AW_SAFE_OUTPUTS") {
			t.Errorf("workflow %s must not write directly to GH_AW_SAFE_OUTPUTS", path)
		}

		lockPath := strings.TrimSuffix(path, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("failed to read Aider lock file %s: %v", lockPath, err)
		}
		lock := string(lockContent)
		if !strings.Contains(lock, `GH_AW_MCP_CLI_SERVERS='["safeoutputs"]'`) {
			t.Errorf("expected safeoutputs MCP CLI to be mounted for %s", path)
		}
		if strings.Contains(lock, `--mount "${RUNNER_TEMP}/gh-aw/safeoutputs:${RUNNER_TEMP}/gh-aw/safeoutputs:rw"`) {
			t.Errorf("Aider execution must not mount the safe-output directory read-write for %s", path)
		}
	}
}
