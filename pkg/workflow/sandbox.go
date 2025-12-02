package workflow

import (
	"encoding/json"
	"fmt"

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
// New format: { agent: "awf"|"srt"|{type, config}, mcp: {...} }
// Legacy format: "default"|"sandbox-runtime" or { type, config }
type SandboxConfig struct {
	// New fields
	Agent *AgentSandboxConfig `yaml:"agent,omitempty"` // Agent sandbox configuration
	MCP   *MCPGatewayConfig   `yaml:"mcp,omitempty"`   // MCP gateway configuration

	// Legacy fields (for backward compatibility)
	Type   SandboxType           `yaml:"type,omitempty"`   // Sandbox type: "default" or "sandbox-runtime"
	Config *SandboxRuntimeConfig `yaml:"config,omitempty"` // Custom SRT config (optional)
}

// AgentSandboxConfig represents the agent sandbox configuration
type AgentSandboxConfig struct {
	Type   SandboxType           `yaml:"type,omitempty"`   // Sandbox type: "awf" or "srt"
	Config *SandboxRuntimeConfig `yaml:"config,omitempty"` // Custom SRT config (optional)
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

// isSRTEnabled checks if Sandbox Runtime is enabled for the workflow
func isSRTEnabled(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		sandboxLog.Print("No sandbox config, SRT disabled")
		return false
	}

	config := workflowData.SandboxConfig

	// Check new format: sandbox.agent
	if config.Agent != nil {
		enabled := config.Agent.Type == SandboxTypeSRT || config.Agent.Type == SandboxTypeRuntime
		sandboxLog.Printf("SRT enabled check (new format): %v (type=%s)", enabled, config.Agent.Type)
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

// validateSandboxConfig validates the sandbox configuration
// Returns an error if the configuration is invalid
func validateSandboxConfig(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return nil // No sandbox config is valid
	}

	sandboxConfig := workflowData.SandboxConfig

	// Validate that SRT is only used with Copilot engine
	if isSRTEnabled(workflowData) {
		// Check if the sandbox-runtime feature flag is enabled
		if !isFeatureEnabled("sandbox-runtime", workflowData) {
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
