package lint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/shellpattern"
)

func SafeOutputShellExpansionRule() Rule {
	return Rule{
		ID: "safe-output-shell-expansion", Description: "Reject shell expansions blocked by the safe-output harness in authored steps",
		ThreatRuleID: "CTR-009",
		Severity:     Error, Targets: AWAST, Check: checkSafeOutputShellExpansion,
	}
}

func checkSafeOutputShellExpansion(ctx context.Context, input Input) ([]Diagnostic, error) {
	outputs, err := authoredPolicyMapping(ctx, input, "safe-outputs")
	if err != nil {
		return nil, err
	}
	raw, exists := outputs["steps"]
	if !exists {
		return nil, nil
	}
	steps, ok := raw.([]any)
	if !ok {
		return nil, errors.New("safe-output-shell-expansion requires a step list at safe-outputs.steps")
	}
	var diagnostics []Diagnostic
	var failures []error
	for index, raw := range steps {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		path := fmt.Sprintf("safe-outputs.steps[%d]", index)
		step, err := policyMapping(raw, path)
		if err != nil || step == nil {
			failures = append(failures, fmt.Errorf("safe-output-shell-expansion requires a step mapping at %s", path))
			continue
		}
		run, exists := step["run"]
		if !exists {
			continue
		}
		script, ok := run.(string)
		if !ok {
			failures = append(failures, fmt.Errorf("safe-output-shell-expansion requires a script string at %s.run", path))
			continue
		}
		if expansion, found := shellpattern.FirstUnsafeExpansion(script); found {
			diagnostics = append(diagnostics, Diagnostic{Path: path + ".run",
				Line:    1 + strings.Count(script[:expansion.Start], "\n"),
				Message: "Safe-output script contains " + expansion.Description + ", which the runtime harness blocks; write dynamic content to agent files and consume the files without shell expansion",
			})
		}
	}
	return diagnostics, errors.Join(failures...)
}
