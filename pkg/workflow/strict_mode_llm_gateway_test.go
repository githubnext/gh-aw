//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestValidateStrictFirewall_LLMGatewaySupport tests the LLM gateway validation in strict mode
func TestValidateStrictFirewall_LLMGatewaySupport(t *testing.T) {
	t.Run("codex engine allows truly custom domains in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"custom-domain.com", "another-custom.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("codex", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for codex engine with truly custom domains in strict mode, got: %v", err)
		}
	})

	t.Run("copilot engine allows truly custom domains in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"custom-domain.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for copilot engine with truly custom domains in strict mode, got: %v", err)
		}
	})

	t.Run("copilot engine allows defaults in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"defaults"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for copilot engine with 'defaults', got: %v", err)
		}
	})

	t.Run("copilot engine allows known ecosystems in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"python", "node", "github"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for copilot engine with known ecosystem identifiers, got: %v", err)
		}
	})

	t.Run("copilot engine allows domains from known ecosystems with warning suggesting ecosystem identifier in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		// These domains are from known ecosystems (python, node) and will emit warnings suggesting ecosystem identifiers
		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "registry.npmjs.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		// Should have incremented warning count
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("codex engine also allows known ecosystems", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"python", "node", "github"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("codex", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for codex engine with known ecosystem identifiers, got: %v", err)
		}
	})

	t.Run("codex engine allows domains from known ecosystems with warning suggesting ecosystem identifier", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		// These domains are from known ecosystems (python, node) and will emit warnings suggesting ecosystem identifiers
		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "registry.npmjs.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("codex", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		// Should have incremented warning count
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("copilot engine allows mixed ecosystems and truly custom domains in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"python", "custom-domain.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for copilot engine with mixed ecosystems and truly custom domains in strict mode, got: %v", err)
		}
	})

	t.Run("claude engine allows truly custom domains in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"custom-domain.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("claude", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for claude engine with truly custom domains in strict mode, got: %v", err)
		}
	})

	t.Run("copilot engine with LLM gateway support requires sandbox.agent to be enabled in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"defaults"},
		}

		sandboxConfig := &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Disabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, sandboxConfig)
		if err == nil {
			t.Error("Expected error for copilot engine with sandbox.agent: false, got nil")
		}
		// Since copilot now supports LLM gateway, it should get the general error message
		// (not the engine-specific "does not support LLM gateway" message)
		if err != nil && !strings.Contains(err.Error(), "sandbox.agent: false") {
			t.Errorf("Expected error about sandbox.agent: false, got: %v", err)
		}
		if err != nil && !strings.Contains(err.Error(), "disables the agent sandbox firewall") {
			t.Errorf("Expected error about disabling agent sandbox firewall, got: %v", err)
		}
	})

	t.Run("codex engine with LLM gateway rejects sandbox.agent: false in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"defaults"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		sandboxConfig := &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Disabled: true,
			},
		}

		// codex engine now supports LLM gateway, so it should get the general sandbox.agent error
		err := compiler.validateStrictFirewall("codex", networkPerms, sandboxConfig)
		if err == nil {
			t.Error("Expected error for sandbox.agent: false in strict mode, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "sandbox.agent: false") {
			t.Errorf("Expected error about sandbox.agent: false, got: %v", err)
		}
		if err != nil && !strings.Contains(err.Error(), "disables the agent sandbox firewall") {
			t.Errorf("Expected error about disabling sandbox firewall, got: %v", err)
		}
	})

	t.Run("strict mode disabled allows custom domains for any engine", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = false

		networkPerms := &NetworkPermissions{
			Allowed: []string{"custom-domain.com"},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error when strict mode is disabled, got: %v", err)
		}
	})

	t.Run("copilot engine with wildcard allows bypass without LLM gateway check", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"*"},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for wildcard (skips all validation), got: %v", err)
		}
	})
}

