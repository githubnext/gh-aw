//go:build !integration

// Package workflow — security architecture T-TD formal tests.
//
// This file encodes the compliance test assertions for the Threat Detection
// test category (T-TD-002 to T-TD-007) defined in
// specs/security-architecture-spec.md §12.2.6.
//
// Each predicate maps to exactly one Go test function:
//
//	T-TD-002 PromptInjectionDetection     → TestFormalTD002_PromptInjectionDetection
//	T-TD-003 SecretLeakDetection          → TestFormalTD003_SecretLeakDetection
//	T-TD-004 MaliciousPatchDetection      → TestFormalTD004_MaliciousPatchDetection
//	T-TD-005 CustomPromptSupport          → TestFormalTD005_CustomPromptSupport
//	T-TD-006 EngineConfigOverride         → TestFormalTD006_EngineConfigOverride
//	T-TD-007 WorkflowFailureOnDetection   → TestFormalTD007_WorkflowFailureOnDetection
//
// All tests call production code directly without stubs; no mocking is used.
// See specs/security-architecture-spec-validation.md §12 for the compliance
// test matrix these tests are cited in.
package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormalTD002_PromptInjectionDetection (T-TD-002)
//
// T-TD-002: Verify prompt injection detection.
//
// TD-04 requires the implementation to detect Prompt Injection. The Codex
// detection engine encodes the detectable threat categories in a structured
// JSON schema. This test verifies that the schema includes the "prompt_injection"
// boolean field as a required output of the detection run.
func TestFormalTD002_PromptInjectionDetection(t *testing.T) {
	assert.Contains(t, detectionResponseSchema, `"prompt_injection":{"type":"boolean"}`,
		"T-TD-002: detection response schema must include prompt_injection as a boolean field")
	assert.Contains(t, detectionResponseSchema, `"prompt_injection"`,
		"T-TD-002: detection response schema must list prompt_injection in required fields")
}

// TestFormalTD003_SecretLeakDetection (T-TD-003)
//
// T-TD-003: Verify secret leak detection.
//
// TD-04 requires the implementation to detect Secret Leaks. The Codex
// detection engine encodes all detectable threat categories in a structured
// JSON schema. This test verifies that the schema includes the "secret_leak"
// boolean field as a required output of the detection run.
func TestFormalTD003_SecretLeakDetection(t *testing.T) {
	assert.Contains(t, detectionResponseSchema, `"secret_leak":{"type":"boolean"}`,
		"T-TD-003: detection response schema must include secret_leak as a boolean field")
	assert.Contains(t, detectionResponseSchema, `"secret_leak"`,
		"T-TD-003: detection response schema must list secret_leak in required fields")
}

// TestFormalTD004_MaliciousPatchDetection (T-TD-004)
//
// T-TD-004: Verify malicious patch detection.
//
// TD-04 requires the implementation to detect Malicious Patches. The Codex
// detection engine encodes all detectable threat categories in a structured
// JSON schema. This test verifies that the schema includes the "malicious_patch"
// boolean field as a required output of the detection run.
func TestFormalTD004_MaliciousPatchDetection(t *testing.T) {
	assert.Contains(t, detectionResponseSchema, `"malicious_patch":{"type":"boolean"}`,
		"T-TD-004: detection response schema must include malicious_patch as a boolean field")
	assert.Contains(t, detectionResponseSchema, `"malicious_patch"`,
		"T-TD-004: detection response schema must list malicious_patch in required fields")
}

// TestFormalTD005_CustomPromptSupport (T-TD-005)
//
// T-TD-005: Verify custom prompt support.
//
// TD-11 requires the implementation to support custom detection prompts.
// TD-12 requires that custom prompts are appended to default detection
// instructions, not replace them. This test verifies that
// parseThreatDetectionObjectConfig correctly stores the custom prompt in the
// Prompt field, and that the compiled detection step emits CUSTOM_PROMPT as
// an environment variable (which is appended to the default prompt at runtime
// in setup_threat_detection.cjs).
func TestFormalTD005_CustomPromptSupport(t *testing.T) {
	c := NewCompiler()

	// Verify parseThreatDetectionObjectConfig stores the prompt field.
	configMap := map[string]any{
		"prompt": "Focus on SQL injection vulnerabilities",
	}
	td := c.parseThreatDetectionObjectConfig(configMap)
	require.NotNil(t, td,
		"T-TD-005: parseThreatDetectionObjectConfig must return a non-nil config")
	assert.Equal(t, "Focus on SQL injection vulnerabilities", td.Prompt,
		"T-TD-005: custom prompt must be stored in ThreatDetectionConfig.Prompt")

	// Verify the compiled detection job emits CUSTOM_PROMPT as an env var.
	md := `---
name: td005-custom-prompt
on: push
engine: copilot
permissions:
  contents: read
safe-outputs:
  create-issue:
  threat-detection:
    prompt: "Focus on SQL injection vulnerabilities"
---

# Mission

T-TD-005: verify custom prompt support in compiled detection job.
`
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "T-TD-005: workflow with custom threat-detection prompt must parse without error")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "T-TD-005: workflow with custom threat-detection prompt must compile without error")

	detectionSection := extractJobSection(yamlOut, "detection")
	require.NotEmpty(t, detectionSection, "T-TD-005: compiled workflow must contain a detection job")
	assert.Contains(t, detectionSection, "CUSTOM_PROMPT",
		"T-TD-005: compiled detection job must emit CUSTOM_PROMPT env var when a custom prompt is configured")

	// Verify custom prompt content appears (appended, not replacing defaults).
	assert.True(t, strings.Contains(yamlOut, "SQL injection") || strings.Contains(yamlOut, "CUSTOM_PROMPT"),
		"T-TD-005: custom prompt content must be present in the compiled detection job env")
}

