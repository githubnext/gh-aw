package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ClaudeSettingsGenerator generates Claude Code settings configurations
type ClaudeSettingsGenerator struct{}

// PermissionsConfiguration represents the permissions section of Claude settings
type PermissionsConfiguration struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// ClaudeSettings represents the structure of Claude Code settings.json
type ClaudeSettings struct {
	Permissions *PermissionsConfiguration `json:"permissions,omitempty"`
	Hooks       *HookConfiguration        `json:"hooks,omitempty"`
}

// HookConfiguration represents the hooks section of settings
type HookConfiguration struct {
	PreToolUse []PreToolUseHook `json:"PreToolUse,omitempty"`
}

// PreToolUseHook represents a pre-tool-use hook configuration
type PreToolUseHook struct {
	Matcher string      `json:"matcher"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry represents a single hook entry
type HookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// GenerateSettingsJSON generates Claude Code settings JSON for network permissions
func (g *ClaudeSettingsGenerator) GenerateSettingsJSON() string {
	settings := ClaudeSettings{
		Permissions: &PermissionsConfiguration{
			Allow: []string{
				"Bash(npm run lint)",
				"Bash(npm run test:*)",
				"Read(~/.zshrc)",
			},
			Deny: []string{
				"Bash(curl:*)",
				"Read(./.env)",
				"Read(./.env.*)",
				"Read(./secrets/**)",
			},
		},
		Hooks: &HookConfiguration{
			PreToolUse: []PreToolUseHook{
				{
					Matcher: "WebFetch|WebSearch",
					Hooks: []HookEntry{
						{
							Type:    "command",
							Command: ".claude/hooks/network_permissions.py",
						},
					},
				},
			},
		},
	}

	settingsJSON, _ := json.MarshalIndent(settings, "", "  ")
	return string(settingsJSON)
}

// GenerateSettingsWorkflowStep generates a GitHub Actions workflow step that creates the settings file
func (g *ClaudeSettingsGenerator) GenerateSettingsWorkflowStep() GitHubActionStep {
	settingsJSON := g.GenerateSettingsJSON()

	runContent := fmt.Sprintf(`mkdir -p /tmp/.claude
cat > /tmp/.claude/settings.json << 'EOF'
%s
EOF`, settingsJSON)

	var lines []string
	lines = append(lines, "      - name: Generate Claude Settings")
	lines = append(lines, "        run: |")

	// Split the run content into lines and properly indent
	runLines := strings.Split(runContent, "\n")
	for _, line := range runLines {
		lines = append(lines, fmt.Sprintf("          %s", line))
	}

	return GitHubActionStep(lines)
}
