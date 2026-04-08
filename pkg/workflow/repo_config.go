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
//	    "runs_on": "custom runner" // string or string[] – runner label(s) for all
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
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var repoConfigLog = logger.New("workflow:repo_config")

// RepoConfigFileName is the path of the repository-level configuration file
// relative to the git root.
const RepoConfigFileName = ".github/workflows/aw.json"

// MaintenanceConfig holds maintenance-workflow-specific settings from aw.json.
type MaintenanceConfig struct {
	// RunsOn is the runner label or list of labels used for all jobs in
	// agentics-maintenance.yml. It is the raw value from JSON (string or []any).
	RunsOn any
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

	// Validate against the embedded JSON schema before parsing into typed structs.
	if err := validateRepoConfigJSON(data, configPath); err != nil {
		return nil, err
	}

	// Parse the raw JSON into a loosely-typed map so we can inspect the
	// maintenance value type before converting to typed structs.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", RepoConfigFileName, err)
	}

	return parseRepoConfig(raw), nil
}

// validateRepoConfigJSON validates raw JSON bytes against the repo config schema.
func validateRepoConfigJSON(data []byte, filePath string) error {
	schema, err := parser.GetCompiledRepoConfigSchema()
	if err != nil {
		return fmt.Errorf("failed to compile repo config schema: %w", err)
	}

	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse %s as JSON: %w", filePath, err)
	}

	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("invalid %s: %w", RepoConfigFileName, err)
	}

	return nil
}

// parseRepoConfig converts the raw JSON map into a typed *RepoConfig.
func parseRepoConfig(raw map[string]any) *RepoConfig {
	cfg := &RepoConfig{}

	maintenanceVal, ok := raw["maintenance"]
	if !ok {
		return cfg
	}

	// maintenance: false – disabled
	if b, ok := maintenanceVal.(bool); ok && !b {
		cfg.MaintenanceDisabled = true
		return cfg
	}

	// maintenance: { ... }
	if obj, ok := maintenanceVal.(map[string]any); ok {
		mc := &MaintenanceConfig{}
		if runsOn, ok := obj["runs_on"]; ok {
			mc.RunsOn = normaliseRunsOn(runsOn)
		}
		cfg.Maintenance = mc
	}

	return cfg
}

// normaliseRunsOn converts the JSON runs_on value into a canonical form:
//   - string → string
//   - []any (JSON array) → []string
func normaliseRunsOn(v any) any {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		labels := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				labels = append(labels, s)
			}
		}
		return labels
	default:
		return v
	}
}

// FormatRunsOn serialises a RunsOn value to a YAML-compatible string that can
// be inlined directly after "runs-on: " in a generated workflow.
//
//   - string  → the string value (no quoting needed for common runner names)
//   - []string → inline JSON array notation, e.g. ["self-hosted", "linux"]
//   - nil / other → defaultRunsOn is returned unchanged
func FormatRunsOn(runsOn any, defaultRunsOn string) string {
	if runsOn == nil {
		return defaultRunsOn
	}
	switch val := runsOn.(type) {
	case string:
		if val == "" {
			return defaultRunsOn
		}
		return val
	case []string:
		if len(val) == 0 {
			return defaultRunsOn
		}
		// Produce inline YAML sequence notation: ["a", "b", "c"]
		var sb strings.Builder
		sb.WriteString("[")
		for i, s := range val {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(`"`)
			sb.WriteString(s)
			sb.WriteString(`"`)
		}
		sb.WriteString("]")
		return sb.String()
	default:
		return defaultRunsOn
	}
}
