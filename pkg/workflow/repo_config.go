// Package workflow provides the repo-level configuration loader for aw.json.
//
// This file loads and validates .github/workflows/aw.json, which provides
// repository-level settings for agentic workflows such as customising the
// agentics-maintenance runner.
//
// Configuration reference:
//
//	{
//	  "ghes": true,               // enables GHES compatibility mode (artifact pins remain latest non-v3)
//	  "help_command": false,      // disables builtin centralized /help comment handler
//	  "utc": "-08:00", // project home UTC offset for rendered local times
//	  "auto_upgrade": true, // set to true to generate agentic-auto-upgrade.yml with weekly schedule
//	  "maintenance": {              // enables generation of agentics-maintenance.yml
//	    "runs_on": "custom runner", // string or string[] – runner label(s) for all
//	    "action_failure_issue_expires": 72, // expiration (hours) for conclusion failure issues
//	    "label_triggers": true, // set to true to enable all label-triggered jobs (opt-in)
//	    "disabled_jobs": ["close-expired-entities"], // optional maintenance jobs to omit
//	    "compile": {
//	      "create_pull_request_github_token": "MY_REPO_TOKEN" // create/update a deduplicated PR instead of an issue
//	    }
//	  }                            // maintenance jobs (default: ubuntu-slim)
//	}
//
//	{
//	  "maintenance": false          // disables agentic maintenance entirely
//	}
package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var repoConfigLog = logger.New("workflow:repo_config")
var repoConfigSecretNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// RepoConfigFileName is the path of the repository-level configuration file
// relative to the git root.
const RepoConfigFileName = ".github/workflows/aw.json"

// DefaultActionFailureIssueExpiresHours is the default expiration (in hours)
// for action failure issues created by the conclusion job.
const DefaultActionFailureIssueExpiresHours = 24 * 7

// RunsOnValue is a JSON-deserializable type for the runs_on field in aw.json.
// It accepts either a single runner label string or an array of runner label strings.
// When unmarshalled, a plain string is normalised to a single-element slice so the
// rest of the code works with a uniform []string type.
type RunsOnValue []string

// UnmarshalJSON implements json.Unmarshaler, accepting either a JSON string or
// a JSON array of strings for the runs_on field.
func (r *RunsOnValue) UnmarshalJSON(data []byte) error {
	// Try plain string first (runs_on: "ubuntu-latest")
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*r = RunsOnValue{s}
		return nil
	}

	// Try array of strings (runs_on: ["self-hosted", "linux"])
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return fmt.Errorf("runs_on must be a string or array of strings: %w", err)
	}
	*r = RunsOnValue(ss)
	return nil
}

// MaintenanceConfig holds maintenance-workflow-specific settings from aw.json.
type MaintenanceCompileConfig struct {
	// CreatePullRequestGitHubToken is the secret name used by the compile-workflows
	// maintenance job for GitHub API calls and branch pushes. When configured,
	// out-of-sync compiled workflows are reported via a deduplicated pull request
	// instead of an issue.
	CreatePullRequestGitHubToken string `json:"create_pull_request_github_token,omitempty"`
}

type MaintenanceConfig struct {
	// RunsOn is the runner label or labels used for all jobs in agentics-maintenance.yml.
	RunsOn RunsOnValue `json:"runs_on,omitempty"`

	// ActionFailureIssueExpires configures expiration (in hours) for action
	// failure issues opened by the conclusion job. Defaults to 168 (7 days).
	ActionFailureIssueExpires int `json:"action_failure_issue_expires,omitempty"`

	// LabelTriggers controls all label-triggered jobs (disable_agentic_workflow,
	// label_apply_safe_outputs, etc.).
	// The value is treated as an opt-in flag: only true enables the jobs.
	// nil (omitted) or false both disable label-triggered jobs.
	// To opt in, set label_triggers: true in aw.json.
	LabelTriggers *bool `json:"label_triggers,omitempty"`

	// DisabledJobs lists maintenance job IDs that should be omitted from generated
	// agentics-maintenance workflows.
	DisabledJobs []string `json:"disabled_jobs,omitempty"`

	// Compile controls compile-workflows maintenance job behavior.
	Compile *MaintenanceCompileConfig `json:"compile,omitempty"`
}

var validDisabledMaintenanceJobs = map[string]string{
	normalizeMaintenanceJobName("close-expired-entities"):         "close-expired-entities",
	normalizeMaintenanceJobName("apply_safe_outputs"):             "apply_safe_outputs",
	normalizeMaintenanceJobName("label_disable_agentic_workflow"): "label_disable_agentic_workflow",
	normalizeMaintenanceJobName("label_apply_safe_outputs"):       "label_apply_safe_outputs",
}

// IsLabelTriggerEnabled returns true only when label_triggers is explicitly set to true.
// The default (nil / omitted) is treated as disabled (false) — opt-in semantics.
func (m *MaintenanceConfig) IsLabelTriggerEnabled() bool {
	if m == nil || m.LabelTriggers == nil {
		return false
	}
	return *m.LabelTriggers
}

