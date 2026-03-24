package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var observabilityPolicyLog = logger.New("cli:observability_policy")

type ObservabilityPolicyEvalConfig struct {
	PolicyPath string
	ReportPath string
	JSONOutput bool
}

type ObservabilityPolicyEvaluation struct {
	PolicyPath string                               `json:"policy_path"`
	ReportPath string                               `json:"report_path"`
	Summary    ObservabilityPolicyEvaluationSummary `json:"summary"`
	Violations []ObservabilityPolicyViolation       `json:"violations,omitempty"`
}

type ObservabilityPolicyEvaluationSummary struct {
	Status          string `json:"status"`
	TotalViolations int    `json:"total_violations"`
	FailViolations  int    `json:"fail_violations"`
	GateViolations  int    `json:"gate_violations"`
	WarnViolations  int    `json:"warn_violations"`
	Blocking        bool   `json:"blocking"`
}

// NewObservabilityPolicyCommand creates the observability-policy command.
func NewObservabilityPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observability-policy",
		Short: "Evaluate observability reports against guardrail policies",
		Long: `Evaluate an observability report against a policy file to surface guardrail decisions.

This command reads two JSON files:
- A policy file that defines fail, gate, or warn rules
- An observability report payload produced for a workflow run

The result can be rendered for people or emitted as JSON for automation.
Blocking actions (fail and gate) return a non-zero exit status.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` observability-policy eval --policy policy.json --report observability-report.json
  ` + string(constants.CLIExtensionPrefix) + ` observability-policy eval --policy policy.json --report observability-report.json --json`,
	}

	cmd.AddCommand(newObservabilityPolicyEvalCommand())

	return cmd
}

func newObservabilityPolicyEvalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Evaluate a policy against an observability report",
		Long: `Evaluate an observability policy against a workflow observability report.

This command is intended for immediate guardrail checks in local development,
CI, or follow-up analysis after running gh aw logs or gh aw audit.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` observability-policy eval --policy policy.json --report observability-report.json
  ` + string(constants.CLIExtensionPrefix) + ` observability-policy eval --policy policy.json --report observability-report.json --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			policyPath, _ := cmd.Flags().GetString("policy")
			reportPath, _ := cmd.Flags().GetString("report")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			config := ObservabilityPolicyEvalConfig{
				PolicyPath: policyPath,
				ReportPath: reportPath,
				JSONOutput: jsonOutput,
			}

			return RunObservabilityPolicyEval(config)
		},
	}

	cmd.Flags().String("policy", "", "Path to the observability policy JSON file")
	cmd.Flags().String("report", "", "Path to the observability report JSON file")
	addJSONFlag(cmd)
	_ = cmd.MarkFlagRequired("policy")
	_ = cmd.MarkFlagRequired("report")

	return cmd
}

// RunObservabilityPolicyEval executes observability policy evaluation.
func RunObservabilityPolicyEval(config ObservabilityPolicyEvalConfig) error {
	if config.PolicyPath == "" {
		return fmt.Errorf("policy path is required")
	}
	if config.ReportPath == "" {
		return fmt.Errorf("report path is required")
	}

	policy, err := readObservabilityPolicyFile(config.PolicyPath)
	if err != nil {
		return err
	}

	payload, err := readObservabilityPayloadFile(config.ReportPath)
	if err != nil {
		return err
	}

	observabilityPolicyLog.Printf("Evaluating policy=%s report=%s", config.PolicyPath, config.ReportPath)

	result := EvaluateObservabilityPolicy(policy, payload)
	evaluation := buildObservabilityPolicyEvaluation(config, result)

	if config.JSONOutput {
		output, err := json.MarshalIndent(evaluation, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal observability policy result: %w", err)
		}
		fmt.Println(string(output))
	} else {
		renderObservabilityPolicyEvaluation(evaluation)
	}

	return evaluation.summaryError()
}

func readObservabilityPolicyFile(path string) (ObservabilityPolicy, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ObservabilityPolicy{}, fmt.Errorf("failed to read observability policy file: %w", err)
	}

	var policy ObservabilityPolicy
	if err := json.Unmarshal(content, &policy); err != nil {
		return ObservabilityPolicy{}, fmt.Errorf("failed to parse observability policy file: %w", err)
	}

	return policy, nil
}

func readObservabilityPayloadFile(path string) (ObservabilityPayload, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ObservabilityPayload{}, fmt.Errorf("failed to read observability report file: %w", err)
	}

	var payload ObservabilityPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return ObservabilityPayload{}, fmt.Errorf("failed to parse observability report file: %w", err)
	}

	return payload, nil
}

func buildObservabilityPolicyEvaluation(config ObservabilityPolicyEvalConfig, result ObservabilityPolicyResult) ObservabilityPolicyEvaluation {
	summary := summarizeObservabilityPolicyResult(result)

	return ObservabilityPolicyEvaluation{
		PolicyPath: config.PolicyPath,
		ReportPath: config.ReportPath,
		Summary:    summary,
		Violations: result.Violations,
	}
}

func summarizeObservabilityPolicyResult(result ObservabilityPolicyResult) ObservabilityPolicyEvaluationSummary {
	summary := ObservabilityPolicyEvaluationSummary{
		Status: "pass",
	}

	for _, violation := range result.Violations {
		summary.TotalViolations++
		switch violation.Action {
		case "fail":
			summary.FailViolations++
		case "gate":
			summary.GateViolations++
		case "warn":
			summary.WarnViolations++
		}
	}

	summary.Blocking = summary.FailViolations > 0 || summary.GateViolations > 0

	switch {
	case summary.FailViolations > 0:
		summary.Status = "fail"
	case summary.GateViolations > 0:
		summary.Status = "gate"
	case summary.WarnViolations > 0:
		summary.Status = "warn"
	}

	return summary
}

func renderObservabilityPolicyEvaluation(evaluation ObservabilityPolicyEvaluation) {
	summary := evaluation.Summary

	if summary.TotalViolations == 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("No observability policy violations detected"))
		return
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("Observability policy evaluation found %d violation(s)", summary.TotalViolations),
	))

	for _, violation := range evaluation.Violations {
		message := fmt.Sprintf("%s: %s", violation.RuleID, violation.Message)
		if violation.Evidence != "" {
			message += " (" + violation.Evidence + ")"
		}

		switch violation.Action {
		case "fail":
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(message))
		case "gate":
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(message))
		case "warn":
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(message))
		default:
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(message))
		}
	}

	if summary.FailViolations > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(
			fmt.Sprintf("Evaluation failed with %d fail violation(s)", summary.FailViolations),
		))
		return
	}

	if summary.GateViolations > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("Evaluation requires approval because %d gate violation(s) matched", summary.GateViolations),
		))
		return
	}

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		fmt.Sprintf("Evaluation completed with %d warning violation(s)", summary.WarnViolations),
	))
}

func (evaluation ObservabilityPolicyEvaluation) summaryError() error {
	switch evaluation.Summary.Status {
	case "fail":
		return fmt.Errorf("observability policy evaluation failed with %d fail violation(s)", evaluation.Summary.FailViolations)
	case "gate":
		return fmt.Errorf("observability policy evaluation requires approval because %d gate violation(s) matched", evaluation.Summary.GateViolations)
	default:
		return nil
	}
}
