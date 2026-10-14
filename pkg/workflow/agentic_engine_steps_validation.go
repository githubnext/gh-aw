package workflow

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

var (
	engineSecretExpressionPattern = regexp.MustCompile(`\$\{\{[^}]*\}\}`)
	engineSecretReferencePattern  = regexp.MustCompile(`(?i)\bsecrets(?:\.[A-Za-z_][A-Za-z0-9_]*|\[['"][A-Za-z_][A-Za-z0-9_]*['"]\])`)
	agenticEngineReferencePattern = regexp.MustCompile(`(?i)(?:\b(?:copilot|claude|codex|gemini|agy|pi)(?:\.exe)?\b|@(?:github/copilot|anthropic-ai/claude-code|openai/codex|google/gemini-cli|earendil-works/pi-coding-agent)|(?:anthropics/claude-code-action|openai/codex-action|github/copilot-cli-action|google-github-actions/run-gemini-cli)|(?:copilot_github_token|anthropic_api_key|openai_api_key|gemini_api_key))`)
)

func hasAgenticEngineReference(content string) bool {
	return agenticEngineReferencePattern.MatchString(content)
}

func agenticEngineSecretNames(registry *EngineRegistry) map[string]struct{} {
	names := make(map[string]struct{})
	if registry == nil {
		return names
	}
	workflowData := &WorkflowData{}
	for _, engineID := range registry.GetSupportedEngines() {
		engine, err := registry.GetEngine(engineID)
		if err != nil {
			continue
		}
		for _, name := range engine.GetRequiredSecretNames(workflowData) {
			names[strings.ToUpper(name)] = struct{}{}
		}
	}
	return names
}

func referencesAgenticEngineSecret(value any, secretNames map[string]struct{}) bool {
	switch value := value.(type) {
	case string:
		for _, expression := range engineSecretExpressionPattern.FindAllString(value, -1) {
			for _, reference := range engineSecretReferencePattern.FindAllString(expression, -1) {
				reference = strings.ToLower(reference)
				name := strings.TrimPrefix(reference, "secrets.")
				if name == reference {
					name = strings.TrimPrefix(reference, "secrets[")
				}
				name = strings.Trim(name, `'" ]`)
				if _, ok := secretNames[strings.ToUpper(name)]; ok {
					return true
				}
			}
		}
	case map[string]any:
		for _, nested := range value {
			if referencesAgenticEngineSecret(nested, secretNames) {
				return true
			}
		}
	case []any:
		for _, nested := range value {
			if referencesAgenticEngineSecret(nested, secretNames) {
				return true
			}
		}
	}
	return false
}

func stepUsesProviderModelCatalogAPI(step map[string]any) bool {
	run, ok := step["run"].(string)
	if !ok {
		return false
	}
	// Provider keys are also used for direct model-list API requests that do not run an engine.
	run = strings.ToLower(run)
	for _, endpoint := range []string{
		"api.openai.com/v1/models",
		"api.anthropic.com/v1/models",
		"generativelanguage.googleapis.com/v1beta/models",
	} {
		if strings.Contains(run, endpoint) {
			return true
		}
	}
	return false
}

func stepReferencesAgenticEngineSecret(step map[string]any, secretNames map[string]struct{}) bool {
	return !stepUsesProviderModelCatalogAPI(step) && referencesAgenticEngineSecret(step, secretNames)
}

func jobReferencesAgenticEngineSecret(job map[string]any, secretNames map[string]struct{}) bool {
	for key, value := range job {
		if key != "steps" && referencesAgenticEngineSecret(value, secretNames) {
			return true
		}
	}
	return false
}

func isActivationSecretValidationStep(jobName string, step map[string]any) bool {
	if jobName != string(constants.ActivationJobName) {
		return false
	}
	run, ok := step["run"].(string)
	if !ok {
		return false
	}
	return strings.Contains(run, "validate_multi_secret.sh") || strings.Contains(run, "check_oauth_tokens.sh")
}

// validateAgenticEngineSteps inspects the final jobs so imported steps, safe-output
// steps, and built-in job augmentations are covered as well as custom jobs.
func (c *Compiler) validateAgenticEngineSteps(workflow map[string]any, strict bool, markdownPath string) error {
	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		return nil
	}
	jobNames := make([]string, 0, len(jobs))
	for name := range jobs {
		jobNames = append(jobNames, name)
	}
	slices.Sort(jobNames)
	engineSecrets := agenticEngineSecretNames(c.engineRegistry)
	for _, jobName := range jobNames {
		switch jobName {
		case string(constants.AgentJobName), string(constants.DetectionJobName), string(constants.EvalsJobName):
			continue
		}
		job, ok := jobs[jobName].(map[string]any)
		if !ok {
			continue
		}
		steps, ok := job["steps"].([]any)
		if !ok {
			continue
		}
		jobHasEngineSecret := jobReferencesAgenticEngineSecret(job, engineSecrets)
		if jobHasEngineSecret {
			msg := fmt.Sprintf("agentic engine credential in job %q suggests engine execution outside the agent sandbox; configure the engine with 'engine:' and run it in the agent job instead", jobName)
			if strict {
				return fmt.Errorf("strict mode: %s", msg)
			}
			fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", msg))
			c.IncrementWarningCount()
		}
		for i, value := range steps {
			step, ok := value.(map[string]any)
			if !ok {
				continue
			}
			stepHasEngineSecret := !jobHasEngineSecret && !isActivationSecretValidationStep(jobName, step) &&
				stepReferencesAgenticEngineSecret(step, engineSecrets)
			if !stepHasEngineSecret && !stepInvokesAgenticEngine(step) {
				continue
			}
			msg := fmt.Sprintf("agentic engine installation, invocation, or credential in job %q, step %d runs outside the agent sandbox; configure the engine with 'engine:' and run it in the agent job instead", jobName, i+1)
			if strict {
				return fmt.Errorf("strict mode: %s", msg)
			}
			fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", msg))
			c.IncrementWarningCount()
		}
	}
	return nil
}
