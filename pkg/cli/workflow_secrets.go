package cli

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
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

// getSecretRequirementsForWorkflow extracts the engine from a workflow file and returns its required secrets
//
// NOTE: In future we will want to analyse more parts of the
// workflow to work out other secrets required, or detect that the particular
// authorization being used in a workflow means certain secrets are not required.
// For now we are only looking at the secrets implied by the engine used.
func getSecretRequirementsForWorkflow(workflowFile string) []SecretRequirement {
	workflowSecretsLog.Printf("Extracting secrets for workflow: %s", workflowFile)

	// Extract engine from workflow file
	engine := extractEngineIDFromFile(workflowFile)
	if engine == "" {
		workflowSecretsLog.Printf("No engine found in workflow %s, skipping", workflowFile)
		return nil
	}

	workflowSecretsLog.Printf("Workflow %s uses engine: %s", workflowFile, engine)

	// Get engine-specific secrets only (no system secrets, no optional)
	// System secrets will be added separately to avoid duplication
	return getSecretRequirementsForEngine(engine, true, true)
}
