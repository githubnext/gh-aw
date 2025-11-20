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
	SandboxTypeDefault  SandboxType = "default"           // Uses AWF (Agent Workflow Firewall)
	SandboxTypeRuntime  SandboxType = "sandbox-runtime"   // Uses Anthropic Sandbox Runtime
)

// SandboxConfig represents the top-level sandbox configuration from front matter
type SandboxConfig struct {
	Type   SandboxType            `yaml:"type,omitempty"`   // Sandbox type: "default" or "sandbox-runtime"
	Config *SandboxRuntimeConfig  `yaml:"config,omitempty"` // Custom SRT config (optional)
}

// SandboxRuntimeConfig represents the Anthropic Sandbox Runtime configuration
// This matches the TypeScript SandboxRuntimeConfig interface
type SandboxRuntimeConfig struct {
	Network                    *SRTNetworkConfig            `json:"network,omitempty"`
	Filesystem                 *SRTFilesystemConfig         `json:"filesystem,omitempty"`
	IgnoreViolations           map[string][]string          `json:"ignoreViolations,omitempty"`
	EnableWeakerNestedSandbox  bool                         `json:"enableWeakerNestedSandbox"`
	AllowAllUnixSockets        bool                         `json:"allowAllUnixSockets"`
}

// SRTNetworkConfig represents network configuration for SRT
type SRTNetworkConfig struct {
	AllowedDomains    []string `json:"allowedDomains,omitempty"`
	DeniedDomains     []string `json:"deniedDomains,omitempty"`
	AllowUnixSockets  []string `json:"allowUnixSockets,omitempty"`
	AllowLocalBinding bool     `json:"allowLocalBinding"`
}

// SRTFilesystemConfig represents filesystem configuration for SRT
type SRTFilesystemConfig struct {
	DenyRead   []string `json:"denyRead,omitempty"`
	AllowWrite []string `json:"allowWrite,omitempty"`
	DenyWrite  []string `json:"denyWrite,omitempty"`
}

// isSRTEnabled checks if Sandbox Runtime is enabled for the workflow
func isSRTEnabled(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		sandboxLog.Print("No sandbox config, SRT disabled")
		return false
	}

	enabled := workflowData.SandboxConfig.Type == SandboxTypeRuntime
	sandboxLog.Printf("SRT enabled check: %v (type=%s)", enabled, workflowData.SandboxConfig.Type)
	return enabled
}

// getSandboxConfig returns the sandbox configuration
func getSandboxConfig(workflowData *WorkflowData) *SandboxConfig {
	if workflowData == nil {
		return nil
	}

	if workflowData.SandboxConfig != nil {
		if sandboxLog.Enabled() {
			sandboxLog.Printf("Retrieved sandbox config: type=%s", workflowData.SandboxConfig.Type)
		}
		return workflowData.SandboxConfig
	}

	return nil
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

	// If user provided custom config, use it
	if sandboxConfig.Config != nil {
		sandboxLog.Print("Using user-provided custom SRT config")
		srtConfig = sandboxConfig.Config
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
				AllowedDomains:    allowedDomains,
				DeniedDomains:     []string{},
				AllowUnixSockets:  []string{"/var/run/docker.sock"},
				AllowLocalBinding: false,
			},
			Filesystem: &SRTFilesystemConfig{
				DenyRead:   []string{},
				AllowWrite: []string{".", "/home/runner/.copilot", "/tmp"},
				DenyWrite:  []string{},
			},
			IgnoreViolations:          map[string][]string{},
			EnableWeakerNestedSandbox: true,
			AllowAllUnixSockets:       true,
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