func normalizeMaintenanceJobName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.ReplaceAll(normalized, "_", "-")
}

// IsJobDisabled reports whether the provided maintenance job ID is explicitly
// disabled in aw.json.
func (m *MaintenanceConfig) IsJobDisabled(jobName string) bool {
	if m == nil || len(m.DisabledJobs) == 0 {
		return false
	}
	normalizedJobName := normalizeMaintenanceJobName(jobName)
	for _, disabledJob := range m.DisabledJobs {
		if normalizeMaintenanceJobName(disabledJob) == normalizedJobName {
			return true
		}
	}
	return false
}

// RepoConfig is the parsed representation of aw.json.
type RepoConfig struct {
	// GHES enables GitHub Enterprise Server compatibility mode.
	// When true, the compiler enables GHES compatibility behavior. Artifact actions
	// continue to use latest non-v3 pins because v3 artifact actions are deprecated.
	GHES bool

	// UTC is the project's home UTC offset used for rendering local times in CLI output.
	// The value must be a numeric UTC offset such as "+00:00" or "-08:00".
	UTC string

	// HelpCommand controls builtin centralized /help command behavior.
	// When nil or true, the builtin help command is enabled.
	// Set to false in aw.json to disable it.
	HelpCommand *bool

	// AutoUpgrade enables generation of agentic-auto-upgrade.yml when true.
	// The workflow runs on a fuzzy weekly schedule and runs the upgrade operation
	// to check for and report available workflow upgrades.
	// Opt-in: nil (omitted) or false both disable generation.
	AutoUpgrade *bool

	// MaintenanceDisabled is true when maintenance has been explicitly set to false
	// in aw.json, disabling agentic-maintenance generation and any features that
	// depend on it (such as expires).
	MaintenanceDisabled bool

	// Maintenance holds maintenance-specific settings when maintenance is enabled
	// and an object was provided (nil when maintenance is not configured or is
	// disabled).
	Maintenance *MaintenanceConfig

	// ActionPins maps action repository@version references to replacement
	// repository@version references. Enterprises running in a private cloud
	// can use this to redirect actions to internal mirrors. Keys and values
	// must use the format "owner/repo@ref".
	ActionPins map[string]string
}

// IsAutoUpgradeEnabled returns true only when auto_upgrade is explicitly set to true.
// The default (nil / omitted) is treated as disabled (false) — opt-in semantics.
func (r *RepoConfig) IsAutoUpgradeEnabled() bool {
	if r == nil || r.AutoUpgrade == nil {
		return false
	}
	return *r.AutoUpgrade
}

