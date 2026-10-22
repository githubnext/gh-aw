package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var serenaMCPContainerLocationCodemodLog = logger.New("cli:codemod_serena_mcp_location")

// getSerenaMCPContainerLocationCodemod upgrades the legacy GitHub-hosted Serena MCP image
// to the project-maintained ghcr.io/oraios/serena container and the matching venv entrypoint.
func getSerenaMCPContainerLocationCodemod() Codemod {
	return Codemod{
		ID:           "serena-mcp-location-migration",
		Name:         "Update Serena MCP to the project-maintained container location",
		Description:  "Updates the legacy ghcr.io/github/serena-mcp-server image and the old 'serena' entrypoint to the current project-maintained ghcr.io/oraios/serena container and venv path.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasSerenaMCPServer(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				servers, ok := f.Document.Frontmatter["mcp-servers"].(map[string]any)
				if !ok {
					return nil
				}
				for name, value := range servers {
					server, ok := value.(map[string]any)
					if !ok || !strings.EqualFold(name, "serena") {
						continue
					}
					container, ok := server["container"].(string)
					if !ok {
						continue
					}
					lower := strings.ToLower(container)
					if !strings.Contains(lower, "serena-mcp-server") || strings.HasPrefix(lower, "ghcr.io/oraios/serena") {
						continue
					}
					if err := f.Set([]string{"mcp-servers", name, "container"}, "ghcr.io/oraios/serena:latest"); err != nil {
						return err
					}
					entrypoint, ok := server["entrypoint"].(string)
					if ok && (strings.EqualFold(entrypoint, "serena") || strings.HasSuffix(strings.ToLower(entrypoint), "/serena") || strings.HasSuffix(strings.ToLower(entrypoint), "\\serena")) {
						if err := f.Set([]string{"mcp-servers", name, "entrypoint"}, "/workspaces/serena/.venv/bin/serena"); err != nil {
							return err
						}
					}
				}
				return nil
			})
			if applied {
				serenaMCPContainerLocationCodemodLog.Print("Updated Serena MCP server to the project-maintained container location")
			}
			return newContent, applied, err
		},
	}
}

func hasSerenaMCPServer(frontmatter map[string]any) bool {
	mcpServersValue, hasMCPServers := frontmatter["mcp-servers"]
	if !hasMCPServers {
		return false
	}
	mcpServersMap, ok := mcpServersValue.(map[string]any)
	if !ok {
		return false
	}
	for serverName := range mcpServersMap {
		if strings.EqualFold(serverName, "serena") {
			return true
		}
	}
	return false
}
