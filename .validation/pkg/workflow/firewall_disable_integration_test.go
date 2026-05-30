//go:build integration

package workflow

import (
	"testing"
)

func TestFirewallDisableIntegration(t *testing.T) {
	t.Run("sandbox agent false with allowed domains does not warn", func(t *testing.T) {
		frontmatter := map[string]any{
			"on":     "workflow_dispatch",
			"engine": "copilot",
			"network": map[string]any{
				"allowed": []any{"example.com"},
			},
			"sandbox": map[string]any{
				"agent": false,
			},
		}

		compiler := NewCompiler(
			WithSkipValidation(true),
		)

		// Extract network permissions
		networkPerms := compiler.extractNetworkPermissions(frontmatter)
		if networkPerms == nil {
			t.Fatal("Expected network permissions to be extracted")
		}

		// sandbox.agent: false replaces deprecated network.firewall: "disable" and should
		// not trigger warnings from deprecated network.firewall validation paths.
		initialWarnings := compiler.warningCount
		err := compiler.checkFirewallDisable(networkPerms)
		if err != nil {
			t.Errorf("Expected no error when using sandbox.agent: false, got: %v", err)
		}
		if compiler.warningCount != initialWarnings {
			t.Error("Should not emit warning when deprecated network.firewall is not used")
		}
	})

	t.Run("sandbox agent false in strict mode does not error", func(t *testing.T) {
		frontmatter := map[string]any{
			"on":     "workflow_dispatch",
			"engine": "copilot",
			"strict": true,
			"network": map[string]any{
				"allowed": []any{"example.com"},
			},
			"sandbox": map[string]any{
				"agent": false,
			},
		}

		compiler := NewCompiler()
		compiler.strictMode = true
		compiler.SetSkipValidation(true)

		networkPerms := compiler.extractNetworkPermissions(frontmatter)
		if networkPerms == nil {
			t.Fatal("Expected network permissions to be extracted")
		}

		err := compiler.checkFirewallDisable(networkPerms)
		if err != nil {
			t.Errorf("Expected no error in strict mode when using sandbox.agent: false, got: %v", err)
		}
	})
}
