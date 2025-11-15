package workflow

import (
"strings"
"testing"
)

// TestPlaywrightExplicitLatestVersion verifies that when user explicitly sets version: "latest"
// it is respected and used in the rendered output
func TestPlaywrightExplicitLatestVersion(t *testing.T) {
t.Run("Explicit latest version for Copilot engine", func(t *testing.T) {
var yaml strings.Builder
playwrightTool := map[string]any{
"version":         "latest",
"allowed_domains": []string{"example.com"},
}

renderPlaywrightMCPConfigWithOptions(&yaml, playwrightTool, false, true, true)
output := yaml.String()

if !strings.Contains(output, "@playwright/mcp@latest") {
t.Errorf("Expected @playwright/mcp@latest when user explicitly sets version: latest, got: %s", output)
}
})

t.Run("Explicit latest version for Claude engine", func(t *testing.T) {
var yaml strings.Builder
playwrightTool := map[string]any{
"version":         "latest",
"allowed_domains": []string{"example.com"},
}

renderPlaywrightMCPConfigWithOptions(&yaml, playwrightTool, false, false, false)
output := yaml.String()

if !strings.Contains(output, "@playwright/mcp@latest") {
t.Errorf("Expected @playwright/mcp@latest when user explicitly sets version: latest, got: %s", output)
}
})

t.Run("Explicit latest version for Codex engine (TOML)", func(t *testing.T) {
var yaml strings.Builder
playwrightTool := map[string]any{
"version":         "latest",
"allowed_domains": []string{"example.com"},
}

renderPlaywrightMCPConfigTOML(&yaml, playwrightTool)
output := yaml.String()

if !strings.Contains(output, "@playwright/mcp@latest") {
t.Errorf("Expected @playwright/mcp@latest when user explicitly sets version: latest, got: %s", output)
}
})
}
