package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var updateProjectLog = logger.New("workflow:update_project")

// ProjectView defines a project view configuration
type ProjectView struct {
	Name          string `yaml:"name" json:"name"`
	Layout        string `yaml:"layout" json:"layout"`
	Filter        string `yaml:"filter,omitempty" json:"filter,omitempty"`
	VisibleFields []int  `yaml:"visible-fields,omitempty" json:"visible_fields,omitempty"`
	Description   string `yaml:"description,omitempty" json:"description,omitempty"`
}

// UpdateProjectConfig holds configuration for unified project board management
type UpdateProjectConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string        `yaml:"github-token,omitempty"`
	Views                []ProjectView `yaml:"views,omitempty"`
}

// parseUpdateProjectConfig handles update-project configuration
func (c *Compiler) parseUpdateProjectConfig(outputMap map[string]any) *UpdateProjectConfig {
	if configData, exists := outputMap["update-project"]; exists {
		updateProjectLog.Print("Parsing update-project configuration")
		updateProjectConfig := &UpdateProjectConfig{}
		updateProjectConfig.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &updateProjectConfig.BaseSafeOutputConfig, 10)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					updateProjectConfig.GitHubToken = tokenStr
					updateProjectLog.Print("Using custom GitHub token for update-project")
				}
			}

			// Parse views if specified
			if viewsData, exists := configMap["views"]; exists {
				if viewsList, ok := viewsData.([]any); ok {
					for i, viewItem := range viewsList {
						if viewMap, ok := viewItem.(map[string]any); ok {
							view := ProjectView{}

							// Parse name (required)
							if name, exists := viewMap["name"]; exists {
								if nameStr, ok := name.(string); ok {
									view.Name = nameStr
								}
							}

							// Parse layout (required)
							if layout, exists := viewMap["layout"]; exists {
								if layoutStr, ok := layout.(string); ok {
									view.Layout = layoutStr
								}
							}

							// Parse filter (optional)
							if filter, exists := viewMap["filter"]; exists {
								if filterStr, ok := filter.(string); ok {
									view.Filter = filterStr
								}
							}

							// Parse visible-fields (optional)
							if visibleFields, exists := viewMap["visible-fields"]; exists {
								if fieldsList, ok := visibleFields.([]any); ok {
									for _, field := range fieldsList {
										if fieldInt, ok := field.(int); ok {
											view.VisibleFields = append(view.VisibleFields, fieldInt)
										}
									}
								}
							}

							// Parse description (optional)
							if description, exists := viewMap["description"]; exists {
								if descStr, ok := description.(string); ok {
									view.Description = descStr
								}
							}

							// Only add view if it has required fields
							if view.Name != "" && view.Layout != "" {
								updateProjectConfig.Views = append(updateProjectConfig.Views, view)
								updateProjectLog.Printf("Parsed view %d: %s (%s)", i+1, view.Name, view.Layout)
							} else {
								updateProjectLog.Printf("Skipping invalid view %d: missing required fields", i+1)
							}
						}
					}
				}
			}
		}

		updateProjectLog.Printf("Parsed update-project config: max=%d, hasCustomToken=%v, viewCount=%d",
			updateProjectConfig.Max, updateProjectConfig.GitHubToken != "", len(updateProjectConfig.Views))
		return updateProjectConfig
	}
	updateProjectLog.Print("No update-project configuration found")
	return nil
}
