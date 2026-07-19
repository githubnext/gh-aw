package workflow

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

func cloneRuntimeWithActionOverrides(base *Runtime, actionRepo, actionVersion string) *Runtime {
	if base == nil {
		return nil
	}

	customRuntime := *base
	customRuntime.Commands = slices.Clone(base.Commands)
	customRuntime.ManifestFiles = slices.Clone(base.ManifestFiles)
	customRuntime.ExtraWithFields = maps.Clone(base.ExtraWithFields)

	if actionRepo != "" {
		customRuntime.ActionRepo = actionRepo
	}
	if actionVersion != "" {
		customRuntime.ActionVersion = actionVersion
	}

	return &customRuntime
}

// applyRuntimeOverrides applies runtime version overrides from frontmatter
func applyRuntimeOverrides(runtimes map[string]any, requirements map[string]*RuntimeRequirement) {
	runtimeSetupLog.Printf("Applying runtime overrides for %d configured runtimes", len(runtimes))
	for runtimeID, configAny := range runtimes {
		configMap, ok := configAny.(map[string]any)
		if !ok {
			continue
		}
		override, ok := parseRuntimeOverride(configMap)
		if !ok {
			continue
		}

		if existing, exists := requirements[runtimeID]; exists {
			applyRuntimeOverrideToRequirement(runtimeID, existing, override)
		} else {
			addRuntimeOverrideRequirement(runtimeID, override, requirements)
		}
	}
}

type runtimeOverride struct {
	version       string
	hasVersion    bool
	actionRepo    string
	actionVersion string
	ifCondition   string
	cooldown      bool
	hasCooldown   bool
}

func parseRuntimeOverride(configMap map[string]any) (runtimeOverride, bool) {
	var override runtimeOverride
	versionAny, hasVersion := configMap["version"]
	if hasVersion {
		version, ok := runtimeVersionToString(versionAny)
		if !ok {
			return runtimeOverride{}, false
		}
		override.version = version
		override.hasVersion = true
	}
	override.actionRepo, _ = configMap["action-repo"].(string)
	override.actionVersion, _ = configMap["action-version"].(string)
	override.ifCondition, _ = configMap["if"].(string)
	override.cooldown, override.hasCooldown = configMap["cooldown"].(bool)
	return override, true
}

func runtimeVersionToString(versionAny any) (string, bool) {
	switch v := versionAny.(type) {
	case string:
		return v, true
	case int:
		return strconv.Itoa(v), true
	case float64:
		if v == float64(int(v)) {
			return strconv.Itoa(int(v)), true
		}
		return fmt.Sprintf("%g", v), true
	default:
		return "", false
	}
}

func applyRuntimeOverrideToRequirement(runtimeID string, existing *RuntimeRequirement, override runtimeOverride) {
	if override.hasVersion {
		runtimeSetupLog.Printf("Overriding version for runtime %s: %s", runtimeID, override.version)
		existing.Version = override.version
	}
	if override.ifCondition != "" {
		runtimeSetupLog.Printf("Setting if condition for runtime %s: %s", runtimeID, override.ifCondition)
		existing.IfCondition = override.ifCondition
	}
	if override.hasCooldown {
		runtimeSetupLog.Printf("Setting cooldown for runtime %s: %v", runtimeID, override.cooldown)
		existing.Cooldown = override.cooldown
	}
	if override.actionRepo != "" || override.actionVersion != "" {
		runtimeSetupLog.Printf("Applying custom action config for runtime %s: repo=%s, version=%s", runtimeID, override.actionRepo, override.actionVersion)
		existing.Runtime = cloneRuntimeWithActionOverrides(existing.Runtime, override.actionRepo, override.actionVersion)
	}
}

func addRuntimeOverrideRequirement(runtimeID string, override runtimeOverride, requirements map[string]*RuntimeRequirement) {
	runtimeSetupLog.Printf("Runtime %s not in requirements, checking known runtimes", runtimeID)
	runtime := findKnownRuntimeWithOverrides(runtimeID, override)
	if runtime == nil {
		runtimeSetupLog.Printf("Skipping unknown runtime %s: not in known runtimes and no action-repo specified", runtimeID)
		return
	}
	runtimeSetupLog.Printf("Adding new requirement for runtime %s: version=%s", runtimeID, override.version)
	requirements[runtimeID] = &RuntimeRequirement{
		Runtime:     runtime,
		Version:     override.version,
		IfCondition: override.ifCondition,
		Cooldown:    true,
	}
	if override.hasCooldown {
		requirements[runtimeID].Cooldown = override.cooldown
	}
}

func findKnownRuntimeWithOverrides(runtimeID string, override runtimeOverride) *Runtime {
	for _, knownRuntime := range knownRuntimes {
		if knownRuntime.ID != runtimeID {
			continue
		}
		if override.actionRepo != "" || override.actionVersion != "" {
			runtimeSetupLog.Printf("Cloning known runtime %s with custom action config: repo=%s, version=%s", runtimeID, override.actionRepo, override.actionVersion)
			return cloneRuntimeWithActionOverrides(knownRuntime, override.actionRepo, override.actionVersion)
		}
		runtimeSetupLog.Printf("Using known runtime %s as-is", runtimeID)
		return knownRuntime
	}
	return nil
}
