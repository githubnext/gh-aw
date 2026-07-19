package workflow

import (
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var jobLog = logger.New("workflow:jobs")

const runtimeFeaturesEnvVarName = "GH_AW_RUNTIME_FEATURES"
const runtimeFeaturesEnvVarExpression = "${{ vars.GH_AW_RUNTIME_FEATURES }}"

const pushExperimentsStateJobName = "push_experiments_state"
const pushEvalsStateJobName = "push_evals_state"
const pushRepoMemoryJobName = "push_repo_memory"
const updateCacheMemoryJobName = "update_cache_memory"

var runtimeFeaturesBuiltInJobNames = map[string]struct{}{
	string(constants.AgentJobName):              {},
	string(constants.ActivationJobName):         {},
	string(constants.PreActivationJobName):      {},
	string(constants.DetectionJobName):          {},
	string(constants.EvalsJobName):              {},
	string(constants.SafeOutputsJobName):        {},
	string(constants.UploadAssetsJobName):       {},
	string(constants.UploadCodeScanningJobName): {},
	string(constants.ConclusionJobName):         {},
	string(constants.UnlockJobName):             {},
	pushExperimentsStateJobName:                 {},
	pushEvalsStateJobName:                       {},
	pushRepoMemoryJobName:                       {},
	updateCacheMemoryJobName:                    {},
}

// Job represents a GitHub Actions job with all its properties
type Job struct {
	Name                       string
	DisplayName                string // Optional display name for the job (name property in YAML)
	RunsOn                     string
	If                         string
	HasWorkflowRunSafetyChecks bool // If true, the job's if condition includes workflow_run safety checks
	PermissionsComment         string
	Permissions                string
	TimeoutMinutes             int
	TimeoutMinutesExpression   string
	Concurrency                string            // Job-level concurrency configuration
	Environment                string            // Job environment configuration
	Strategy                   string            // Job strategy configuration (matrix strategy)
	Container                  string            // Job container configuration
	Services                   string            // Job services configuration
	Env                        map[string]string // Job-level environment variables
	ContinueOnError            *bool             // continue-on-error flag for the job (nil means unset)
	Steps                      []string
	Needs                      []string // Job dependencies (needs clause)
	Outputs                    map[string]string

	// Reusable workflow call properties
	Uses           string            // Path to reusable workflow (e.g., ./.github/workflows/reusable.yml)
	With           map[string]any    // Input parameters for reusable workflow
	Secrets        map[string]string // Secrets for reusable workflow (explicit mappings)
	SecretsInherit bool              // When true, emits "secrets: inherit" (passes all caller secrets)
}

// JobManager manages a collection of jobs and handles dependency validation
type JobManager struct {
	jobs     map[string]*Job
	jobOrder []string // Job names in sorted alphabetical order
}

// NewJobManager creates a new JobManager instance
func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*Job),
	}
}

// AddJob adds a job to the manager
func (jm *JobManager) AddJob(job *Job) error {
	if err := validateJobDefinition(job); err != nil {
		return err
	}

	if job.Name == "" {
		return errors.New("job name cannot be empty")
	}

	if _, exists := jm.jobs[job.Name]; exists {
		return fmt.Errorf("job '%s' already exists", job.Name)
	}

	jobLog.Printf("Adding job: %s", job.Name)
	jm.jobs[job.Name] = job
	jm.jobOrder = append(jm.jobOrder, job.Name)
	// Keep jobOrder sorted alphabetically after each addition
	sort.Strings(jm.jobOrder)
	return nil
}

// GetJob retrieves a job by name
func (jm *JobManager) GetJob(name string) (*Job, bool) {
	job, exists := jm.jobs[name]
	return job, exists
}

// GetAllJobs returns all jobs in the manager
func (jm *JobManager) GetAllJobs() map[string]*Job {
	// Return a copy to prevent external modification
	result := make(map[string]*Job)
	maps.Copy(result, jm.jobs)
	return result
}

// extractStepName extracts the step name from a YAML step string
// Returns empty string if no name is found
func extractStepName(stepYAML string) string {
	// Look for "name: " in the step YAML
	// Format is typically "      - name: Step Name" with various indentation
	lines := strings.SplitSeq(stepYAML, "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		// Remove leading dash if present
		trimmed = strings.TrimPrefix(trimmed, "-")
		trimmed = strings.TrimSpace(trimmed)

		if after, ok := strings.CutPrefix(trimmed, "name:"); ok {
			// Extract the name value after "name:"
			name := strings.TrimSpace(after)
			// Remove quotes if present
			name = strings.Trim(name, "\"'")
			return name
		}
	}
	return ""
}

