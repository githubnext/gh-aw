package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var mountAsCLIsCodemodLog = logger.New("cli:codemod_mount_as_clis")

// getMountAsCLIsToCLIProxyCodemod creates a codemod that:
//  1. Renames tools.mount-as-clis to tools.cli-proxy.
//  2. Removes the deprecated features.mcp-cli flag.
func getMountAsCLIsToCLIProxyCodemod() Codemod {
	return Codemod{
		ID:           "mount-as-clis-to-cli-proxy",
		Name:         "Rename 'tools.mount-as-clis' to 'tools.cli-proxy' and remove 'features.mcp-cli'",
		Description:  "Renames the deprecated 'mount-as-clis:' field to 'cli-proxy:' inside the tools block, and removes the now-unnecessary 'features.mcp-cli: true' flag.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			hasMountAsCLIs := hasToolsMountAsCLIs(frontmatter)
			hasMCPCLIFeature := hasMCPCLIFeatureFlag(frontmatter)

			if !hasMountAsCLIs && !hasMCPCLIFeature {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if hasMountAsCLIs {
					if err := f.Rename([]string{"tools", "mount-as-clis"}, "cli-proxy"); err != nil {
						return err
					}
				}
				if hasMCPCLIFeature {
					if err := f.Remove("features", "mcp-cli"); err != nil {
						return err
					}
					return f.PruneEmpty("features")
				}
				return nil
			})
			if applied {
				mountAsCLIsCodemodLog.Print("Renamed tools.mount-as-clis to tools.cli-proxy and removed features.mcp-cli")
			}
			return newContent, applied, err
		},
	}
}

// hasToolsMountAsCLIs returns true if tools.mount-as-clis exists.
func hasToolsMountAsCLIs(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasMountAsCLIs := toolsMap["mount-as-clis"]
	return hasMountAsCLIs
}

// hasMCPCLIFeatureFlag returns true if features.mcp-cli is set.
func hasMCPCLIFeatureFlag(frontmatter map[string]any) bool {
	featuresAny, hasFeatures := frontmatter["features"]
	if !hasFeatures {
		return false
	}
	featuresMap, ok := featuresAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasMCPCLI := featuresMap["mcp-cli"]
	return hasMCPCLI
}
