package workflow

import (
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
)

// validReactionContents is the set of valid GitHub ReactionContent enum values.
var validReactionContents = map[string]bool{
	"THUMBS_UP":   true,
	"THUMBS_DOWN": true,
	"HEART":       true,
	"HOORAY":      true,
	"CONFUSED":    true,
	"ROCKET":      true,
	"EYES":        true,
	"LAUGH":       true,
}

// validDisapprovalIntegrityLevels is the set of valid integrity levels for disapproval-integrity.
var validDisapprovalIntegrityLevels = map[string]bool{
	"none":       true,
	"unapproved": true,
	"approved":   true,
	"merged":     true,
}

// validEndorserMinIntegrityLevels is the set of valid integrity levels for endorser-min-integrity.
var validEndorserMinIntegrityLevels = map[string]bool{
	"unapproved": true,
	"approved":   true,
	"merged":     true,
}

// validateIntegrityReactions validates the integrity-reactions feature configuration.
// It checks that:
//   - endorsement-reactions and disapproval-reactions contain valid ReactionContent values
//   - the integrity-reactions feature flag requires min-integrity to be set (defaults will be injected)
//   - disapproval-integrity and endorser-min-integrity use valid integrity levels
//   - the integrity-reactions feature flag requires MCPG >= v0.2.18
func validateIntegrityReactions(tools *Tools, workflowName string, data *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig) error {
	if tools == nil || tools.GitHub == nil {
		return nil
	}

	github := tools.GitHub
	reactionFields := getIntegrityReactionFieldState(github)
	featureEnabled := isFeatureEnabled(constants.IntegrityReactionsFeatureFlag, data)

	if !reactionFields.hasExplicitFields && !featureEnabled {
		return nil
	}
	if err := validateIntegrityReactionFeatureGate(reactionFields.hasExplicitFields, featureEnabled, workflowName); err != nil {
		return err
	}
	if err := validateIntegrityReactionGatewaySupport(gatewayConfig, workflowName); err != nil {
		return err
	}
	if err := validateIntegrityReactionMinIntegrity(github, workflowName); err != nil {
		return err
	}
	return validateIntegrityReactionValues(github, reactionFields, workflowName)
}

type integrityReactionFieldState struct {
	hasExplicitFields       bool
	hasDisapprovalIntegrity bool
	hasEndorserMinIntegrity bool
}

func getIntegrityReactionFieldState(github *GitHubToolConfig) integrityReactionFieldState {
	hasDisapprovalIntegrity := github.DisapprovalIntegrity != ""
	hasEndorserMinIntegrity := github.EndorserMinIntegrity != ""
	return integrityReactionFieldState{
		hasExplicitFields: len(github.EndorsementReactions) > 0 ||
			len(github.DisapprovalReactions) > 0 ||
			hasDisapprovalIntegrity ||
			hasEndorserMinIntegrity,
		hasDisapprovalIntegrity: hasDisapprovalIntegrity,
		hasEndorserMinIntegrity: hasEndorserMinIntegrity,
	}
}

func validateIntegrityReactionFeatureGate(hasExplicitReactionFields, featureEnabled bool, workflowName string) error {
	if hasExplicitReactionFields && !featureEnabled {
		toolsValidationLog.Printf("Reaction fields present but integrity-reactions feature flag not enabled in workflow: %s", workflowName)
		return errors.New("invalid guard policy: 'endorsement-reactions', 'disapproval-reactions', 'disapproval-integrity', and 'endorser-min-integrity' require the 'integrity-reactions' feature flag to be enabled. Add 'features: integrity-reactions: true' to your workflow")
	}
	return nil
}

func validateIntegrityReactionGatewaySupport(gatewayConfig *MCPGatewayRuntimeConfig, workflowName string) error {
	if !mcpgSupportsIntegrityReactions(gatewayConfig) {
		version := string(constants.DefaultMCPGatewayVersion)
		if gatewayConfig != nil && gatewayConfig.Version != "" {
			version = gatewayConfig.Version
		}
		toolsValidationLog.Printf("integrity-reactions feature flag enabled but MCPG version %s < %s in workflow: %s", version, constants.MCPGIntegrityReactionsMinVersion, workflowName)
		return fmt.Errorf("invalid guard policy: 'integrity-reactions' feature flag requires MCPG >= %s, but the configured version is %s. Update the MCP gateway version to use this feature",
			constants.MCPGIntegrityReactionsMinVersion, version)
	}
	return nil
}

func validateIntegrityReactionMinIntegrity(github *GitHubToolConfig, workflowName string) error {
	if github.MinIntegrity == "" {
		toolsValidationLog.Printf("integrity-reactions feature flag enabled without min-integrity in workflow: %s", workflowName)
		return errors.New("invalid guard policy: 'integrity-reactions' feature flag requires 'github.min-integrity' to be set")
	}
	return nil
}

func validateIntegrityReactionValues(github *GitHubToolConfig, fields integrityReactionFieldState, workflowName string) error {
	for i, reaction := range github.EndorsementReactions {
		if !validReactionContents[reaction] {
			toolsValidationLog.Printf("Invalid endorsement-reactions value '%s' at index %d in workflow: %s", reaction, i, workflowName)
			return fmt.Errorf("invalid guard policy: 'endorsement-reactions' contains invalid value '%s'. Valid values: THUMBS_UP, THUMBS_DOWN, HEART, HOORAY, CONFUSED, ROCKET, EYES, LAUGH", reaction)
		}
	}

	// Validate disapproval-reactions values (if explicitly provided)
	for i, reaction := range github.DisapprovalReactions {
		if !validReactionContents[reaction] {
			toolsValidationLog.Printf("Invalid disapproval-reactions value '%s' at index %d in workflow: %s", reaction, i, workflowName)
			return fmt.Errorf("invalid guard policy: 'disapproval-reactions' contains invalid value '%s'. Valid values: THUMBS_UP, THUMBS_DOWN, HEART, HOORAY, CONFUSED, ROCKET, EYES, LAUGH", reaction)
		}
	}
	if fields.hasDisapprovalIntegrity && !validDisapprovalIntegrityLevels[github.DisapprovalIntegrity] {
		toolsValidationLog.Printf("Invalid disapproval-integrity value '%s' in workflow: %s", github.DisapprovalIntegrity, workflowName)
		return fmt.Errorf("invalid guard policy: 'disapproval-integrity' must be one of: 'none', 'unapproved', 'approved', 'merged'. Got: '%s'", github.DisapprovalIntegrity)
	}
	if fields.hasEndorserMinIntegrity && !validEndorserMinIntegrityLevels[github.EndorserMinIntegrity] {
		toolsValidationLog.Printf("Invalid endorser-min-integrity value '%s' in workflow: %s", github.EndorserMinIntegrity, workflowName)
		return fmt.Errorf("invalid guard policy: 'endorser-min-integrity' must be one of: 'unapproved', 'approved', 'merged'. Got: '%s'", github.EndorserMinIntegrity)
	}

	return nil
}
