package lint

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/checkoutpolicy"
)

func PullRequestTargetCheckoutRule() Rule {
	return Rule{
		ID: "pull-request-target-checkout", Description: "Restrict authored pull_request_target checkouts to trusted base refs or exact allowed pairs",
		ThreatRuleID: "CTR-008",
		Severity:     Warning, Targets: AWAST, Check: checkPullRequestTargetCheckouts,
	}
}

func checkPullRequestTargetCheckouts(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil || input.Document == nil {
		return nil, err
	}
	trigger, found, err := authoredTrigger(ctx, input.Document.Frontmatter["on"], "pull_request_target")
	if err != nil || !found {
		return nil, err
	}
	raw := input.Document.Frontmatter["checkout"]
	if raw == false {
		return nil, nil
	}
	severity, err := sourcePolicySeverity(input, "pull-request-target-checkout")
	if err != nil {
		return nil, err
	}
	pending, err := authoredImportsPending(input)
	if err != nil {
		return nil, err
	}
	settings, err := policyMapping(trigger, "on.pull_request_target")
	if err != nil {
		return nil, err
	}
	allowed, err := targetAllowedPairs(ctx, settings["allowed-checkouts"])
	if err != nil {
		return nil, err
	}
	var entries []any
	switch value := raw.(type) {
	case nil:
	case map[string]any, map[string]string:
		entries = []any{value}
	case []any:
		entries = value
	default:
		return nil, errors.New("pull-request-target-checkout requires checkout to be false, a mapping, or a list of mappings")
	}
	if len(entries) == 0 && !pending {
		return []Diagnostic{targetCheckoutDiagnostic("checkout", severity)}, nil
	}
	return inspectTargetCheckouts(ctx, entries, raw, allowed, pending, severity)
}

func inspectTargetCheckouts(ctx context.Context, entries []any, raw any, allowed [][2]string, pending bool, severity Severity) ([]Diagnostic, error) {
	var diagnostics []Diagnostic
	var failures []error
	for index, entry := range entries {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		path := "checkout"
		if _, list := raw.([]any); list {
			path = fmt.Sprintf("checkout[%d]", index)
		}
		config, err := policyMapping(entry, path)
		if err != nil || config == nil {
			failures = append(failures, fmt.Errorf("pull-request-target-checkout requires a checkout mapping at %s", path))
			continue
		}
		safe, err := targetCheckout(ctx, config, path, allowed, pending)
		if err != nil {
			failures = append(failures, err)
		} else if !safe {
			diagnostics = append(diagnostics, targetCheckoutDiagnostic(path, severity))
		}
	}
	return diagnostics, errors.Join(append(failures, ctx.Err())...)
}

func targetCheckoutDiagnostic(path string, severity Severity) Diagnostic {
	return Diagnostic{Severity: severity, Path: path,
		Message: "pull_request_target checkout is not restricted to a trusted base ref or exact allowed repository/ref pair without fetch; disable checkout or use the supported base checkout or an explicit allowed-checkouts pair. Trigger-risk acknowledgment does not authorize checkout",
	}
}

func targetCheckout(ctx context.Context, config map[string]any, path string, allowed [][2]string, pending bool) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	repository, ref, err := targetCheckoutPair(config, path)
	if err != nil {
		return false, err
	}
	if fetch, exists := config["fetch"]; exists {
		if text, scalar := fetch.(string); scalar {
			if strings.TrimSpace(text) == "" {
				return false, fmt.Errorf("pull-request-target-checkout requires a non-empty fetch ref at %s.fetch", path)
			}
			return false, nil
		}
		refs, err := policyStrings(ctx, fetch, path+".fetch")
		if err != nil || len(refs) > 0 {
			return false, err
		}
	}
	wiki := false
	if value, exists := config["wiki"]; exists {
		var ok bool
		wiki, ok = value.(bool)
		if !ok {
			return false, fmt.Errorf("pull-request-target-checkout requires a boolean at %s.wiki", path)
		}
	}
	if checkoutpolicy.Base(repository, ref) {
		return true, nil
	}
	if wiki {
		return false, nil
	}
	// Imports can supply missing literal operands or the allowlist, but cannot
	// make an explicit expression, wildcard, wiki, or additional fetch eligible.
	if pending {
		return (repository == "" || checkoutpolicy.LiteralRepository(repository)) &&
			(ref == "" || checkoutpolicy.LiteralRef(ref)), nil
	}
	if !checkoutpolicy.LiteralPair(repository, ref) {
		return false, nil
	}
	for _, pair := range allowed {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if pair == [2]string{repository, ref} {
			return true, nil
		}
	}
	return false, nil
}

func targetCheckoutPair(config map[string]any, path string) (string, string, error) {
	repository, ref := "", ""
	for _, key := range []string{"repository", "repo", "ref"} {
		value, exists := config[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return "", "", fmt.Errorf("pull-request-target-checkout requires a string at %s.%s", path, key)
		}
		if key == "ref" {
			ref = text
		} else {
			repository = text
		}
	}
	if _, alias := config["repo"]; alias {
		if _, canonical := config["repository"]; canonical {
			return "", "", errors.New("pull-request-target-checkout requires only one of checkout.repo and checkout.repository")
		}
	}
	return repository, ref, nil
}

func targetAllowedPairs(ctx context.Context, raw any) ([][2]string, error) {
	if raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, errors.New("pull-request-target-checkout requires an allowed-checkouts list")
	}
	var pairs [][2]string
	for index, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := fmt.Sprintf("on.pull_request_target.allowed-checkouts[%d]", index)
		fields, err := policyMapping(entry, path)
		if err != nil || fields == nil {
			return nil, fmt.Errorf("pull-request-target-checkout requires a repository/ref mapping at %s", path)
		}
		repository, repoOK := fields["repository"].(string)
		ref, refOK := fields["ref"].(string)
		if !repoOK || !refOK {
			return nil, fmt.Errorf("pull-request-target-checkout requires repository and ref strings at %s", path)
		}
		pairs = append(pairs, [2]string{repository, ref})
	}
	return pairs, nil
}

func authoredTrigger(ctx context.Context, raw any, name string) (any, bool, error) {
	switch on := raw.(type) {
	case nil:
	case string:
		return nil, on == name, nil
	case map[string]any:
		settings, found := on[name]
		return settings, found, nil
	case []string, []any:
		names, err := policyStrings(ctx, on, "on")
		return nil, slices.Contains(names, name), err
	default:
		return nil, false, errors.New("AST policy lint requires on to be a trigger name, list, or mapping")
	}
	return nil, false, nil
}
