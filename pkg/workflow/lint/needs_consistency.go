package lint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/goccy/go-yaml"
)

func NeedsConsistencyRule() Rule {
	return Rule{
		ID: "needs-consistency", Description: "Keep job dependencies and needs expressions connected to declared producers",
		Severity: Warning, Targets: ActionsAST | AWAST, Check: checkNeedsConsistency,
	}
}

type needsJob struct {
	path    string
	fields  map[string]any
	needs   map[string]struct{}
	outputs map[string]any
}

type needsChecker struct {
	jobs        map[string]*needsJob
	authored    bool
	imports     bool
	diagnostics []Diagnostic
	reported    map[string]struct{}
	root        map[string]any
}

func checkNeedsConsistency(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	checker := needsChecker{jobs: make(map[string]*needsJob), authored: input.Workflow == nil}
	if err := checker.index(ctx, input); err != nil {
		return nil, err
	}
	if checker.root != nil {
		if err := checker.expressions(ctx, &needsJob{fields: checker.root}); err != nil {
			return checker.diagnostics, err
		}
	}
	for _, id := range sliceutil.SortedKeys(checker.jobs) {
		job := checker.jobs[id]
		for _, dep := range sliceutil.SortedKeys(job.needs) {
			switch {
			case dep == id:
				checker.report(job.path+".needs", "Job depends on itself; remove the self dependency")
			case checker.jobs[dep] == nil && !checker.deferred(dep):
				checker.report(job.path+".needs", "Dependency names an unknown job; use an existing job ID in needs")
			}
		}
		if err := checker.expressions(ctx, job); err != nil {
			return checker.diagnostics, err
		}
	}
	if err := checker.cycles(ctx); err != nil {
		return checker.diagnostics, err
	}
	return checker.diagnostics, nil
}

func (c *needsChecker) index(ctx context.Context, input Input) error {
	if c.authored {
		imports, err := authoredImportsPending(input)
		if err != nil {
			return err
		}
		c.imports = imports
		blocks, err := aw.ExecutionBlocks(ctx, input.Document)
		if err != nil {
			return err
		}
		for _, block := range blocks {
			var id string
			switch {
			case strings.HasPrefix(block.Path, "jobs."):
				id = strings.TrimPrefix(block.Path, "jobs.")
			case strings.HasPrefix(block.Path, "safe-jobs."):
				id = stringutil.NormalizeSafeOutputIdentifier(strings.TrimPrefix(block.Path, "safe-jobs."))
			case strings.HasPrefix(block.Path, "safe-outputs.jobs."):
				id = stringutil.NormalizeSafeOutputIdentifier(strings.TrimPrefix(block.Path, "safe-outputs.jobs."))
			default:
				continue // These scopes receive compiler-owned dependencies.
			}
			if err := c.add(ctx, id, block.Path, block.Fields); err != nil {
				return err
			}
		}
		return nil
	}
	root, err := actions.Lower(input.Workflow)
	if err != nil {
		return errors.New("needs-consistency requires a valid Actions graph; correct job dependencies and output bindings before linting")
	}
	c.root = make(map[string]any)
	for _, item := range root {
		if item.Key != "jobs" {
			if key, ok := item.Key.(string); ok {
				c.root[key] = item.Value
			}
			continue
		}
		jobs, ok := item.Value.(yaml.MapSlice)
		if !ok {
			return errors.New("needs-consistency requires an Actions jobs mapping")
		}
		for _, entry := range jobs {
			id, validID := entry.Key.(string)
			fields, validFields := entry.Value.(map[string]any)
			if !validID || !validFields {
				return errors.New("needs-consistency requires Actions job mappings with string IDs")
			}
			if err := c.add(ctx, id, "jobs."+id, fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *needsChecker) add(ctx context.Context, id, path string, fields map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := strings.ToLower(id)
	if c.jobs[key] != nil {
		return fmt.Errorf("needs-consistency requires unambiguous job IDs at %s; rename the conflicting job", path)
	}
	job := &needsJob{path: path, fields: fields, needs: make(map[string]struct{})}
	if fields["needs"] != nil {
		deps, valid, err := workflowRunStrings(ctx, fields["needs"], true)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("needs-consistency requires literal job IDs at %s.needs; use needs: [build]", path)
		}
		for _, dep := range deps {
			if strings.TrimSpace(dep) == "" || strings.Contains(dep, "${{") {
				return fmt.Errorf("needs-consistency requires literal job IDs at %s.needs; use needs: [build]", path)
			}
			if strings.HasPrefix(path, "safe-jobs.") || strings.HasPrefix(path, "safe-outputs.jobs.") {
				dep = stringutil.NormalizeSafeOutputIdentifier(dep)
			}
			job.needs[strings.ToLower(dep)] = struct{}{}
		}
	}
	outputs, err := policyMapping(fields["outputs"], path+".outputs")
	if err != nil {
		return err
	}
	job.outputs = make(map[string]any, len(outputs))
	for name, value := range outputs {
		job.outputs[strings.ToLower(name)] = value
	}
	c.jobs[key] = job
	return nil
}

func (c *needsChecker) deferred(id string) bool {
	if !c.authored {
		return false
	}
	_, builtin := constants.KnownBuiltInJobNames[id]
	return builtin || c.imports
}

func (c *needsChecker) report(path, message string) {
	if c.reported == nil {
		c.reported = make(map[string]struct{})
	}
	key := path + "\n" + message
	if _, reported := c.reported[key]; reported {
		return
	}
	c.reported[key] = struct{}{}
	c.diagnostics = append(c.diagnostics, Diagnostic{Path: path, Message: message})
}

func (c *needsChecker) cycles(ctx context.Context) error {
	state := make(map[string]int)
	var visit func(string) error
	visit = func(id string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		state[id] = 1
		job := c.jobs[id]
		for _, dep := range sliceutil.SortedKeys(job.needs) {
			if dep == id || c.jobs[dep] == nil {
				continue
			}
			switch state[dep] {
			case 1:
				c.report(job.path+".needs", "Job dependencies contain a cycle; remove a needs edge to make the graph acyclic")
			case 0:
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		state[id] = 2
		return nil
	}
	for _, id := range sliceutil.SortedKeys(c.jobs) {
		if state[id] == 0 {
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
}
