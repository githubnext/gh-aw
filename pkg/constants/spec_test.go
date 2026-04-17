//go:build !integration

package constants_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/constants"
)

// TestSpec_EngineConstants_NameValues validates the documented engine name constant values.
// Spec section: "## Engine Constants"
func TestSpec_EngineConstants_NameValues(t *testing.T) {
	tests := []struct {
		name     string
		constant constants.EngineName
		expected string
	}{
		// From spec: constants.CopilotEngine // "copilot"
		{name: "CopilotEngine value", constant: constants.CopilotEngine, expected: "copilot"},
		// From spec: constants.ClaudeEngine // "claude"
		{name: "ClaudeEngine value", constant: constants.ClaudeEngine, expected: "claude"},
		// From spec: constants.CodexEngine // "codex"
		{name: "CodexEngine value", constant: constants.CodexEngine, expected: "codex"},
		// From spec: constants.GeminiEngine // "gemini"
		{name: "GeminiEngine value", constant: constants.GeminiEngine, expected: "gemini"},
		// From spec: constants.DefaultEngine // "copilot"
		{name: "DefaultEngine is copilot", constant: constants.DefaultEngine, expected: "copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant),
				"engine constant %s should have documented value %q", tt.name, tt.expected)
		})
	}
}

// TestSpec_EngineConstants_AgenticEngines validates the documented AgenticEngines list.
// Spec section: "// All supported engine names"
// Spec documents: constants.AgenticEngines // []string{"claude", "codex", "copilot", "gemini"}
//
// SPEC_MISMATCH: README documents 4 engines {"claude", "codex", "copilot", "gemini"} but
// the implementation includes a 5th engine "crush" (CrushEngine). The spec is outdated
// and does not reflect the current engine catalog.
func TestSpec_EngineConstants_AgenticEngines(t *testing.T) {
	engines := constants.AgenticEngines
	require.NotEmpty(t, engines, "AgenticEngines should be non-empty")

	// Spec documents at least these four engines; implementation may include additional engines.
	documentedEngines := []string{"claude", "codex", "copilot", "gemini"}
	for _, expected := range documentedEngines {
		assert.Contains(t, engines, expected,
			"AgenticEngines should contain documented engine %q", expected)
	}
}

// TestSpec_PublicAPI_GetEngineOption validates the documented GetEngineOption function.
// Spec section: "// Get engine metadata"
func TestSpec_PublicAPI_GetEngineOption(t *testing.T) {
	t.Run("GetEngineOption returns EngineOption for known engine", func(t *testing.T) {
		// Spec documents: opt := constants.GetEngineOption("copilot")
		// opt.Label = "GitHub Copilot"
		// opt.SecretName = "COPILOT_GITHUB_TOKEN"
		opt := constants.GetEngineOption("copilot")
		require.NotNil(t, opt, "GetEngineOption should return non-nil for documented engine 'copilot'")
		assert.Equal(t, "GitHub Copilot", opt.Label,
			"copilot EngineOption.Label should be 'GitHub Copilot' as documented")
		assert.Equal(t, "COPILOT_GITHUB_TOKEN", opt.SecretName,
			"copilot EngineOption.SecretName should be 'COPILOT_GITHUB_TOKEN' as documented")
	})

	t.Run("GetEngineOption returns nil for unknown engine", func(t *testing.T) {
		// Spec documents GetEngineOption returns nil for unknown engine values
		opt := constants.GetEngineOption("unknown-engine-xyz")
		assert.Nil(t, opt, "GetEngineOption should return nil for unknown engine names")
	})

	t.Run("EngineOption has documented fields", func(t *testing.T) {
		// Spec documents EngineOption fields: Value, Label, Description, SecretName,
		// AlternativeSecrets, EnvVarName, KeyURL, WhenNeeded
		opt := constants.GetEngineOption("copilot")
		require.NotNil(t, opt)

		assert.NotEmpty(t, opt.Value, "EngineOption.Value should be non-empty")
		assert.NotEmpty(t, opt.Label, "EngineOption.Label should be non-empty")
		assert.NotEmpty(t, opt.SecretName, "EngineOption.SecretName should be non-empty")
		assert.NotEmpty(t, opt.KeyURL, "EngineOption.KeyURL should be non-empty")
	})
}

// TestSpec_PublicAPI_GetAllEngineSecretNames validates the documented helper function.
// Spec section: "// Get all secret names for all engines"
func TestSpec_PublicAPI_GetAllEngineSecretNames(t *testing.T) {
	secrets := constants.GetAllEngineSecretNames()
	require.NotEmpty(t, secrets, "GetAllEngineSecretNames should return non-empty slice")

	// Spec documents COPILOT_GITHUB_TOKEN as one of the secrets
	assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN",
		"GetAllEngineSecretNames should include COPILOT_GITHUB_TOKEN as documented")
}

