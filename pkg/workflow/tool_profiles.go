package workflow

import (
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"path"

	"github.com/github/gh-aw/pkg/parser"
)

//go:embed data/tool-profiles/*.md
var builtinToolProfiles embed.FS

func init() {
	err := fs.WalkDir(builtinToolProfiles, "data/tool-profiles", func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := builtinToolProfiles.ReadFile(file)
		if err != nil {
			return err
		}
		parser.RegisterBuiltinVirtualFile(parser.BuiltinToolProfilePathPrefix+path.Base(file), content)
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("loading built-in tool profiles: %v", err))
	}
}

func mergeToolProfileRuntimes(runtimes map[string]any, profileRuntimes string) (map[string]any, error) {
	defaults, err := mergeRuntimes(nil, profileRuntimes)
	if err != nil {
		return nil, fmt.Errorf("invalid tool profile runtimes: %w", err)
	}
	for name, config := range runtimes {
		current, currentIsMap := config.(map[string]any)
		profile, profileIsMap := defaults[name].(map[string]any)
		if currentIsMap && profileIsMap {
			maps.Copy(profile, current)
		} else {
			defaults[name] = config
		}
	}
	return defaults, nil
}

func validateToolProfileSandbox(data *WorkflowData) error {
	if data == nil || data.ParsedTools == nil || data.ParsedTools.Profile == "" {
		return nil
	}
	agent := getAgentConfig(data)
	if (agent != nil && agent.Disabled) || isCloudHypervisorRuntime(data) || isNVXRuntime(data) {
		return fmt.Errorf("tools.profile: %s requires an enabled Docker agent sandbox with profile mounts; use sandbox.agent.runtime: docker", data.ParsedTools.Profile)
	}
	return nil
}
