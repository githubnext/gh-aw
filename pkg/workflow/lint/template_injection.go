package lint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/rhysd/actionlint"
)

// TemplateInjectionRule checks the direct executable-interpolation trigger of
// CTR-006, not whole-rule conformance or arbitrary shell-code data flow.
func TemplateInjectionRule() Rule {
	return Rule{
		ID:          "template-injection",
		Description: "Detect attacker-controlled Actions expressions interpolated into run scripts",
		Severity:    Error,
		Check:       checkTemplateInjection,
	}
}

func checkTemplateInjection(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Workflow == nil {
		return nil, errors.New("template-injection requires an Actions AST")
	}
	resolved, err := actions.Resolve(input.Workflow)
	if err != nil {
		return nil, err
	}
	checker := templateInjectionChecker{rule: taint.Rule{
		ID: "template-injection", IsSource: templateInjectionSource,
		IsFilter: templateInjectionFilter,
	}}
	jobs := slices.Clone(resolved.Jobs)
	slices.SortFunc(jobs, func(a, b *actions.Job) int { return strings.Compare(string(a.ID), string(b.ID)) })
	for _, job := range jobs {
		bindings, err := checker.environment(ctx, resolved.Env, "env", job.ID, resolved, nil)
		if err != nil {
			return checker.diagnostics, err
		}
		path := "jobs." + string(job.ID)
		bindings, err = checker.environment(ctx, job.Env, path+".env", job.ID, resolved, bindings)
		if err != nil {
			return checker.diagnostics, err
		}
		for i, step := range job.Steps {
			stepPath := fmt.Sprintf("%s.steps[%d]", path, i)
			if err := checker.step(ctx, step, stepPath, job.ID, resolved, bindings); err != nil {
				return checker.diagnostics, err
			}
		}
	}
	sortDiagnostics(checker.diagnostics)
	return checker.diagnostics, nil
}

type templateInjectionChecker struct {
	rule        taint.Rule
	diagnostics []Diagnostic
}

func (c *templateInjectionChecker) step(ctx context.Context, step actions.Step, path string, job actions.JobID, workflow *actions.Workflow, bindings map[string][]taint.Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var script string
	var env map[string][]taint.Source
	var err error
	switch value := step.(type) {
	case actions.RunStep:
		script = value.Run
		env, err = c.environment(ctx, value.Env, path+".env", job, workflow, bindings)
	case actions.RawStep:
		run, exists := value.Fields["run"]
		if !exists {
			return nil
		}
		var ok bool
		script, ok = run.(string)
		if !ok {
			return fmt.Errorf("run script must be a string at %s.run", path)
		}
		env, err = c.rawEnvironment(ctx, value.Fields["env"], path+".env", bindings)
	case actions.ActionStep:
		return nil
	default:
		return fmt.Errorf("unsupported step at %s", path)
	}
	if err != nil {
		return err
	}
	// Interpolation happens before every shell/interpreter starts, even
	// within apparent quotes, comments, or quoted heredoc bodies.
	expressions, err := c.text(ctx, script, path+".run", env)
	for _, expression := range expressions {
		if len(expression.Sources) == 0 {
			continue
		}
		related := make([]RelatedLocation, 0, len(expression.Sources))
		for _, source := range expression.Sources {
			related = append(related, RelatedLocation{Path: source.Path})
		}
		c.diagnostics = append(c.diagnostics, Diagnostic{
			Path: path + ".run", Line: expression.Line, Related: slices.Compact(related),
			Message: "Attacker-controlled Actions value is interpolated into executable script text; " +
				"bind it through env and use a quoted variable expansion instead",
		})
	}
	return err
}

func (c *templateInjectionChecker) text(ctx context.Context, text, path string, bindings map[string][]taint.Source) ([]taint.TextExpression, error) {
	expressions, err := taint.AnalyzeText(ctx, text, path, c.rule, bindings)
	var parseError *taint.ExpressionError
	if errors.As(err, &parseError) {
		c.diagnostics = append(c.diagnostics, Diagnostic{
			Severity: Warning, Path: path, Line: parseError.Line,
			Message: "Incomplete template-injection analysis: invalid Actions expression; " +
				"fix the expression and rerun the check",
		})
		return expressions, nil
	}
	return expressions, err
}

