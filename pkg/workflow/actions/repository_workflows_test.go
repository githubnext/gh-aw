package actions

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryWorkflowGraphs(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", ".github", "workflows", "*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "must discover repository workflows, including generated .lock.yml files")
	var jobs, edges int
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			input := workflowGraphFromYAML(t, content)
			require.NotEmpty(t, input.Jobs)
			resolved, err := Resolve(input)
			require.NoError(t, err)
			require.Len(t, resolved.Jobs, len(input.Jobs))
			for i, job := range resolved.Jobs {
				expected := slices.Clone(input.Jobs[i].Needs)
				slices.Sort(expected)
				assert.Equal(t, slices.Compact(expected), job.Needs)
				assert.Equal(t, input.Jobs[i].ID, job.ID)
				assert.Len(t, job.Steps, len(input.Jobs[i].Steps))
				jobs++
				edges += len(job.Needs)
			}
			_, err = Lower(input)
			require.NoError(t, err, "all authored steps must also lower as native GitHub Actions")
		})
	}
	t.Logf("Built Actions graphs for %d workflow files: %d jobs, %d dependency edges", len(files), jobs, edges)
}

// Authored expressions already contain their needs bindings. Keep them opaque
// while validating the complete explicit graph and job-scoped step IDs.
func workflowGraphFromYAML(t *testing.T, content []byte) *Workflow {
	t.Helper()
	var authored struct {
		Name string `yaml:"name"`
		Jobs map[string]struct {
			RunsOn  any               `yaml:"runs-on"`
			Uses    string            `yaml:"uses"`
			Needs   any               `yaml:"needs"`
			Outputs map[string]string `yaml:"outputs"`
			Steps   []map[string]any  `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(content, &authored))
	result := &Workflow{Name: authored.Name}
	for _, id := range sortedKeys(authored.Jobs) {
		source := authored.Jobs[id]
		job := &Job{ID: JobID(id), RunsOn: source.RunsOn, Uses: source.Uses}
		switch needs := source.Needs.(type) {
		case nil:
		case string:
			job.Needs = []JobID{JobID(needs)}
		case []any:
			for _, dependency := range needs {
				name, ok := dependency.(string)
				require.True(t, ok, "needs entries must be job IDs")
				job.Needs = append(job.Needs, JobID(name))
			}
		default:
			t.Fatalf("job %q has unsupported needs type %T", id, needs)
		}
		job.Outputs = make(map[OutputName]Expression, len(source.Outputs))
		for name, value := range source.Outputs {
			job.Outputs[OutputName(name)] = Literal(value)
		}
		for _, fields := range source.Steps {
			job.Steps = append(job.Steps, RawStep{Fields: fields})
		}
		result.Jobs = append(result.Jobs, job)
	}
	return result
}

func TestAuthoredWorkflowGraphValidation(t *testing.T) {
	tests := []struct {
		name string
		jobs string
		want string
	}{
		{"scalar needs", "a: {runs-on: ubuntu-latest, steps: [{run: echo a}]}\nb: {uses: './worker.yml', needs: a}", ""},
		{"list needs", "a: {runs-on: ubuntu-latest, steps: [{run: echo a}]}\nb: {uses: './worker.yml', needs: [a]}", ""},
		{"missing dependency", "a: {uses: './worker.yml', needs: missing}", "non-existent job"},
		{"cycle", "a: {uses: './worker.yml', needs: b}\nb: {uses: './worker.yml', needs: a}", "cycle detected"},
		{"duplicate step IDs", "a: {runs-on: ubuntu-latest, steps: [{id: same, run: echo a}, {id: same, run: echo b}]}", "duplicate workflow step"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workflowGraphFromYAML(t, []byte("jobs:\n  "+strings.ReplaceAll(tt.jobs, "\n", "\n  ")+"\n"))
			resolved, err := Resolve(input)
			if tt.want != "" {
				require.ErrorContains(t, err, tt.want)
				return
			}
			require.NoError(t, err)
			require.Len(t, resolved.Jobs, 2)
			assert.Equal(t, []JobID{"a"}, resolved.Jobs[1].Needs)
		})
	}
}
