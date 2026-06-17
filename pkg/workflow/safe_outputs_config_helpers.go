package workflow

import (
	"encoding/json"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Configuration Helpers
// ========================================
//
// This file contains helper utilities used by the safe-outputs compiler:
// - JSON serialisation of custom job/script names for the handler manager

var safeOutputsConfigGenLog = logger.New("workflow:safe_outputs_config_generation_helpers")

func buildNormalizedSortedJSON(names []string, valueFn func(string) string) (string, error) {
	values := make(map[string]string, len(names))
	for _, name := range names {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(name)
		values[normalizedName] = valueFn(normalizedName)
	}

	keys := sortedMapKeys(values)

	ordered := make(map[string]string, len(keys))
	for _, k := range keys {
		ordered[k] = values[k]
	}

	jsonBytes, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// buildCustomSafeOutputJobsJSON builds a JSON mapping of custom safe output job names to empty
// strings, for use in the GH_AW_SAFE_OUTPUT_JOBS env var of the handler manager step.
// This allows the handler manager to silently skip messages handled by custom safe-output job
// steps rather than reporting them as "No handler loaded for message type '...'".
func buildCustomSafeOutputJobsJSON(data *WorkflowData) string {
	if data.SafeOutputs == nil || len(data.SafeOutputs.Jobs) == 0 {
		return ""
	}

	jobNames := make([]string, 0, len(data.SafeOutputs.Jobs))
	for jobName := range data.SafeOutputs.Jobs {
		jobNames = append(jobNames, jobName)
	}

	jsonStr, err := buildNormalizedSortedJSON(jobNames, func(string) string { return "" })
	if err != nil {
		safeOutputsConfigGenLog.Printf("Warning: failed to marshal custom safe output jobs: %v", err)
		return ""
	}
	return jsonStr
}
