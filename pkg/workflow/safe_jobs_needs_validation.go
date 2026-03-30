package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var safeJobsNeedsValidationLog = logger.New("workflow:safe_jobs_needs_validation")

// validateSafeJobNeeds validates the needs: declarations on custom safe-output jobs.
//
// For each custom safe-job, every entry in its needs: list must refer to a job that
// will actually exist in the compiled workflow. Valid targets are:
//
//   - "agent"        — main agent job (always present)
//   - "detection"    — threat-detection job (only when threat detection is enabled)
//   - "safe_outputs" — consolidated safe-outputs job (only when builtin safe-output types,
//     custom scripts, custom actions, or user-provided steps are configured)
//   - "upload_assets"— upload-assets job (only when upload-asset is configured)
//   - "unlock"       — unlock job (only when lock-for-agent is enabled)
//   - other custom safe-job names (normalised to underscore format)
//
// Each validated needs: entry is also rewritten to its normalized (underscore) form so
// that the compiled YAML references the correct job ID regardless of whether the author
// wrote "safe-outputs" or "safe_outputs".
//
// Additionally, cycles between custom safe-jobs are detected and reported as errors.
func validateSafeJobNeeds(data *WorkflowData) error {
	if data.SafeOutputs == nil || len(data.SafeOutputs.Jobs) == 0 {
		return nil
	}

	safeJobsNeedsValidationLog.Printf("Validating needs: declarations for %d safe-jobs", len(data.SafeOutputs.Jobs))

	validIDs := computeValidSafeJobNeeds(data)

	for originalName, jobConfig := range data.SafeOutputs.Jobs {
		if jobConfig == nil || len(jobConfig.Needs) == 0 {
			continue
		}

		normalizedJobName := stringutil.NormalizeSafeOutputIdentifier(originalName)
		for i, need := range jobConfig.Needs {
			normalizedNeed := stringutil.NormalizeSafeOutputIdentifier(need)
			if !validIDs[normalizedNeed] {
				return fmt.Errorf(
					"safe-outputs.jobs.%s: unknown needs target %q\n\nValid dependency targets for custom safe-jobs are:\n%s\n\n"+
						"Custom safe-jobs cannot depend on workflow control jobs such as 'conclusion' or 'activation'",
					originalName,
					need,
					formatValidNeedsTargets(validIDs),
				)
			}
			// Prevent a job from listing itself as a dependency
			if normalizedNeed == normalizedJobName {
				return fmt.Errorf(
					"safe-outputs.jobs.%s: a job cannot depend on itself in needs",
					originalName,
				)
			}
			// Rewrite the needs entry to its canonical underscore form so the compiled
			// YAML references the correct job ID (e.g. "safe-outputs" → "safe_outputs").
			jobConfig.Needs[i] = normalizedNeed
		}
	}

	// Detect cycles between custom safe-jobs
	if err := detectSafeJobCycles(data.SafeOutputs.Jobs); err != nil {
		return err
	}

	safeJobsNeedsValidationLog.Print("safe-job needs: validation passed")
	return nil
}

// computeValidSafeJobNeeds returns the set of job IDs that custom safe-jobs are
// allowed to depend on, based on the workflow configuration.
func computeValidSafeJobNeeds(data *WorkflowData) map[string]bool {
	valid := map[string]bool{
		string(constants.AgentJobName): true, // agent is always present
	}

	if data.SafeOutputs == nil {
		safeJobsNeedsValidationLog.Print("No safe-outputs configured, only agent job is a valid needs target")
		return valid
	}

	// safe_outputs consolidated job only exists when builtin safe-output types, custom scripts,
	// custom actions, or user-provided steps are configured. Custom safe-jobs (safe-outputs.jobs)
	// compile to separate jobs and do NOT create steps in the consolidated job.
	if consolidatedSafeOutputsJobWillExist(data.SafeOutputs) {
		valid[string(constants.SafeOutputsJobName)] = true
	}

	// detection job exists when threat detection is enabled
	if IsDetectionJobEnabled(data.SafeOutputs) {
		valid[string(constants.DetectionJobName)] = true
	}

	// upload_assets job exists when upload-asset is configured
	if data.SafeOutputs.UploadAssets != nil {
		valid[string(constants.UploadAssetsJobName)] = true
	}

	// unlock job exists when lock-for-agent is enabled
	if data.LockForAgent {
		valid[string(constants.UnlockJobName)] = true
	}

	// other custom safe-job names (normalized) are also valid targets
	for jobName := range data.SafeOutputs.Jobs {
		normalized := stringutil.NormalizeSafeOutputIdentifier(jobName)
		valid[normalized] = true
	}

	safeJobsNeedsValidationLog.Printf("Computed valid safe-job needs targets: %d total", len(valid))
	return valid
}

