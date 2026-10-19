package lint

import (
	"context"
	"errors"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/rhysd/actionlint"
)

// PromptExpressionRule checks independently observable forbidden forms, not
// the compiler's complete context allowlist or imported prompt content.
func PromptExpressionRule() Rule {
	return Rule{
		ID: "prompt-expression", Description: "Reject multiline and dangerous property accesses in authored AW prompt expressions",
		ThreatRuleID: "CTR-010",
		Severity:     Error, Targets: AWAST, Check: checkPromptExpressions,
	}
}

func checkPromptExpressions(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil || input.Document == nil {
		return nil, err
	}
	text := input.Document.Markdown
	var diagnostics []Diagnostic
	err := taint.VisitInterpolations(ctx, text, "prompt", func(start, end, line int, node actionlint.ExprNode) error {
		if strings.ContainsAny(text[start:end], "\r\n") {
			diagnostics = append(diagnostics, Diagnostic{Path: "prompt", Line: line,
				Message: "AW prompt expressions cannot span multiple lines; keep each Actions expression on one line",
			})
		}
		dangerous, err := promptDangerousProperty(ctx, node)
		if dangerous {
			diagnostics = append(diagnostics, Diagnostic{Path: "prompt", Line: line,
				Message: "AW prompt expression accesses a blocked JavaScript property; remove prototype or built-in property traversal",
			})
		}
		return err
	})
	var parseError *taint.ExpressionError
	if errors.As(err, &parseError) {
		diagnostics = append(diagnostics, Diagnostic{Path: "prompt", Line: parseError.Line,
			Message: "AW prompt contains an invalid Actions expression; correct its syntax before linting",
		})
		return diagnostics, nil
	}
	return diagnostics, err
}

func promptDangerousProperty(ctx context.Context, node actionlint.ExprNode) (bool, error) {
	dangerous := false
	actionlint.VisitExprNode(node, func(child, _ actionlint.ExprNode, entering bool) {
		if !entering || ctx.Err() != nil {
			return
		}
		var property string
		switch value := child.(type) {
		case *actionlint.ObjectDerefNode:
			property = value.Property
		case *actionlint.IndexAccessNode:
			if key, literal := value.Index.(*actionlint.StringNode); literal {
				property = key.Value
			}
		}
		// The Actions parser canonicalizes dot-property names to lowercase.
		for _, blocked := range constants.DangerousPropertyNames {
			if strings.EqualFold(property, blocked) {
				dangerous = true
				break
			}
		}
	})
	return dangerous, ctx.Err()
}
