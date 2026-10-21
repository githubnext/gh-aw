package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var cliProxyBashCodemodLog = logger.New("cli:codemod_cli_proxy_bash")

// getCLIProxyBashDisabledCodemod disables shell-backed CLI modes when shell execution is refused.
//
// cli-proxy mounts MCP servers as CLI executables that can only be invoked from a shell, so it is
// incompatible with 'tools.bash: false' (or an empty bash allowlist). GitHub gh-proxy mode is
// shell-backed for the same reason.
func getCLIProxyBashDisabledCodemod() Codemod {
	return Codemod{
		ID:           "cli-proxy-false-when-bash-disabled",
		Name:         "Disable shell-backed CLI modes when 'tools.bash' is disabled",
		Description:  "Adds an explicit 'cli-proxy: false' (or disables 'cli-proxy: true') and rewrites GitHub gh-proxy mode to local when bash is disabled, since CLI-mounted MCP servers and gh-proxy require shell access.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return applyCLIProxyBashDisabledCodemod(content, frontmatter, "")
		},
		ApplyWithContext: func(content string, frontmatter map[string]any, filePath string) (string, bool, error) {
			return applyCLIProxyBashDisabledCodemod(content, frontmatter, filePath)
		},
	}
}

func applyCLIProxyBashDisabledCodemod(content string, frontmatter map[string]any, filePath string) (string, bool, error) {
	effectiveTools, err := resolveEffectiveTools(content, frontmatter, filePath)
	if err != nil {
		cliProxyBashCodemodLog.Printf("Failed to resolve effective tools: %v", err)
		authored, ok := frontmatter["tools"].(map[string]any)
		if !ok {
			return content, false, nil
		}
		effectiveTools = authored
	}
	if !toolsRefuseBash(effectiveTools) {
		return content, false, nil
	}

	needsCLIProxyFalse := !frontmatterHasCLIProxyDisabled(frontmatter)
	// Use the effective tools map here on purpose: if an import contributes gh-proxy while
	// bash is disabled, adding a local tools.github.mode override is the codemod's fix.
	_, needsGitHubLocal := workflow.IsGitHubCLIProxyMode(effectiveTools)
	if !needsCLIProxyFalse && !needsGitHubLocal {
		return content, false, nil
	}

	newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
		if needsCLIProxyFalse {
			if err := f.Set([]string{"tools", "cli-proxy"}, false); err != nil {
				return err
			}
		}
		if needsGitHubLocal {
			if github, found := f.Lookup("tools", "github"); found {
				if _, mapping := github.(map[string]any); !mapping {
					return f.Set([]string{"tools", "github"}, map[string]any{"mode": "local"})
				}
			}
			return f.Set([]string{"tools", "github", "mode"}, "local")
		}
		return nil
	})
	if applied {
		cliProxyBashCodemodLog.Print("Disabled shell-backed CLI modes because tools.bash is disabled")
	}
	return newContent, applied, err
}

func toolsRefuseBash(toolsMap map[string]any) bool {
	// Keep these semantics aligned with workflow.isBashExplicitlyRefused: bash is fully refused
	// only by bash: false or an empty bash allowlist.
	bashValue, hasBash := toolsMap["bash"]
	if !hasBash {
		return false
	}
	switch v := bashValue.(type) {
	case bool:
		return !v
	case []any:
		return len(v) == 0
	}
	return false
}

// frontmatterHasCLIProxyDisabled reports whether tools.cli-proxy is already explicitly false.
func frontmatterHasCLIProxyDisabled(frontmatter map[string]any) bool {
	toolsMap, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return false
	}
	enabled, isBool := toolsMap["cli-proxy"].(bool)
	return isBool && !enabled
}