// TestSpec_SemanticTypes_StringAndIsValid validates the documented String() and IsValid()
// methods on semantic types that implement them.
// Spec section: "## Semantic Types" and "## Design Notes"
// Spec: "All semantic types implement String() string and IsValid() bool methods."
//
// SPEC_MISMATCH: README claims all semantic types implement String() and IsValid(), but
// EngineName and FeatureFlag do not have these methods in the implementation. Only
// JobName, StepID, CommandPrefix, Version, DocURL, URL, and MCPServerID (String only)
// implement these methods.
func TestSpec_SemanticTypes_StringAndIsValid(t *testing.T) {
	t.Run("EngineName string representation", func(t *testing.T) {
		// EngineName does not implement String()/IsValid() despite spec claiming all
		// semantic types do. Use string() conversion directly.
		e := constants.CopilotEngine
		assert.Equal(t, "copilot", string(e),
			"CopilotEngine underlying string value should be 'copilot' as documented")

		empty := constants.EngineName("")
		assert.Empty(t, string(empty),
			"empty EngineName should have empty string representation")
	})

	t.Run("FeatureFlag string representation", func(t *testing.T) {
		// FeatureFlag does not implement String()/IsValid() despite spec claiming all
		// semantic types do. Use string() conversion directly.
		f := constants.MCPGatewayFeatureFlag
		assert.NotEmpty(t, string(f),
			"MCPGatewayFeatureFlag should have non-empty string value")
	})

	t.Run("JobName implements String and IsValid", func(t *testing.T) {
		j := constants.AgentJobName
		// From spec: AgentJobName // "agent"
		assert.Equal(t, "agent", j.String(),
			"AgentJobName.String() should return 'agent' as documented")
		assert.True(t, j.IsValid(),
			"non-empty JobName.IsValid() should return true")

		empty := constants.JobName("")
		assert.False(t, empty.IsValid(),
			"empty JobName.IsValid() should return false")
	})

	t.Run("StepID implements String and IsValid", func(t *testing.T) {
		s := constants.CheckMembershipStepID
		// From spec: CheckMembershipStepID // "check_membership"
		assert.Equal(t, "check_membership", s.String(),
			"CheckMembershipStepID.String() should return 'check_membership' as documented")
		assert.True(t, s.IsValid(),
			"non-empty StepID.IsValid() should return true")

		empty := constants.StepID("")
		assert.False(t, empty.IsValid(),
			"empty StepID.IsValid() should return false")
	})

	t.Run("CommandPrefix implements String and IsValid", func(t *testing.T) {
		// From spec: CLIExtensionPrefix // "gh aw" — user-facing CLI prefix
		p := constants.CLIExtensionPrefix
		assert.Equal(t, "gh aw", p.String(),
			"CLIExtensionPrefix.String() should return 'gh aw' as documented")
		assert.True(t, p.IsValid(),
			"non-empty CommandPrefix.IsValid() should return true")

		empty := constants.CommandPrefix("")
		assert.False(t, empty.IsValid(),
			"empty CommandPrefix.IsValid() should return false")
	})

	t.Run("Version implements String and IsValid", func(t *testing.T) {
		v := constants.Version("1.0.0")
		assert.Equal(t, "1.0.0", v.String(),
			"Version.String() should return the underlying string value")
		assert.True(t, v.IsValid(),
			"non-empty Version.IsValid() should return true")

		empty := constants.Version("")
		assert.False(t, empty.IsValid(),
			"empty Version.IsValid() should return false")
	})
}

// TestSpec_FormattingConstants_Values validates the documented formatting constant values.
// Spec section: "## Formatting Constants"
func TestSpec_FormattingConstants_Values(t *testing.T) {
	// From spec: MaxExpressionLineLength // 120 — maximum line length for YAML expressions
	assert.Equal(t, constants.LineLength(120), constants.MaxExpressionLineLength,
		"MaxExpressionLineLength should be 120 as documented")

	// From spec: ExpressionBreakThreshold // 100 — threshold at which long lines get broken
	assert.Equal(t, constants.LineLength(100), constants.ExpressionBreakThreshold,
		"ExpressionBreakThreshold should be 100 as documented")

	// From spec: CLIExtensionPrefix // "gh aw" — user-facing CLI prefix
	assert.Equal(t, "gh aw", constants.CLIExtensionPrefix.String(),
		"CLIExtensionPrefix should be 'gh aw' as documented")
}

