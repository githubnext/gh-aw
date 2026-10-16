package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestConclusionTimingInventoryUsesFinalJobs(t *testing.T) {
	c := NewCompiler()
	jobs := []*Job{
		{Name: "conclusion", Env: map[string]string{"AUTHOR_ENV": `"preserved"`}},
		{Name: "activation"},
		{Name: "agent", DisplayName: "Author agent"},
		{Name: "unnamed"},
		{Name: "dynamic", DisplayName: "${{ inputs.name }}"},
		{Name: "matrix", DisplayName: "Build", Strategy: "strategy:\n      matrix:\n        os: [linux]"},
		{Name: "reusable", DisplayName: "Deploy", Uses: "./.github/workflows/deploy.yml"},
		{Name: "send_report", DisplayName: "Send report"},
		{Name: "push_repo_memory"},
	}
	for _, job := range jobs {
		require.NoError(t, c.jobManager.AddJob(job))
	}
	require.NoError(t, c.finalizeBuiltJobs(&WorkflowData{}))
	var output strings.Builder
	c.jobManager.WriteJobsYAML(&output)
	var compiled struct {
		Jobs map[string]struct {
			Name string            `yaml:"name"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(output.String()), &compiled))
	env := compiled.Jobs["conclusion"].Env
	require.Equal(t, "preserved", env["AUTHOR_ENV"])
	var names map[string]string
	var eligible map[string]bool
	require.NoError(t, json.Unmarshal([]byte(env["GH_AW_JOB_DISPLAY_NAMES"]), &names))
	require.NoError(t, json.Unmarshal([]byte(env["GH_AW_JOB_TIMING_ELIGIBLE"]), &eligible))
	require.Equal(t, map[string]bool{
		"conclusion": true, "activation": true, "agent": true, "unnamed": true,
		"dynamic": false, "matrix": false, "reusable": false,
		"send_report": true, "push_repo_memory": true,
	}, eligible)
	for id, job := range compiled.Jobs {
		if id == "dynamic" {
			require.NotContains(t, names, id, "dynamic expressions must never be interpolated into serialized JSON")
		} else {
			require.Equal(t, job.Name, names[id], "inventory must match the emitted display name for %s", id)
		}
	}
	require.Equal(t, "activation", names["activation"])
	require.Equal(t, "Author agent", names["agent"])
	require.Equal(t, 1, strings.Count(output.String(), "GH_AW_JOB_DISPLAY_NAMES:"))
	require.Equal(t, 1, strings.Count(output.String(), "GH_AW_JOB_TIMING_ELIGIBLE:"))
}

func TestCompiledJobNamesSupplyTimingLookup(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the compiler/runtime inventory contract test")
	}
	dir := testutil.TempDir(t, "compiled-job-timing")
	path := filepath.Join(dir, "timing.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
on: workflow_dispatch
engine: copilot
strict: false
safe-outputs:
  add-comment:
jobs:
  agent:
    name: Author agent
  build:
    name: Build project
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
---
Run the agent.
`), 0600))
	require.NoError(t, NewCompiler().CompileWorkflow(path))
	lock, err := os.ReadFile(filepath.Join(dir, "timing.lock.yml"))
	require.NoError(t, err)
	var compiled struct {
		Jobs map[string]struct {
			Name string            `yaml:"name"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(lock, &compiled))
	env := compiled.Jobs["conclusion"].Env
	results := make(map[string]map[string]string)
	var jobs []map[string]any
	expected := make(map[string]int)
	for _, id := range []string{"activation", "agent", "safe_outputs", "build"} {
		results[id] = map[string]string{"result": "success"}
		jobs = append(jobs, map[string]any{
			"name": compiled.Jobs[id].Name, "status": "completed", "conclusion": "success",
			"started_at": "2026-10-16T10:00:00Z", "completed_at": "2026-10-16T10:00:02Z",
			"run_attempt": 1,
		})
		expected[id] = 2000
	}
	resultsJSON, err := json.Marshal(results)
	require.NoError(t, err)
	env["GH_AW_JOB_RESULTS"] = string(resultsJSON)
	env["GITHUB_RUN_ATTEMPT"] = "1"
	fixture, err := json.Marshal(map[string]any{"env": env, "jobs": jobs})
	require.NoError(t, err)
	cmd := exec.Command(node, "-e", `
const { collectJobDurations } = require("../../actions/setup/js/workflow_job_durations.cjs");
const fixture = JSON.parse(require("fs").readFileSync(0, "utf8"));
collectJobDurations({ ...fixture, runContext: { repo: {} }, githubClient: {},
  warn: message => { throw new Error(message); }
}).then(result => process.stdout.write(JSON.stringify(result)));
`)
	cmd.Stdin = strings.NewReader(string(fixture))
	actualJSON, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", actualJSON)
	var actual map[string]int
	require.NoError(t, json.Unmarshal(actualJSON, &actual))
	require.Equal(t, expected, actual)
}
