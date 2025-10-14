package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// SafeJobConfig defines a safe job configuration with GitHub Actions job properties
type SafeJobConfig struct {
	// Standard GitHub Actions job properties
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	RunsOn      any               `yaml:"runs-on,omitempty"`
	If          string            `yaml:"if,omitempty"`
	Needs       []string          `yaml:"needs,omitempty"`
	Steps       []any             `yaml:"steps,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Permissions map[string]string `yaml:"permissions,omitempty"`

	// Additional safe-job specific properties
	Inputs      map[string]*SafeJobInput `yaml:"inputs,omitempty"`
	GitHubToken string                   `yaml:"github-token,omitempty"`
	Output      string                   `yaml:"output,omitempty"`
}

// SafeJobInput defines an input parameter for a safe job, using workflow_dispatch syntax
type SafeJobInput struct {
	Description string   `yaml:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty"`
	Default     string   `yaml:"default,omitempty"`
	Type        string   `yaml:"type,omitempty"`
	Options     []string `yaml:"options,omitempty"`
}

// HasSafeJobsEnabled checks if any safe-jobs are enabled at the top level
func HasSafeJobsEnabled(safeJobs map[string]*SafeJobConfig) bool {
	return len(safeJobs) > 0
}

// parseSafeJobsConfig parses the safe-jobs configuration from top-level frontmatter
func (c *Compiler) parseSafeJobsConfig(frontmatter map[string]any) map[string]*SafeJobConfig {
	safeJobsSection, exists := frontmatter["safe-jobs"]
	if !exists {
		return nil
	}

	safeJobsMap, ok := safeJobsSection.(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]*SafeJobConfig)

	for jobName, jobValue := range safeJobsMap {
		jobConfig, ok := jobValue.(map[string]any)
		if !ok {
			continue
		}

		safeJob := &SafeJobConfig{}

		// Parse name
		if name, exists := jobConfig["name"]; exists {
			if nameStr, ok := name.(string); ok {
				safeJob.Name = nameStr
			}
		}

		// Parse description
		if description, exists := jobConfig["description"]; exists {
			if descStr, ok := description.(string); ok {
				safeJob.Description = descStr
			}
		}

		// Parse runs-on
		if runsOn, exists := jobConfig["runs-on"]; exists {
			safeJob.RunsOn = runsOn
		}

		// Parse if condition
		if ifCond, exists := jobConfig["if"]; exists {
			if ifStr, ok := ifCond.(string); ok {
				safeJob.If = c.extractExpressionFromIfString(ifStr)
			}
		}

		// Parse needs
		if needs, exists := jobConfig["needs"]; exists {
			if needsList, ok := needs.([]any); ok {
				for _, need := range needsList {
					if needStr, ok := need.(string); ok {
						safeJob.Needs = append(safeJob.Needs, needStr)
					}
				}
			} else if needStr, ok := needs.(string); ok {
				safeJob.Needs = append(safeJob.Needs, needStr)
			}
		}

		// Parse steps
		if steps, exists := jobConfig["steps"]; exists {
			if stepsList, ok := steps.([]any); ok {
				safeJob.Steps = stepsList
			}
		}

		// Parse env
		if env, exists := jobConfig["env"]; exists {
			if envMap, ok := env.(map[string]any); ok {
				safeJob.Env = make(map[string]string)
				for key, value := range envMap {
					if valueStr, ok := value.(string); ok {
						safeJob.Env[key] = valueStr
					}
				}
			}
		}

		// Parse permissions
		if permissions, exists := jobConfig["permissions"]; exists {
			if permMap, ok := permissions.(map[string]any); ok {
				safeJob.Permissions = make(map[string]string)
				for key, value := range permMap {
					if valueStr, ok := value.(string); ok {
						safeJob.Permissions[key] = valueStr
					}
				}
			}
		}

		// Parse github-token
		if token, exists := jobConfig["github-token"]; exists {
			if tokenStr, ok := token.(string); ok {
				safeJob.GitHubToken = tokenStr
			}
		}

		// Parse output
		if output, exists := jobConfig["output"]; exists {
			if outputStr, ok := output.(string); ok {
				safeJob.Output = outputStr
			}
		}

		// Parse inputs
		if inputs, exists := jobConfig["inputs"]; exists {
			if inputsMap, ok := inputs.(map[string]any); ok {
				safeJob.Inputs = make(map[string]*SafeJobInput)
				for inputName, inputValue := range inputsMap {
					if inputConfig, ok := inputValue.(map[string]any); ok {
						input := &SafeJobInput{}

						if desc, exists := inputConfig["description"]; exists {
							if descStr, ok := desc.(string); ok {
								input.Description = descStr
							}
						}

						if req, exists := inputConfig["required"]; exists {
							if reqBool, ok := req.(bool); ok {
								input.Required = reqBool
							}
						}

						if def, exists := inputConfig["default"]; exists {
							if defStr, ok := def.(string); ok {
								input.Default = defStr
							}
						}

						if typ, exists := inputConfig["type"]; exists {
							if typStr, ok := typ.(string); ok {
								input.Type = typStr
							}
						}

						if opts, exists := inputConfig["options"]; exists {
							if optsList, ok := opts.([]any); ok {
								for _, opt := range optsList {
									if optStr, ok := opt.(string); ok {
										input.Options = append(input.Options, optStr)
									}
								}
							}
						}

						safeJob.Inputs[inputName] = input
					}
				}
			}
		}

		result[jobName] = safeJob
	}

	return result
}

