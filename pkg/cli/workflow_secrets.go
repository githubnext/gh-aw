package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var workflowSecretsLog = logger.New("cli:workflow_secrets")

// getSecretsRequirementsForWorkflows collects secrets from all provided workflow files
// and returns a deduplicated list including system secrets
func getSecretsRequirementsForWorkflows(workflowFiles []string) []SecretRequirement {
	workflowSecretsLog.Printf("Collecting secrets from %d workflow files", len(workflowFiles))

	var allRequirements []SecretRequirement
	seenSecrets := make(map[string]bool)

	// Map getRequiredSecretsForWorkflow over all workflows and union results
	for _, workflowFile := range workflowFiles {
		secrets := getSecretRequirementsForWorkflow(workflowFile)
		for _, req := range secrets {
			if !seenSecrets[req.Name] {
				seenSecrets[req.Name] = true
				allRequirements = append(allRequirements, req)
			}
		}
	}

	// Always add system secrets (deduplicated)
	for _, sys := range constants.SystemSecrets {
		if seenSecrets[sys.Name] {
			continue
		}
		seenSecrets[sys.Name] = true
		allRequirements = append(allRequirements, SecretRequirement{
			Name:           sys.Name,
			WhenNeeded:     sys.WhenNeeded,
			Description:    sys.Description,
			Optional:       sys.Optional,
			IsEngineSecret: false,
		})
	}

	workflowSecretsLog.Printf("Returning %d deduplicated secret requirements", len(allRequirements))
	return allRequirements
}

// getSecretRequirementsForWorkflow extracts the engine from a workflow file and returns its required secrets.
// It also extracts AuthDefinition secrets from inline engine definitions.
//
// NOTE: In future we will want to analyse more parts of the
// workflow to work out other secrets required, or detect that the particular
// authorization being used in a workflow means certain secrets are not required.
// For now we are only looking at the secrets implied by the engine used.
func getSecretRequirementsForWorkflow(workflowFile string) []SecretRequirement {
	workflowSecretsLog.Printf("Extracting secrets for workflow: %s", workflowFile)

	// Extract engine from workflow file
	engine, engineConfig := extractEngineConfigFromFile(workflowFile)
	if engine == "" {
		workflowSecretsLog.Printf("No engine found in workflow %s, skipping", workflowFile)
		return nil
	}

	workflowSecretsLog.Printf("Workflow %s uses engine: %s", workflowFile, engine)

	// Get engine-specific secrets only (no system secrets, no optional)
	// System secrets will be added separately to avoid duplication
	reqs := getSecretRequirementsForEngine(engine, false, false)

	// For inline engine definitions with an AuthDefinition, also include auth secrets.
	if engineConfig != nil && engineConfig.InlineProviderAuth != nil {
		authReqs := secretRequirementsFromAuthDefinition(engineConfig.InlineProviderAuth, engine)
		workflowSecretsLog.Printf("Adding %d auth definition secret(s) for workflow %s", len(authReqs), workflowFile)
		reqs = append(reqs, authReqs...)
	}

	return reqs
}

// extractEngineConfigFromFile parses a workflow file and returns the engine ID and config.
// Returns ("", nil) when the file cannot be read or parsed.
func extractEngineConfigFromFile(filePath string) (string, *workflow.EngineConfig) {
	content, err := readWorkflowFileContent(filePath)
	if err != nil {
		return "", nil
	}

	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil
	}

	compiler := &workflow.Compiler{}
	engineSetting, engineConfig := compiler.ExtractEngineConfig(result.Frontmatter)

	if engineConfig != nil && engineConfig.ID != "" {
		return engineConfig.ID, engineConfig
	}
	if engineSetting != "" {
		return engineSetting, engineConfig
	}
	return "copilot", engineConfig // Default engine
}

// readWorkflowFileContent reads a workflow file's content as a string.
func readWorkflowFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading workflow file %s: %w", filePath, err)
	}
	return string(content), nil
}
