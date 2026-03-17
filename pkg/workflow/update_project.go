package workflow

import "github.com/github/gh-aw/pkg/logger"

var updateProjectLog = logger.New("workflow:update_project")

// ProjectView defines a project view configuration
type ProjectView struct {
	Name          string `yaml:"name" json:"name"`
	Layout        string `yaml:"layout" json:"layout"`
	Filter        string `yaml:"filter,omitempty" json:"filter,omitempty"`
	VisibleFields []int  `yaml:"visible-fields,omitempty" json:"visible_fields,omitempty"`
	Description   string `yaml:"description,omitempty" json:"description,omitempty"`
}

// ProjectFieldDefinition defines a project custom field configuration
// used by update_project operation=create_fields.
type ProjectFieldDefinition struct {
	Name     string   `yaml:"name" json:"name"`
	DataType string   `yaml:"data-type" json:"data_type"`
	Options  []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// UpdateProjectConfig holds configuration for unified project board management
type UpdateProjectConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string                   `yaml:"github-token,omitempty"`
	Project              string                   `yaml:"project,omitempty"`       // Default project URL for operations
	TargetRepoSlug       string                   `yaml:"target-repo,omitempty"`   // Default repository for cross-repo content resolution in "owner/repo" format
	AllowedRepos         []string                 `yaml:"allowed-repos,omitempty"` // List of additional repositories allowed for target_repo resolution
	Views                []ProjectView            `yaml:"views,omitempty"`
	FieldDefinitions     []ProjectFieldDefinition `yaml:"field-definitions,omitempty" json:"field_definitions,omitempty"`
}

// parseUpdateProjectConfig handles update-project configuration
func (c *Compiler) parseUpdateProjectConfig(outputMap map[string]any) *UpdateProjectConfig {
	if configData, exists := outputMap["update-project"]; exists {
		updateProjectLog.Print("Parsing update-project configuration")
		updateProjectConfig := &UpdateProjectConfig{}
		updateProjectConfig.Max = defaultIntStr(10) // Default max is 10

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

			// Parse project URL override if specified
			if project, exists := configMap["project"]; exists {
				if projectStr, ok := project.(string); ok {
					updateProjectConfig.Project = projectStr
					updateProjectLog.Printf("Using custom project URL for update-project: %s", projectStr)
				}
			}

			// Parse target-repo for cross-repo content resolution (no wildcard allowed)
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil
			}
			updateProjectConfig.TargetRepoSlug = targetRepoSlug

			// Parse allowed-repos for cross-repo content resolution
			updateProjectConfig.AllowedRepos = parseAllowedReposFromConfig(configMap)

			// Parse views if specified
			updateProjectConfig.Views = parseProjectViews(configMap, updateProjectLog)

			// Parse field-definitions if specified
			updateProjectConfig.FieldDefinitions = parseProjectFieldDefinitions(configMap, updateProjectLog)
		}

		updateProjectLog.Printf("Parsed update-project config: max=%d, hasCustomToken=%v, hasCustomProject=%v, targetRepo=%q, allowedReposCount=%d, viewCount=%d, fieldDefinitionCount=%d",
			updateProjectConfig.Max, updateProjectConfig.GitHubToken != "", updateProjectConfig.Project != "", updateProjectConfig.TargetRepoSlug, len(updateProjectConfig.AllowedRepos), len(updateProjectConfig.Views), len(updateProjectConfig.FieldDefinitions))
		return updateProjectConfig
	}
	updateProjectLog.Print("No update-project configuration found")
	return nil
}
