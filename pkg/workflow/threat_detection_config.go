// Package workflow - data model and config parsing for threat detection.
package workflow

import "strings"

// ThreatDetectionConfig holds configuration for threat detection in agent output
type ThreatDetectionConfig struct {
	Prompt              string        `yaml:"prompt,omitempty"`            // Additional custom prompt instructions to append
	Steps               []any         `yaml:"steps,omitempty"`             // Array of extra job steps to run before engine execution
	PostSteps           []any         `yaml:"post-steps,omitempty"`        // Array of extra job steps to run after engine execution
	MaxAICredits        int64         `yaml:"max-ai-credits,omitempty"`    // Maximum AI credits budget for threat-detection engine execution
	Model               string        `yaml:"model,omitempty"`             // Model override for threat detection engine execution
	EngineConfig        *EngineConfig `yaml:"engine-config,omitempty"`     // Extended engine configuration for threat detection
	EngineDisabled      bool          `yaml:"-"`                           // Internal flag: true when engine is explicitly set to false
	RunsOn              string        `yaml:"runs-on,omitempty"`           // Runner override for the detection job
	Environment         string        `yaml:"environment,omitempty"`       // GitHub Actions environment override for the detection job (defaults to top-level environment when OIDC is used)
	ContinueOnError     *bool         `yaml:"continue-on-error,omitempty"` // When true (default), detection failures produce warnings instead of blocking safe outputs
	EnabledExpr         *string       `yaml:"-"`                           // Expression form of the enabled flag, e.g. "${{ inputs.enable-threat-detection }}"
	ContinueOnErrorExpr *string       `yaml:"-"`                           // Expression form of continue-on-error, e.g. "${{ inputs.coe }}"
}

// IsContinueOnError reports whether detection failures should produce warnings instead of errors.
// Defaults to true (continue) when not explicitly set.
// Note: when ContinueOnErrorExpr is set, the value is determined at runtime; this method returns
// true as a safe compile-time default (matches the default behaviour).
func (td *ThreatDetectionConfig) IsContinueOnError() bool {
	return td.ContinueOnError == nil || *td.ContinueOnError
}

// HasRunnableDetection reports whether this config will produce a detection job
// that actually executes. Returns false when the engine is disabled and no
// custom steps are configured, since the job would have nothing to run.
// When EnabledExpr is set, detection is conditionally enabled at runtime so we always
// compile the detection job.
func (td *ThreatDetectionConfig) HasRunnableDetection() bool {
	if td.EnabledExpr != nil {
		return true
	}
	return !td.EngineDisabled || len(td.Steps) > 0 || len(td.PostSteps) > 0
}

// IsConditional reports whether detection is expression-controlled (enabled/disabled at runtime).
// When true the detection job is always compiled but its if: condition includes the caller
// expression so GitHub Actions evaluates it at runtime.
func (td *ThreatDetectionConfig) IsConditional() bool {
	return td.EnabledExpr != nil
}

// parseThreatDetectionConfig handles threat-detection configuration
func (c *Compiler) parseThreatDetectionConfig(outputMap map[string]any) *ThreatDetectionConfig {
	if configData, exists := outputMap["threat-detection"]; exists {
		threatLog.Print("Found threat-detection configuration")
		// Handle boolean values
		if boolVal, ok := configData.(bool); ok {
			if !boolVal {
				threatLog.Print("Threat detection explicitly disabled")
				// When explicitly disabled, return nil
				return nil
			}
			threatLog.Print("Threat detection enabled with default settings")
			// When enabled as boolean, return empty config
			return &ThreatDetectionConfig{}
		}

		// Handle expression string values (e.g. "${{ inputs.enable-threat-detection }}")
		if strVal, ok := configData.(string); ok {
			if isExpression(strVal) {
				threatLog.Printf("Threat detection controlled by runtime expression: %s", strVal)
				// Detection is conditionally enabled at runtime; always compile the detection job.
				return &ThreatDetectionConfig{EnabledExpr: &strVal}
			}
			// Non-expression strings are rejected by the JSON schema validator; log and fall through.
			threatLog.Printf("Ignoring invalid non-expression string for threat-detection: %s", strVal)
		}

		// Handle object configuration
		if configMap, ok := configData.(map[string]any); ok {
			// Check for enabled field – supports both literal bool and expression string.
			if enabled, exists := configMap["enabled"]; exists {
				switch v := enabled.(type) {
				case bool:
					if !v {
						threatLog.Print("Threat detection disabled via enabled field")
						// When explicitly disabled, return nil
						return nil
					}
				case string:
					if isExpression(v) {
						threatLog.Printf("Threat detection enabled field is a runtime expression: %s", v)
						// Parse remaining fields but record the expression for runtime evaluation.
						config := c.parseThreatDetectionObjectConfig(configMap)
						config.EnabledExpr = &v
						return config
					}
					// Non-expression strings are invalid; fall through to parse remaining fields.
					threatLog.Printf("Ignoring invalid non-expression string for enabled: %s", v)
				}
			}

			return c.parseThreatDetectionObjectConfig(configMap)
		}
	}

	// Default behavior: enabled if any safe-outputs are configured
	threatLog.Print("Using default threat detection configuration")
	return &ThreatDetectionConfig{}
}

