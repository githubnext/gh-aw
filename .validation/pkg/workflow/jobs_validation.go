// This file provides validation for GitHub Actions job configurations.
//
// # Job Validation
//
// This file validates that job definitions are correct before workflow compilation,
// catching issues that would cause silent failures or confusing runtime errors.
//
// # Validation Functions
//
//   - ValidateDependencies() - Checks job dependencies exist and contain no cycles
//   - ValidateDuplicateSteps() - Detects duplicate step definitions (compiler bugs)
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - Adding new job-level structural constraints
//   - Adding new dependency graph validation rules

package workflow

import (
	"fmt"
	"strings"
)

// ValidateDependencies checks that all job dependencies exist and there are no cycles
func (jm *JobManager) ValidateDependencies() error {
	jobLog.Printf("Validating dependencies for %d jobs", len(jm.jobs))
	// First check that all dependencies reference existing jobs
	for jobName, job := range jm.jobs {
		for _, dep := range job.Needs {
			if _, exists := jm.jobs[dep]; !exists {
				jobLog.Printf("Validation failed: job %s depends on non-existent job %s", jobName, dep)
				return fmt.Errorf("job '%s' depends on non-existent job '%s'", jobName, dep)
			}
		}
	}

	// Check for cycles using DFS
	return jm.detectCycles()
}

// ValidateDuplicateSteps checks that no job has duplicate steps.
// This detects compiler bugs where the same step is added multiple times.
func (jm *JobManager) ValidateDuplicateSteps() error {
	jobLog.Printf("Validating for duplicate steps in %d jobs", len(jm.jobs))

	for jobName, job := range jm.jobs {
		if len(job.Steps) == 0 {
			continue
		}

		// Track seen steps to detect duplicates
		seen := make(map[string]int)

		for i, step := range job.Steps {
			// job.Steps entries may be either complete step blocks (multi-line) or
			// individual YAML line fragments. Only elements that begin with the step
			// leader "- " represent a new step definition; property lines (e.g.,
			// "continue-on-error:", "name:" inside a "with:" block) start with
			// plain indentation and should not be treated as step definitions.
			if !strings.HasPrefix(strings.TrimSpace(step), "-") {
				continue
			}

			// Extract step name from YAML for comparison
			stepName := extractStepName(step)
			if stepName == "" {
				// Steps without names can't be checked for duplicates
				continue
			}

			if firstIndex, exists := seen[stepName]; exists {
				jobLog.Printf("Duplicate step detected in job '%s': step '%s' at positions %d and %d", jobName, stepName, firstIndex, i)
				return fmt.Errorf("compiler bug: duplicate step '%s' found in job '%s' (positions %d and %d)", stepName, jobName, firstIndex, i)
			}

			seen[stepName] = i
		}
	}

	jobLog.Print("No duplicate steps detected in any job")
	return nil
}
