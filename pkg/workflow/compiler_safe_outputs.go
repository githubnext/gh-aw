package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerSafeOutputsLog = logger.New("workflow:compiler_safe_outputs")

// mergeSafeJobsFromIncludedConfigs merges safe-jobs from included safe-outputs configurations
func (c *Compiler) mergeSafeJobsFromIncludedConfigs(topSafeJobs map[string]*SafeJobConfig, includedConfigs []string) (map[string]*SafeJobConfig, error) {
	compilerSafeOutputsLog.Printf("Merging safe-jobs from included configs: includedCount=%d", len(includedConfigs))
	result := topSafeJobs
	if result == nil {
		result = make(map[string]*SafeJobConfig)
	}

	for _, configJSON := range includedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the safe-outputs configuration
		var safeOutputsConfig map[string]any
		if err := json.Unmarshal([]byte(configJSON), &safeOutputsConfig); err != nil {
			continue // Skip invalid JSON
		}

		// Extract safe-jobs from the safe-outputs.jobs field
		includedSafeJobs := extractSafeJobsFromFrontmatter(map[string]any{
			"safe-outputs": safeOutputsConfig,
		})

		// Merge with conflict detection
		var err error
		result, err = mergeSafeJobs(result, includedSafeJobs)
		if err != nil {
			return nil, fmt.Errorf("failed to merge safe-jobs from includes: %w", err)
		}
	}

	return result, nil
}

// needsGitCommands checks if safe outputs configuration requires Git commands
func needsGitCommands(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil
}
