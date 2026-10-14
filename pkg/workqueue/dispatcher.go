package workqueue

import (
	"context"
	"regexp"

	"github.com/github/gh-aw/pkg/parser"
	"gopkg.in/yaml.v3"
)

var dispatcherNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

type DispatcherRoute struct {
	Repository string `json:"repository"`
	Workflow   string `json:"workflow"`
	Ref        string `json:"ref"`
	Revision   string `json:"revision"`
}

// DispatcherFromConfig resolves a dispatcher, never a worker assignment, at the
// repository's verified default revision. It does not grant or launch Claims.
func (b Branch) DispatcherFromConfig(ctx context.Context, name string) (DispatcherRoute, error) {
	if !dispatcherNamePattern.MatchString(name) {
		return DispatcherRoute{}, queueError("dispatcher_invalid", "expected a dispatcher basename, for example eslint-factory-dispatcher")
	}
	b, err := b.withClient()
	if err != nil {
		return DispatcherRoute{}, err
	}
	repository, ref, revision, err := b.verifiedDefaultRoute(ctx)
	if err != nil {
		return DispatcherRoute{}, err
	}
	b.Remote = repository
	source, err := b.configFile(ctx, workerWorkflowDirectory+name+".md", revision)
	if err != nil {
		return DispatcherRoute{}, err
	}
	if err := validateDispatcherSource(source); err != nil {
		return DispatcherRoute{}, err
	}
	workflow := name + ".lock.yml"
	compiled, err := b.configFile(ctx, workerWorkflowDirectory+workflow, revision)
	if err != nil {
		return DispatcherRoute{}, err
	}
	if err := validateDispatcherLock(compiled); err != nil {
		return DispatcherRoute{}, err
	}
	return DispatcherRoute{Repository: repository, Workflow: workflow, Ref: ref, Revision: revision}, nil
}

func validateDispatcherSource(source []byte) error {
	frontmatter, err := parser.ExtractFrontmatterFromContent(string(source))
	if err != nil {
		return err
	}
	if frontmatter == nil {
		return queueError("dispatcher_invalid", "dispatcher source requires agentic workflow frontmatter")
	}
	tools, ok := frontmatter.Frontmatter["tools"].(map[string]any)
	if !ok {
		return queueError("dispatcher_invalid", "dispatcher must declare work-queue tools")
	}
	queue := tools["work-queue"]
	enabled := queue == true
	if options, ok := queue.(map[string]any); ok {
		enabled = options["worker"] != true && options["require-assignment"] != true
	}
	if !enabled {
		return queueError("dispatcher_invalid", "expected an unassigned work-queue dispatcher; workers require authenticated Claims")
	}
	outputs, ok := frontmatter.Frontmatter["safe-outputs"].(map[string]any)
	if !ok {
		return queueError("dispatcher_invalid", "dispatcher must declare safe-outputs")
	}
	dispatch, ok := outputs["dispatch-workflow"].(map[string]any)
	if !ok {
		return queueError("dispatcher_invalid", "dispatcher must declare safe-outputs.dispatch-workflow")
	}
	workers, ok := dispatch["workflows"].([]any)
	if !ok || len(workers) == 0 {
		return queueError("dispatcher_invalid", "dispatcher must declare safe-outputs.dispatch-workflow.workflows")
	}
	for _, worker := range workers {
		name, ok := worker.(string)
		if !ok || !dispatcherNamePattern.MatchString(name) {
			return queueError("dispatcher_invalid", "dispatcher worker allowlist must contain workflow basenames")
		}
	}
	return nil
}

func validateDispatcherLock(compiled []byte) error {
	var workflow struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]struct {
			Env map[string]any `yaml:"env"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(compiled, &workflow); err != nil {
		return err
	}
	trigger, ok := workflow.On["workflow_dispatch"]
	if !ok {
		return queueError("dispatcher_invalid", "compiled dispatcher must accept workflow_dispatch")
	}
	if dispatch, ok := trigger.(map[string]any); ok {
		if inputs, ok := dispatch["inputs"].(map[string]any); ok {
			if _, reserved := inputs["work_queue_assignment"]; reserved {
				return queueError("dispatcher_invalid", "cannot directly dispatch a Claim-assigned worker")
			}
		}
	}
	for _, job := range workflow.Jobs {
		if job.Env["GH_AW_WORK_QUEUE_ROLE"] == "dispatcher" {
			return nil
		}
	}
	return queueError("dispatcher_invalid", "compiled workflow must contain the trusted work-queue dispatcher role")
}
