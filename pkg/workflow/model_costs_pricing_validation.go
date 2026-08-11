package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var modelCostsPricingValidationLog = logger.New("workflow:model_costs_pricing_validation")

// validateDefaultAiCreditsPricing returns an error when the workflow's
// models.default-ai-credits-pricing frontmatter is present and either:
//   - the effective AWF version is older than AWFDefaultAiCreditsPricingMinVersion
//     (the field is silently dropped during config resolution in older versions), or
//   - any price field has a non-positive value.
//
// Absent pricing (nil) is allowed; both checks are only enforced once a value
// is explicitly configured.
//
// The AWF api-proxy rejects zero rates as "not configured", so requiring
// positive values here prevents silent runtime failures for self-hosted models.
func validateDefaultAiCreditsPricing(workflowData *WorkflowData) error {
	p := workflowData.DefaultAiCreditsPricing
	if p == nil {
		return nil
	}
	firewallConfig := getFirewallConfig(workflowData)
	if !awfSupportsDefaultAiCreditsPricing(firewallConfig) {
		awfTag := getAWFImageTag(firewallConfig)
		modelCostsPricingValidationLog.Printf("Rejecting default-ai-credits-pricing: AWF tag %q predates %s", awfTag, constants.AWFDefaultAiCreditsPricingMinVersion)
		return NewValidationError(
			"models.default-ai-credits-pricing",
			awfTag,
			fmt.Sprintf("requires AWF %s or newer; pinned version %q drops apiProxy.defaultAiCreditsPricing during config resolution", constants.AWFDefaultAiCreditsPricingMinVersion, awfTag),
			fmt.Sprintf("Set network.firewall.version or sandbox.agent.version to %s or newer:\n\nnetwork:\n  firewall:\n    version: %s", constants.AWFDefaultAiCreditsPricingMinVersion, constants.AWFDefaultAiCreditsPricingMinVersion),
		)
	}
	if p.Input <= 0 {
		return NewValidationError(
			"models.default-ai-credits-pricing.input",
			fmt.Sprintf("%g", p.Input),
			fmt.Sprintf("input must be a positive value, got %g. Expected a value greater than 0.", p.Input),
			"Set a positive input rate.\n\nExample:\nmodels:\n  default-ai-credits-pricing:\n    input: 0.000001\n    output: 0.000001",
		)
	}
	if p.Output <= 0 {
		return NewValidationError(
			"models.default-ai-credits-pricing.output",
			fmt.Sprintf("%g", p.Output),
			fmt.Sprintf("output must be a positive value, got %g. Expected a value greater than 0.", p.Output),
			"Set a positive output rate.\n\nExample:\nmodels:\n  default-ai-credits-pricing:\n    input: 0.000001\n    output: 0.000001",
		)
	}
	if p.CachedInput != nil && *p.CachedInput <= 0 {
		return NewValidationError(
			"models.default-ai-credits-pricing.cache_read",
			fmt.Sprintf("%g", *p.CachedInput),
			fmt.Sprintf("cache_read must be a positive value when set, got %g. Expected a value greater than 0.", *p.CachedInput),
			"Set cache_read to a positive value when you configure it.\n\nExample:\nmodels:\n  default-ai-credits-pricing:\n    cache_read: 0.000001",
		)
	}
	if p.CacheWrite != nil && *p.CacheWrite <= 0 {
		return NewValidationError(
			"models.default-ai-credits-pricing.cache_write",
			fmt.Sprintf("%g", *p.CacheWrite),
			fmt.Sprintf("cache_write must be a positive value when set, got %g. Expected a value greater than 0.", *p.CacheWrite),
			"Set cache_write to a positive value when you configure it.\n\nExample:\nmodels:\n  default-ai-credits-pricing:\n    cache_write: 0.000001",
		)
	}
	modelCostsPricingValidationLog.Printf("Validated default-ai-credits-pricing: input=%g output=%g", p.Input, p.Output)
	return nil
}
