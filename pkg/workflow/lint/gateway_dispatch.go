package lint

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

func GatewayDispatchLiteralsRule() Rule {
	return Rule{
		ID: "gateway-dispatch-literals", Description: "Require literal keys and values throughout on.gateway-dispatch",
		Severity: Error, Targets: AWAST, Check: checkGatewayDispatchLiterals,
	}
}

func checkGatewayDispatchLiterals(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil || input.Document == nil {
		return nil, err
	}
	on, ok := input.Document.Frontmatter["on"].(map[string]any)
	if !ok {
		return nil, nil
	}
	return CheckGatewayDispatchLiterals(on["gateway-dispatch"]), ctx.Err()
}

// CheckGatewayDispatchLiterals also enforces the rule during compilation,
// before raw gateway declarations are normalized or lowered.
func CheckGatewayDispatchLiterals(raw any) []Diagnostic {
	var diagnostics []Diagnostic
	var visit func(any, string)
	visit = func(raw any, path string) {
		switch value := raw.(type) {
		case string:
			if strings.Contains(value, "${{") {
				diagnostics = append(diagnostics, Diagnostic{
					Path:    path,
					Message: "GitHub Actions expressions are not supported in on.gateway-dispatch; keys and values must be literals (for example, repos: [github/gh-aw])",
				})
			}
		case map[string]any:
			for _, key := range slices.Sorted(maps.Keys(value)) {
				if strings.Contains(key, "${{") {
					visit(key, path+" property name")
					continue
				}
				visit(value[key], path+"."+key)
			}
		case []any:
			for i, item := range value {
				visit(item, fmt.Sprintf("%s[%d]", path, i))
			}
		case []string:
			for i, item := range value {
				visit(item, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	visit(raw, "on.gateway-dispatch")
	return diagnostics
}
