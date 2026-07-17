// Package workflow - stateless predicates and utility helpers for threat detection.
package workflow

import (
	"encoding/json"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var threatLog = logger.New("workflow:threat_detection")

// detectionStepCondition is the if condition applied to inline detection steps.
// Detection steps only run when the detection guard determines there's output to analyze.
const detectionStepCondition = "always() && steps.detection_guard.outputs.run_detection == 'true'"

// stepEnvIndent is the indentation prefix used for env var lines in rendered step YAML.
const stepEnvIndent = "        "

// IsDetectionJobEnabled reports whether a detection job should be created for
// the given safe-outputs configuration. This is the single source of truth
// used by all codepaths that decide whether to create, depend on, or reference
// the detection job.
func IsDetectionJobEnabled(so *SafeOutputsConfig) bool {
	return so != nil && so.ThreatDetection != nil && so.ThreatDetection.HasRunnableDetection()
}

// IsConditionalDetection reports whether the safe-outputs configuration uses an expression
// to control threat detection at runtime. When true, the detection job is always compiled
// but may be skipped at runtime; downstream jobs must handle the skipped result.
func IsConditionalDetection(so *SafeOutputsConfig) bool {
	return so != nil && so.ThreatDetection != nil && so.ThreatDetection.IsConditional()
}

// isThreatDetectionExplicitlyDisabledInConfigs checks whether any of the provided
// safe-outputs config JSON strings has threat-detection explicitly set to disabled.
// Supports both the boolean form (threat-detection: false) and the object form
// (threat-detection: { enabled: false }), mirroring parseThreatDetectionConfig.
// This is used to determine whether the default detection should be applied when
// safe-outputs comes from imports/includes (i.e. no safe-outputs: section in the
// main workflow frontmatter).
func isThreatDetectionExplicitlyDisabledInConfigs(configs []string) bool {
	for _, configJSON := range configs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}
		var config map[string]any
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			continue
		}
		if tdVal, exists := config["threat-detection"]; exists {
			// Boolean form: threat-detection: false
			if tdBool, ok := tdVal.(bool); ok && !tdBool {
				threatLog.Print("Threat detection explicitly disabled (boolean form) in safe-outputs config")
				return true
			}
			// Object form: threat-detection: { enabled: false }
			if tdMap, ok := tdVal.(map[string]any); ok {
				if enabled, exists := tdMap["enabled"]; exists {
					if enabledBool, ok := enabled.(bool); ok && !enabledBool {
						threatLog.Print("Threat detection explicitly disabled (object form) in safe-outputs config")
						return true
					}
				}
			}
		}
	}
	return false
}

func getThreatDetectionAdditionalAllowedDomains(data *WorkflowData) []string {
	if data == nil || data.NetworkPermissions == nil {
		return []string{}
	}

	// Evaluate the effective merged detection environment (main + detection-specific
	// overrides) so that a custom base URL configured only in
	// safe-outputs.threat-detection.engine.env also triggers domain propagation.
	var detectionSpecificEnv map[string]string
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.EngineConfig != nil {
		detectionSpecificEnv = data.SafeOutputs.ThreatDetection.EngineConfig.Env
	}
	effectiveEnv := mergeThreatDetectionEngineEnv(data, detectionSpecificEnv)

	hasCustomTarget := effectiveEnv["OPENAI_BASE_URL"] != "" ||
		effectiveEnv["ANTHROPIC_BASE_URL"] != "" ||
		effectiveEnv[constants.CopilotProviderBaseURL] != ""
	if !hasCustomTarget {
		return []string{}
	}

	additional := make([]string, 0, len(data.NetworkPermissions.Allowed))
	seen := make(map[string]struct{})
	for _, entry := range data.NetworkPermissions.Allowed {
		if entry == "" || strings.Contains(entry, "${{") {
			continue
		}
		if len(getEcosystemDomains(entry)) > 0 {
			continue
		}
		if _, exists := seen[entry]; exists {
			continue
		}
		seen[entry] = struct{}{}
		additional = append(additional, entry)
	}

	threatLog.Printf("Computed %d additional allowed domains for threat detection", len(additional))
	return additional
}

func canReuseThreatDetectionEngineConfigForExternalDetector(data *WorkflowData, engineID string) bool {
	return data.SafeOutputs != nil &&
		data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig != nil &&
		(data.SafeOutputs.ThreatDetection.EngineConfig.ID == "" || data.SafeOutputs.ThreatDetection.EngineConfig.ID == engineID)
}

// mergeThreatDetectionEngineEnv composes detection engine env vars from the main
// engine env and detection-specific overrides.
//
// Detection values take precedence when keys overlap. When detectionEnv is empty,
// it still returns a copy of the main env map to avoid aliasing/mutation of the
// parent WorkflowData.EngineConfig.Env by downstream detection-specific updates.
func mergeThreatDetectionEngineEnv(data *WorkflowData, detectionEnv map[string]string) map[string]string {
	if data == nil || data.EngineConfig == nil || len(data.EngineConfig.Env) == 0 {
		return detectionEnv
	}
	if len(detectionEnv) == 0 {
		// Return a copy (not the original map) so subsequent detection-specific
		// env merges cannot mutate the main engine's env map by aliasing.
		return maps.Clone(data.EngineConfig.Env)
	}

	merged := make(map[string]string, len(data.EngineConfig.Env)+len(detectionEnv))
	maps.Copy(merged, data.EngineConfig.Env)
	maps.Copy(merged, detectionEnv)
	return merged
}

// cloneThreatDetectionEngineConfig returns a shallow copy of source with engine ID
// normalized to the provided detection engineID. If source is nil, it returns a
// minimal config containing only the ID.
func cloneThreatDetectionEngineConfig(engineID string, source *EngineConfig) *EngineConfig {
	if source == nil {
		return &EngineConfig{ID: engineID}
	}
	cloned := *source
	cloned.ID = engineID
	return &cloned
}

// engineCoreSecretVarNames returns the secret-backed env var names for the given engine ID
// that must be excluded from the AWF container via --exclude-env. These are the credentials
// that AWF's API proxy intercepts, so the container itself does not need them.
func engineCoreSecretVarNames(engineID string) []string {
	switch engineID {
	case "copilot":
		return []string{"COPILOT_GITHUB_TOKEN"}
	case "claude":
		return []string{"ANTHROPIC_API_KEY"}
	case "codex":
		return []string{"OPENAI_API_KEY", "CODEX_API_KEY"}
	case "gemini", "antigravity":
		return []string{"GEMINI_API_KEY", "ANTIGRAVITY_API_KEY"}
	default:
		return []string{}
	}
}
