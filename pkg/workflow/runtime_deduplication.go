package workflow

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
)

var runtimeDeduplicationLog = logger.New("workflow:runtime_deduplication")

// DeduplicateRuntimeSetupStepsFromCustomSteps removes runtime setup action steps from custom steps
// to avoid duplication when runtime steps are added before custom steps.
// This function parses the YAML custom steps, removes any steps that use runtime setup actions,
// and returns the deduplicated YAML.
//
// It preserves user-customized setup actions (e.g., with specific versions) and filters the corresponding
// runtime from the requirements so we don't generate a duplicate runtime setup step.
func DeduplicateRuntimeSetupStepsFromCustomSteps(customSteps string, runtimeRequirements []RuntimeRequirement) (string, []RuntimeRequirement, error) {
	if customSteps == "" || len(runtimeRequirements) == 0 {
		return customSteps, runtimeRequirements, nil
	}

	runtimeDeduplicationLog.Printf("Deduplicating runtime setup steps from custom steps (%d runtimes)", len(runtimeRequirements))

	versionComments := extractUsesVersionComments(customSteps)

	// Parse custom steps YAML
	var stepsWrapper map[string]any
	if err := yaml.Unmarshal([]byte(customSteps), &stepsWrapper); err != nil {
		return customSteps, runtimeRequirements, fmt.Errorf("failed to parse custom workflow steps from frontmatter. Custom steps must be valid GitHub Actions step syntax. Example:\nsteps:\n  - name: Setup\n    run: echo 'hello'\n  - name: Build\n    run: make build\nError: %w", err)
	}

	stepsVal, hasSteps := stepsWrapper["steps"]
	if !hasSteps {
		return customSteps, runtimeRequirements, nil
	}

	steps, ok := stepsVal.([]any)
	if !ok {
		return customSteps, runtimeRequirements, nil
	}

	actionRepoToReq := buildRuntimeActionRepoMap(runtimeRequirements)
	filterResult := filterRuntimeSetupSteps(steps, actionRepoToReq)
	filteredSteps := filterResult.steps
	filteredRuntimeIDs := filterResult.runtimeIDs
	removedCount := filterResult.removedCount
	preservedCount := filterResult.preservedCount

	if removedCount == 0 && preservedCount == 0 {
		runtimeDeduplicationLog.Print("  No duplicate runtime setup steps found")
		return customSteps, runtimeRequirements, nil
	}

	runtimeDeduplicationLog.Printf("  Removed %d duplicate runtime setup steps, preserved %d user-customized steps", removedCount, preservedCount)

	filteredRequirements := filterRuntimeRequirements(runtimeRequirements, filteredRuntimeIDs)

	// Convert back to YAML
	stepsWrapper["steps"] = filteredSteps

	restoreRuntimeSetupVersionComments(filteredSteps, versionComments)

	deduplicatedYAML, err := yaml.Marshal(stepsWrapper)
	if err != nil {
		return customSteps, runtimeRequirements, fmt.Errorf("failed to marshal deduplicated workflow steps to YAML. Step deduplication removes duplicate runtime setup actions (like actions/setup-node) from custom steps to avoid conflicts when automatic runtime detection adds them. This optimization ensures runtime setup steps appear before custom steps. Error: %w", err)
	}

	// Remove quotes from uses values with version comments
	// The YAML marshaller quotes strings containing # (for inline version comments)
	// but GitHub Actions expects unquoted uses values
	deduplicatedStr := unquoteUsesWithComments(string(deduplicatedYAML))

	return deduplicatedStr, filteredRequirements, nil
}

type runtimeSetupFilterResult struct {
	steps          []any
	runtimeIDs     map[string]struct{}
	removedCount   int
	preservedCount int
}

func buildRuntimeActionRepoMap(runtimeRequirements []RuntimeRequirement) map[string]*RuntimeRequirement {
	actionRepoToReq := make(map[string]*RuntimeRequirement)
	for i := range runtimeRequirements {
		if runtimeRequirements[i].Runtime.ActionRepo != "" {
			actionRepoToReq[runtimeRequirements[i].Runtime.ActionRepo] = &runtimeRequirements[i]
			runtimeDeduplicationLog.Printf("  Will check steps using action: %s", runtimeRequirements[i].Runtime.ActionRepo)
		}
	}
	return actionRepoToReq
}

func filterRuntimeSetupSteps(steps []any, actionRepoToReq map[string]*RuntimeRequirement) runtimeSetupFilterResult {
	result := runtimeSetupFilterResult{
		runtimeIDs: make(map[string]struct{}),
	}
	for _, stepAny := range steps {
		keep, preserved, removed := processRuntimeSetupStep(stepAny, actionRepoToReq, result.runtimeIDs)
		if keep {
			result.steps = append(result.steps, stepAny)
		}
		if preserved {
			result.preservedCount++
		}
		if removed {
			result.removedCount++
		}
	}
	return result
}

