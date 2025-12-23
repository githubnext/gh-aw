package workflow

import (
	"strings"
	"testing"
)

func TestRenderGitHubMCPDockerConfig(t *testing.T) {
	tests := []struct {
		name     string
		options  GitHubMCPDockerOptions
		expected []string // Expected substrings in the output
		notFound []string // Substrings that should NOT be in the output
	}{
		{
			name: "Claude engine configuration (no type field, with effective token)",
			options: GitHubMCPDockerOptions{
				ReadOnly:           false,
				Toolsets:           "default",
				DockerImageVersion: "latest",
				CustomArgs:         nil,
				IncludeTypeField:   false,
				AllowedTools:       nil,
				EffectiveToken:     "${{ secrets.GITHUB_TOKEN }}",
			},
			expected: []string{
				`"command": "docker"`,
				`"run"`,
				`"-i"`,
				`"--rm"`,
				`"-e"`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN"`,
				`"GITHUB_TOOLSETS=default"`,
				`"ghcr.io/github/github-mcp-server:latest"`,
				`"env": {`,
				// Security fix: Now uses shell environment variable instead of GitHub Actions expression
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "$GITHUB_MCP_SERVER_TOKEN"`,
			},
			notFound: []string{
				`"type": "local"`,
				`"tools":`,
				`"GITHUB_READ_ONLY=1"`,
			},
		},
		{
			name: "Copilot engine configuration (with type field, no effective token)",
			options: GitHubMCPDockerOptions{
				ReadOnly:           false,
				Toolsets:           "default",
				DockerImageVersion: "latest",
				CustomArgs:         nil,
				IncludeTypeField:   true,
				AllowedTools:       []string{"create_issue", "issue_read"},
				EffectiveToken:     "",
			},
			expected: []string{
				`"type": "local"`,
				`"command": "docker"`,
				`"tools": [`,
				`"create_issue"`,
				`"issue_read"`,
				// Security fix: Now uses shell environment variable (with backslash for Copilot CLI interpolation)
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "\${GITHUB_MCP_SERVER_TOKEN}"`,
			},
			notFound: []string{
				`"GITHUB_READ_ONLY=1"`,
			},
		},
		{
			name: "Read-only mode enabled",
			options: GitHubMCPDockerOptions{
				ReadOnly:           true,
				Toolsets:           "default",
				DockerImageVersion: "v1.0.0",
				CustomArgs:         nil,
				IncludeTypeField:   false,
				AllowedTools:       nil,
				EffectiveToken:     "${{ secrets.TOKEN }}",
			},
			expected: []string{
				`"GITHUB_READ_ONLY=1"`,
				`"ghcr.io/github/github-mcp-server:v1.0.0"`,
			},
			notFound: []string{},
		},
		{
			name: "Custom args provided",
			options: GitHubMCPDockerOptions{
				ReadOnly:           false,
				Toolsets:           "default",
				DockerImageVersion: "latest",
				CustomArgs:         []string{"--verbose", "--debug"},
				IncludeTypeField:   false,
				AllowedTools:       nil,
				EffectiveToken:     "${{ secrets.TOKEN }}",
			},
			expected: []string{
				`"--verbose"`,
				`"--debug"`,
			},
			notFound: []string{},
		},
		{
			name: "Copilot with wildcard tools (no allowed tools specified)",
			options: GitHubMCPDockerOptions{
				ReadOnly:           false,
				Toolsets:           "default",
				DockerImageVersion: "latest",
				CustomArgs:         nil,
				IncludeTypeField:   true,
				AllowedTools:       nil,
				EffectiveToken:     "",
			},
			expected: []string{
				`"type": "local"`,
				`"tools": ["*"]`,
			},
			notFound: []string{},
		},
		{
			name: "Custom toolsets",
			options: GitHubMCPDockerOptions{
				ReadOnly:           false,
				Toolsets:           "repos,issues,pull_requests",
				DockerImageVersion: "latest",
				CustomArgs:         nil,
				IncludeTypeField:   false,
				AllowedTools:       nil,
				EffectiveToken:     "${{ secrets.TOKEN }}",
			},
			expected: []string{
				`"GITHUB_TOOLSETS=repos,issues,pull_requests"`,
			},
			notFound: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			RenderGitHubMCPDockerConfig(&yaml, tt.options)
			output := yaml.String()

			// Check for expected substrings
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
				}
			}

			// Check that unwanted substrings are not present
			for _, notFound := range tt.notFound {
				if strings.Contains(output, notFound) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nOutput:\n%s", notFound, output)
				}
			}
		})
	}
}

func TestRenderGitHubMCPDockerConfig_OutputStructure(t *testing.T) {
	// Test that the output has the expected JSON structure
	var yaml strings.Builder
	RenderGitHubMCPDockerConfig(&yaml, GitHubMCPDockerOptions{
		ReadOnly:           true,
		Toolsets:           "default",
		DockerImageVersion: "latest",
		CustomArgs:         []string{"--test"},
		IncludeTypeField:   true,
		AllowedTools:       []string{"tool1", "tool2"},
		EffectiveToken:     "",
	})

	output := yaml.String()

	// Verify the order of key elements
	typeIndex := strings.Index(output, `"type": "local"`)
	commandIndex := strings.Index(output, `"command": "docker"`)
	argsIndex := strings.Index(output, `"args": [`)
	toolsIndex := strings.Index(output, `"tools": [`)
	envIndex := strings.Index(output, `"env": {`)

	if typeIndex == -1 || commandIndex == -1 || argsIndex == -1 || toolsIndex == -1 || envIndex == -1 {
		t.Fatalf("Missing required JSON structure elements in output:\n%s", output)
	}

	// Verify order: type -> command -> args -> tools -> env
	if typeIndex >= commandIndex || commandIndex >= argsIndex || argsIndex >= toolsIndex || toolsIndex >= envIndex {
		t.Errorf("JSON elements are not in expected order. Indices: type=%d, command=%d, args=%d, tools=%d, env=%d\nOutput:\n%s",
			typeIndex, commandIndex, argsIndex, toolsIndex, envIndex, output)
	}
}