// buildSafeJobs creates custom safe-output jobs defined in SafeOutputs.Jobs
func (c *Compiler) buildSafeJobs(data *WorkflowData, threatDetectionEnabled bool) error {
	if data.SafeOutputs == nil || len(data.SafeOutputs.Jobs) == 0 {
		return nil
	}

	for jobName, jobConfig := range data.SafeOutputs.Jobs {
		// Normalize job name to use underscores for consistency
		normalizedJobName := normalizeSafeOutputIdentifier(jobName)

		job := &Job{
			Name: normalizedJobName,
		}

		// Set custom job name if specified
		if jobConfig.Name != "" {
			job.DisplayName = jobConfig.Name
		}

		// Safe-jobs should depend on agent job (always) AND detection job (if enabled)
		job.Needs = append(job.Needs, constants.AgentJobName)
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
		}

		// Add any additional dependencies from the config
		job.Needs = append(job.Needs, jobConfig.Needs...)

		// Set runs-on
		if jobConfig.RunsOn != nil {
			if runsOnStr, ok := jobConfig.RunsOn.(string); ok {
				job.RunsOn = fmt.Sprintf("runs-on: %s", runsOnStr)
			} else if runsOnList, ok := jobConfig.RunsOn.([]any); ok {
				// Handle array format
				var runsOnItems []string
				for _, item := range runsOnList {
					if itemStr, ok := item.(string); ok {
						runsOnItems = append(runsOnItems, fmt.Sprintf("      - %s", itemStr))
					}
				}
				if len(runsOnItems) > 0 {
					job.RunsOn = fmt.Sprintf("runs-on:\n%s", strings.Join(runsOnItems, "\n"))
				}
			}
		} else {
			job.RunsOn = "runs-on: ubuntu-latest" // Default
		}

		// Set if condition - combine safe output type check with user-provided condition
		// Custom safe jobs should only run if the agent output contains the job name (tool call)
		// Use normalized job name to match the underscore format in output_types
		safeOutputCondition := BuildSafeOutputType(normalizedJobName, 0) // min=0 means check for the tool in output_types

		if jobConfig.If != "" {
			// If user provided a custom condition, combine it with the safe output type check
			userConditionStr := c.extractExpressionFromIfString(jobConfig.If)
			userCondition := &ExpressionNode{Expression: userConditionStr}
			job.If = buildAnd(safeOutputCondition, userCondition).Render()
		} else {
			// Otherwise, just use the safe output type check
			job.If = safeOutputCondition.Render()
		}

		// Build job steps
		var steps []string

		// Add step to download agent output artifact
		steps = append(steps, "      - name: Download agent output artifact\n")
		steps = append(steps, "        continue-on-error: true\n")
		steps = append(steps, "        uses: actions/download-artifact@v5\n")
		steps = append(steps, "        with:\n")
		steps = append(steps, fmt.Sprintf("          name: %s\n", constants.AgentOutputArtifactName))
		steps = append(steps, "          path: /tmp/gh-aw/sh-aw/safe-jobs/\n")

		// Add environment variables step
		steps = append(steps, "      - name: Setup Safe Job Environment Variables\n")
		steps = append(steps, "        run: |\n")
		steps = append(steps, "          echo \"Setting up environment for safe job\"\n")

		// Configure GITHUB_AW_AGENT_OUTPUT to point to downloaded artifact file
		steps = append(steps, fmt.Sprintf("          echo \"GITHUB_AW_AGENT_OUTPUT=/tmp/gh-aw/safe-jobs/%s\" >> $GITHUB_ENV\n", constants.AgentOutputArtifactName))

		// Add job-specific environment variables
		if jobConfig.Env != nil {
			for key, value := range jobConfig.Env {
				steps = append(steps, fmt.Sprintf("          echo \"%s=%s\" >> $GITHUB_ENV\n", key, value))
			}
		}

		// Add custom steps from the job configuration
		if len(jobConfig.Steps) > 0 {
			for _, step := range jobConfig.Steps {
				if stepMap, ok := step.(map[string]any); ok {
					stepYAML, err := c.convertStepToYAML(stepMap)
					if err != nil {
						return fmt.Errorf("failed to convert step to YAML for safe job %s: %w", jobName, err)
					}
					steps = append(steps, stepYAML)
				}
			}
		}

		job.Steps = steps

		// Set permissions if specified
		if len(jobConfig.Permissions) > 0 {
			var perms []string
			for perm, level := range jobConfig.Permissions {
				perms = append(perms, fmt.Sprintf("      %s: %s", perm, level))
			}
			job.Permissions = fmt.Sprintf("permissions:\n%s", strings.Join(perms, "\n"))
		}

		// Add the job to the job manager
		if err := c.jobManager.AddJob(job); err != nil {
			return fmt.Errorf("failed to add safe job %s: %w", jobName, err)
		}
	}

	return nil
}

