package lint

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/taint"
)

func CheckoutModeRule() Rule {
	return Rule{
		ID: "checkout-mode", Description: "Require explicit checkout expansion and its authored safety prerequisites",
		ThreatRuleID: "CTR-033",
		Severity:     Error, Targets: AWAST, Check: checkCheckoutModes,
	}
}

func checkCheckoutModes(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Document == nil {
		return nil, nil
	}
	var entries []any
	switch raw := input.Document.Frontmatter["checkout"].(type) {
	case nil:
		return nil, nil
	case bool:
		if raw {
			return nil, errors.New("checkout-mode requires false, a mapping, or a list of checkout mappings")
		}
		return nil, nil
	case []any:
		entries = raw
	case map[string]any:
		entries = []any{raw}
	default:
		return nil, errors.New("checkout-mode requires false, a mapping, or a list of checkout mappings")
	}
	pending, err := authoredImportsPending(input)
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	var failures []error
	for index, raw := range entries {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		prefix := "checkout"
		if _, list := input.Document.Frontmatter["checkout"].([]any); list {
			prefix = fmt.Sprintf("%s[%d]", prefix, index)
		}
		config, err := policyMapping(raw, prefix)
		failures = append(failures, err)
		found, err := checkoutMode(ctx, config, prefix, pending)
		diagnostics = append(diagnostics, found...)
		failures = append(failures, err)
	}
	return diagnostics, errors.Join(failures...)
}

func checkoutMode(ctx context.Context, config map[string]any, prefix string, pending bool) ([]Diagnostic, error) {
	raw, specified := config["multi"]
	multi, boolean := raw.(bool)
	if specified && !boolean {
		return []Diagnostic{{Path: prefix + ".multi", Message: "Checkout expansion mode must be a literal boolean; only multi: true enables repository-list expansion"}}, nil
	}
	var diagnostics []Diagnostic
	var failures []error
	for _, key := range []string{"repository", "repo"} {
		if value, exists := config[key]; exists && !multi {
			parts, err := checkoutLiteral(ctx, value, prefix+"."+key)
			failures = append(failures, err)
			if len(parts) > 1 {
				diagnostics = append(diagnostics, Diagnostic{Path: prefix + "." + key,
					Message: "A literal repository list requires explicit multi: true; single mode must resolve to one repository"})
			}
		}
	}
	if multi {
		for _, key := range []string{"lfs", "current", "wiki"} {
			if config[key] == true {
				diagnostics = append(diagnostics, Diagnostic{Path: prefix + "." + key,
					Message: "Expanded checkouts cannot enable LFS, current-repository identity, or wiki checkout; remove this option"})
			}
		}
		switch config["submodules"] {
		case true, "true", "recursive":
			diagnostics = append(diagnostics, Diagnostic{Path: prefix + ".submodules",
				Message: "Expanded checkouts cannot enable submodules because their URLs bypass the repository allowlist"})
		}
		found, err := checkoutExpansionPrerequisites(ctx, config, prefix, pending)
		diagnostics = append(diagnostics, found...)
		failures = append(failures, err)
	}
	return diagnostics, errors.Join(failures...)
}

func checkoutLiteral(ctx context.Context, raw any, field string) ([]string, error) {
	text, dynamic, err := checkoutString(ctx, raw, field)
	if err != nil || dynamic {
		return nil, err
	}
	return strings.Fields(text), nil
}

func checkoutString(ctx context.Context, raw any, field string) (string, bool, error) {
	text, ok := raw.(string)
	if !ok {
		return "", false, fmt.Errorf("checkout-mode requires a repository or path string at %s", field)
	}
	expressions, err := taint.AnalyzeText(ctx, text, field, taint.Rule{IsSource: func(string) bool { return false }}, nil)
	return text, len(expressions) > 0, err
}

func checkoutExpansionPrerequisites(ctx context.Context, config map[string]any, prefix string, pending bool) ([]Diagnostic, error) {
	var diagnostics []Diagnostic
	var failures []error
	if value, exists := config["path"]; exists || !pending {
		valid := false
		if value != nil {
			text, dynamic, err := checkoutString(ctx, value, prefix+".path")
			failures = append(failures, err)
			if err == nil && !dynamic && text != "" {
				cleaned := path.Clean(strings.ReplaceAll(text, "\\", "/"))
				valid = cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "../") && !strings.HasPrefix(cleaned, "/") && !strings.Contains(cleaned, ":")
			}
		}
		if !valid {
			diagnostics = append(diagnostics, Diagnostic{Path: prefix + ".path",
				Message: "Expanded checkouts require an explicit literal non-root relative path prefix"})
		}
	}
	value, exists := config["allowed-repos"]
	if !exists && !pending {
		diagnostics = append(diagnostics, Diagnostic{Path: prefix + ".allowed-repos",
			Message: "Expanded checkouts require a scoped repository allowlist"})
	} else if exists {
		var patterns []string
		var err error
		dynamic := false
		if _, literal := value.(string); literal {
			var text string
			text, dynamic, err = checkoutString(ctx, value, prefix+".allowed-repos")
			if !dynamic {
				patterns = strings.Fields(text)
			}
		} else {
			patterns, err = policyStrings(ctx, value, prefix+".allowed-repos")
		}
		failures = append(failures, err)
		if err == nil && !dynamic && len(patterns) == 0 {
			diagnostics = append(diagnostics, Diagnostic{Path: prefix + ".allowed-repos", Message: "Expanded checkouts require a non-empty scoped repository allowlist"})
		}
		for _, pattern := range patterns {
			if strings.TrimSpace(pattern) == "*" || strings.TrimSpace(pattern) == "*/*" {
				diagnostics = append(diagnostics, Diagnostic{Path: prefix + ".allowed-repos",
					Message: "Expanded checkouts cannot use an unrestricted repository allowlist; scope patterns to an owner or repository"})
			}
		}
	}
	return diagnostics, errors.Join(failures...)
}
