package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineLatestVersionCodemodLog = logger.New("cli:codemod_engine_latest_version")

// getPinEngineLatestVersionCodemod removes engine.version: latest so workflows use
// the compiler's pinned default engine version instead of an unpinned latest tag.
func getPinEngineLatestVersionCodemod() Codemod {
	return Codemod{
		ID:           "pin-engine-version",
		Name:         "Pin engine latest version",
		Description:  "Removes engine.version: latest so workflows use a pinned default engine version",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			engineAny, hasEngine := frontmatter["engine"]
			if !hasEngine {
				return content, false, nil
			}
			engineMap, ok := engineAny.(map[string]any)
			if !ok {
				return content, false, nil
			}
			versionAny, hasVersion := engineMap["version"]
			if !hasVersion {
				return content, false, nil
			}
			version, ok := versionAny.(string)
			if !ok || strings.TrimSpace(version) != "latest" {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return removeFieldFromBlock(lines, "version", "engine")
			})
			if applied {
				engineLatestVersionCodemodLog.Print("Removed engine.version: latest to use pinned default engine version")
			}
			return newContent, applied, err
		},
	}
}
