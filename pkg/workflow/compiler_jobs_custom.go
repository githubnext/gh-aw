package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
)

// This file contains functions for building custom jobs from frontmatter configuration.
// Custom jobs can be either reusable workflow calls or regular jobs with steps.

// extractJobsFromFrontmatter extracts job configuration from frontmatter.
// This uses the structured extraction helper for consistency.
func (c *Compiler) extractJobsFromFrontmatter(frontmatter map[string]any) map[string]any {
	return ExtractMapField(frontmatter, "jobs")
}

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section.
// Custom jobs can be:
// - Reusable workflow calls (with uses, with, and secrets)
// - Regular jobs with steps
// Jobs without explicit needs dependencies automatically depend on the activation job.
func (c *Compiler) buildCustomJobs(data *WorkflowData, activationJobCreated bool) error {
	compilerJobsLog.Printf("Building %d custom jobs", len(data.Jobs))
	for jobName, jobConfig := range data.Jobs {
		if configMap, ok := jobConfig.(map[string]any); ok {
			job := &Job{
				Name: jobName,
			}

			// Extract job dependencies
			hasExplicitNeeds := false
			if needs, hasNeeds := configMap["needs"]; hasNeeds {
				hasExplicitNeeds = true
				if needsList, ok := needs.([]any); ok {
					for _, need := range needsList {
						if needStr, ok := need.(string); ok {
							job.Needs = append(job.Needs, needStr)
						}
					}
				} else if needStr, ok := needs.(string); ok {
					// Single dependency as string
					job.Needs = append(job.Needs, needStr)
				}
			}

			// If no explicit needs and activation job exists, automatically add activation as dependency
			// This ensures custom jobs wait for workflow validation before executing
			if !hasExplicitNeeds && activationJobCreated {
				job.Needs = append(job.Needs, constants.ActivationJobName)
				compilerJobsLog.Printf("Added automatic dependency: custom job '%s' now depends on '%s'", jobName, constants.ActivationJobName)
			}

			// Extract other job properties
			if runsOn, hasRunsOn := configMap["runs-on"]; hasRunsOn {
				if runsOnStr, ok := runsOn.(string); ok {
					job.RunsOn = fmt.Sprintf("runs-on: %s", runsOnStr)
				}
			}

			if ifCond, hasIf := configMap["if"]; hasIf {
				if ifStr, ok := ifCond.(string); ok {
					job.If = c.extractExpressionFromIfString(ifStr)
				}
			}

			// Extract permissions
			if permissions, hasPermissions := configMap["permissions"]; hasPermissions {
				if permsMap, ok := permissions.(map[string]any); ok {
					// Use gopkg.in/yaml.v3 to marshal permissions
					yamlBytes, err := yaml.Marshal(permsMap)
					if err != nil {
						return fmt.Errorf("failed to convert permissions to YAML for job '%s': %w", jobName, err)
					}
					// Indent the YAML properly for job-level permissions
					permsYAML := string(yamlBytes)
					lines := strings.Split(strings.TrimSpace(permsYAML), "\n")
					var formattedPerms strings.Builder
					formattedPerms.WriteString("permissions:\n")
					for _, line := range lines {
						formattedPerms.WriteString("      " + line + "\n")
					}
					job.Permissions = formattedPerms.String()
				}
			}

			// Extract outputs for custom jobs
			if outputs, hasOutputs := configMap["outputs"]; hasOutputs {
				if outputsMap, ok := outputs.(map[string]any); ok {
					job.Outputs = make(map[string]string)
					for key, val := range outputsMap {
						if valStr, ok := val.(string); ok {
							job.Outputs[key] = valStr
						} else {
							compilerJobsLog.Printf("Warning: output '%s' in job '%s' has non-string value (type: %T), ignoring", key, jobName, val)
						}
					}
				}
			}

			// Check if this is a reusable workflow call
			if uses, hasUses := configMap["uses"]; hasUses {
				if usesStr, ok := uses.(string); ok {
					compilerJobsLog.Printf("Custom job '%s' is a reusable workflow call: %s", jobName, usesStr)
					job.Uses = usesStr

					// Extract with parameters for reusable workflow
					if with, hasWith := configMap["with"]; hasWith {
						if withMap, ok := with.(map[string]any); ok {
							job.With = withMap
						}
					}

					// Extract secrets for reusable workflow
					if secrets, hasSecrets := configMap["secrets"]; hasSecrets {
						if secretsMap, ok := secrets.(map[string]any); ok {
							job.Secrets = make(map[string]string)
							for key, val := range secretsMap {
								if valStr, ok := val.(string); ok {
									job.Secrets[key] = valStr
								}
							}
						}
					}
				}
			} else {
				// Add basic steps if specified (only for non-reusable workflow jobs)
				if steps, hasSteps := configMap["steps"]; hasSteps {
					if stepsList, ok := steps.([]any); ok {
						for _, step := range stepsList {
							if stepMap, ok := step.(map[string]any); ok {
								// Apply action pinning before converting to YAML
								stepMap = ApplyActionPinToStep(stepMap, data)

								stepYAML, err := c.convertStepToYAML(stepMap)
								if err != nil {
									return fmt.Errorf("failed to convert step to YAML for job '%s': %w", jobName, err)
								}
								job.Steps = append(job.Steps, stepYAML)
							}
						}
					}
				}
			}

			if err := c.jobManager.AddJob(job); err != nil {
				return fmt.Errorf("failed to add custom job '%s': %w", jobName, err)
			}
			compilerJobsLog.Printf("Successfully added custom job '%s' with %d needs dependencies", jobName, len(job.Needs))
		}
	}

	compilerJobsLog.Print("Completed building all custom jobs")
	return nil
}
