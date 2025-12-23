// Package workflow provides sandbox configuration and validation for agentic workflows.
//
// This file handles:
//   - Sandbox type definitions (AWF, SRT)
//   - Sandbox configuration structures and parsing
//   - Sandbox runtime config generation
//   - Domain-specific validation for sandbox configurations
//
// # Validation Functions
//
// This file contains domain-specific validation functions for sandbox configuration:
//   - validateMountsSyntax() - Validates container mount syntax
//   - validateSandboxConfig() - Validates complete sandbox configuration
//
// These validation functions are co-located with sandbox logic following the principle
// that domain-specific validation belongs in domain files. See validation.go for the
// validation architecture documentation.
package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var sandboxLog = logger.New("workflow:sandbox")

// SandboxType represents the type of sandbox to use
type SandboxType string

const (
	SandboxTypeAWF     SandboxType = "awf"             // Uses AWF (Agent Workflow Firewall)
	SandboxTypeSRT     SandboxType = "srt"             // Uses Anthropic Sandbox Runtime
	SandboxTypeDefault SandboxType = "default"         // Alias for AWF (backward compat)
	SandboxTypeRuntime SandboxType = "sandbox-runtime" // Alias for SRT (backward compat)
)

// SandboxConfig represents the top-level sandbox configuration from front matter
// New format: { agent: "awf"|"srt"|{type, config} }
// Legacy format: "default"|"sandbox-runtime" or { type, config }
type SandboxConfig struct {
	// New fields
	Agent *AgentSandboxConfig `yaml:"agent,omitempty"` // Agent sandbox configuration

	// Legacy fields (for backward compatibility)
	Type   SandboxType           `yaml:"type,omitempty"`   // Sandbox type: "default" or "sandbox-runtime"
	Config *SandboxRuntimeConfig `yaml:"config,omitempty"` // Custom SRT config (optional)
}

// AgentSandboxConfig represents the agent sandbox configuration
type AgentSandboxConfig struct {
	ID       string                `yaml:"id,omitempty"`      // Agent ID: "awf" or "srt" (replaces Type in new object format)
	Type     SandboxType           `yaml:"type,omitempty"`    // Sandbox type: "awf" or "srt" (legacy, use ID instead)
	Disabled bool                  `yaml:"-"`                 // True when agent is explicitly set to false (disables firewall). This is a runtime flag, not serialized to YAML.
	Config   *SandboxRuntimeConfig `yaml:"config,omitempty"`  // Custom SRT config (optional)
	Command  string                `yaml:"command,omitempty"` // Custom command to replace AWF or SRT installation
	Args     []string              `yaml:"args,omitempty"`    // Additional arguments to append to the command
	Env      map[string]string     `yaml:"env,omitempty"`     // Environment variables to set on the step
	Mounts   []string              `yaml:"mounts,omitempty"`  // Container mounts to add for AWF (format: "source:dest:mode")
}

// SandboxRuntimeConfig represents the Anthropic Sandbox Runtime configuration
// This matches the TypeScript SandboxRuntimeConfig interface
// Note: Network configuration is controlled by the top-level 'network' field, not this struct
type SandboxRuntimeConfig struct {
	// Network is only used internally for generating SRT settings JSON output.
	// It is NOT user-configurable from sandbox.agent.config (yaml:"-" prevents parsing).
	// The json tag is needed for output serialization to .srt-settings.json.
	Network                   *SRTNetworkConfig    `yaml:"-" json:"network,omitempty"`
	Filesystem                *SRTFilesystemConfig `yaml:"filesystem,omitempty" json:"filesystem,omitempty"`
	IgnoreViolations          map[string][]string  `yaml:"ignoreViolations,omitempty" json:"ignoreViolations,omitempty"`
	EnableWeakerNestedSandbox bool                 `yaml:"enableWeakerNestedSandbox" json:"enableWeakerNestedSandbox"`
}

// SRTNetworkConfig represents network configuration for SRT
type SRTNetworkConfig struct {
	AllowedDomains      []string `yaml:"allowedDomains,omitempty" json:"allowedDomains,omitempty"`
	DeniedDomains       []string `yaml:"deniedDomains,omitempty" json:"deniedDomains"`
	AllowUnixSockets    []string `yaml:"allowUnixSockets,omitempty" json:"allowUnixSockets,omitempty"`
	AllowLocalBinding   bool     `yaml:"allowLocalBinding" json:"allowLocalBinding"`
	AllowAllUnixSockets bool     `yaml:"allowAllUnixSockets" json:"allowAllUnixSockets"`
}

