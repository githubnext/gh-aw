package workflow

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

const artifactValidationJobName = "artifact_validation"

var artifactValidationLog = logger.New("workflow:safe_outputs_artifact_validation")
var artifactValidationUnsafeContext = regexp.MustCompile(`(?i)\b(?:secrets|needs|inputs|vars)\b`)
var artifactValidatorEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func parseArtifactValidationConfig(value any) (*ArtifactValidationConfig, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("CTR-033: safe-outputs.artifact-validation must be an object with steps; example: artifact-validation: {steps: [{run: 'node validate.cjs'}]}")
	}
	for _, key := range slices.Sorted(maps.Keys(object)) {
		if key != "steps" {
			return nil, fmt.Errorf("CTR-033: unknown artifact-validation field %q; use steps to configure the read-only validator", key)
		}
	}
	steps, ok := object["steps"].([]any)
	if !ok || len(steps) == 0 {
		return nil, fmt.Errorf("CTR-033: artifact-validation.steps must contain at least one required validator step")
	}
	return &ArtifactValidationConfig{Steps: steps}, nil
}

func validateArtifactValidation(data *WorkflowData) error {
	if data == nil || data.SafeOutputs == nil {
		return nil
	}
	config := data.SafeOutputs
	if config.artifactValidationParseError != nil {
		return config.artifactValidationParseError
	}
	if !usesPatchesAndCheckouts(config) {
		if config.ArtifactValidation != nil {
			return fmt.Errorf("CTR-033: artifact-validation requires a publishing create-pull-request or push-to-pull-request-branch handler")
		}
		return nil
	}
	if len(config.Steps) > 0 && !isWorkQueueParticipant(data) {
		return fmt.Errorf("CTR-034: safe-outputs.steps run with publication credentials; move artifact validators to safe-outputs.artifact-validation.steps and move trusted publishing code to a dedicated safe-output handler")
	}
	if config.ArtifactValidation == nil {
		return nil
	}
	if isWorkQueueEnabled(data) {
		return fmt.Errorf("CTR-033: artifact-validation is not supported with work-queue; use the queue's trusted claim adapter validation boundary")
	}
	for _, checkout := range data.CheckoutConfigs {
		if checkout != nil && (checkout.Repository != "" || checkout.Path != "" || checkout.Ref != "") {
			return fmt.Errorf("CTR-033: artifact-validation currently requires the workflow repository checkout at its trusted base; remove custom checkout repository, path, or ref")
		}
	}
	if create := config.CreatePullRequests; create != nil && (create.TargetRepoSlug != "" || create.HeadRepoSlug != "" || len(create.AllowedRepos) != 0 || create.BaseBranch != "" || len(create.AllowedBaseBranches) != 0) {
		return fmt.Errorf("CTR-033: artifact-validation currently supports the workflow repository and trusted event base only; remove target-repo, head-repo, allowed-repos, base-branch, or allowed-base-branches")
	}
	if push := config.PushToPullRequestBranch; push != nil && (push.TargetRepoSlug != "" || push.HeadRepoSlug != "" || len(push.AllowedRepos) != 0 || push.BaseBranch != "") {
		return fmt.Errorf("CTR-033: artifact-validation currently supports the workflow repository and trusted event base only; remove target-repo, head-repo, allowed-repos, or base-branch")
	}
	if len(config.ArtifactValidation.Steps) == 0 {
		return fmt.Errorf("CTR-033: artifact-validation.steps must contain at least one required validator step")
	}
	if len(config.Scripts) > 0 || len(config.Actions) > 0 || len(config.Jobs) > 0 {
		return fmt.Errorf("CTR-034: artifact-validation cannot share publication with custom safe-output jobs, scripts, or actions; put custom effects in a separate workflow")
	}
	for _, key := range slices.Sorted(maps.Keys(config.Env)) {
		if strings.HasPrefix(key, "GH_AW_ARTIFACT_") || key == "GH_AW_AGENT_OUTPUT" {
			return fmt.Errorf("CTR-033: safe-outputs.env cannot override trusted artifact environment %q", key)
		}
		if artifactHostExecutionEnv(key) {
			return fmt.Errorf("CTR-034: safe-outputs.env %q can alter trusted publisher execution; use publisher-specific authentication configuration instead", key)
		}
	}
	var globalEnv any = data.Env
	if data.RawFrontmatter != nil {
		globalEnv = data.RawFrontmatter["env"]
	}
	keys := make(map[string]any)
	switch typed := globalEnv.(type) {
	case map[string]any:
		keys = typed
	case map[string]string:
		for key := range typed {
			keys[key] = nil
		}
	case string:
		keys = parseEnvYAMLSection(typed)
	}
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		if artifactHostExecutionEnv(key) {
			return fmt.Errorf("CTR-034: workflow env %q can alter trusted publisher execution; remove it when using artifact-validation", key)
		}
	}
	if err := validateArtifactValidatorValue(globalEnv); err != nil {
		return fmt.Errorf("CTR-034: protected publication cannot expose credential-capable expressions through workflow env: %w", err)
	}
	for _, name := range slices.Sorted(maps.Keys(data.Jobs)) {
		raw := data.Jobs[name]
		if name == artifactValidationJobName {
			return fmt.Errorf("CTR-033: jobs.%s is reserved for the trusted artifact validation job; configure artifact-validation.steps instead", name)
		}
		if normalizeBuiltinJobAlias(name) == "safe_outputs" {
			if object, ok := raw.(map[string]any); ok {
				for _, key := range []string{"setup-steps", "pre-steps", "steps", "env", "if", "needs"} {
					if _, exists := object[key]; exists {
						return fmt.Errorf("CTR-034: jobs.%s.%s can bypass the protected publication boundary; configure validators in artifact-validation.steps instead", name, key)
					}
				}
			}
		}
	}
	for index, raw := range config.ArtifactValidation.Steps {
		if err := validateArtifactValidatorStep(raw, index); err != nil {
			return err
		}
	}
	artifactValidationLog.Printf("Validated %d required read-only artifact validator steps", len(config.ArtifactValidation.Steps))
	return nil
}

