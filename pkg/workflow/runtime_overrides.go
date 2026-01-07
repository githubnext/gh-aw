package workflow

import "fmt"

// applyRuntimeOverrides applies runtime version overrides from frontmatter
func applyRuntimeOverrides(runtimes map[string]any, requirements map[string]*RuntimeRequirement) {
	runtimeSetupLog.Printf("Applying runtime overrides for %d configured runtimes", len(runtimes))
	for runtimeID, configAny := range runtimes {
		// Parse runtime configuration
		configMap, ok := configAny.(map[string]any)
		if !ok {
			continue
		}

		// Extract version from config
		versionAny, hasVersion := configMap["version"]
		var version string
		if hasVersion {
			// Convert version to string (handle both string and numeric types)
			switch v := versionAny.(type) {
			case string:
				version = v
			case int:
				version = fmt.Sprintf("%d", v)
			case float64:
				// Check if it's a whole number
				if v == float64(int(v)) {
					version = fmt.Sprintf("%d", int(v))
				} else {
					version = fmt.Sprintf("%g", v)
				}
			default:
				continue
			}
		}

		// Extract action-repo and action-version from config
		actionRepo, _ := configMap["action-repo"].(string)
		actionVersion, _ := configMap["action-version"].(string)

		// Find or create runtime requirement
		if existing, exists := requirements[runtimeID]; exists {
			// Override version for existing requirement
			if hasVersion {
				runtimeSetupLog.Printf("Overriding version for runtime %s: %s", runtimeID, version)
				existing.Version = version
			}

			// If action-repo or action-version is specified, create a custom Runtime
			if actionRepo != "" || actionVersion != "" {
				runtimeSetupLog.Printf("Applying custom action config for runtime %s: repo=%s, version=%s", runtimeID, actionRepo, actionVersion)
				// Clone the existing runtime to avoid modifying the global knownRuntimes
				customRuntime := &Runtime{
					ID:              existing.Runtime.ID,
					Name:            existing.Runtime.Name,
					ActionRepo:      existing.Runtime.ActionRepo,
					ActionVersion:   existing.Runtime.ActionVersion,
					VersionField:    existing.Runtime.VersionField,
					DefaultVersion:  existing.Runtime.DefaultVersion,
					Commands:        existing.Runtime.Commands,
					ExtraWithFields: existing.Runtime.ExtraWithFields,
				}

				// Apply overrides
				if actionRepo != "" {
					customRuntime.ActionRepo = actionRepo
				}
				if actionVersion != "" {
					customRuntime.ActionVersion = actionVersion
				}

				existing.Runtime = customRuntime
			}
		} else {
			// Check if this is a known runtime
			var runtime *Runtime
			for _, knownRuntime := range knownRuntimes {
				if knownRuntime.ID == runtimeID {
					// Clone the known runtime if we need to customize it
					if actionRepo != "" || actionVersion != "" {
						runtime = &Runtime{
							ID:              knownRuntime.ID,
							Name:            knownRuntime.Name,
							ActionRepo:      knownRuntime.ActionRepo,
							ActionVersion:   knownRuntime.ActionVersion,
							VersionField:    knownRuntime.VersionField,
							DefaultVersion:  knownRuntime.DefaultVersion,
							Commands:        knownRuntime.Commands,
							ExtraWithFields: knownRuntime.ExtraWithFields,
						}

						// Apply overrides
						if actionRepo != "" {
							runtime.ActionRepo = actionRepo
						}
						if actionVersion != "" {
							runtime.ActionVersion = actionVersion
						}
					} else {
						runtime = knownRuntime
					}
					break
				}
			}

			// If runtime is known or we have custom action configuration, create a new requirement
			if runtime != nil {
				requirements[runtimeID] = &RuntimeRequirement{
					Runtime: runtime,
					Version: version,
				}
			}
			// If runtime is unknown and no action-repo specified, skip it (user might have typo)
		}
	}
}