// TestSpec_NetworkPorts_Values validates the documented network port constant values.
// Spec section: "## Network Port Constants"
func TestSpec_NetworkPorts_Values(t *testing.T) {
	tests := []struct {
		name     string
		actual   int
		expected int
	}{
		// From spec: DefaultMCPGatewayPort // 80
		{name: "DefaultMCPGatewayPort", actual: constants.DefaultMCPGatewayPort, expected: 80},
		// From spec: DefaultMCPServerPort // 3000
		{name: "DefaultMCPServerPort", actual: constants.DefaultMCPServerPort, expected: 3000},
		// From spec: DefaultMCPInspectorPort // 3001
		{name: "DefaultMCPInspectorPort", actual: constants.DefaultMCPInspectorPort, expected: 3001},
		// From spec: MinNetworkPort // 1
		{name: "MinNetworkPort", actual: constants.MinNetworkPort, expected: 1},
		// From spec: MaxNetworkPort // 65535
		{name: "MaxNetworkPort", actual: constants.MaxNetworkPort, expected: 65535},
		// From spec: ClaudeLLMGatewayPort // 10000
		{name: "ClaudeLLMGatewayPort", actual: constants.ClaudeLLMGatewayPort, expected: 10000},
		// From spec: CodexLLMGatewayPort // 10001
		{name: "CodexLLMGatewayPort", actual: constants.CodexLLMGatewayPort, expected: 10001},
		// From spec: CopilotLLMGatewayPort // 10002
		{name: "CopilotLLMGatewayPort", actual: constants.CopilotLLMGatewayPort, expected: 10002},
		// From spec: GeminiLLMGatewayPort // 10003
		{name: "GeminiLLMGatewayPort", actual: constants.GeminiLLMGatewayPort, expected: 10003},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.actual,
				"port constant %s should have documented value %d", tt.name, tt.expected)
		})
	}
}

// TestSpec_RuntimeConfiguration_Timeouts validates the documented timeout constants.
// Spec section: "## Runtime Configuration"
func TestSpec_RuntimeConfiguration_Timeouts(t *testing.T) {
	// From spec: DefaultAgenticWorkflowTimeout // 20 * time.Minute
	assert.Equal(t, 20*time.Minute, constants.DefaultAgenticWorkflowTimeout,
		"DefaultAgenticWorkflowTimeout should be 20 minutes as documented")

	// From spec: DefaultToolTimeout // 60 * time.Second
	assert.Equal(t, 60*time.Second, constants.DefaultToolTimeout,
		"DefaultToolTimeout should be 60 seconds as documented")

	// From spec: DefaultMCPStartupTimeout // 120 * time.Second
	assert.Equal(t, 120*time.Second, constants.DefaultMCPStartupTimeout,
		"DefaultMCPStartupTimeout should be 120 seconds as documented")
}

// TestSpec_RuntimeConfiguration_RateLimits validates the documented rate limit constants.
// Spec section: "// Rate limits"
func TestSpec_RuntimeConfiguration_RateLimits(t *testing.T) {
	// From spec: DefaultRateLimitMax // 5 — max runs per window
	assert.Equal(t, 5, constants.DefaultRateLimitMax,
		"DefaultRateLimitMax should be 5 as documented")

	// From spec: DefaultRateLimitWindow // 60 — window in minutes
	assert.Equal(t, 60, constants.DefaultRateLimitWindow,
		"DefaultRateLimitWindow should be 60 as documented")
}

// TestSpec_FeatureFlags_Values validates the documented feature flag constant values.
// Spec section: "## Feature Flags"
func TestSpec_FeatureFlags_Values(t *testing.T) {
	tests := []struct {
		name     string
		constant constants.FeatureFlag
		expected string
	}{
		// From spec: MCPScriptsFeatureFlag // "mcp-scripts"
		{name: "MCPScriptsFeatureFlag", constant: constants.MCPScriptsFeatureFlag, expected: "mcp-scripts"},
		// From spec: MCPGatewayFeatureFlag // "mcp-gateway"
		{name: "MCPGatewayFeatureFlag", constant: constants.MCPGatewayFeatureFlag, expected: "mcp-gateway"},
		// From spec: DisableXPIAPromptFeatureFlag // "disable-xpia-prompt"
		{name: "DisableXPIAPromptFeatureFlag", constant: constants.DisableXPIAPromptFeatureFlag, expected: "disable-xpia-prompt"},
		// From spec: CopilotRequestsFeatureFlag // "copilot-requests"
		{name: "CopilotRequestsFeatureFlag", constant: constants.CopilotRequestsFeatureFlag, expected: "copilot-requests"},
		// From spec: CliProxyFeatureFlag // "cli-proxy"
		{name: "CliProxyFeatureFlag", constant: constants.CliProxyFeatureFlag, expected: "cli-proxy"},
		// From spec: IntegrityReactionsFeatureFlag // "integrity-reactions"
		{name: "IntegrityReactionsFeatureFlag", constant: constants.IntegrityReactionsFeatureFlag, expected: "integrity-reactions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant),
				"feature flag %s should have documented string value %q", tt.name, tt.expected)
		})
	}
}