// detectCycles uses DFS to detect cycles in the job dependency graph
func (jm *JobManager) detectCycles() error {
	jobLog.Print("Detecting cycles in job dependency graph")
	// Track visit states: 0=unvisited, 1=visiting, 2=visited
	visitState := make(map[string]int)

	// Initialize all jobs as unvisited
	for jobName := range jm.jobs {
		visitState[jobName] = 0
	}

	// Run DFS from each unvisited job
	for jobName := range jm.jobs {
		if visitState[jobName] == 0 {
			if err := jm.dfsVisit(jobName, visitState); err != nil {
				return err
			}
		}
	}

	jobLog.Print("No cycles detected in job dependencies")
	return nil
}

// dfsVisit performs DFS visit for cycle detection
func (jm *JobManager) dfsVisit(jobName string, visitState map[string]int) error {
	visitState[jobName] = 1 // Mark as visiting

	job := jm.jobs[jobName]
	for _, dep := range job.Needs {
		if visitState[dep] == 1 {
			// Found a back edge - cycle detected
			jobLog.Printf("Cycle detected: job %s has circular dependency through %s", jobName, dep)
			return fmt.Errorf("cycle detected in job dependencies: job '%s' has circular dependency through '%s'", jobName, dep)
		}
		if visitState[dep] == 0 {
			if err := jm.dfsVisit(dep, visitState); err != nil {
				return err
			}
		}
	}

	visitState[jobName] = 2 // Mark as visited
	return nil
}

// WriteJobsYAML writes the jobs section of a GitHub Actions workflow directly
// to b, avoiding an intermediate string copy.  Callers that already hold a
// *strings.Builder (e.g. generateWorkflowBody) should prefer this method over
// RenderToYAML to reduce allocations.
func (jm *JobManager) WriteJobsYAML(b *strings.Builder) {
	jobLog.Printf("Rendering %d jobs to YAML", len(jm.jobs))
	if len(jm.jobs) == 0 {
		b.WriteString("jobs:\n")
		return
	}

	b.WriteString("jobs:\n")

	// jobOrder is kept sorted alphabetically by AddJob
	for _, jobName := range jm.jobOrder {
		jm.renderJobTo(b, jm.jobs[jobName])
	}
}

// renderJobTo writes a single job to b directly, with no intermediate string allocation.
func (jm *JobManager) renderJobTo(b *strings.Builder, job *Job) {
	jobLog.Printf("Rendering job: %s (steps=%d, needs=%d, reusable=%t)", job.Name, len(job.Steps), len(job.Needs), job.Uses != "")
	fmt.Fprintf(b, "  %s:\n", job.Name)
	if job.DisplayName != "" {
		fmt.Fprintf(b, "    name: %s\n", job.DisplayName)
	}
	renderJobNeedsTo(b, job)
	renderJobIfTo(b, job)
	renderJobExecutionConfigTo(b, job)
	renderJobSecurityAndLimitsTo(b, job)
	renderJobEnvAndOutputsTo(b, job)
	if job.Uses != "" {
		renderReusableJobTo(b, job)
	} else {
		renderJobStepsTo(b, job)
	}
	b.WriteString("\n")
}

func renderJobNeedsTo(b *strings.Builder, job *Job) {
	if len(job.Needs) > 0 {
		if len(job.Needs) == 1 {
			fmt.Fprintf(b, "    needs: %s\n", job.Needs[0])
		} else {
			b.WriteString("    needs:\n")
			// Sort needs for consistent output
			sortedNeeds := make([]string, len(job.Needs))
			copy(sortedNeeds, job.Needs)
			sort.Strings(sortedNeeds)
			for _, dep := range sortedNeeds {
				fmt.Fprintf(b, "      - %s\n", dep)
			}
		}
	}
}

func renderJobIfTo(b *strings.Builder, job *Job) {
	if job.If != "" {
		if job.HasWorkflowRunSafetyChecks {
			b.WriteString("    # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation\n")
		}
		if hasNewlineInStringLiteral(job.If) {
			fmt.Fprintf(b, "    if: \"%s\"\n", escapeForYAMLDoubleQuoted(job.If))
		} else if strings.Contains(job.If, "\n") || len(job.If) > int(constants.MaxExpressionLineLength) {
			renderFoldedJobIfTo(b, job.If)
		} else {
			fmt.Fprintf(b, "    if: %s\n", job.If)
		}
	}
}

func renderFoldedJobIfTo(b *strings.Builder, condition string) {
	b.WriteString("    if: >\n")
	if strings.Contains(condition, "\n") {
		for line := range strings.SplitSeq(condition, "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintf(b, "      %s\n", strings.TrimSpace(line))
			}
		}
		return
	}
	for _, line := range BreakLongExpression(condition) {
		fmt.Fprintf(b, "      %s\n", strings.TrimSpace(line))
	}
}

