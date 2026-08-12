package authz

import (
	"errors"
	"fmt"
	"slices"

	"github.com/github/gh-aw/pkg/intent"
)

var (
	ErrToolDenied          = errors.New("tool denied by intent policy")
	ErrToolNotAllowed      = errors.New("tool not allowed by intent policy")
	ErrWriteScopeDenied    = errors.New("write denied by intent policy")
	ErrHumanApprovalNeeded = errors.New("human approval required by intent policy")
	ErrRequiredChecks      = errors.New("required checks missing by intent policy")
	ErrAutoMergeDenied     = errors.New("auto-merge denied by intent policy")
	ErrMaxAttemptsExceeded = errors.New("max attempts exceeded by intent policy")
)

type ToolContext struct {
	IsWrite       bool
	IsAutoMerge   bool
	Branch        string
	DefaultBranch string
	Approved      bool
	PassedChecks  []string
	Attempt       int
}

type Authorizer struct{}

func (a Authorizer) AuthorizeTool(policy intent.ExecutionPolicy, tool string, ctx ToolContext) error {
	if slices.Contains(policy.DeniedTools, tool) {
		return fmt.Errorf("%w: %s", ErrToolDenied, tool)
	}
	if policy.AllowedTools != nil && !slices.Contains(policy.AllowedTools, tool) {
		return fmt.Errorf("%w: %s", ErrToolNotAllowed, tool)
	}
	if policy.MaxAttempts > 0 && ctx.Attempt > policy.MaxAttempts {
		return fmt.Errorf("%w: attempt %d exceeds max_attempts %d", ErrMaxAttemptsExceeded, ctx.Attempt, policy.MaxAttempts)
	}
	if ctx.IsWrite {
		if policy.WriteScope == "none" || policy.Autonomy == "propose_only" {
			return ErrWriteScopeDenied
		}
		if policy.WriteScope == "feature_branch" && ctx.Branch != "" && ctx.DefaultBranch != "" && ctx.Branch == ctx.DefaultBranch {
			return fmt.Errorf("%w: feature_branch policy cannot write to %s", ErrWriteScopeDenied, ctx.Branch)
		}
		if policy.HumanApprovalRequired && !ctx.Approved {
			return ErrHumanApprovalNeeded
		}
	}
	if len(policy.RequiredChecks) > 0 {
		for _, check := range policy.RequiredChecks {
			if !slices.Contains(ctx.PassedChecks, check) {
				return fmt.Errorf("%w: %s", ErrRequiredChecks, check)
			}
		}
	}
	if ctx.IsAutoMerge && policy.AutoMergeAllowed != nil && !*policy.AutoMergeAllowed {
		return ErrAutoMergeDenied
	}
	return nil
}