// parseThreatDetectionObjectConfig parses the object form of threat-detection config,
// assuming enabled has already been checked and is truthy. It extracts prompt, steps,
// post-steps, runs-on, continue-on-error, and engine fields.
func (c *Compiler) parseThreatDetectionObjectConfig(configMap map[string]any) *ThreatDetectionConfig {
	threatConfig := &ThreatDetectionConfig{}

	// Parse prompt field
	if prompt, exists := configMap["prompt"]; exists {
		if promptStr, ok := prompt.(string); ok {
			threatConfig.Prompt = promptStr
		}
	}

	// Parse steps field (pre-execution steps, run before engine execution)
	if steps, exists := configMap["steps"]; exists {
		if stepsArray, ok := steps.([]any); ok {
			threatConfig.Steps = stepsArray
		}
	}

	// Parse post-steps field (post-execution steps, run after engine execution)
	if postSteps, exists := configMap["post-steps"]; exists {
		if postStepsArray, ok := postSteps.([]any); ok {
			threatConfig.PostSteps = postStepsArray
		}
	}

	// Parse max-ai-credits field
	if maxAICredits, exists := configMap["max-ai-credits"]; exists {
		threatConfig.MaxAICredits = parseMaxAICreditsValue(maxAICredits)
	}

	// Parse runs-on field
	if runOn, exists := configMap["runs-on"]; exists {
		threatConfig.RunsOn = renderRunsOnSnippet(runOn)
	}

	// Parse continue-on-error field (default: true).
	// Accepts a literal bool or a GitHub Actions expression string.
	if coe, exists := configMap["continue-on-error"]; exists {
		switch v := coe.(type) {
		case bool:
			threatConfig.ContinueOnError = &v
			threatLog.Printf("Threat detection continue-on-error set to: %v", v)
		case string:
			if isExpression(v) {
				threatLog.Printf("Threat detection continue-on-error is a runtime expression: %s", v)
				threatConfig.ContinueOnErrorExpr = &v
			}
		}
	}

	// Parse engine field (supports string, object, and boolean false formats)
	if engine, exists := configMap["engine"]; exists {
		// Handle boolean false to disable AI engine
		if engineBool, ok := engine.(bool); ok {
			if !engineBool {
				threatLog.Print("Threat detection AI engine disabled")
				// engine: false means no AI engine steps
				threatConfig.EngineConfig = nil
				threatConfig.EngineDisabled = true
			}
		} else if engineStr, ok := engine.(string); ok {
			threatLog.Printf("Threat detection engine set to: %s", engineStr)
			// Handle string format
			threatConfig.EngineConfig = &EngineConfig{ID: engineStr}
		} else if engineObj, ok := engine.(map[string]any); ok {
			threatLog.Print("Parsing threat detection engine configuration")
			// Handle object format - use extractEngineConfig logic
			_, engineConfig, model := c.ExtractEngineConfig(map[string]any{"engine": engineObj})
			threatConfig.EngineConfig = engineConfig
			threatConfig.Model = model
		}
	}

	threatLog.Printf("Threat detection configured with custom prompt: %v, custom pre-steps: %v, custom post-steps: %v", threatConfig.Prompt != "", len(threatConfig.Steps) > 0, len(threatConfig.PostSteps) > 0)
	return threatConfig
}

// extractRawExpression strips the "${{" prefix and "}}" suffix from a GitHub Actions
// expression string (e.g. "${{ inputs.flag }}" → "inputs.flag"). The result can be
// embedded directly into a YAML if: condition expression tree.
// Callers must ensure the input is a valid expression (verified by isExpression()) before
// calling this function; non-expression strings are returned with no modification.
func extractRawExpression(expr string) string {
	s := strings.TrimPrefix(expr, "${{")
	s = strings.TrimSuffix(s, "}}")
	return strings.TrimSpace(s)
}
