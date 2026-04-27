package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var circuitBreakerLog = logger.New("workflow:circuit_breaker")

// defaultCircuitBreakerMaxConsecutiveFailures is the default number of consecutive failures
// before the circuit breaker opens.
const defaultCircuitBreakerMaxConsecutiveFailures = 5

// defaultCircuitBreakerTimeWindow is the default time window for counting failures.
const defaultCircuitBreakerTimeWindow = "24h"

// defaultCircuitBreakerCooldown is the default cooldown period after the circuit opens.
const defaultCircuitBreakerCooldown = "1h"

// extractCircuitBreakerConfig extracts the 'circuit-breaker' field from frontmatter.
// It also handles the feature flag form: features.circuit-breaker: true.
func (c *Compiler) extractCircuitBreakerConfig(frontmatter map[string]any) *CircuitBreakerConfig {
	// Check for explicit circuit-breaker configuration
	if cbValue, exists := frontmatter["circuit-breaker"]; exists && cbValue != nil {
		switch v := cbValue.(type) {
		case map[string]any:
			config := &CircuitBreakerConfig{}

			// Extract max-consecutive-failures (default: 5)
			if maxValue, ok := v["max-consecutive-failures"]; ok {
				switch max := maxValue.(type) {
				case int:
					config.MaxConsecutiveFailures = max
				case int64:
					config.MaxConsecutiveFailures = int(max)
				case uint64:
					config.MaxConsecutiveFailures = int(max)
				case float64:
					config.MaxConsecutiveFailures = int(max)
				}
			}

			// Extract time-window (default: "24h")
			if windowValue, ok := v["time-window"]; ok {
				if str, ok := windowValue.(string); ok {
					config.TimeWindow = str
				}
			}

			// Extract cooldown (default: "1h")
			if cooldownValue, ok := v["cooldown"]; ok {
				if str, ok := cooldownValue.(string); ok {
					config.Cooldown = str
				}
			}

			// Extract notify (default: true)
			if notifyValue, ok := v["notify"]; ok {
				if b, ok := notifyValue.(bool); ok {
					config.Notify = &b
				}
			}

			applyCircuitBreakerDefaults(config)
			circuitBreakerLog.Printf("Extracted circuit-breaker config: max=%d, window=%s, cooldown=%s",
				config.MaxConsecutiveFailures, config.TimeWindow, config.Cooldown)
			return config

		case bool:
			if v {
				// circuit-breaker: true → use all defaults
				config := &CircuitBreakerConfig{}
				applyCircuitBreakerDefaults(config)
				circuitBreakerLog.Print("Circuit-breaker enabled via boolean flag (using defaults)")
				return config
			}
			// circuit-breaker: false → explicitly disabled
			circuitBreakerLog.Print("Circuit-breaker explicitly disabled via boolean false")
			return nil
		}
	}

	// Check the feature flag: features.circuit-breaker: true
	if featuresValue, exists := frontmatter["features"]; exists && featuresValue != nil {
		if features, ok := featuresValue.(map[string]any); ok {
			if cbFeature, exists := features["circuit-breaker"]; exists {
				if b, ok := cbFeature.(bool); ok && b {
					config := &CircuitBreakerConfig{}
					applyCircuitBreakerDefaults(config)
					circuitBreakerLog.Print("Circuit-breaker enabled via features.circuit-breaker: true (using defaults)")
					return config
				}
			}
		}
	}

	circuitBreakerLog.Print("No circuit-breaker configuration specified")
	return nil
}

// applyCircuitBreakerDefaults fills in default values for any unset circuit-breaker config fields.
func applyCircuitBreakerDefaults(config *CircuitBreakerConfig) {
	if config.MaxConsecutiveFailures <= 0 {
		config.MaxConsecutiveFailures = defaultCircuitBreakerMaxConsecutiveFailures
	}
	if config.TimeWindow == "" {
		config.TimeWindow = defaultCircuitBreakerTimeWindow
	}
	if config.Cooldown == "" {
		config.Cooldown = defaultCircuitBreakerCooldown
	}
	if config.Notify == nil {
		t := true
		config.Notify = &t
	}
}

// circuitBreakerDurationToMinutes parses a duration string (e.g. "24h", "30m") and returns
// the equivalent number of minutes as an integer. Sub-minute durations are rounded up to 1 minute
// to prevent a 0-minute window/cooldown from breaking the check logic.
func circuitBreakerDurationToMinutes(d string) (int, error) {
	dur, err := time.ParseDuration(d)
	if err != nil {
		return 0, fmt.Errorf("invalid circuit-breaker duration %q: %w", d, err)
	}
	// Round up to 1 minute to avoid a 0-minute window that would disable the check.
	return max(int(dur.Minutes()), 1), nil
}

