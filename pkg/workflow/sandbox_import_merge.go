package workflow

import (
	"encoding/json"
	"strings"
)

// mergeImportedSandboxMounts merges sandbox.agent.mounts from imported shared workflows into
// the main workflow sandbox config using union + dedup semantics.
func (c *Compiler) mergeImportedSandboxMounts(mainSandbox *SandboxConfig, mergedSandbox string) *SandboxConfig {
	if strings.TrimSpace(mergedSandbox) == "" {
		return mainSandbox
	}
	if mainSandbox != nil && mainSandbox.Agent != nil && mainSandbox.Agent.Disabled {
		return mainSandbox
	}

	importedMounts := collectImportedSandboxMounts(c, mergedSandbox)
	if len(importedMounts) == 0 {
		return mainSandbox
	}

	if mainSandbox == nil {
		mainSandbox = &SandboxConfig{}
	}
	if mainSandbox.Agent == nil {
		mainSandbox.Agent = &AgentSandboxConfig{}
	}
	mainSandbox.Agent.Mounts = mergeUniqueSandboxMounts(importedMounts, mainSandbox.Agent.Mounts)
	return mainSandbox
}

func collectImportedSandboxMounts(c *Compiler, mergedSandbox string) []string {
	var mounts []string
	for line := range strings.SplitSeq(mergedSandbox, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" || line == "null" {
			continue
		}

		var rawSandbox any
		if err := json.Unmarshal([]byte(line), &rawSandbox); err != nil {
			continue
		}

		importedConfig := c.extractSandboxConfig(map[string]any{"sandbox": rawSandbox})
		if importedConfig == nil || importedConfig.Agent == nil || len(importedConfig.Agent.Mounts) == 0 {
			continue
		}
		mounts = append(mounts, importedConfig.Agent.Mounts...)
	}
	return mounts
}

func mergeUniqueSandboxMounts(primary []string, secondary []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	merged := make([]string, 0, len(primary)+len(secondary))

	appendUnique := func(mounts []string) {
		for _, mount := range mounts {
			if mount == "" {
				continue
			}
			if _, ok := seen[mount]; ok {
				continue
			}
			seen[mount] = struct{}{}
			merged = append(merged, mount)
		}
	}

	appendUnique(primary)
	appendUnique(secondary)
	return merged
}