// SRTFilesystemConfig represents filesystem configuration for SRT
type SRTFilesystemConfig struct {
	DenyRead   []string `yaml:"denyRead" json:"denyRead"`
	AllowWrite []string `yaml:"allowWrite,omitempty" json:"allowWrite,omitempty"`
	DenyWrite  []string `yaml:"denyWrite" json:"denyWrite"`
}

// getAgentType returns the effective agent type from AgentSandboxConfig
// Prefers ID field (new format) over Type field (legacy)
func getAgentType(agent *AgentSandboxConfig) SandboxType {
	if agent == nil {
		return ""
	}
	// New format: use ID field if set
	if agent.ID != "" {
		return SandboxType(agent.ID)
	}
	// Legacy format: use Type field
	return agent.Type
}

// isSupportedSandboxType checks if a sandbox type is valid/supported
func isSupportedSandboxType(sandboxType SandboxType) bool {
	return sandboxType == SandboxTypeAWF ||
		sandboxType == SandboxTypeSRT ||
		sandboxType == SandboxTypeDefault ||
		sandboxType == SandboxTypeRuntime
}

// isSRTEnabled checks if Sandbox Runtime is enabled for the workflow
func isSRTEnabled(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		sandboxLog.Print("No sandbox config, SRT disabled")
		return false
	}

	config := workflowData.SandboxConfig

	// Check new format: sandbox.agent
	if config.Agent != nil {
		// Get effective type from ID or Type field
		agentType := getAgentType(config.Agent)
		enabled := agentType == SandboxTypeSRT || agentType == SandboxTypeRuntime
		sandboxLog.Printf("SRT enabled check (new format): %v (type=%s)", enabled, agentType)
		return enabled
	}

	// Check legacy format: sandbox.type
	enabled := config.Type == SandboxTypeRuntime || config.Type == SandboxTypeSRT
	sandboxLog.Printf("SRT enabled check (legacy format): %v (type=%s)", enabled, config.Type)
	return enabled
}

// generateSRTConfigJSON generates the .srt-settings.json content
// Network configuration is always derived from the top-level 'network' field.
// User-provided sandbox config can override filesystem, ignoreViolations, and enableWeakerNestedSandbox.
func generateSRTConfigJSON(workflowData *WorkflowData) (string, error) {
	if workflowData == nil {
		return "", fmt.Errorf("workflowData is nil")
	}

	sandboxConfig := workflowData.SandboxConfig
	if sandboxConfig == nil {
		return "", fmt.Errorf("sandbox config is nil")
	}

	// Start with base SRT config
	sandboxLog.Print("Generating SRT config from network permissions")

	// Generate network config from top-level network field (always)
	// Network config is NOT user-configurable from sandbox.agent.config
	domainMap := make(map[string]bool)

	// Add Copilot default domains
	for _, domain := range CopilotDefaultDomains {
		domainMap[domain] = true
	}

	// Add NetworkPermissions domains (if specified)
	if workflowData.NetworkPermissions != nil && len(workflowData.NetworkPermissions.Allowed) > 0 {
		// Expand ecosystem identifiers and add individual domains
		expandedDomains := GetAllowedDomains(workflowData.NetworkPermissions)
		for _, domain := range expandedDomains {
			domainMap[domain] = true
		}
	}

	// Convert to slice
	allowedDomains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		allowedDomains = append(allowedDomains, domain)
	}
	SortStrings(allowedDomains)

	srtConfig := &SandboxRuntimeConfig{
		Network: &SRTNetworkConfig{
			AllowedDomains:      allowedDomains,
			DeniedDomains:       []string{},
			AllowUnixSockets:    []string{"/var/run/docker.sock"},
			AllowLocalBinding:   false,
			AllowAllUnixSockets: true,
		},
		Filesystem: &SRTFilesystemConfig{
			DenyRead:   []string{},
			AllowWrite: []string{".", "/home/runner/.copilot", "/tmp"},
			DenyWrite:  []string{},
		},
		IgnoreViolations:          map[string][]string{},
		EnableWeakerNestedSandbox: true,
	}

	// Apply user-provided non-network config (filesystem, ignoreViolations, enableWeakerNestedSandbox)
	var userConfig *SandboxRuntimeConfig
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Config != nil {
		userConfig = sandboxConfig.Agent.Config
	} else if sandboxConfig.Config != nil {
		userConfig = sandboxConfig.Config
	}

	if userConfig != nil {
		sandboxLog.Print("Applying user-provided SRT config (filesystem, ignoreViolations, enableWeakerNestedSandbox)")

		// Apply filesystem config if provided
		if userConfig.Filesystem != nil {
			srtConfig.Filesystem = userConfig.Filesystem
			// Normalize nil slices
			if srtConfig.Filesystem.DenyRead == nil {
				srtConfig.Filesystem.DenyRead = []string{}
			}
			if srtConfig.Filesystem.AllowWrite == nil {
				srtConfig.Filesystem.AllowWrite = []string{}
			}
			if srtConfig.Filesystem.DenyWrite == nil {
				srtConfig.Filesystem.DenyWrite = []string{}
			}
		}

		// Apply ignoreViolations if provided
		if userConfig.IgnoreViolations != nil {
			srtConfig.IgnoreViolations = userConfig.IgnoreViolations
		}

		// Note: EnableWeakerNestedSandbox defaults to true in srtConfig above.
		// We only override it with the user's value if they provided a config.
		// Since Go's bool zero value is false, if user doesn't specify this field,
		// it will be false in userConfig. This means users must explicitly set it
		// to true if they want it enabled when providing custom config.
		// This is intentional: providing custom config opts into full control.
		srtConfig.EnableWeakerNestedSandbox = userConfig.EnableWeakerNestedSandbox
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(srtConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal SRT config to JSON: %w", err)
	}

	sandboxLog.Printf("Generated SRT config: %s", string(jsonBytes))
	return string(jsonBytes), nil
}

