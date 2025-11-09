package workflow

import (
	"fmt"
)

// CampaignProjectConfig holds configuration for creating and managing GitHub Projects v2 boards for campaigns
type CampaignProjectConfig struct {
	Name         string                       `yaml:"name"`                     // Project name (supports template expressions like {{campaign.id}})
	View         string                       `yaml:"view,omitempty"`           // Project view type: board, table, or roadmap (default: board)
	StatusField  string                       `yaml:"status-field"`             // Name of the status field (default: Status)
	AgentField   string                       `yaml:"agent-field,omitempty"`    // Name of the agent field (default: Agent)
	Fields       map[string]string            `yaml:"fields,omitempty"`         // Simple text fields to add to project items
	CustomFields []CampaignProjectCustomField `yaml:"custom-fields,omitempty"`  // Advanced custom fields for analytics (number, date, select, iteration)
	Insights     []string                     `yaml:"insights,omitempty"`       // Insights to generate: agent-velocity, campaign-progress, bottlenecks
	GitHubToken  string                       `yaml:"github-token,omitempty"`   // GitHub token for project operations
}

// CampaignProjectCustomField defines a custom field for advanced analytics
type CampaignProjectCustomField struct {
	Name        string   `yaml:"name"`                  // Field name (e.g., "Priority", "Story Points", "Sprint")
	Type        string   `yaml:"type"`                  // Field type: number, date, single_select, iteration, text
	Value       string   `yaml:"value,omitempty"`       // Default value or template expression
	Options     []string `yaml:"options,omitempty"`     // Options for single_select fields
	Description string   `yaml:"description,omitempty"` // Field description
}

// parseCampaignProjectConfig handles campaign.project configuration
func (c *Compiler) parseCampaignProjectConfig(campaignMap map[string]any) *CampaignProjectConfig {
	if projectData, exists := campaignMap["project"]; exists {
		projectConfig := &CampaignProjectConfig{}

		if projectMap, ok := projectData.(map[string]any); ok {
			// Parse name (required)
			if name, exists := projectMap["name"]; exists {
				if nameStr, ok := name.(string); ok {
					projectConfig.Name = nameStr
				}
			}

			// Parse view (optional, default: board)
			if view, exists := projectMap["view"]; exists {
				if viewStr, ok := view.(string); ok {
					projectConfig.View = viewStr
				}
			}
			if projectConfig.View == "" {
				projectConfig.View = "board"
			}

			// Parse status-field (optional, default: Status)
			if statusField, exists := projectMap["status-field"]; exists {
				if statusFieldStr, ok := statusField.(string); ok {
					projectConfig.StatusField = statusFieldStr
				}
			}
			if projectConfig.StatusField == "" {
				projectConfig.StatusField = "Status"
			}

			// Parse agent-field (optional, default: Agent)
			if agentField, exists := projectMap["agent-field"]; exists {
				if agentFieldStr, ok := agentField.(string); ok {
					projectConfig.AgentField = agentFieldStr
				}
			}
			if projectConfig.AgentField == "" {
				projectConfig.AgentField = "Agent"
			}

			// Parse fields (optional)
			if fields, exists := projectMap["fields"]; exists {
				if fieldsMap, ok := fields.(map[string]any); ok {
					projectConfig.Fields = make(map[string]string)
					for key, value := range fieldsMap {
						if valueStr, ok := value.(string); ok {
							projectConfig.Fields[key] = valueStr
						}
					}
				}
			}

			// Parse insights (optional)
			if insights, exists := projectMap["insights"]; exists {
				if insightsArray, ok := insights.([]any); ok {
					for _, insight := range insightsArray {
						if insightStr, ok := insight.(string); ok {
							projectConfig.Insights = append(projectConfig.Insights, insightStr)
						}
					}
				}
			}

			// Parse custom-fields (optional)
			if customFields, exists := projectMap["custom-fields"]; exists {
				if customFieldsArray, ok := customFields.([]any); ok {
					for _, field := range customFieldsArray {
						if fieldMap, ok := field.(map[string]any); ok {
							customField := CampaignProjectCustomField{}

							if name, exists := fieldMap["name"]; exists {
								if nameStr, ok := name.(string); ok {
									customField.Name = nameStr
								}
							}

							if fieldType, exists := fieldMap["type"]; exists {
								if typeStr, ok := fieldType.(string); ok {
									customField.Type = typeStr
								}
							}

							if value, exists := fieldMap["value"]; exists {
								if valueStr, ok := value.(string); ok {
									customField.Value = valueStr
								}
							}

							if description, exists := fieldMap["description"]; exists {
								if descStr, ok := description.(string); ok {
									customField.Description = descStr
								}
							}

							if options, exists := fieldMap["options"]; exists {
								if optionsArray, ok := options.([]any); ok {
									for _, opt := range optionsArray {
										if optStr, ok := opt.(string); ok {
											customField.Options = append(customField.Options, optStr)
										}
									}
								}
							}

							// Only add if name and type are set
							if customField.Name != "" && customField.Type != "" {
								projectConfig.CustomFields = append(projectConfig.CustomFields, customField)
							}
						}
					}
				}
			}

			// Parse github-token (optional)
			if githubToken, exists := projectMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					projectConfig.GitHubToken = githubTokenStr
				}
			}
		}

		// Return nil if name is not set (invalid configuration)
		if projectConfig.Name == "" {
			return nil
		}

		return projectConfig
	}

	return nil
}

