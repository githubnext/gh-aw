package workflow

import "fmt"

// validateDefaultAiCreditsPricing returns an error when the workflow's
// models.default-ai-credits-pricing frontmatter is present and any price field
// has a non-positive value. Absent pricing (nil) is allowed; the check is only
// enforced once a value is explicitly configured.
//
// The AWF api-proxy rejects zero rates as "not configured", so requiring
// positive values here prevents silent runtime failures for self-hosted models.
func validateDefaultAiCreditsPricing(workflowData *WorkflowData) error {
	p := workflowData.DefaultAiCreditsPricing
	if p == nil {
		return nil
	}
	if p.Input <= 0 {
		return fmt.Errorf("models.default-ai-credits-pricing: input must be a positive value (got %g); use a small positive rate such as 0.000001 for effectively-free self-hosted models", p.Input)
	}
	if p.Output <= 0 {
		return fmt.Errorf("models.default-ai-credits-pricing: output must be a positive value (got %g); use a small positive rate such as 0.000001 for effectively-free self-hosted models", p.Output)
	}
	if p.CachedInput != nil && *p.CachedInput <= 0 {
		return fmt.Errorf("models.default-ai-credits-pricing: cache_read must be a positive value when set (got %g)", *p.CachedInput)
	}
	if p.CacheWrite != nil && *p.CacheWrite <= 0 {
		return fmt.Errorf("models.default-ai-credits-pricing: cache_write must be a positive value when set (got %g)", *p.CacheWrite)
	}
	return nil
}