// TestSupportsLLMGateway tests the SupportsLLMGateway method for each engine
func TestSupportsLLMGateway(t *testing.T) {
	registry := NewEngineRegistry()

	tests := []struct {
		engineID     string
		expectedPort int
		description  string
	}{
		{
			engineID:     "codex",
			expectedPort: constants.CodexLLMGatewayPort,
			description:  "Codex engine uses dedicated port for LLM gateway",
		},
		{
			engineID:     "claude",
			expectedPort: constants.ClaudeLLMGatewayPort,
			description:  "Claude engine uses dedicated port for LLM gateway",
		},
		{
			engineID:     "copilot",
			expectedPort: constants.CopilotLLMGatewayPort,
			description:  "Copilot engine uses dedicated port for LLM gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			engine, err := registry.GetEngine(tt.engineID)
			if err != nil {
				t.Fatalf("Failed to get engine '%s': %v", tt.engineID, err)
			}

			llmGatewayPort := engine.SupportsLLMGateway()
			if llmGatewayPort != tt.expectedPort {
				t.Errorf("Engine '%s': expected SupportsLLMGateway() = %d, got %d",
					tt.engineID, tt.expectedPort, llmGatewayPort)
			}
		})
	}
}

// TestValidateStrictFirewall_EcosystemSuggestions tests ecosystem suggestions in warning messages
func TestValidateStrictFirewall_EcosystemSuggestions(t *testing.T) {
	t.Run("warns with ecosystem suggestion when individual domain from ecosystem is used", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domain in strict mode, got: %v", err)
		}
		// Should have emitted a warning
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("warns with ecosystem suggestion for multiple domains from same ecosystem", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"npmjs.org", "registry.npmjs.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		// Should have emitted a warning
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("warns with ecosystem suggestion for domains from different ecosystems", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "npmjs.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		// Should have emitted a warning
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("truly custom domains are allowed without errors or warnings", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"custom-domain.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for truly custom domain in strict mode, got: %v", err)
		}
		// Should NOT have emitted a warning
		if compiler.GetWarningCount() != initialWarnings {
			t.Errorf("Expected no warnings for truly custom domain, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("mixed custom and ecosystem domains shows warnings only for ecosystem domains", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "custom-domain.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for mixed domains in strict mode, got: %v", err)
		}
		// Should have emitted a warning for pypi.org only
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("allows ecosystem identifiers without warnings", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"python", "node"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for ecosystem identifiers in strict mode, got: %v", err)
		}
		// Should NOT have emitted any warnings
		if compiler.GetWarningCount() != initialWarnings {
			t.Errorf("Expected no warnings for ecosystem identifiers, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})
}

// TestValidateStrictFirewall_CustomDomainBehavior tests the new behavior where truly custom domains are allowed
func TestValidateStrictFirewall_CustomDomainBehavior(t *testing.T) {
	t.Run("truly custom domain is allowed in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"api.example.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for truly custom domain, got: %v", err)
		}
	})

	t.Run("multiple truly custom domains are allowed in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"api.example.com", "cdn.myservice.io", "*.assets.example.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for multiple truly custom domains, got: %v", err)
		}
	})

	t.Run("ecosystem identifier with custom domains are allowed", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"python", "node", "api.example.com", "cdn.example.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for ecosystem identifiers with custom domains, got: %v", err)
		}
	})

	t.Run("ecosystem domain with custom domains emits warning for ecosystem domain only", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "api.example.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for ecosystem domain with custom domain, got: %v", err)
		}
		// Should have emitted a warning for pypi.org
		if compiler.GetWarningCount() != initialWarnings+1 {
			t.Errorf("Expected warning count to increase by 1, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})

	t.Run("defaults with custom domains are allowed without warnings", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"defaults", "api.example.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		initialWarnings := compiler.GetWarningCount()
		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for defaults with custom domains, got: %v", err)
		}
		// Should NOT have emitted any warnings
		if compiler.GetWarningCount() != initialWarnings {
			t.Errorf("Expected no warnings for defaults with custom domains, got %d warnings", compiler.GetWarningCount()-initialWarnings)
		}
	})
}