// buildCampaignProjectJob creates the campaign project management job
func (c *Compiler) buildCampaignProjectJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.CampaignProject == nil {
		return nil, fmt.Errorf("campaign.project configuration is required")
	}

	// Build custom environment variables specific to campaign project
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_NAME: %q\n", data.CampaignProject.Name))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_VIEW: %q\n", data.CampaignProject.View))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_STATUS_FIELD: %q\n", data.CampaignProject.StatusField))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_AGENT_FIELD: %q\n", data.CampaignProject.AgentField))

	// Add custom fields as JSON
	if len(data.CampaignProject.Fields) > 0 {
		fieldsJSON := "{"
		first := true
		for key, value := range data.CampaignProject.Fields {
			if !first {
				fieldsJSON += ","
			}
			fieldsJSON += fmt.Sprintf("%q:%q", key, value)
			first = false
		}
		fieldsJSON += "}"
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_FIELDS: %q\n", fieldsJSON))
	}

	// Add insights configuration
	if len(data.CampaignProject.Insights) > 0 {
		insightsStr := ""
		for i, insight := range data.CampaignProject.Insights {
			if i > 0 {
				insightsStr += ","
			}
			insightsStr += insight
		}
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_INSIGHTS: %q\n", insightsStr))
	}

	// Add custom fields configuration as JSON
	if len(data.CampaignProject.CustomFields) > 0 {
		customFieldsJSON := "["
		for i, field := range data.CampaignProject.CustomFields {
			if i > 0 {
				customFieldsJSON += ","
			}
			customFieldsJSON += "{"
			customFieldsJSON += fmt.Sprintf("%q:%q", "name", field.Name)
			customFieldsJSON += fmt.Sprintf(",%q:%q", "type", field.Type)
			if field.Value != "" {
				customFieldsJSON += fmt.Sprintf(",%q:%q", "value", field.Value)
			}
			if field.Description != "" {
				customFieldsJSON += fmt.Sprintf(",%q:%q", "description", field.Description)
			}
			if len(field.Options) > 0 {
				customFieldsJSON += fmt.Sprintf(",%q:[", "options")
				for j, opt := range field.Options {
					if j > 0 {
						customFieldsJSON += ","
					}
					customFieldsJSON += fmt.Sprintf("%q", opt)
				}
				customFieldsJSON += "]"
			}
			customFieldsJSON += "}"
		}
		customFieldsJSON += "]"
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_CUSTOM_FIELDS: %q\n", customFieldsJSON))
	}

	// Get token from config
	token := data.CampaignProject.GitHubToken

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Manage Campaign Project",
		StepID:        "campaign_project",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getCampaignProjectScript(),
		Token:         token,
	})

	outputs := map[string]string{
		"project_number": "${{ steps.campaign_project.outputs.project_number }}",
		"project_url":    "${{ steps.campaign_project.outputs.project_url }}",
		"item_id":        "${{ steps.campaign_project.outputs.item_id }}",
		"item_count":     "${{ steps.campaign_project.outputs.item_count }}",
		"issue_count":    "${{ steps.campaign_project.outputs.issue_count }}",
	}

	job := &Job{
		Name:           "campaign_project",
		If:             "always()", // Always run to update project status
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadProjectsWrite().RenderToYAML(),
		TimeoutMinutes: 10,
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName},
	}

	return job, nil
}