func processRuntimeSetupStep(stepAny any, actionRepoToReq map[string]*RuntimeRequirement, filteredRuntimeIDs map[string]struct{}) (bool, bool, bool) {
	step, ok := stepAny.(map[string]any)
	if !ok {
		return true, false, false
	}
	usesStr, ok := runtimeSetupStepUses(step)
	if !ok {
		return true, false, false
	}
	for actionRepo, req := range actionRepoToReq {
		if !strings.Contains(usesStr, actionRepo) {
			continue
		}
		if runtimeSetupStepCustomized(step, req) {
			filteredRuntimeIDs[req.Runtime.ID] = struct{}{}
			runtimeDeduplicationLog.Printf("  Preserving user-customized runtime setup step: %s", usesStr)
			return true, true, false
		}
		captureRuntimeSetupExtraFields(step, req)
		runtimeDeduplicationLog.Printf("  Removing duplicate runtime setup step: %s", usesStr)
		return false, false, true
	}
	return true, false, false
}

func runtimeSetupStepUses(step map[string]any) (string, bool) {
	usesVal, hasUses := step["uses"]
	if !hasUses {
		return "", false
	}
	usesStr, ok := usesVal.(string)
	return usesStr, ok
}

func runtimeSetupStepCustomized(step map[string]any, req *RuntimeRequirement) bool {
	withVal, hasWith := step["with"]
	if !hasWith {
		return false
	}
	withMap, isMap := withVal.(map[string]any)
	if !isMap || len(withMap) == 0 {
		return false
	}
	if req.Runtime.ID == "go" {
		return goRuntimeSetupCustomized(withMap)
	}
	if req.Runtime.VersionField == "" {
		return false
	}
	return runtimeVersionSetupCustomized(withMap, req) || nodeRuntimeSetupCustomized(withMap, req)
}

func goRuntimeSetupCustomized(withMap map[string]any) bool {
	for key := range withMap {
		if key != "go-version-file" && key != "cache" {
			return true
		}
	}
	if goVersionFile, ok := withMap["go-version-file"]; ok {
		if goVersionFileStr, isStr := goVersionFile.(string); isStr && goVersionFileStr != "go.mod" {
			return true
		}
	}
	return false
}

func runtimeVersionSetupCustomized(withMap map[string]any, req *RuntimeRequirement) bool {
	userVersion, hasVersion := withMap[req.Runtime.VersionField]
	if !hasVersion {
		return false
	}
	userVersionStr := fmt.Sprintf("%v", userVersion)
	if req.Runtime.DefaultVersion != "" && userVersionStr != req.Runtime.DefaultVersion {
		return true
	}
	if req.Version != "" && userVersionStr != req.Version {
		return true
	}
	return req.Runtime.DefaultVersion == "" && req.Version == ""
}

func nodeRuntimeSetupCustomized(withMap map[string]any, req *RuntimeRequirement) bool {
	if req.Runtime.ID != "node" {
		return false
	}
	nodeVersionFile, ok := withMap["node-version-file"].(string)
	return ok && strings.TrimSpace(nodeVersionFile) != ""
}

func captureRuntimeSetupExtraFields(step map[string]any, req *RuntimeRequirement) {
	withMap, ok := step["with"].(map[string]any)
	if !ok || len(withMap) == 0 {
		return
	}
	if req.ExtraFields == nil {
		req.ExtraFields = make(map[string]any)
	}
	for key, value := range withMap {
		if req.Runtime.VersionField != "" && key == req.Runtime.VersionField {
			continue
		}
		if req.Runtime.ID == "go" && (key == "go-version-file" || key == "cache") {
			continue
		}
		req.ExtraFields[key] = value
		runtimeDeduplicationLog.Printf("  Capturing extra field from setup step: %s = %v", key, value)
	}
}

func filterRuntimeRequirements(runtimeRequirements []RuntimeRequirement, filteredRuntimeIDs map[string]struct{}) []RuntimeRequirement {
	var filteredRequirements []RuntimeRequirement
	for _, req := range runtimeRequirements {
		if !setutil.Contains(filteredRuntimeIDs, req.Runtime.ID) {
			filteredRequirements = append(filteredRequirements, req)
		} else {
			runtimeDeduplicationLog.Printf("  Excluding runtime %s from generated setup steps (user has custom setup)", req.Runtime.ID)
		}
	}
	return filteredRequirements
}

func restoreRuntimeSetupVersionComments(filteredSteps []any, versionComments map[string]string) {
	for i, step := range filteredSteps {
		if stepMap, ok := step.(map[string]any); ok {
			if usesStr, ok := runtimeSetupStepUses(stepMap); ok {
				if versionComment, hasComment := versionComments[usesStr]; hasComment {
					stepMap["uses"] = usesStr + versionComment
					filteredSteps[i] = stepMap
				}
			}
		}
	}
}