// TestSpec_MCPServerIDs_Values validates the documented MCP server ID constants.
// Spec section: "### MCP Server IDs"
func TestSpec_MCPServerIDs_Values(t *testing.T) {
	// From spec: SafeOutputsMCPServerID // "safeoutputs"
	assert.Equal(t, "safeoutputs", string(constants.SafeOutputsMCPServerID),
		"SafeOutputsMCPServerID should be 'safeoutputs' as documented")

	// From spec: MCPScriptsMCPServerID // "mcpscripts"
	assert.Equal(t, "mcpscripts", string(constants.MCPScriptsMCPServerID),
		"MCPScriptsMCPServerID should be 'mcpscripts' as documented")

	// From spec: MCPScriptsMCPVersion // "1.0.0"
	assert.Equal(t, "1.0.0", string(constants.MCPScriptsMCPVersion),
		"MCPScriptsMCPVersion should be '1.0.0' as documented")

	// From spec: AgenticWorkflowsMCPServerID // "agenticworkflows"
	assert.Equal(t, "agenticworkflows", string(constants.AgenticWorkflowsMCPServerID),
		"AgenticWorkflowsMCPServerID should be 'agenticworkflows' as documented")
}

// TestSpec_JobNames_Values validates the documented job name constant values.
// Spec section: "### Job Names"
func TestSpec_JobNames_Values(t *testing.T) {
	tests := []struct {
		name     string
		constant constants.JobName
		expected string
	}{
		// From spec: AgentJobName // "agent"
		{name: "AgentJobName", constant: constants.AgentJobName, expected: "agent"},
		// From spec: ActivationJobName // "activation"
		{name: "ActivationJobName", constant: constants.ActivationJobName, expected: "activation"},
		// From spec: DetectionJobName // "detection"
		{name: "DetectionJobName", constant: constants.DetectionJobName, expected: "detection"},
		// From spec: ConclusionJobName // "conclusion"
		{name: "ConclusionJobName", constant: constants.ConclusionJobName, expected: "conclusion"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant.String(),
				"job name %s should have documented value %q", tt.name, tt.expected)
			assert.True(t, tt.constant.IsValid(),
				"documented job name %s should report IsValid() = true", tt.name)
		})
	}
}

// TestSpec_VersionConstraints_MinVersionValues validates the documented minimum version constraints.
// Spec section: "### Minimum Version Constraints"
func TestSpec_VersionConstraints_MinVersionValues(t *testing.T) {
	tests := []struct {
		name     string
		constant constants.Version
		expected string
	}{
		// From spec: AWFExcludeEnvMinVersion // "v0.25.3"
		{name: "AWFExcludeEnvMinVersion", constant: constants.AWFExcludeEnvMinVersion, expected: "v0.25.3"},
		// From spec: AWFCliProxyMinVersion // "v0.25.17"
		{name: "AWFCliProxyMinVersion", constant: constants.AWFCliProxyMinVersion, expected: "v0.25.17"},
		// From spec: CopilotNoAskUserMinVersion // "1.0.19"
		{name: "CopilotNoAskUserMinVersion", constant: constants.CopilotNoAskUserMinVersion, expected: "1.0.19"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant.String(),
				"version constraint %s should have documented value %q", tt.name, tt.expected)
		})
	}
}

// TestSpec_SystemSecrets_GlobalSlice validates the documented SystemSecrets global variable.
// Spec section: "### SystemSecretSpec"
func TestSpec_SystemSecrets_GlobalSlice(t *testing.T) {
	// Spec: "SystemSecrets is the global []SystemSecretSpec slice containing
	// GH_AW_GITHUB_TOKEN, GH_AW_AGENT_TOKEN, and GH_AW_GITHUB_MCP_SERVER_TOKEN."
	secrets := constants.SystemSecrets
	require.Len(t, secrets, 3, "SystemSecrets should contain exactly 3 documented secrets")

	names := make([]string, len(secrets))
	for i, s := range secrets {
		names[i] = s.Name
	}
	assert.Contains(t, names, "GH_AW_GITHUB_TOKEN",
		"SystemSecrets should include GH_AW_GITHUB_TOKEN as documented")
	assert.Contains(t, names, "GH_AW_AGENT_TOKEN",
		"SystemSecrets should include GH_AW_AGENT_TOKEN as documented")
	assert.Contains(t, names, "GH_AW_GITHUB_MCP_SERVER_TOKEN",
		"SystemSecrets should include GH_AW_GITHUB_MCP_SERVER_TOKEN as documented")
}