func validateArtifactValidatorStep(raw any, index int) error {
	step, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("CTR-033: artifact-validation.steps[%d] must be a step object", index)
	}
	if _, exists := step["if"]; exists {
		return fmt.Errorf("CTR-033: artifact-validation.steps[%d] cannot be conditional; every validator must succeed before publication", index)
	}
	for _, key := range slices.Sorted(maps.Keys(step)) {
		switch key {
		case "name", "id", "run", "env", "timeout-minutes", "continue-on-error":
		default:
			return fmt.Errorf("CTR-034: artifact-validation.steps[%d] cannot use %q; validators run as shell commands in the isolated Node container", index, key)
		}
	}
	run, _ := step["run"].(string)
	if strings.TrimSpace(run) == "" {
		return fmt.Errorf("CTR-033: artifact-validation.steps[%d] must specify a non-empty run", index)
	}
	if ExpressionPatternDotAll.MatchString(run) {
		return fmt.Errorf("CTR-034: artifact-validation.steps[%d] cannot interpolate expressions into run; pass non-secret values through env instead", index)
	}
	if id, _ := step["id"].(string); id == "artifact_prepare" || id == "artifact_finalize" || id == "approved_artifacts" {
		return fmt.Errorf("CTR-033: artifact-validation.steps[%d] uses reserved step id %q; choose a different id", index, id)
	}
	if value, exists := step["continue-on-error"]; exists && value != false {
		return fmt.Errorf("CTR-033: artifact-validation.steps[%d] cannot ignore failures; remove continue-on-error", index)
	}
	if _, err := MapToStep(step); err != nil {
		return fmt.Errorf("CTR-033: invalid artifact-validation.steps[%d]: %w", index, err)
	}
	for _, value := range step {
		if err := validateArtifactValidatorValue(value); err != nil {
			return fmt.Errorf("CTR-034: artifact-validation.steps[%d]: %w", index, err)
		}
	}
	switch env := step["env"].(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(env)) {
			if !artifactValidatorEnvAllowed(key) {
				return fmt.Errorf("CTR-033: artifact-validation.steps[%d] cannot override trusted artifact environment %q", index, key)
			}
		}
	case map[string]string:
		for _, key := range slices.Sorted(maps.Keys(env)) {
			if !artifactValidatorEnvAllowed(key) {
				return fmt.Errorf("CTR-033: artifact-validation.steps[%d] cannot override trusted artifact environment %q", index, key)
			}
		}
	}
	return nil
}

func artifactHostExecutionEnv(key string) bool {
	for _, prefix := range []string{"GH_AW_", "GITHUB_", "RUNNER_", "ACTIONS_", "NODE_", "LD_", "DYLD_", "GIT_", "DOCKER_"} {
		if strings.HasPrefix(strings.ToUpper(key), prefix) {
			return true
		}
	}
	switch strings.ToUpper(key) {
	case "PATH", "HOME", "BASH_ENV", "ENV", "SHELLOPTS", "BASHOPTS":
		return true
	}
	return false
}

func artifactValidatorEnvAllowed(key string) bool {
	if !artifactValidatorEnvName.MatchString(key) {
		return false
	}
	for _, prefix := range []string{"GH_AW_", "GITHUB_", "RUNNER_", "ACTIONS_", "NODE_", "LD_", "DYLD_", "DOCKER_"} {
		if strings.HasPrefix(strings.ToUpper(key), prefix) {
			return false
		}
	}
	switch strings.ToUpper(key) {
	case "PATH", "HOME", "BASH_ENV", "ENV", "SHELLOPTS", "BASHOPTS":
		return false
	}
	return true
}

func validateArtifactValidatorValue(value any) error {
	switch typed := value.(type) {
	case string:
		for _, match := range ExpressionPatternDotAll.FindAllStringSubmatch(typed, -1) {
			if len(match) > 1 && artifactValidationUnsafeContext.MatchString(maskQuotedExpressionLiterals(match[1])) {
				return fmt.Errorf("validator expressions cannot read secrets, custom job outputs, inputs, or variables; use trusted checkout files and the shared artifact manifest")
			}
		}
	case map[string]any:
		for _, child := range typed {
			if err := validateArtifactValidatorValue(child); err != nil {
				return err
			}
		}
	case map[string]string:
		for _, child := range typed {
			if err := validateArtifactValidatorValue(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := validateArtifactValidatorValue(child); err != nil {
				return err
			}
		}
	}
	return nil
}
