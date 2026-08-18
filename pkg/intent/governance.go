package intent

import (
	"errors"
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var governanceLog = logger.New("intent:governance")

// ErrToolDenied is returned by Authorizer.AuthorizeTool when the tool appears in
// the policy's DeniedTools list. A deny always wins, even if the same tool is
// also present in AllowedTools.
var ErrToolDenied = errors.New("intent: tool denied by policy")

// ErrToolNotAllowed is returned by Authorizer.AuthorizeTool when the policy's
// AllowedTools is non-nil (restricted) and does not contain the requested tool.
var ErrToolNotAllowed = errors.New("intent: tool not allowed by policy")

// ResolveRisk returns rec.Risk when explicitly set; otherwise it derives a risk
// classification from rec.Domains and rec.Priority using deterministic,
// precedence-ordered rules:
//
//	security + critical priority -> high
//	production                   -> high
//	infrastructure                -> medium
//	documentation                 -> low
//	anything else                 -> unknown
//
// An explicit Risk always wins over any derived value, even when the record's
// domains or priority would otherwise match a different rule.
func ResolveRisk(rec IntentRecord) string {
	if rec.Risk != "" {
		governanceLog.Printf("ResolveRisk: using explicit risk=%s", rec.Risk)
		return rec.Risk
	}

	if slices.Contains(rec.Domains, "security") && rec.Priority == "critical" {
		governanceLog.Print("ResolveRisk: security+critical -> high")
		return "high"
	}
	if slices.Contains(rec.Domains, "production") {
		governanceLog.Print("ResolveRisk: production -> high")
		return "high"
	}
	if slices.Contains(rec.Domains, "infrastructure") {
		governanceLog.Print("ResolveRisk: infrastructure -> medium")
		return "medium"
	}
	if slices.Contains(rec.Domains, "documentation") {
		governanceLog.Print("ResolveRisk: documentation -> low")
		return "low"
	}

	governanceLog.Print("ResolveRisk: no matching rule -> unknown")
	return "unknown"
}

// Authorizer authorizes individual tool calls against a compiled ExecutionPolicy.
type Authorizer struct{}

// AuthorizeTool reports whether tool may be called under policy. DeniedTools is
// checked first and always wins, even if tool also appears in AllowedTools. A
// nil AllowedTools means unrestricted (any tool not explicitly denied is
// allowed); a non-nil AllowedTools (including an empty, non-nil slice) restricts
// calls to the listed tools, so a non-nil empty slice denies every tool.
func (a Authorizer) AuthorizeTool(policy ExecutionPolicy, tool string) error {
	if slices.Contains(policy.DeniedTools, tool) {
		return ErrToolDenied
	}
	if policy.AllowedTools != nil && !slices.Contains(policy.AllowedTools, tool) {
		return ErrToolNotAllowed
	}
	return nil
}