func renderJobExecutionConfigTo(b *strings.Builder, job *Job) {
	if job.RunsOn != "" {
		fmt.Fprintf(b, "    %s\n", job.RunsOn)
	}
	if job.Strategy != "" {
		fmt.Fprintf(b, "    %s\n", strings.TrimRight(job.Strategy, "\n"))
	}
	if job.Environment != "" {
		fmt.Fprintf(b, "    %s\n", job.Environment)
	}
	if job.Container != "" {
		fmt.Fprintf(b, "    %s\n", job.Container)
	}
	if job.Services != "" {
		fmt.Fprintf(b, "    %s\n", job.Services)
	}
}

func renderJobSecurityAndLimitsTo(b *strings.Builder, job *Job) {
	if job.PermissionsComment != "" {
		for line := range strings.SplitSeq(strings.TrimRight(job.PermissionsComment, "\n"), "\n") {
			fmt.Fprintf(b, "    %s\n", line)
		}
	}
	if job.Permissions != "" {
		fmt.Fprintf(b, "    %s\n", job.Permissions)
	}
	if job.Concurrency != "" {
		fmt.Fprintf(b, "    %s\n", job.Concurrency)
	}
	if job.TimeoutMinutesExpression != "" {
		fmt.Fprintf(b, "    timeout-minutes: %s\n", job.TimeoutMinutesExpression)
	} else if job.TimeoutMinutes > 0 {
		fmt.Fprintf(b, "    timeout-minutes: %d\n", job.TimeoutMinutes)
	}
	if job.ContinueOnError != nil {
		fmt.Fprintf(b, "    continue-on-error: %t\n", *job.ContinueOnError)
	}
}

func renderJobEnvAndOutputsTo(b *strings.Builder, job *Job) {
	env := buildRenderedJobEnv(job)
	if len(env) > 0 {
		b.WriteString("    env:\n")
		envKeys := sliceutil.SortedKeys(env)
		for _, key := range envKeys {
			fmt.Fprintf(b, "      %s: %s\n", key, env[key])
		}
	}
	if len(job.Outputs) > 0 {
		b.WriteString("    outputs:\n")
		outputKeys := sliceutil.SortedKeys(job.Outputs)
		for _, key := range outputKeys {
			fmt.Fprintf(b, "      %s: %s\n", key, job.Outputs[key])
		}
	}
}

func renderReusableJobTo(b *strings.Builder, job *Job) {
	jobLog.Printf("Rendering reusable workflow call: %s uses=%s with=%d secrets=%d", job.Name, job.Uses, len(job.With), len(job.Secrets))
	fmt.Fprintf(b, "    uses: %s\n", job.Uses)
	renderReusableJobWithTo(b, job)
	renderReusableJobSecretsTo(b, job)
}

func renderReusableJobWithTo(b *strings.Builder, job *Job) {
	if len(job.With) == 0 {
		return
	}
	b.WriteString("    with:\n")
	for _, key := range sliceutil.SortedKeys(job.With) {
		switch v := job.With[key].(type) {
		case string:
			fmt.Fprintf(b, "      %s: %s\n", key, v)
		case int, int64, float64:
			fmt.Fprintf(b, "      %s: %v\n", key, v)
		case bool:
			fmt.Fprintf(b, "      %s: %t\n", key, v)
		default:
			fmt.Fprintf(b, "      %s: %v\n", key, v)
		}
	}
}

func renderReusableJobSecretsTo(b *strings.Builder, job *Job) {
	if job.SecretsInherit {
		b.WriteString("    secrets: inherit\n")
	} else if len(job.Secrets) > 0 {
		b.WriteString("    secrets:\n")
		for _, key := range sliceutil.SortedKeys(job.Secrets) {
			fmt.Fprintf(b, "      %s: %s\n", key, job.Secrets[key])
		}
	}
}

func renderJobStepsTo(b *strings.Builder, job *Job) {
	if len(job.Steps) == 0 {
		return
	}
	b.WriteString("    steps:\n")
	for _, step := range job.Steps {
		b.WriteString(step)
	}
}

func buildRenderedJobEnv(job *Job) map[string]string {
	if job == nil {
		return nil
	}
	env := maps.Clone(job.Env)
	if shouldInjectRuntimeFeaturesEnv(job) {
		if env == nil {
			env = make(map[string]string)
		}
		if _, exists := env[runtimeFeaturesEnvVarName]; !exists {
			env[runtimeFeaturesEnvVarName] = runtimeFeaturesEnvVarExpression
		}
	}
	return env
}

func shouldInjectRuntimeFeaturesEnv(job *Job) bool {
	if job == nil || job.Uses != "" {
		return false
	}
	_, ok := runtimeFeaturesBuiltInJobNames[job.Name]
	return ok
}
