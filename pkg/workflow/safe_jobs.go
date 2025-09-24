package workflow

import "fmt"

// SafeJobConfig defines a safe job configuration with GitHub Actions job properties
type SafeJobConfig struct {
	// Standard GitHub Actions job properties
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

// extractSafeJobsFromFrontmatter extracts safe-jobs section from frontmatter map
func extractSafeJobsFromFrontmatter(frontmatter map[string]any) map[string]*SafeJobConfig {
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
