package workflow

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
)

const localArtifactPrefix = "${{ steps.artifact-prefix.outputs.prefix }}"

var artifactStepStart = regexp.MustCompile(`(?m)^      - `)

type artifactProtocolStep struct {
	raw             string
	start           int
	end             int
	ID              string           `yaml:"id"`
	Uses            string           `yaml:"uses"`
	If              string           `yaml:"if"`
	ContinueOnError *TemplatableBool `yaml:"continue-on-error"`
	With            map[string]any   `yaml:"with"`
}

func parseArtifactProtocolSteps(job *Job) ([]artifactProtocolStep, error) {
	raw := strings.Join(job.Steps, "")
	offsets := artifactStepStart.FindAllStringIndex(raw, -1)
	var steps []artifactProtocolStep
	for i, offset := range offsets {
		bounds := [2]int(offset)
		start, end := bounds[0], len(raw)
		if i == 0 {
			start = 0
		}
		next := i + 1
		if next >= 0 && next < len(offsets) {
			nextBounds := [2]int(offsets[next])
			end = nextBounds[0]
		}
		block := raw[start:end]
		var parsed []artifactProtocolStep
		if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
			return nil, fmt.Errorf("cannot inspect artifact steps in job %s: %w", job.Name, err)
		}
		if len(parsed) > 1 {
			return nil, fmt.Errorf("expected one artifact step block in job %s", job.Name)
		}
		if len(parsed) > 0 {
			step := parsed[0]
			step.raw = block
			step.start, step.end = start, end
			steps = append(steps, step)
		} else {
			return nil, fmt.Errorf("expected an artifact step block in job %s", job.Name)
		}
	}
	return steps, nil
}

// bindArtifactProducerAttempts separates current uploads from retained upstream
// artifacts after all jobs exist, so every download binds to its actual producer.
func (c *Compiler) bindArtifactProducerAttempts(data *WorkflowData) error {
	if !hasWorkflowCallTrigger(data.On) {
		return nil
	}
	jobs := c.jobManager.GetAllJobs()
	parsed := make(map[string][]artifactProtocolStep)
	producers := make(map[string]string)
	for name, job := range jobs {
		steps, err := parseArtifactProtocolSteps(job)
		if err != nil {
			return err
		}
		parsed[name] = steps
		for _, step := range steps {
			artifact, ok := step.With["name"].(string)
			if !ok {
				continue
			}
			if strings.HasPrefix(step.Uses, "actions/upload-artifact@") && strings.HasPrefix(artifact, localArtifactPrefix) {
				if prior, exists := producers[artifact]; exists && prior != name {
					return fmt.Errorf("artifact %s has ambiguous producers %s and %s", artifact, prior, name)
				}
				producers[artifact] = name
			}
		}
	}
	// The optional tiny agent fallback shares its producer even when not emitted.
	if producer, exists := producers[localArtifactPrefix+constants.AgentArtifactName.String()]; exists {
		producers[localArtifactPrefix+constants.AgentOutputFallbackArtifactName.String()] = producer
	}
	for name, job := range jobs {
		if err := c.bindJobArtifactSources(job, parsed[name], producers); err != nil {
			return err
		}
	}
	return nil
}

func artifactDownloadSources(step artifactProtocolStep, producers map[string]string) (map[string]string, error) {
	sources := make(map[string]string)
	if !strings.HasPrefix(step.Uses, "actions/download-artifact@") {
		return sources, nil
	}
	for _, key := range []string{"name", "pattern"} {
		value, ok := step.With[key].(string)
		if !ok || !strings.Contains(value, localArtifactPrefix) {
			continue
		}
		for pattern := range strings.SplitSeq(strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}"), ",") {
			found := false
			for artifact, producer := range producers {
				matched, err := path.Match(strings.TrimPrefix(pattern, localArtifactPrefix), strings.TrimPrefix(artifact, localArtifactPrefix))
				if err != nil {
					return nil, fmt.Errorf("invalid artifact pattern %q: %w", pattern, err)
				}
				if matched {
					if prior, exists := sources[pattern]; exists && prior != producer {
						return nil, fmt.Errorf("artifact pattern %q has multiple producers", pattern)
					}
					sources[pattern], found = producer, true
				}
			}
			if !found {
				if templatableBoolIsTrue(step.ContinueOnError) {
					// No job can supply this optional channel in this configuration.
					sources[pattern] = ""
					continue
				}
				return nil, fmt.Errorf("artifact download %q has no bound producer", pattern)
			}
		}
	}
	return sources, nil
}

func (c *Compiler) bindJobArtifactSources(job *Job, steps []artifactProtocolStep, producers map[string]string) error {
	required := make(map[string]bool)
	var edits []artifactStepEdit
	for _, step := range steps {
		sources, err := artifactDownloadSources(step, producers)
		if err != nil {
			return fmt.Errorf("job %s: %w", job.Name, err)
		}
		var guards []string
		for _, producer := range sources {
			if producer == "" {
				guards = append(guards, "false")
				continue
			}
			if producer == job.Name {
				continue
			}
			required[producer] = required[producer] || !templatableBoolIsTrue(step.ContinueOnError)
			guards = append(guards, "steps.artifact-source-"+producer+".outcome == 'success'")
		}
		if len(guards) > 0 {
			sort.Strings(guards)
			edits = append(edits, artifactDownloadEdits(step, sources, job.Name)...)
			edits = append(edits, artifactDownloadGuardEdit(step, strings.Join(slices.Compact(guards), " && ")))
		}
	}
	return c.addArtifactAttemptSteps(job, steps, edits, producers, required)
}

func (c *Compiler) addArtifactAttemptSteps(job *Job, steps []artifactProtocolStep, edits []artifactStepEdit, producers map[string]string, required map[string]bool) error {
	uploads := false
	for _, producer := range producers {
		uploads = uploads || producer == job.Name
	}
	if !uploads && len(required) == 0 {
		if len(edits) > 0 {
			var err error
			job.Steps, err = applyArtifactStepEdits(job.Steps, edits)
			return err
		}
		return nil
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !slices.Contains(job.Needs, name) {
			job.Needs = append(job.Needs, name)
		}
	}
	found := false
	for _, step := range steps {
		if step.ID != "artifact-prefix" {
			continue
		}
		if found {
			return fmt.Errorf("job %s has duplicate artifact prefix producers", job.Name)
		}
		found = true
		var added []string
		if uploads {
			added = append(added, c.artifactProducerPublishSteps(job.Name)...)
		}
		for _, producer := range names {
			added = append(added, c.artifactProducerResolveSteps(producer, required[producer])...)
		}
		edits = append(edits, artifactStepEdit{start: step.end, end: step.end, steps: added})
	}
	if !found {
		return fmt.Errorf("job %s consumes artifact metadata without local prefix computation", job.Name)
	}
	var err error
	job.Steps, err = applyArtifactStepEdits(job.Steps, edits)
	return err
}
