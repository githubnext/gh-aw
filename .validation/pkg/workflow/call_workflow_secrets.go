package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var callWorkflowSecretsLog = logger.New("workflow:call_workflow_secrets")

// extractCallWorkflowSecrets returns the list of secret names declared in the worker
// workflow's on.workflow_call.secrets section. This is used by orchestrator workflows
// to map secrets explicitly instead of using secrets: inherit.
//
// Priority: .lock.yml > .yml (same as extractCallWorkflowPermissions).
// .md sources are not consulted since they have not yet been compiled and do not
// have an on.workflow_call.secrets section.
//
// Returns nil when no compiled workflow file is found or when no secrets are declared,
// which signals the caller to fall back to secrets: inherit for backward compatibility.
func extractCallWorkflowSecrets(workflowName, markdownPath string) ([]string, error) {
	fileResult, err := findWorkflowFile(workflowName, markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow file for '%s': %w", workflowName, err)
	}

	// Priority: .lock.yml > .yml
	if fileResult.lockExists {
		return extractSecretsFromWorkflowFile(fileResult.lockPath)
	}

	if fileResult.ymlExists {
		return extractSecretsFromWorkflowFile(fileResult.ymlPath)
	}

	// No compiled file found — return nil so the caller falls back to secrets: inherit.
	callWorkflowSecretsLog.Printf("No compiled workflow file found for '%s', falling back to secrets: inherit", workflowName)
	return nil, nil
}

// extractSecretsFromWorkflowFile parses a .lock.yml or .yml workflow file and returns
// the secret names declared in its on.workflow_call.secrets section.
func extractSecretsFromWorkflowFile(filePath string) ([]string, error) {
	workflow, err := readWorkflowYAML(filePath)
	if err != nil {
		return nil, err
	}

	secrets := extractWorkflowCallSecretsFromParsed(workflow)
	callWorkflowSecretsLog.Printf("Extracted %d workflow_call secrets from %s", len(secrets), filePath)
	return secrets, nil
}

// extractWorkflowCallSecretsFromParsed extracts the secret names declared in
// on.workflow_call.secrets from an already-parsed workflow map.
func extractWorkflowCallSecretsFromParsed(workflow map[string]any) []string {
	onSection, ok := workflow["on"]
	if !ok {
		return nil
	}
	onMap, ok := onSection.(map[string]any)
	if !ok {
		return nil
	}
	workflowCallVal, ok := onMap["workflow_call"]
	if !ok {
		return nil
	}
	workflowCallMap, ok := workflowCallVal.(map[string]any)
	if !ok {
		return nil
	}
	secretsSection, ok := workflowCallMap["secrets"]
	if !ok {
		return nil
	}
	secretsMap, ok := secretsSection.(map[string]any)
	if !ok {
		return nil
	}

	names := make([]string, 0, len(secretsMap))
	for name := range secretsMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