// validateMountsSyntax validates that mount strings follow the correct syntax
// Expected format: "source:destination:mode" where mode is either "ro" or "rw"
func validateMountsSyntax(mounts []string) error {
	for i, mount := range mounts {
		// Split the mount string by colons
		parts := strings.Split(mount, ":")

		// Must have exactly 3 parts: source, destination, mode
		if len(parts) != 3 {
			return fmt.Errorf("invalid mount syntax at index %d: '%s'. Expected format: 'source:destination:mode' (e.g., '/host/path:/container/path:ro')", i, mount)
		}

		source := parts[0]
		dest := parts[1]
		mode := parts[2]

		// Validate that source and destination are not empty
		if source == "" {
			return fmt.Errorf("invalid mount at index %d: source path is empty in '%s'", i, mount)
		}
		if dest == "" {
			return fmt.Errorf("invalid mount at index %d: destination path is empty in '%s'", i, mount)
		}

		// Validate mode is either "ro" or "rw"
		if mode != "ro" && mode != "rw" {
			return fmt.Errorf("invalid mount at index %d: mode must be 'ro' (read-only) or 'rw' (read-write), got '%s' in '%s'", i, mode, mount)
		}

		sandboxLog.Printf("Validated mount %d: source=%s, dest=%s, mode=%s", i, source, dest, mode)
	}

	return nil
}

// validateSandboxConfig validates the sandbox configuration
// Returns an error if the configuration is invalid
func validateSandboxConfig(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return nil // No sandbox config is valid
	}

	sandboxConfig := workflowData.SandboxConfig

	// Validate mounts syntax if specified
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Mounts) > 0 {
		if err := validateMountsSyntax(agentConfig.Mounts); err != nil {
			return err
		}
	}

	// Validate that SRT is only used with Copilot engine
	if isSRTEnabled(workflowData) {
		// Check if the sandbox-runtime feature flag is enabled
		if !isFeatureEnabled(constants.SandboxRuntimeFeatureFlag, workflowData) {
			return fmt.Errorf("sandbox-runtime feature is experimental and requires the 'sandbox-runtime' feature flag to be enabled. Set 'features: { sandbox-runtime: true }' in frontmatter or set GH_AW_FEATURES=sandbox-runtime")
		}

		if workflowData.EngineConfig == nil || workflowData.EngineConfig.ID != "copilot" {
			engineID := "none"
			if workflowData.EngineConfig != nil {
				engineID = workflowData.EngineConfig.ID
			}
			return fmt.Errorf("sandbox-runtime is only supported with Copilot engine (current engine: %s)", engineID)
		}

		// Check for mutual exclusivity with AWF
		if workflowData.NetworkPermissions != nil && workflowData.NetworkPermissions.Firewall != nil && workflowData.NetworkPermissions.Firewall.Enabled {
			return fmt.Errorf("sandbox-runtime and AWF firewall cannot be used together; please use either 'sandbox: sandbox-runtime' or 'network.firewall' but not both")
		}
	}

	// Validate config structure if provided
	if sandboxConfig.Config != nil {
		if sandboxConfig.Type != SandboxTypeRuntime {
			return fmt.Errorf("custom sandbox config can only be provided when type is 'sandbox-runtime'")
		}
	}

	return nil
}
