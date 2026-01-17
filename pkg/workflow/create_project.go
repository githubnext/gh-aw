package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var createProjectLog = logger.New("workflow:create_project")

// CreateProjectsConfig holds configuration for creating GitHub Projects V2
type CreateProjectsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string        `yaml:"github-token,omitempty"`
	TargetOwner          string        `yaml:"target-owner,omitempty"` // Default target owner (org/user) for the new project
	TitlePrefix          string        `yaml:"title-prefix,omitempty"` // Default prefix for auto-generated project titles
	Views                []ProjectView `yaml:"views,omitempty"`        // Project views to create automatically after project creation
}

// parseCreateProjectsConfig handles create-project configuration
func (c *Compiler) parseCreateProjectsConfig(outputMap map[string]any) *CreateProjectsConfig {
	if configData, exists := outputMap["create-project"]; exists {
		createProjectLog.Print("Parsing create-project configuration")
		createProjectsConfig := &CreateProjectsConfig{}
		createProjectsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &createProjectsConfig.BaseSafeOutputConfig, 1)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					createProjectsConfig.GitHubToken = tokenStr
					createProjectLog.Print("Using custom GitHub token for create-project")
				}
			}

			// Parse target-owner if specified
			if targetOwner, exists := configMap["target-owner"]; exists {
				if targetOwnerStr, ok := targetOwner.(string); ok {
					createProjectsConfig.TargetOwner = targetOwnerStr
					createProjectLog.Printf("Default target owner configured: %s", targetOwnerStr)
				}
			}

			// Parse title-prefix if specified
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					createProjectsConfig.TitlePrefix = titlePrefixStr
					createProjectLog.Printf("Title prefix configured: %s", titlePrefixStr)
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
								createProjectsConfig.Views = append(createProjectsConfig.Views, view)
								createProjectLog.Printf("Parsed view %d: %s (%s)", i+1, view.Name, view.Layout)
							} else {
								createProjectLog.Printf("Skipping invalid view %d: missing required fields", i+1)
							}
						}
					}
				}
			}
		}

		createProjectLog.Printf("Parsed create-project config: max=%d, hasCustomToken=%v, hasTargetOwner=%v, hasTitlePrefix=%v, viewCount=%d",
			createProjectsConfig.Max, createProjectsConfig.GitHubToken != "", createProjectsConfig.TargetOwner != "", createProjectsConfig.TitlePrefix != "", len(createProjectsConfig.Views))
		return createProjectsConfig
	}
	createProjectLog.Print("No create-project configuration found")
	return nil
}
