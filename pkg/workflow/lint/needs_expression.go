package lint

import (
	"context"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/rhysd/actionlint"
)

func (c *needsChecker) expressions(ctx context.Context, job *needsJob) error {
	return walkExpressionScalars(ctx, job.fields, job.path, func(path, text string) error {
		visit := func(_, _, _ int, node actionlint.ExprNode) error {
			actionlint.VisitExprNode(node, func(child, _ actionlint.ExprNode, entering bool) {
				if !entering || ctx.Err() != nil {
					return
				}
				parts, ok := needsReference(child)
				if !ok || len(parts) < 2 {
					return
				}
				c.reference(job, path, parts)
			})
			return ctx.Err()
		}
		if needsConditionPath(job.path, path) && !strings.Contains(text, "${{") {
			lexer := actionlint.NewExprLexer(text + " }}")
			node, err := actionlint.NewExprParser().Parse(lexer)
			if err != nil || lexer.Offset() != len(text+" }}") {
				return &taint.ExpressionError{Path: path, Line: 1}
			}
			return visit(0, 0, 1, node)
		}
		return taint.VisitInterpolations(ctx, text, path, visit)
	})
}

func needsConditionPath(jobPath, path string) bool {
	if path == jobPath+".if" {
		return true
	}
	for _, field := range []string{"steps", "post-steps"} {
		rest, found := strings.CutPrefix(path, jobPath+"."+field+"[")
		if found && strings.HasSuffix(rest, "].if") && strings.Count(rest, ".") == 1 {
			return true
		}
	}
	return false
}

// Keep computed indexes and filters distinct from literal keys, including '*'.
// Their nested expressions are still visited for independent needs accesses.
func needsReference(node actionlint.ExprNode) ([]string, bool) {
	switch node := node.(type) {
	case *actionlint.VariableNode:
		return []string{"needs"}, strings.EqualFold(node.Name, "needs")
	case *actionlint.ObjectDerefNode:
		parts, ok := needsReference(node.Receiver)
		return append(parts, strings.ToLower(node.Property)), ok
	case *actionlint.IndexAccessNode:
		parts, ok := needsReference(node.Operand)
		key, literal := node.Index.(*actionlint.StringNode)
		if !literal {
			return nil, false
		}
		return append(parts, strings.ToLower(key.Value)), ok
	}
	return nil, false
}

func (c *needsChecker) reference(job *needsJob, path string, parts []string) {
	if len(parts) < 2 {
		return
	}
	var id, property, output string
	for index, part := range parts {
		switch index {
		case 1:
			id = part
		case 2:
			property = part
		case 3:
			output = part
		}
	}
	producer := c.jobs[id]
	if producer == nil && !c.deferred(id) {
		c.report(path, "Needs expression references an unknown job; use an existing producer job ID")
		return
	}
	// AW compiler-owned scopes and imports may supply dependencies not visible
	// in authored source. The compiled graph is authoritative when available.
	_, builtin := constants.KnownBuiltInJobNames[strings.TrimPrefix(job.path, "jobs.")]
	safeJob := strings.HasPrefix(job.path, "safe-jobs.") || strings.HasPrefix(job.path, "safe-outputs.jobs.")
	checkEdge := !c.authored || !builtin && !c.imports && (!safeJob || !c.deferred(id))
	if c.authored && id == string(constants.ActivationJobName) && job.fields["needs"] == nil {
		checkEdge = false // The compiler can add this default edge.
	}
	_, connected := job.needs[id]
	if checkEdge && !connected {
		c.report(path, "Needs expression references a job outside direct needs; add the producer to this job's needs list, even if it is a transitive dependency")
		return
	}
	if len(parts) < 3 {
		return
	}
	if property != "outputs" && property != "result" {
		c.report(path, "Needs expression references an unknown job property; use needs.build.result or needs.build.outputs.value")
		return
	}
	if len(parts) < 4 {
		return
	}
	if property == "outputs" && producer != nil &&
		producer.fields["uses"] == nil && (!c.authored || !c.deferred(id)) {
		if _, exists := producer.outputs[output]; !exists {
			c.report(path, "Needs expression references an undeclared job output; declare it on the producer's outputs mapping or use an existing output name")
		}
	}
}
