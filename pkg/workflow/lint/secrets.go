package lint

import (
	"context"

	"github.com/github/gh-aw/pkg/workflow/taint"
)

func SecretLeakRule() Rule {
	return Rule{
		ID:          "secret-leak",
		Description: "Detect secret values flowing to disclosure sinks",
		Severity:    Error,
		Check: func(ctx context.Context, input Input) ([]Diagnostic, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			var result taint.Result
			var err error
			if input.Document != nil {
				result, err = taint.AnalyzeDocument(input.Document, input.Workflow, taint.SecretLeakRule())
			} else {
				result, err = taint.CheckSecretLeaks(input.Workflow)
			}
			if err != nil {
				return nil, err
			}
			diagnostics := make([]Diagnostic, 0, len(result.Findings)+len(result.Gaps))
			for _, finding := range result.Findings {
				related := make([]RelatedLocation, 0, len(finding.Sources))
				for _, source := range finding.Sources {
					related = append(related, RelatedLocation{Path: source.Path, Symbol: source.Reference})
				}
				diagnostics = append(diagnostics, Diagnostic{
					Path: finding.Site.Path, Line: finding.Site.Line,
					Related: related,
					Message: "Secret-tainted value reaches " + string(finding.Site.Kind) +
						"; remove the disclosure and pass credentials only to authentication inputs",
				})
			}
			for _, gap := range result.Gaps {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: Warning, Path: gap.Path,
					Message: "Incomplete secret-leak analysis: " + gap.Reason,
				})
			}
			return diagnostics, nil
		},
	}
}
