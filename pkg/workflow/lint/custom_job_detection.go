package lint

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/rhysd/actionlint"
)

func CustomJobDetectionRule() Rule {
	return Rule{
		ID: "custom-job-detection", Description: "Require custom jobs downstream of agent to wait for successful threat detection",
		Severity: Error, Targets: AWAST, Check: checkCustomJobDetection,
	}
}

func checkCustomJobDetection(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Document == nil {
		return nil, nil
	}
	disabled, err := customJobDetectionDisabled(input)
	if err != nil || disabled {
		return nil, err
	}
	jobs, err := policyMapping(input.Document.Frontmatter["jobs"], "jobs")
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	for _, id := range sliceutil.SortedKeys(jobs) {
		if err := ctx.Err(); err != nil {
			return diagnostics, err
		}
		if _, builtin := constants.KnownBuiltInJobNames[id]; builtin {
			continue
		}
		fields, err := policyMapping(jobs[id], "jobs."+id)
		if err != nil {
			return diagnostics, err
		}
		found, err := customJobDetectionDiagnostics(ctx, fields, "jobs."+id)
		diagnostics = append(diagnostics, found...)
		if err != nil {
			return diagnostics, err
		}
	}
	return diagnostics, nil
}

func customJobDetectionDisabled(input Input) (bool, error) {
	if input.Workflow != nil {
		return !slices.ContainsFunc(input.Workflow.Jobs, func(job *actions.Job) bool {
			return string(job.ID) == string(constants.DetectionJobName)
		}), nil
	}
	outputs, err := policyMapping(input.Document.Frontmatter["safe-outputs"], "safe-outputs")
	if err != nil {
		return false, err
	}
	detection := outputs["threat-detection"]
	if enabled, ok := detection.(bool); ok {
		return !enabled, nil
	}
	fields, ok := detection.(map[string]any)
	if !ok {
		return false, nil
	}
	engine, literal := fields["engine"].(bool)
	if !literal || engine {
		return false, nil
	}
	for _, key := range []string{"steps", "post-steps"} {
		if fields[key] == nil {
			continue
		}
		steps, valid := fields[key].([]any)
		if !valid {
			return false, fmt.Errorf("custom-job-detection requires a step sequence at safe-outputs.threat-detection.%s", key)
		}
		if len(steps) > 0 {
			return false, nil
		}
	}
	return true, nil
}

func customJobDetectionDiagnostics(ctx context.Context, fields map[string]any, path string) ([]Diagnostic, error) {
	if fields["needs"] == nil {
		return nil, nil
	}
	needs, valid, err := workflowRunStrings(ctx, fields["needs"], true)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("custom-job-detection requires literal job IDs at %s.needs; correct the job dependencies before linting", path)
	}
	if !slices.Contains(needs, string(constants.AgentJobName)) {
		return nil, nil
	}
	if !slices.Contains(needs, string(constants.DetectionJobName)) {
		return []Diagnostic{{
			Path:    path + ".needs",
			Message: "Custom jobs depending on agent must also need detection; add detection to needs and require needs.detection.result == 'success' when using a status override",
		}}, nil
	}
	gated, err := customJobDetectionCondition(fields["if"])
	if err != nil {
		return nil, fmt.Errorf("cannot inspect %s.if: expected a valid Actions condition; correct the condition before linting", path)
	}
	if gated {
		return nil, nil
	}
	return []Diagnostic{{
		Path:    path + ".if",
		Message: "Custom job condition can bypass successful threat detection; require needs.detection.result == 'success' on every runnable branch, including with always(), failure(), or cancelled()",
	}}, nil
}

func customJobDetectionCondition(value any) (bool, error) {
	if value == nil {
		return true, nil // Actions implicitly requires success() for all needs.
	}
	if _, ok := value.(bool); ok {
		return true, nil
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return false, errors.New("invalid job condition")
	}
	text = strings.TrimSpace(text)
	if body, wrapped := strings.CutPrefix(text, "${{"); wrapped {
		var closed bool
		text, closed = strings.CutSuffix(strings.TrimSpace(body), "}}")
		if !closed {
			return false, errors.New("invalid job condition")
		}
	}
	lexer := actionlint.NewExprLexer(text + " }}")
	node, err := actionlint.NewExprParser().Parse(lexer)
	if err != nil || lexer.Offset() != len(text+" }}") {
		return false, errors.New("invalid job condition")
	}
	statusOverride := false
	actionlint.VisitExprNode(node, func(child, _ actionlint.ExprNode, entering bool) {
		if call, ok := child.(*actionlint.FuncCallNode); ok && entering {
			switch strings.ToLower(call.Callee) {
			case "always", "success", "failure", "cancelled":
				statusOverride = true
			}
		}
	})
	return !statusOverride || detectionSuccessRequired(node), nil
}

// Prove a success guard on every disjunction, not merely its textual presence.
func detectionSuccessRequired(node actionlint.ExprNode) bool {
	switch node := node.(type) {
	case *actionlint.LogicalOpNode:
		if node.Kind.String() == "&&" {
			return detectionSuccessRequired(node.Left) || detectionSuccessRequired(node.Right)
		}
		return detectionSuccessRequired(node.Left) && detectionSuccessRequired(node.Right)
	case *actionlint.FuncCallNode:
		return strings.EqualFold(node.Callee, "success") && len(node.Args) == 0
	case *actionlint.CompareOpNode:
		return node.Kind.String() == "==" &&
			(detectionResultReference(node.Left) && detectionSuccessLiteral(node.Right) ||
				detectionResultReference(node.Right) && detectionSuccessLiteral(node.Left))
	}
	return false
}

func detectionSuccessLiteral(node actionlint.ExprNode) bool {
	value, ok := node.(*actionlint.StringNode)
	return ok && strings.EqualFold(value.Value, "success")
}

func detectionResultReference(node actionlint.ExprNode) bool {
	parts := []string{"result", string(constants.DetectionJobName), "needs"}
	for _, part := range parts {
		switch value := node.(type) {
		case *actionlint.ObjectDerefNode:
			if !strings.EqualFold(value.Property, part) {
				return false
			}
			node = value.Receiver
		case *actionlint.IndexAccessNode:
			key, ok := value.Index.(*actionlint.StringNode)
			if !ok || !strings.EqualFold(key.Value, part) {
				return false
			}
			node = value.Operand
		case *actionlint.VariableNode:
			return part == "needs" && strings.EqualFold(value.Name, part)
		default:
			return false
		}
	}
	return false
}