// generateCircuitBreakerCheckSteps generates the pre-activation steps that check whether
// the circuit breaker is open. Three steps are generated:
//  1. Find the previous run with a circuit-breaker-state artifact (GitHub Script).
//  2. Download the artifact if found (actions/download-artifact@v4).
//  3. Read the JSON file and evaluate the circuit state, outputting circuit_breaker_ok.
func (c *Compiler) generateCircuitBreakerCheckSteps(data *WorkflowData, steps []string) []string {
	cfg := data.CircuitBreaker
	if cfg == nil {
		return steps
	}

	timeWindowMinutes, err := circuitBreakerDurationToMinutes(cfg.TimeWindow)
	if err != nil {
		circuitBreakerLog.Printf("Warning: could not parse circuit-breaker time-window %q, using default 1440 minutes: %v", cfg.TimeWindow, err)
		timeWindowMinutes = 1440
	}
	cooldownMinutes, err := circuitBreakerDurationToMinutes(cfg.Cooldown)
	if err != nil {
		circuitBreakerLog.Printf("Warning: could not parse circuit-breaker cooldown %q, using default 60 minutes: %v", cfg.Cooldown, err)
		cooldownMinutes = 60
	}

	notify := "true"
	if cfg.Notify != nil && !*cfg.Notify {
		notify = "false"
	}

	// Step 1: Find the previous run with the circuit-breaker-state artifact.
	steps = append(steps, "      - name: Find previous circuit breaker state\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", constants.FindCircuitBreakerArtifactStepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")
	steps = append(steps, generateGitHubScriptWithRequire("find_circuit_breaker_artifact.cjs"))

	// Step 2: Download the artifact (actions/download-artifact handles ZIP extraction natively).
	steps = append(steps, "      - name: Download previous circuit breaker state\n")
	steps = append(steps, fmt.Sprintf("        if: steps.%s.outputs.previous_run_id != ''\n", constants.FindCircuitBreakerArtifactStepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/download-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", constants.CircuitBreakerArtifactName))
	steps = append(steps, fmt.Sprintf("          run-id: ${{ steps.%s.outputs.previous_run_id }}\n", constants.FindCircuitBreakerArtifactStepID))
	steps = append(steps, "          path: /tmp/gh-aw\n")
	steps = append(steps, "          github-token: ${{ secrets.GITHUB_TOKEN }}\n")

	// Step 3: Read the JSON and evaluate the circuit breaker state.
	steps = append(steps, "      - name: Check circuit breaker\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckCircuitBreakerStepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_CB_MAX_FAILURES: \"%d\"\n", cfg.MaxConsecutiveFailures))
	steps = append(steps, fmt.Sprintf("          GH_AW_CB_TIME_WINDOW_MINUTES: \"%d\"\n", timeWindowMinutes))
	steps = append(steps, fmt.Sprintf("          GH_AW_CB_COOLDOWN_MINUTES: \"%d\"\n", cooldownMinutes))
	steps = append(steps, fmt.Sprintf("          GH_AW_CB_NOTIFY: \"%s\"\n", notify))
	steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")
	steps = append(steps, generateGitHubScriptWithRequire("check_circuit_breaker.cjs"))

	circuitBreakerLog.Printf("Added circuit breaker check steps: max=%d, window=%dm, cooldown=%dm",
		cfg.MaxConsecutiveFailures, timeWindowMinutes, cooldownMinutes)
	return steps
}

// generateCircuitBreakerUpdateSteps generates the post-execution steps that update the circuit
// breaker state artifact. These steps use if: always() so they run regardless of job outcome.
func (c *Compiler) generateCircuitBreakerUpdateSteps(yaml *strings.Builder, data *WorkflowData) {
	if data.CircuitBreaker == nil {
		return
	}

	timeWindowMinutes, err := circuitBreakerDurationToMinutes(data.CircuitBreaker.TimeWindow)
	if err != nil {
		circuitBreakerLog.Printf("Warning: could not parse circuit-breaker time-window %q for update step, using default 1440 minutes: %v", data.CircuitBreaker.TimeWindow, err)
		timeWindowMinutes = 1440
	}

	circuitBreakerLog.Print("Adding circuit breaker state update steps to agent job")

	yaml.WriteString("      - name: Update circuit breaker state\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        id: %s\n", constants.UpdateCircuitBreakerStepID)
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_CB_JOB_STATUS: ${{ job.status }}\n")
	fmt.Fprintf(yaml, "          GH_AW_CB_MAX_FAILURES: \"%d\"\n", data.CircuitBreaker.MaxConsecutiveFailures)
	fmt.Fprintf(yaml, "          GH_AW_CB_TIME_WINDOW_MINUTES: \"%d\"\n", timeWindowMinutes)
	fmt.Fprintf(yaml, "          GH_AW_WORKFLOW_NAME: %q\n", data.Name)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString(generateGitHubScriptWithRequire("update_circuit_breaker.cjs"))

	yaml.WriteString("      - name: Upload circuit breaker state\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("actions/upload-artifact"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", constants.CircuitBreakerArtifactName)
	yaml.WriteString("          path: /tmp/gh-aw/circuit-breaker-state.json\n")
	yaml.WriteString("          if-no-files-found: ignore\n")
	yaml.WriteString("          overwrite: true\n")
}
