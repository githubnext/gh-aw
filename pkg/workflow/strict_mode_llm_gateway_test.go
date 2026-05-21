//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
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

	t.Run("copilot engine allows domains from known ecosystems with informational ecosystem guidance in strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		// These domains are from known ecosystems (python, node) and will emit warnings suggesting ecosystem identifiers
		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "registry.npmjs.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
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

	t.Run("codex engine allows domains from known ecosystems with informational ecosystem guidance", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		// These domains are from known ecosystems (python, node) and will emit warnings suggesting ecosystem identifiers
		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "registry.npmjs.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("codex", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
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

	t.Run("copilot engine requires sandbox.agent to be enabled in strict mode", func(t *testing.T) {
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
		// All engines use the general error message now (LLM gateway is always present)
		if err != nil && !strings.Contains(err.Error(), "sandbox.agent: false") {
			t.Errorf("Expected error about sandbox.agent: false, got: %v", err)
		}
		if err != nil && !strings.Contains(err.Error(), "disables the agent sandbox firewall") {
			t.Errorf("Expected error about disabling agent sandbox firewall, got: %v", err)
		}
	})

	t.Run("codex engine rejects sandbox.agent: false in strict mode", func(t *testing.T) {
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

// TestValidateStrictFirewall_EcosystemSuggestions tests ecosystem suggestions in informational messages
func TestValidateStrictFirewall_EcosystemSuggestions(t *testing.T) {
	t.Run("prints informational ecosystem suggestion when individual domain from ecosystem is used", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		output := testutil.CaptureStderr(t, func() {
			err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
			if err != nil {
				t.Errorf("Expected no error for individual ecosystem domain in strict mode, got: %v", err)
			}
		})
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
		}
		if !strings.Contains(output, "recommend using ecosystem identifiers") ||
			!strings.Contains(output, "'pypi.org'") ||
			!strings.Contains(output, "'python'") {
			t.Errorf("Expected informational ecosystem guidance in stderr, got: %q", output)
		}
	})

	t.Run("prints informational ecosystem suggestion without strict mode warning prefix", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"files.pythonhosted.org", "pypi.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		output := testutil.CaptureStderr(t, func() {
			err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
			if err != nil {
				t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
			}
		})
		if !strings.Contains(output, "recommend using ecosystem identifiers") {
			t.Errorf("Expected informational ecosystem guidance in stderr, got: %q", output)
		}
		if strings.Contains(output, "strict mode:") {
			t.Errorf("Expected informational ecosystem guidance to omit strict mode warning prefix, got: %q", output)
		}
		if !strings.Contains(output, "'files.pythonhosted.org' → 'python', 'pypi.org' → 'python'") {
			t.Errorf("Expected informational ecosystem guidance to include both python suggestions, got: %q", output)
		}
	})

	t.Run("prints informational ecosystem suggestion for multiple domains from same ecosystem", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"npmjs.org", "registry.npmjs.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
		}
	})

	t.Run("prints informational ecosystem suggestion for domains from different ecosystems", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "npmjs.org"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for individual ecosystem domains in strict mode, got: %v", err)
		}
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
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

	t.Run("mixed custom and ecosystem domains print informational guidance without warning count", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "custom-domain.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for mixed domains in strict mode, got: %v", err)
		}
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
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

	t.Run("ecosystem domain with custom domains prints informational guidance without warning count", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = true

		networkPerms := &NetworkPermissions{
			Allowed: []string{"pypi.org", "api.example.com"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		}

		err := compiler.validateStrictFirewall("copilot", networkPerms, nil)
		if err != nil {
			t.Errorf("Expected no error for ecosystem domain with custom domain, got: %v", err)
		}
		if compiler.GetWarningCount() != 0 {
			t.Errorf("Expected no warnings for informational ecosystem guidance, got %d", compiler.GetWarningCount())
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
