package workflow

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var unavailableTopLevelEnvContextPattern = regexp.MustCompile(
	`(?:^|[^A-Za-z0-9_.-])(env|job|jobs|matrix|needs|runner|steps|strategy)(?:[^A-Za-z0-9_-]|$)`,
)

// validateTopLevelEnvExpressions rejects expression contexts that GitHub Actions
// only makes available within jobs.
func validateTopLevelEnvExpressions(env map[string]any) error {
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		value := env[name]
		stringValue, ok := value.(string)
		if !ok {
			continue
		}

		var unavailableContexts []string
		for _, expressionMatch := range ExpressionPatternDotAll.FindAllStringSubmatch(stringValue, -1) {
			if len(expressionMatch) < 2 {
				continue
			}
			expression := maskQuotedExpressionLiterals(expressionMatch[1])
			for _, contextMatch := range unavailableTopLevelEnvContextPattern.FindAllStringSubmatch(expression, -1) {
				if len(contextMatch) >= 2 && !slices.Contains(unavailableContexts, contextMatch[1]) {
					unavailableContexts = append(unavailableContexts, contextMatch[1])
				}
			}
		}

		if len(unavailableContexts) == 0 {
			continue
		}
		slices.Sort(unavailableContexts)
		return NewValidationError(
			"env."+name,
			stringValue,
			"top-level env expression references context(s) unavailable outside jobs: "+strings.Join(unavailableContexts, ", "),
			fmt.Sprintf("Move this environment variable to a job or step env block. Example:\njobs:\n  agent:\n    env:\n      %s: %s", name, stringValue),
		)
	}
	return nil
}
