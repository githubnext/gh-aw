package lint

import (
	"context"
	"errors"
	"maps"
	"strings"
	"unicode"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/rhysd/actionlint"
)

func (c *needsChecker) prompt(ctx context.Context, input Input) error {
	if input.Document == nil {
		return nil
	}
	consumer, err := c.promptConsumer(ctx, input)
	if err != nil || consumer == nil {
		return err
	}
	localConsumer := *consumer
	localConsumer.needs = maps.Clone(consumer.needs)
	localConsumer.needs[string(constants.ActivationJobName)] = struct{}{}
	lateBindings, err := c.promptLateBindings(ctx)
	if err != nil {
		return err
	}
	check := func(line int, node actionlint.ExprNode, allowDeferred bool) error {
		return c.promptExpression(ctx, &localConsumer, lateBindings, line, node, allowDeferred)
	}
	err = taint.VisitInterpolations(ctx, input.Document.Markdown, "prompt", func(start, _, line int, node actionlint.ExprNode) error {
		return check(line, node, !promptConditionContext(input.Document.Markdown[:start]))
	})
	var parseError *taint.ExpressionError
	if err != nil && !errors.As(err, &parseError) {
		return err
	}
	// Prompt-expression owns syntax diagnostics. A coverage gap in the prompt
	// must not suppress independently observable dependency graph diagnostics.
	return visitPromptConditions(ctx, input.Document.Markdown, func(line int, node actionlint.ExprNode) error {
		return check(line, node, false)
	})
}

func (c *needsChecker) promptExpression(ctx context.Context, consumer *needsJob, lateBindings map[string]struct{}, line int, node actionlint.ExprNode, allowDeferred bool) error {
	checker := *c
	checker.diagnostics = nil
	checker.reported = nil
	actionlint.VisitExprNode(node, func(child, parent actionlint.ExprNode, entering bool) {
		if !entering || ctx.Err() != nil {
			return
		}
		parts, ok := needsReference(child)
		if !ok || len(parts) < 2 {
			return
		}
		producer := parts[1] //nolint:uncheckedsliceindex // The guard above requires a producer component.
		if parentParts, ok := needsReference(parent); ok && len(parentParts) > len(parts) {
			return
		}
		selectedConsumer := consumer
		if _, late := lateBindings[strings.Join(parts, ".")]; late && allowDeferred && node == child {
			selectedConsumer = c.jobs[string(constants.AgentJobName)]
		}
		// Prompt dependencies are compiler-owned, but activation cannot read
		// outputs of jobs that run after it, even when the agent needs them.
		if _, connected := selectedConsumer.needs[producer]; !connected && c.jobs[producer] != nil {
			checker.report("prompt", "Prompt references a job unavailable during activation; read data staged by agent steps instead, or move an independent producer before activation with on.needs")
		} else {
			checker.reference(selectedConsumer, "prompt", parts)
		}
	})
	for _, diagnostic := range checker.diagnostics {
		diagnostic.Line = line
		c.diagnostics = append(c.diagnostics, diagnostic)
	}
	return ctx.Err()
}

func promptConditionContext(text string) bool {
	if open := strings.LastIndex(text, "{{"); open >= 0 {
		prefix := text[open+2:]
		if split := strings.IndexFunc(prefix, unicode.IsSpace); split >= 0 && !strings.Contains(prefix, "}}") {
			return promptConditionTag(prefix[:split])
		}
	}
	return false
}

func visitPromptConditions(ctx context.Context, text string, visit func(line int, node actionlint.ExprNode) error) error {
	offset, line := 0, 1
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		start := strings.Index(text[offset:], "{{")
		if start < 0 {
			return nil
		}
		start += offset
		line += strings.Count(text[offset:start], "\n")
		end := strings.Index(text[start+2:], "}}")
		if end < 0 {
			return nil
		}
		end += start + 4
		tag := text[start+2 : end-2]
		if nested := strings.Index(tag, "{{"); nested >= 0 {
			offset = start + 2 + nested
			line += strings.Count(text[start:offset], "\n")
			continue
		}
		split := strings.IndexFunc(tag, unicode.IsSpace)
		if split >= 0 && promptConditionTag(tag[:split]) {
			expression := strings.TrimSpace(tag[split:])
			if !strings.HasPrefix(expression, "${") && !strings.HasPrefix(expression, "__") {
				err := taint.VisitInterpolations(ctx, "${{ "+expression+" }}", "prompt", func(_, _, _ int, node actionlint.ExprNode) error {
					return visit(line, node)
				})
				var parseError *taint.ExpressionError
				if err != nil && !errors.As(err, &parseError) {
					return err
				}
			}
		}
		line += strings.Count(text[start:end], "\n")
		offset = end
	}
}

func promptConditionTag(tag string) bool {
	switch tag {
	case "#if", "#elseif", "#else-if", "#else_if", "elseif", "else-if", "else_if":
		return true
	default:
		return false
	}
}

func (c *needsChecker) promptConsumer(ctx context.Context, input Input) (*needsJob, error) {
	consumer := c.jobs[string(constants.ActivationJobName)]
	if c.authored {
		consumer = &needsJob{path: "prompt", needs: map[string]struct{}{
			string(constants.PreActivationJobName): {},
			string(constants.ActivationJobName):    {}, // Local activation producers are rewritten by the compiler.
		}}
		// Scalar and array triggers cannot declare on.needs.
		var on map[string]any
		var err error
		switch input.Document.Frontmatter["on"].(type) {
		case string, []string, []any:
		default:
			on, err = policyMapping(input.Document.Frontmatter["on"], "on")
		}
		if err != nil {
			return nil, err
		}
		explicit, valid, err := workflowRunStrings(ctx, on["needs"], true)
		if err != nil {
			return nil, err
		}
		if on["needs"] != nil && !valid {
			return nil, errors.New("needs-consistency requires literal job IDs at on.needs; use on.needs: [build]")
		}
		if valid {
			for _, id := range explicit {
				consumer.needs[strings.ToLower(id)] = struct{}{}
			}
		}
		for id, job := range c.jobs {
			if !strings.HasPrefix(job.path, "jobs.") {
				continue
			}
			_, preActivation := job.needs[string(constants.PreActivationJobName)]
			_, activation := job.needs[string(constants.ActivationJobName)]
			if job.fields["needs"] == nil || preActivation && !activation {
				consumer.needs[id] = struct{}{}
			}
		}
	}
	return consumer, nil
}

func (c *needsChecker) promptLateBindings(ctx context.Context) (map[string]struct{}, error) {
	bindings := make(map[string]struct{})
	agent := c.jobs[string(constants.AgentJobName)]
	if c.authored || agent == nil {
		return bindings, nil
	}
	steps, ok := agent.fields["steps"].([]map[string]any)
	if !ok {
		return bindings, nil
	}
	for _, step := range steps {
		if step["id"] != string(constants.DeferredPromptStepID) {
			continue
		}
		env, err := policyMapping(step["env"], "jobs.agent.steps.env")
		if err != nil {
			return nil, err
		}
		for key, value := range env {
			text, ok := value.(string)
			if !ok || !strings.HasPrefix(key, "GH_AW_DEFERRED_") {
				continue
			}
			text = strings.TrimSpace(text)
			if err := taint.VisitInterpolations(ctx, text, "jobs.agent.steps.env", func(start, end, _ int, node actionlint.ExprNode) error {
				if parts, ok := needsReference(node); ok && start == 0 && end == len(text) {
					bindings[strings.Join(parts, ".")] = struct{}{}
				}
				return ctx.Err()
			}); err != nil {
				return nil, err
			}
		}
	}
	return bindings, nil
}
