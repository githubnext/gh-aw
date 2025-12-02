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
type SandboxRuntimeConfig struct {
	Network                   *SRTNetworkConfig    `yaml:"network,omitempty" json:"network,omitempty"`
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
// If custom config is provided, uses that; otherwise auto-generates based on network permissions
func generateSRTConfigJSON(workflowData *WorkflowData) (string, error) {
	if workflowData == nil {
		return "", fmt.Errorf("workflowData is nil")
	}

	sandboxConfig := workflowData.SandboxConfig
	if sandboxConfig == nil {
		return "", fmt.Errorf("sandbox config is nil")
	}

	var srtConfig *SandboxRuntimeConfig

	// Check new format first: sandbox.agent.config
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Config != nil {
		sandboxLog.Print("Using user-provided custom SRT config (new format)")
		srtConfig = sandboxConfig.Agent.Config
	} else if sandboxConfig.Config != nil {
		// Legacy format: sandbox.config
		sandboxLog.Print("Using user-provided custom SRT config (legacy format)")
		srtConfig = sandboxConfig.Config
	}

	// If user provided custom config, normalize and use it
	if srtConfig != nil {
		// Normalize nil slices to empty slices to ensure proper JSON serialization
		// (YAML parsing creates nil slices for [], but JSON marshals nil as null)
		if srtConfig.Network != nil {
			if srtConfig.Network.AllowedDomains == nil {
				srtConfig.Network.AllowedDomains = []string{}
			}
			if srtConfig.Network.DeniedDomains == nil {
				srtConfig.Network.DeniedDomains = []string{}
			}
			if srtConfig.Network.AllowUnixSockets == nil {
				srtConfig.Network.AllowUnixSockets = []string{}
			}
		}
		if srtConfig.Filesystem != nil {
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
	} else {
		// Auto-generate config based on network permissions
		sandboxLog.Print("Auto-generating SRT config from network permissions")

		// Merge Copilot default domains with network permissions
		// Similar logic to GetCopilotAllowedDomains but returns []string instead of comma-separated string
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

		srtConfig = &SandboxRuntimeConfig{
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
