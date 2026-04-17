// Package workflow provides the repo-level configuration loader for aw.json.
//
// This file loads and validates .github/workflows/aw.json, which provides
// repository-level settings for agentic workflows such as customising the
// agentics-maintenance runner.
//
// Configuration reference:
//
//	{
//	  "maintenance": {              // enables generation of agentics-maintenance.yml
//	    "runs_on": "custom runner", // string or string[] – runner label(s) for all
//	    "action_failure_issue_expires": 72 // expiration (hours) for conclusion failure issues
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

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var repoConfigLog = logger.New("workflow:repo_config")

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
type MaintenanceConfig struct {
	// RunsOn is the runner label or labels used for all jobs in agentics-maintenance.yml.
	RunsOn RunsOnValue `json:"runs_on,omitempty"`

	// ActionFailureIssueExpires configures expiration (in hours) for action
	// failure issues opened by the conclusion job. Defaults to 168 (7 days).
	ActionFailureIssueExpires int `json:"action_failure_issue_expires,omitempty"`
}

// RepoConfig is the parsed representation of aw.json.
type RepoConfig struct {
	// MaintenanceDisabled is true when maintenance has been explicitly set to false
	// in aw.json, disabling agentic-maintenance generation and any features that
	// depend on it (such as expires).
	MaintenanceDisabled bool

	// Maintenance holds maintenance-specific settings when maintenance is enabled
	// and an object was provided (nil when maintenance is not configured or is
	// disabled).
	Maintenance *MaintenanceConfig
}

// UnmarshalJSON implements json.Unmarshaler to handle the polymorphic maintenance
// field, which can be either the boolean false (disable) or a configuration object.
func (r *RepoConfig) UnmarshalJSON(data []byte) error {
	// Use an intermediate struct with json.RawMessage to defer maintenance parsing.
	var raw struct {
		Maintenance json.RawMessage `json:"maintenance,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

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
