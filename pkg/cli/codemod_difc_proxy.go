package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var difcProxyCodemodLog = logger.New("cli:codemod_difc_proxy")

// getDIFCProxyToIntegrityProxyCodemod creates a codemod that migrates the deprecated
// 'features.difc-proxy' flag to the new 'tools.github.integrity-proxy' field.
//
// Migration rules:
//   - features.difc-proxy: true  → remove from features (proxy is enabled by default)
//   - features.difc-proxy: false → remove from features AND add
//     tools.github.integrity-proxy: false (to preserve the opt-out intent)
func getDIFCProxyToIntegrityProxyCodemod() Codemod {
	return Codemod{
		ID:           "features-difc-proxy-to-tools-github",
		Name:         "Migrate 'features.difc-proxy' to 'tools.github.integrity-proxy'",
		Description:  "Removes the deprecated 'features.difc-proxy' flag. The DIFC proxy is now enabled by default when guard policies are configured. If the flag was set to false, adds 'tools.github.integrity-proxy: false' to preserve the opt-out.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if features.difc-proxy exists
			flagValue, hasDIFCProxy := getDIFCProxyFlagValue(frontmatter)
			if !hasDIFCProxy {
				return content, false, nil
			}

			// Determine if we need to add integrity-proxy: false to tools.github
			addDisableFlag := !flagValue && hasToolsGithubMap(frontmatter)

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if err := f.Remove("features", "difc-proxy"); err != nil {
					return err
				}
				if err := f.PruneEmpty("features"); err != nil {
					return err
				}
				if _, exists := f.Lookup("tools", "github", "integrity-proxy"); addDisableFlag && !exists {
					return f.Set([]string{"tools", "github", "integrity-proxy"}, false)
				}
				return nil
			})
			if applied {
				if addDisableFlag {
					difcProxyCodemodLog.Print("Migrated features.difc-proxy: false → tools.github.integrity-proxy: false")
				} else {
					difcProxyCodemodLog.Print("Removed features.difc-proxy: true (proxy is now enabled by default)")
				}
			}
			return newContent, applied, err
		},
	}
}

// getDIFCProxyFlagValue returns the boolean value of features.difc-proxy and whether it exists.
// Note: string values are checked case-insensitively; "false" returns false, any other non-empty
// string returns true. This matches the existing behavior of isFeatureEnabled in features.go.
func getDIFCProxyFlagValue(frontmatter map[string]any) (bool, bool) {
	featuresAny, hasFeatures := frontmatter["features"]
	if !hasFeatures {
		return false, false
	}
	featuresMap, ok := featuresAny.(map[string]any)
	if !ok {
		return false, false
	}
	val, hasFlag := featuresMap["difc-proxy"]
	if !hasFlag {
		return false, false
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal, true
	}
	// Handle string values: "false" is treated as false; other non-empty strings as true.
	if strVal, ok := val.(string); ok {
		if strings.EqualFold(strVal, "false") {
			return false, true
		}
		return strVal != "", true
	}
	return false, true
}

// hasToolsGithubMap returns true if frontmatter has tools.github as a map (not just true/false).
func hasToolsGithubMap(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return false
	}
	githubAny, hasGitHub := toolsMap["github"]
	if !hasGitHub {
		return false
	}
	_, ok = githubAny.(map[string]any)
	return ok
}
