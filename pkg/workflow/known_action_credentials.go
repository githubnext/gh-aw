// This file handles detection and cleanup of credentials left by known GitHub Actions.
//
// # Known Action Credentials Cleanup
//
// Certain well-known GitHub Actions authenticate to cloud providers or container
// registries and leave credentials on disk. If these actions are detected in a
// workflow, a cleanup step is injected before the agentic engine executes to
// remove those credentials.
//
// # Known Actions
//
// The following actions are recognized and their credential locations cleared:
//   - google-github-actions/auth  → ./gha-creds-*.json (GCP service account keys)
//   - aws-actions/configure-aws-credentials → ~/.aws/credentials (AWS access keys)
//   - azure/login                 → ~/.azure/ (Azure service principal credentials)
//   - docker/login-action         → ~/.docker/config.json (Docker registry auth tokens)
//   - actions/checkout            → ~/.ssh/ (SSH private keys from deploy key)
//
// # Integration
//
// Detection scans all merged workflow steps (custom steps, pre-steps, and
// pre-agent-steps) and stores the set of required cleanups in WorkflowData.
// The compiler then injects the cleanup step immediately before the agent
// execution step, at the same point as the git credentials cleaner.

package workflow

import (
	"fmt"
	"maps"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/logger"
)

var knownActionCredentialsLog = logger.New("workflow:known_action_credentials")

// knownCredentialLeakingAction describes a GitHub Action that leaves credentials
// on disk that must be cleaned before the agentic engine executes.
type knownCredentialLeakingAction struct {
	// actionPrefix is the action reference without the @version suffix
	// (e.g., "aws-actions/configure-aws-credentials")
	actionPrefix string
	// envVar is the environment variable set to "true" when this action is detected,
	// controlling which cleanup is performed by the shell script
	envVar string
	// credentialPaths describes the credentials the action creates (used in log messages)
	credentialPaths string
}

// knownCredentialLeakingActions is the ordered list of GitHub Actions known to leave
// credentials on disk. Order determines the env-var order in the generated YAML step.
var knownCredentialLeakingActions = []knownCredentialLeakingAction{
	{
		actionPrefix:    "google-github-actions/auth",
		envVar:          "GH_AW_CLEAN_GCP",
		credentialPaths: "./gha-creds-*.json (GCP service account keys)",
	},
	{
		actionPrefix:    "aws-actions/configure-aws-credentials",
		envVar:          "GH_AW_CLEAN_AWS",
		credentialPaths: "~/.aws/credentials (AWS access keys)",
	},
	{
		actionPrefix:    "azure/login",
		envVar:          "GH_AW_CLEAN_AZURE",
		credentialPaths: "~/.azure/ (Azure service principal credentials)",
	},
	{
		actionPrefix:    "docker/login-action",
		envVar:          "GH_AW_CLEAN_DOCKER",
		credentialPaths: "~/.docker/config.json (Docker registry auth tokens)",
	},
	{
		actionPrefix:    "actions/checkout",
		envVar:          "GH_AW_CLEAN_SSH",
		credentialPaths: "~/.ssh/ (SSH private keys from deploy key)",
	},
}

// DetectKnownCredentialLeakingActions scans a list of workflow steps and returns a
// map of environment variable names to true for each known credential-leaking action
// found. The returned map is used to generate the cleanup step env block.
// Returns nil when no known actions are detected.
func DetectKnownCredentialLeakingActions(steps []any) map[string]bool {
	detected := map[string]bool{}

	for _, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		uses, ok := stepMap["uses"].(string)
		if !ok || uses == "" {
			continue
		}

		// Strip inline comment annotations (e.g., "actions/checkout@v4 # pinned")
		uses = strings.TrimSpace(strings.SplitN(uses, " #", 2)[0])
		// Strip the @version suffix
		actionRef, _, _ := strings.Cut(uses, "@")

		for _, known := range knownCredentialLeakingActions {
			if actionRef == known.actionPrefix {
				detected[known.envVar] = true
				knownActionCredentialsLog.Printf(
					"Detected known credential-leaking action: %s → will clean %s",
					actionRef, known.credentialPaths,
				)
			}
		}
	}

	if len(detected) == 0 {
		return nil
	}
	return detected
}

// detectKnownCredentialLeakingActionsFromYAML parses a YAML steps string and delegates
// to DetectKnownCredentialLeakingActions. The YAML string may use a "steps:" wrapper
// (as produced by processAndMergeSteps) or be a bare sequence. Returns nil if no
// known actions are detected or if the YAML cannot be parsed.
func detectKnownCredentialLeakingActionsFromYAML(stepsYAML string) map[string]bool {
	if stepsYAML == "" {
		return nil
	}

	// Try wrapped form first: "steps:\n  - ...\n"
	var wrapped map[string]any
	if err := yaml.Unmarshal([]byte(stepsYAML), &wrapped); err == nil {
		if stepsVal, ok := wrapped["steps"]; ok {
			if steps, ok := stepsVal.([]any); ok {
				return DetectKnownCredentialLeakingActions(steps)
			}
		}
	}

	// Fall back to bare sequence form
	var steps []any
	if err := yaml.Unmarshal([]byte(stepsYAML), &steps); err == nil {
		return DetectKnownCredentialLeakingActions(steps)
	}

	return nil
}

// mergeKnownActionEnvVars merges two env-var maps into a single map.
// Either argument may be nil.
func mergeKnownActionEnvVars(a, b map[string]bool) map[string]bool {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	merged := make(map[string]bool, len(a)+len(b))
	maps.Copy(merged, a)
	maps.Copy(merged, b)
	return merged
}

// DetectKnownCredentialLeakingActionsFromWorkflowData scans all step collections in
// workflowData (custom steps, pre-steps, pre-agent-steps) and returns the merged set
// of environment variables required for the known-action credentials cleanup step.
// Returns nil when no known credential-leaking actions are found.
func DetectKnownCredentialLeakingActionsFromWorkflowData(workflowData *WorkflowData) map[string]bool {
	result := mergeKnownActionEnvVars(
		detectKnownCredentialLeakingActionsFromYAML(workflowData.CustomSteps),
		mergeKnownActionEnvVars(
			detectKnownCredentialLeakingActionsFromYAML(workflowData.PreSteps),
			detectKnownCredentialLeakingActionsFromYAML(workflowData.PreAgentSteps),
		),
	)
	if len(result) > 0 {
		knownActionCredentialsLog.Printf("Known credential-leaking actions detected, env vars: %v", result)
	}
	return result
}

// generateKnownActionsCredentialCleanerStep generates the YAML lines for a step that
// removes credentials left by known GitHub Actions before the agentic engine runs.
// Returns nil when envVars is empty (no known actions detected).
func (c *Compiler) generateKnownActionsCredentialCleanerStep(envVars map[string]bool) []string {
	if len(envVars) == 0 {
		return nil
	}

	lines := []string{
		"      - name: Clean known action credentials\n",
		"        continue-on-error: true\n",
		"        env:\n",
	}

	// Emit env vars in a stable, deterministic order (knownCredentialLeakingActions order)
	for _, known := range knownCredentialLeakingActions {
		if envVars[known.envVar] {
			lines = append(lines, fmt.Sprintf("          %s: \"true\"\n", known.envVar))
		}
	}

	lines = append(lines, "        run: bash \"${RUNNER_TEMP}/gh-aw/actions/clean_known_action_credentials.sh\"\n")
	return lines
}