// extractSafeJobsFromFrontmatter extracts safe-jobs section from frontmatter map
// First checks the new location under safe-outputs.jobs, then falls back to old location safe-jobs (for backwards compatibility during transition)
func extractSafeJobsFromFrontmatter(frontmatter map[string]any) map[string]*SafeJobConfig {
	// Check new location: safe-outputs.jobs
	if safeOutputs, exists := frontmatter["safe-outputs"]; exists {
		if safeOutputsMap, ok := safeOutputs.(map[string]any); ok {
			if jobs, exists := safeOutputsMap["jobs"]; exists {
				if jobsMap, ok := jobs.(map[string]any); ok {
					c := &Compiler{} // Create a temporary compiler instance for parsing
					frontmatterCopy := map[string]any{"safe-jobs": jobsMap}
					return c.parseSafeJobsConfig(frontmatterCopy)
				}
			}
		}
	}

	// Fallback to old location: safe-jobs (for backwards compatibility)
	safeJobs, exists := frontmatter["safe-jobs"]
	if !exists {
		return make(map[string]*SafeJobConfig)
	}

	if safeJobsMap, ok := safeJobs.(map[string]any); ok {
		c := &Compiler{} // Create a temporary compiler instance for parsing
		frontmatterCopy := map[string]any{"safe-jobs": safeJobsMap}
		return c.parseSafeJobsConfig(frontmatterCopy)
	}

	return make(map[string]*SafeJobConfig)
}

// mergeSafeJobs merges safe-jobs from multiple sources and detects name conflicts
func mergeSafeJobs(base map[string]*SafeJobConfig, additional map[string]*SafeJobConfig) (map[string]*SafeJobConfig, error) {
	if additional == nil {
		return base, nil
	}

	if base == nil {
		base = make(map[string]*SafeJobConfig)
	}

	result := make(map[string]*SafeJobConfig)

	// Copy base safe-jobs
	for name, config := range base {
		result[name] = config
	}

	// Add additional safe-jobs, checking for conflicts
	for name, config := range additional {
		if _, exists := result[name]; exists {
			return nil, fmt.Errorf("safe-job name conflict: '%s' is defined in both main workflow and included files", name)
		}
		result[name] = config
	}

	return result, nil
}