func (c *templateInjectionChecker) environment(ctx context.Context, env map[string]actions.Expression, path string, job actions.JobID, workflow *actions.Workflow, parent map[string][]taint.Source) (map[string][]taint.Source, error) {
	values := make(map[string]any, len(env))
	for _, key := range slices.Sorted(maps.Keys(env)) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		text, err := actions.RenderExpression(env[key], job, workflow, true)
		if err != nil {
			return nil, fmt.Errorf("cannot render Actions environment binding at %s.%s", path, key)
		}
		values[key] = text
	}
	return c.rawEnvironment(ctx, values, path, parent)
}

func (c *templateInjectionChecker) rawEnvironment(ctx context.Context, raw any, path string, parent map[string][]taint.Source) (map[string][]taint.Source, error) {
	bindings := maps.Clone(parent)
	if bindings == nil {
		bindings = make(map[string][]taint.Source)
	}
	if raw == nil {
		return bindings, nil
	}
	env, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("environment must be a mapping at %s", path)
	}
	for _, key := range slices.Sorted(maps.Keys(env)) {
		text, ok := env[key].(string)
		if !ok {
			switch env[key].(type) {
			case nil, bool, int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64, float32, float64:
				text = ""
			default:
				return nil, fmt.Errorf("environment binding must be a scalar at %s.%s", path, key)
			}
		}
		expressions, err := c.text(ctx, text, path+"."+key, parent)
		if err != nil {
			return nil, err
		}
		var sources []taint.Source
		for _, expression := range expressions {
			sources = append(sources, expression.Sources...)
		}
		bindings["env."+strings.ToLower(key)] = sources
	}
	return bindings, nil
}

func templateInjectionFilter(node actionlint.ExprNode) bool {
	switch n := node.(type) {
	case *actionlint.CompareOpNode, *actionlint.NotOpNode:
		return true
	case *actionlint.FuncCallNode:
		switch strings.ToLower(n.Callee) {
		case "contains", "startswith", "endswith":
			return true
		}
	}
	return false
}

func templateInjectionSource(reference string) bool {
	// Match known string-bearing fields, their aggregate parents, and computed
	// indexes. Numeric IDs and arbitrary compiler step/job outputs are not sources.
	sources := []string{
		"inputs", "github.head_ref", "github.ref", "github.ref_name",
		"github.event.inputs", "github.event.client_payload",
		"github.event.issue.title", "github.event.issue.body",
		"github.event.issue.labels.*.name", "github.event.issue.labels.*.description",
		"github.event.pull_request.title", "github.event.pull_request.body",
		"github.event.pull_request.head.ref", "github.event.pull_request.head.label",
		"github.event.pull_request.head.repo.default_branch",
		"github.event.pull_request.labels.*.name", "github.event.pull_request.labels.*.description",
		"github.event.comment.body", "github.event.review.body", "github.event.review_comment.body",
		"github.event.discussion.title", "github.event.discussion.body",
		"github.event.label.name", "github.event.label.description", "github.event.pages.*.page_name",
		"github.event.milestone.title", "github.event.milestone.description",
		"github.event.head_commit.message", "github.event.head_commit.author.name",
		"github.event.head_commit.author.email", "github.event.head_commit.committer.name",
		"github.event.head_commit.committer.email",
		"github.event.commits.*.message", "github.event.commits.*.author.name",
		"github.event.commits.*.author.email", "github.event.commits.*.committer.name",
		"github.event.commits.*.committer.email",
		"github.event.release.name", "github.event.release.body", "github.event.release.tag_name",
		"github.event.workflow_run.head_branch", "github.event.workflow_run.head_commit.message",
	}
	parts := strings.Split(strings.ToLower(reference), ".")
	for _, source := range sources {
		fields := strings.Split(source, ".")
		length := min(len(parts), len(fields))
		if slices.EqualFunc(parts[:length], fields[:length], func(part, field string) bool {
			return part == field || part == "*" || field == "*"
		}) {
			return true
		}
	}
	return false
}
