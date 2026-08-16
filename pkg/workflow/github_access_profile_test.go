package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveGitHubAccessProfile(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		wantMode GitHubAccessMode
		wantCLI  bool
	}{
		{
			name:     "nil data defaults to mcp-local",
			data:     nil,
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
		{
			name:     "explicit cli",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"mode": "cli"}}},
			wantMode: GitHubAccessModeCLI,
			wantCLI:  true,
		},
		{
			name:     "legacy gh-proxy normalizes to cli",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"mode": "gh-proxy"}}},
			wantMode: GitHubAccessModeCLI,
			wantCLI:  true,
		},
		{
			name:     "explicit mcp-remote",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"mode": "mcp-remote"}}},
			wantMode: GitHubAccessModeMCPRemote,
			wantCLI:  false,
		},
		{
			name:     "legacy mode local normalizes to mcp-local",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"mode": "local"}}},
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
		{
			name:     "legacy mode remote normalizes to mcp-remote",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"mode": "remote"}}},
			wantMode: GitHubAccessModeMCPRemote,
			wantCLI:  false,
		},
		{
			name:     "legacy type remote resolves to mcp-remote",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"type": "remote"}}},
			wantMode: GitHubAccessModeMCPRemote,
			wantCLI:  false,
		},
		{
			name:     "omitted mode with toolsets stays mcp-local",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{"toolsets": []any{"default"}}}},
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
		{
			name:     "omitted mode bare github stays mcp-local (backward compatible)",
			data:     &WorkflowData{Tools: map[string]any{"github": map[string]any{}}},
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
		{
			name:     "no github tool is not cli",
			data:     &WorkflowData{Tools: map[string]any{}},
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
		{
			name:     "github disabled is not cli",
			data:     &WorkflowData{Tools: map[string]any{"github": false}},
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
		{
			name: "features.cli-proxy resolves to cli",
			data: &WorkflowData{
				Tools:    map[string]any{"github": map[string]any{}},
				Features: map[string]any{"cli-proxy": true},
			},
			wantMode: GitHubAccessModeCLI,
			wantCLI:  true,
		},
		{
			name: "features.integrity-reactions resolves to cli",
			data: &WorkflowData{
				Tools:    map[string]any{"github": map[string]any{}},
				Features: map[string]any{"integrity-reactions": true},
			},
			wantMode: GitHubAccessModeCLI,
			wantCLI:  true,
		},
		{
			name: "explicit mcp-local wins over features.cli-proxy",
			data: &WorkflowData{
				Tools:    map[string]any{"github": map[string]any{"mode": "mcp-local"}},
				Features: map[string]any{"cli-proxy": true},
			},
			wantMode: GitHubAccessModeMCPLocal,
			wantCLI:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := resolveGitHubAccessProfile(tt.data)
			assert.Equal(t, tt.wantMode, profile.Mode, "mode")
			assert.Equal(t, tt.wantCLI, profile.IsCLI(), "IsCLI")
		})
	}
}

func TestGitHubAccessProfileMCPTransport(t *testing.T) {
	assert.Equal(t, GitHubMCPModeLocal, GitHubAccessProfile{Mode: GitHubAccessModeMCPLocal}.MCPTransport())
	assert.Equal(t, GitHubMCPModeRemote, GitHubAccessProfile{Mode: GitHubAccessModeMCPRemote}.MCPTransport())
	assert.Equal(t, GitHubMCPModeLocal, GitHubAccessProfile{Mode: GitHubAccessModeCLI}.MCPTransport())
}

func TestGitHubAccessModePrecedence(t *testing.T) {
	githubTool := map[string]any{"mode": "mcp-local", "type": "remote"}
	profile := resolveGitHubAccessProfile(&WorkflowData{Tools: map[string]any{"github": githubTool}})

	assert.Equal(t, GitHubAccessModeMCPLocal, profile.Mode)
	assert.Equal(t, GitHubMCPModeLocal, getGitHubType(githubTool))
}

// TestNonMCPEngineAutoDerivesCLIViaEnforcement verifies that non-MCP engines resolve
// to cli through enforceMCPProxyTools (which writes an explicit cli mode) rather than
// through a registry lookup in the resolver.
func TestNonMCPEngineAutoDerivesCLIViaEnforcement(t *testing.T) {
	piEngine := NewPiEngine()
	tools, err := enforceMCPProxyTools(piEngine, map[string]any{"github": map[string]any{}})
	if err != nil {
		t.Fatalf("enforceMCPProxyTools failed: %v", err)
	}
	data := &WorkflowData{Tools: tools, EngineConfig: &EngineConfig{ID: "pi"}}
	profile := resolveGitHubAccessProfile(data)
	assert.Equal(t, GitHubAccessModeCLI, profile.Mode)
	assert.True(t, profile.IsCLI())
}
