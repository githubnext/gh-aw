package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// Job represents a GitHub Actions job with all its properties
type Job struct {
	Name           string
	RunsOn         string
	If             string
	Permissions    string
	TimeoutMinutes int
	Steps          []string
	Depends        []string // Job dependencies (needs clause)
	Outputs        map[string]string
}

// JobManager manages a collection of jobs and handles dependency validation
type JobManager struct {
	jobs     map[string]*Job
	jobOrder []string // Preserves the order jobs were added
}

// NewJobManager creates a new JobManager instance
func NewJobManager() *JobManager {
	return &JobManager{
		jobs:     make(map[string]*Job),
		jobOrder: make([]string, 0),
	}
}

// AddJob adds a job to the manager
func (jm *JobManager) AddJob(job *Job) error {
	if job.Name == "" {
		return fmt.Errorf("job name cannot be empty")
	}

	if _, exists := jm.jobs[job.Name]; exists {
		return fmt.Errorf("job '%s' already exists", job.Name)
	}

	jm.jobs[job.Name] = job
	jm.jobOrder = append(jm.jobOrder, job.Name)
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
	for name, job := range jm.jobs {
		result[name] = job
	}
	return result
}

// ValidateDependencies checks that all job dependencies exist and there are no cycles
func (jm *JobManager) ValidateDependencies() error {
	// First check that all dependencies reference existing jobs
	for jobName, job := range jm.jobs {
		for _, dep := range job.Depends {
			if _, exists := jm.jobs[dep]; !exists {
				return fmt.Errorf("job '%s' depends on non-existent job '%s'", jobName, dep)
			}
		}
	}

	// Check for cycles using DFS
	return jm.detectCycles()
}

// detectCycles uses DFS to detect cycles in the job dependency graph
func (jm *JobManager) detectCycles() error {
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

	return nil
}

// dfsVisit performs DFS visit for cycle detection
func (jm *JobManager) dfsVisit(jobName string, visitState map[string]int) error {
	visitState[jobName] = 1 // Mark as visiting

	job := jm.jobs[jobName]
	for _, dep := range job.Depends {
		if visitState[dep] == 1 {
			// Found a back edge - cycle detected
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

// RenderToYAML generates the jobs section of a GitHub Actions workflow
func (jm *JobManager) RenderToYAML() string {
	if len(jm.jobs) == 0 {
		return "jobs:\n"
	}

	var yaml strings.Builder
	yaml.WriteString("jobs:\n")

	// Use the insertion order instead of alphabetical sorting
	for _, jobName := range jm.jobOrder {
		job := jm.jobs[jobName]
		yaml.WriteString(jm.renderJob(job))
	}

	return yaml.String()
}

// renderJob renders a single job to YAML
func (jm *JobManager) renderJob(job *Job) string {
	var yaml strings.Builder

	yaml.WriteString(fmt.Sprintf("  %s:\n", job.Name))

	// Add needs clause if there are dependencies
	if len(job.Depends) > 0 {
		if len(job.Depends) == 1 {
			yaml.WriteString(fmt.Sprintf("    needs: %s\n", job.Depends[0]))
		} else {
			yaml.WriteString("    needs:\n")
			for _, dep := range job.Depends {
				yaml.WriteString(fmt.Sprintf("      - %s\n", dep))
			}
		}
	}

	// Add if condition if present
	if job.If != "" {
		yaml.WriteString(fmt.Sprintf("    %s\n", job.If))
	}

	// Add runs-on
	if job.RunsOn != "" {
		yaml.WriteString(fmt.Sprintf("    %s\n", job.RunsOn))
	}

	// Add permissions section
	if job.Permissions != "" {
		yaml.WriteString(fmt.Sprintf("    %s\n", job.Permissions))
	}

	// Add timeout_minutes if specified
	if job.TimeoutMinutes > 0 {
		yaml.WriteString(fmt.Sprintf("    timeout-minutes: %d\n", job.TimeoutMinutes))
	}

	// Add outputs section
	if len(job.Outputs) > 0 {
		yaml.WriteString("    outputs:\n")
		// Sort output keys for consistent output
		outputKeys := make([]string, 0, len(job.Outputs))
		for key := range job.Outputs {
			outputKeys = append(outputKeys, key)
		}
		sort.Strings(outputKeys)

		for _, key := range outputKeys {
			yaml.WriteString(fmt.Sprintf("      %s: %s\n", key, job.Outputs[key]))
		}
	}

	// Add steps section
	if len(job.Steps) > 0 {
		yaml.WriteString("    steps:\n")
		for _, step := range job.Steps {
			// Each step is already formatted with proper indentation
			yaml.WriteString(step)
		}
	}

	// Add newline after each job for proper formatting
	yaml.WriteString("\n")

	return yaml.String()
}

// GetTopologicalOrder returns jobs in topological order (dependencies before dependents)
func (jm *JobManager) GetTopologicalOrder() ([]string, error) {
	// First validate dependencies to ensure no cycles
	if err := jm.ValidateDependencies(); err != nil {
		return nil, err
	}

	// Track in-degree (number of incoming dependencies) for each job
	inDegree := make(map[string]int)
	for jobName := range jm.jobs {
		inDegree[jobName] = 0
	}

	// Calculate in-degrees: count how many dependencies each job has
	for _, job := range jm.jobs {
		inDegree[job.Name] = len(job.Depends)
	}

	// Start with jobs that have no dependencies (in-degree = 0)
	queue := make([]string, 0)
	for jobName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, jobName)
		}
	}

	result := make([]string, 0, len(jm.jobs))

	// Process jobs in topological order
	for len(queue) > 0 {
		// Sort queue for consistent output
		sort.Strings(queue)

		// Take the first job from queue
		currentJob := queue[0]
		queue = queue[1:]
		result = append(result, currentJob)

		// For each job that depends on the current job, reduce its in-degree
		for jobName, job := range jm.jobs {
			for _, dep := range job.Depends {
				if dep == currentJob {
					inDegree[jobName]--
					if inDegree[jobName] == 0 {
						queue = append(queue, jobName)
					}
				}
			}
		}
	}

	return result, nil
}
