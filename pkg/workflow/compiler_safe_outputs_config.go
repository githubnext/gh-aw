package workflow

import (
	"encoding/json"
	"fmt"
)

func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	// Use the new generic builder instead of manual duplication
	config := buildSafeOutputConfigs(data.SafeOutputs)

	// Only add the env var if there are handlers to configure
	if len(config) > 0 {
		configJSON, err := json.Marshal(config)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Failed to marshal handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
	}
}

// addAllSafeOutputConfigEnvVars adds environment variables for all enabled safe output types