// consolidatedSafeOutputsJobWillExist returns true when the compiled workflow will include
// a "safe_outputs" job. The consolidated job is generated only when at least one builtin
// safe-output handler, custom script, custom action, or user-provided step is configured.
// Custom safe-jobs (safe-outputs.jobs) compile to SEPARATE jobs and therefore do not cause
// the consolidated safe_outputs job to be emitted.
func consolidatedSafeOutputsJobWillExist(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	// Scripts, actions, and user-provided steps always add to the consolidated job.
	if len(safeOutputs.Scripts) > 0 || len(safeOutputs.Actions) > 0 || len(safeOutputs.Steps) > 0 {
		return true
	}
	// Reuse the existing reflection-based check with the dynamic fields cleared.
	// hasAnySafeOutputEnabled will then fall through to reflection over safeOutputFieldMapping,
	// which covers every builtin pointer type (create-issue, add-comment, etc.).
	stripped := *safeOutputs
	stripped.Jobs = nil
	stripped.Scripts = nil
	stripped.Actions = nil
	stripped.Steps = nil
	return hasAnySafeOutputEnabled(&stripped)
}

// formatValidNeedsTargets returns a human-readable, sorted list of valid need targets.
func formatValidNeedsTargets(validIDs map[string]bool) string {
	targets := make([]string, 0, len(validIDs))
	for id := range validIDs {
		targets = append(targets, "  - "+id)
	}
	sort.Strings(targets)
	return strings.Join(targets, "\n")
}

// detectSafeJobCycles checks for dependency cycles among custom safe-jobs using DFS.
func detectSafeJobCycles(jobs map[string]*SafeJobConfig) error {
	if len(jobs) == 0 {
		return nil
	}
	safeJobsNeedsValidationLog.Printf("Detecting dependency cycles among %d custom safe-jobs", len(jobs))

	// Build normalized name mapping
	normalized := make(map[string]*SafeJobConfig, len(jobs))
	originalNames := make(map[string]string, len(jobs))
	for name, cfg := range jobs {
		n := stringutil.NormalizeSafeOutputIdentifier(name)
		normalized[n] = cfg
		originalNames[n] = name
	}

	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := make(map[string]int, len(normalized))

	var dfs func(node string, path []string) error
	dfs = func(node string, path []string) error {
		if state[node] == visited {
			return nil
		}
		if state[node] == visiting {
			// Build the cycle description using original names where available
			cycleNodes := make([]string, 0, len(path)+1)
			for _, p := range path {
				if orig, ok := originalNames[p]; ok {
					cycleNodes = append(cycleNodes, orig)
				} else {
					cycleNodes = append(cycleNodes, p)
				}
			}
			origNode := node
			if orig, ok := originalNames[node]; ok {
				origNode = orig
			}
			cycleNodes = append(cycleNodes, origNode)
			return fmt.Errorf(
				"safe-outputs.jobs: dependency cycle detected: %s",
				strings.Join(cycleNodes, " → "),
			)
		}

		state[node] = visiting
		cfg, exists := normalized[node]
		if exists && cfg != nil {
			for _, dep := range cfg.Needs {
				depNorm := stringutil.NormalizeSafeOutputIdentifier(dep)
				// Only recurse into other custom safe-jobs; skip generated jobs
				if _, isSafeJob := normalized[depNorm]; isSafeJob {
					if err := dfs(depNorm, append(path, node)); err != nil {
						return err
					}
				}
			}
		}
		state[node] = visited
		return nil
	}

	for node := range normalized {
		if err := dfs(node, nil); err != nil {
			return err
		}
	}

	return nil
}