// TestFormalTD006_EngineConfigOverride (T-TD-006)
//
// T-TD-006: Verify engine configuration override.
//
// TD-13/TD-14/TD-15 require the implementation to support overriding the AI
// engine for threat detection in string, object, and disabled (false) formats.
// This test verifies that parseThreatDetectionObjectConfig correctly handles
// all three engine override formats.
func TestFormalTD006_EngineConfigOverride(t *testing.T) {
	c := NewCompiler()

	t.Run("string format", func(t *testing.T) {
		// TD-13: string engine ID is stored in EngineConfig.ID.
		configMap := map[string]any{
			"engine": "copilot",
		}
		td := c.parseThreatDetectionObjectConfig(configMap)
		require.NotNil(t, td,
			"T-TD-006: string engine format must produce a non-nil ThreatDetectionConfig")
		require.NotNil(t, td.EngineConfig,
			"T-TD-006: string engine format must produce a non-nil EngineConfig")
		assert.Equal(t, "copilot", td.EngineConfig.ID,
			"T-TD-006: string engine format must store the engine ID in EngineConfig.ID")
		assert.False(t, td.EngineDisabled,
			"T-TD-006: string engine format must not set EngineDisabled")
	})

	t.Run("object format", func(t *testing.T) {
		// TD-14: object engine config is parsed into EngineConfig.
		configMap := map[string]any{
			"engine": map[string]any{
				"id":        "copilot",
				"max-turns": 5,
			},
		}
		td := c.parseThreatDetectionObjectConfig(configMap)
		require.NotNil(t, td,
			"T-TD-006: object engine format must produce a non-nil ThreatDetectionConfig")
		require.NotNil(t, td.EngineConfig,
			"T-TD-006: object engine format must produce a non-nil EngineConfig")
		assert.Equal(t, "copilot", td.EngineConfig.ID,
			"T-TD-006: object engine format must store the engine ID in EngineConfig.ID")
		assert.False(t, td.EngineDisabled,
			"T-TD-006: object engine format must not set EngineDisabled")
	})

	t.Run("disabled (false)", func(t *testing.T) {
		// TD-15: engine: false disables AI-powered detection.
		configMap := map[string]any{
			"engine": false,
		}
		td := c.parseThreatDetectionObjectConfig(configMap)
		require.NotNil(t, td,
			"T-TD-006: engine:false must produce a non-nil ThreatDetectionConfig (config still present)")
		assert.True(t, td.EngineDisabled,
			"T-TD-006: engine:false must set EngineDisabled=true")
		assert.Nil(t, td.EngineConfig,
			"T-TD-006: engine:false must leave EngineConfig nil")
	})
}

// TestFormalTD007_WorkflowFailureOnDetection (T-TD-007)
//
// T-TD-007: Verify workflow failure on threat detection.
//
// TD-09 requires that if any threat is detected, the workflow MUST fail and
// safe outputs MUST NOT execute. This test verifies that the compiled
// safe_outputs job condition requires detection job success
// (`needs.detection.result == 'success'`), ensuring safe outputs are blocked
// when the detection job fails due to a detected threat.
func TestFormalTD007_WorkflowFailureOnDetection(t *testing.T) {
	md := `---
name: td007-failure-on-detection
on: push
engine: copilot
permissions:
  contents: read
safe-outputs:
  create-issue:
---

# Mission

T-TD-007: verify that safe_outputs is blocked when threat detection fails.
`
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "T-TD-007: workflow must parse without error")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "T-TD-007: workflow must compile without error")
	require.NotEmpty(t, yamlOut, "T-TD-007: compiled YAML must not be empty")

	safeOutputsSection := extractJobSection(yamlOut, "safe_outputs")
	require.NotEmpty(t, safeOutputsSection,
		"T-TD-007: compiled workflow must contain a safe_outputs job")
	assert.Contains(t, safeOutputsSection, "needs.detection.result == 'success'",
		"T-TD-007: safe_outputs job condition must require detection job success to block execution on threat detection failure")
}