// UnmarshalJSON implements json.Unmarshaler to handle the polymorphic maintenance
// field, which can be either the boolean false (disable) or a configuration object.
func (r *RepoConfig) UnmarshalJSON(data []byte) error {
	// Use an intermediate struct with json.RawMessage to defer maintenance parsing.
	var raw struct {
		GHES        bool              `json:"ghes,omitempty"`
		HelpCommand *bool             `json:"help_command,omitempty"` // nil = use default (enabled)
		UTC         string            `json:"utc,omitempty"`
		AutoUpgrade *bool             `json:"auto_upgrade,omitempty"`
		Maintenance json.RawMessage   `json:"maintenance,omitempty"`
		ActionPins  map[string]string `json:"action_pins,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.GHES = raw.GHES
	r.HelpCommand = raw.HelpCommand
	r.UTC = strings.TrimSpace(raw.UTC)
	r.AutoUpgrade = raw.AutoUpgrade
	r.ActionPins = raw.ActionPins

	if len(raw.Maintenance) == 0 || string(raw.Maintenance) == "null" {
		return nil
	}

	// Try boolean first: maintenance: false disables the feature.
	var b bool
	if err := json.Unmarshal(raw.Maintenance, &b); err == nil {
		repoConfigLog.Printf("Maintenance field parsed as boolean: disabled=%v", !b)
		r.MaintenanceDisabled = !b
		return nil
	}

	// Otherwise deserialise as an object with JSON annotations.
	var mc MaintenanceConfig
	if err := json.Unmarshal(raw.Maintenance, &mc); err != nil {
		return fmt.Errorf("invalid maintenance configuration: %w", err)
	}
	repoConfigLog.Printf("Maintenance field parsed as object: runsOn=%v, issueExpires=%d", mc.RunsOn, mc.ActionFailureIssueExpires)
	r.Maintenance = &mc
	return nil
}

// IsHelpCommandEnabled returns true when the builtin centralized /help command
// handler should be enabled. The default is enabled.
func (r *RepoConfig) IsHelpCommandEnabled() bool {
	if r == nil || r.HelpCommand == nil {
		return true
	}
	return *r.HelpCommand
}

// LoadRepoConfig loads and validates .github/workflows/aw.json from the
// provided git root directory.  The function returns a non-nil *RepoConfig
// with default values when the file does not exist (the file is optional).
// An error is returned only when the file exists but cannot be read or fails
// schema validation.
func LoadRepoConfig(gitRoot string) (*RepoConfig, error) {
	configPath := filepath.Join(gitRoot, RepoConfigFileName)
	repoConfigLog.Printf("Loading repo config from %s", configPath)

	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			repoConfigLog.Print("Repo config file not found, using defaults")
			return &RepoConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", RepoConfigFileName, err)
	}

	// Validate against the embedded JSON schema before deserialising.
	if err := validateRepoConfigJSON(data, configPath); err != nil {
		return nil, err
	}

	// Deserialise into typed structs via JSON annotations.
	var cfg RepoConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", RepoConfigFileName, err)
	}
	if err := validateRepoConfigValues(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateRepoConfigJSON validates raw JSON bytes against the repo config schema.
func validateRepoConfigJSON(data []byte, filePath string) error {
	repoConfigLog.Printf("Validating repo config JSON schema: %s (%d bytes)", filePath, len(data))
	schema, err := parser.GetCompiledRepoConfigSchema()
	if err != nil {
		return fmt.Errorf("failed to compile repo config schema: %w", err)
	}

	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse %s as JSON: %w", filePath, err)
	}

	if err := schema.Validate(doc); err != nil {
		repoConfigLog.Printf("Repo config schema validation failed: %v", err)
		return fmt.Errorf("invalid %s: %w", RepoConfigFileName, err)
	}

	repoConfigLog.Print("Repo config JSON schema validation passed")
	return nil
}

func validateRepoConfigValues(cfg *RepoConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.UTC != "" {
		normalized, err := NormalizeUTCOffset(cfg.UTC)
		if err != nil {
			return fmt.Errorf("invalid %s: utc %w", RepoConfigFileName, err)
		}
		cfg.UTC = normalized
	}
	if cfg.Maintenance != nil {
		seenDisabledJobs := map[string]string{}
		for _, jobName := range cfg.Maintenance.DisabledJobs {
			normalizedJobName := normalizeMaintenanceJobName(jobName)
			if normalizedJobName == "" {
				return fmt.Errorf("invalid %s: maintenance.disabled_jobs entries must not be blank", RepoConfigFileName)
			}
			if _, ok := validDisabledMaintenanceJobs[normalizedJobName]; !ok {
				return fmt.Errorf("invalid %s: maintenance.disabled_jobs contains unrecognized job %q (valid values: close-expired-entities, apply_safe_outputs, label_disable_agentic_workflow, label_apply_safe_outputs)", RepoConfigFileName, jobName)
			}
			if previous, exists := seenDisabledJobs[normalizedJobName]; exists {
				return fmt.Errorf("invalid %s: maintenance.disabled_jobs contains duplicate entries %q and %q after normalization", RepoConfigFileName, previous, jobName)
			}
			seenDisabledJobs[normalizedJobName] = jobName
		}
	}

	if cfg.Maintenance == nil || cfg.Maintenance.Compile == nil {
		return nil
	}
	compileCfg := cfg.Maintenance.Compile
	secretName := compileCfg.CreatePullRequestGitHubToken
	if secretName != "" && !repoConfigSecretNamePattern.MatchString(secretName) {
		return fmt.Errorf("invalid %s: maintenance.compile.create_pull_request_github_token must match %s", RepoConfigFileName, repoConfigSecretNamePattern.String())
	}
	return nil
}

// FormatRunsOn serialises a RunsOnValue to a YAML-compatible string that can
// be inlined directly after "runs-on: " in a generated workflow.
//
//   - empty / nil  → defaultRunsOn is returned
//   - single label → the label string (e.g. "ubuntu-latest")
//   - multiple labels → JSON-encoded flow sequence, e.g. ["self-hosted","linux"]
//
// For multi-label values json.Marshal is used so that any characters that are
// special in YAML or JSON (quotes, backslashes, …) are properly escaped.
// The schema already forbids newlines and control characters, providing a
// defence-in-depth against YAML injection.
func FormatRunsOn(runsOn RunsOnValue, defaultRunsOn string) string {
	if len(runsOn) == 0 {
		return defaultRunsOn
	}
	if len(runsOn) == 1 {
		if runsOn[0] == "" {
			return defaultRunsOn
		}
		return runsOn[0]
	}
	// Multiple labels: use json.Marshal to produce a properly-escaped YAML
	// flow sequence.  A JSON array is valid YAML flow sequence notation.
	encoded, err := json.Marshal([]string(runsOn))
	if err != nil {
		// []string marshalling never fails; fall back to the default just in case.
		return defaultRunsOn
	}
	return string(encoded)
}

// ActionFailureIssueExpiresHours returns the configured action failure issue
// expiration in hours, or the default value when unset.
func (r *RepoConfig) ActionFailureIssueExpiresHours() int {
	if r != nil && r.Maintenance != nil && r.Maintenance.ActionFailureIssueExpires > 0 {
		return r.Maintenance.ActionFailureIssueExpires
	}
	return DefaultActionFailureIssueExpiresHours
}
