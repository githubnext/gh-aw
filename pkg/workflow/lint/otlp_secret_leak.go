package lint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/taint"
)

func OTLPSecretLeakRule() Rule {
	return Rule{
		ID:           "otlp-secret-leak",
		ThreatRuleID: "CTR-017",
		Description:  "Detect secret values exported as OTLP attributes",
		Severity:     Error,
		Targets:      ActionsAST | AWAST,
		Check:        checkOTLPSecretLeaks,
	}
}

func checkOTLPSecretLeaks(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	rule := taint.SecretLeakRule()
	rule.IsSource = isOTLPSecretReference
	rule.IsSink = func(site taint.Site) bool {
		return site.Kind == taint.OTLPAttribute
	}

	if input.Workflow != nil {
		result, err := taint.Analyze(input.Workflow, rule)
		diagnostics := make([]Diagnostic, 0, len(result.Findings))
		for _, finding := range result.Findings {
			diagnostics = append(diagnostics, Diagnostic{
				Path:    finding.Site.Path,
				Message: "Secret-tainted value is exported to an OTLP backend; remove the secret from telemetry attributes",
			})
		}
		return diagnostics, err
	}

	if input.Document != nil {
		return checkAuthoredOTLPSecretAttributes(ctx, input.Document)
	}
	return nil, errors.New("OTLP secret-leak lint requires an Actions or AW AST")
}

func checkAuthoredOTLPSecretAttributes(ctx context.Context, document *aw.Document) ([]Diagnostic, error) {
	if document == nil {
		return nil, nil
	}
	observability, ok := document.Frontmatter["observability"].(map[string]any)
	if !ok {
		return nil, nil
	}
	otlp, ok := observability["otlp"].(map[string]any)
	if !ok {
		return nil, nil
	}

	var diagnostics []Diagnostic
	var failures []error
	for _, field := range []string{"attributes", "resource-attributes"} {
		attributes, ok := stringValues(otlp[field])
		if !ok {
			continue
		}
		for _, name := range slices.Sorted(maps.Keys(attributes)) {
			if err := ctx.Err(); err != nil {
				return diagnostics, errors.Join(append(failures, err)...)
			}
			path := "observability.otlp." + field + "." + name
			expressions, err := taint.AnalyzeText(ctx, attributes[name], path, authoredOTLPSecretRule(), nil)
			if err != nil {
				failures = append(failures, fmt.Errorf("cannot inspect OTLP attribute at %s: invalid Actions expression; correct the value before linting", path))
				continue
			}
			for _, expression := range expressions {
				if len(expression.Sources) == 0 {
					continue
				}
				diagnostics = append(diagnostics, Diagnostic{
					Path:    path,
					Message: "Secret-tainted value is exported to an OTLP backend; remove the secret from telemetry attributes",
				})
				break
			}
		}
	}
	return diagnostics, errors.Join(append(failures, ctx.Err())...)
}

func authoredOTLPSecretRule() taint.Rule {
	rule := taint.SecretLeakRule()
	rule.IsSource = isOTLPSecretReference
	return rule
}

func isOTLPSecretReference(reference string) bool {
	reference = strings.ToLower(reference)
	return reference == "secrets" || strings.HasPrefix(reference, "secrets.") ||
		reference == "github.token" || strings.HasPrefix(reference, "github.token.")
}

func stringValues(value any) (map[string]string, bool) {
	switch attributes := value.(type) {
	case map[string]any:
		result := make(map[string]string, len(attributes))
		for name, value := range attributes {
			text, ok := value.(string)
			if ok {
				result[name] = text
			}
		}
		return result, true
	case map[string]string:
		return attributes, true
	default:
		return nil, false
	}
}
