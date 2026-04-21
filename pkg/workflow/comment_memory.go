package workflow

import "github.com/github/gh-aw/pkg/logger"

var commentMemoryLog = logger.New("workflow:comment_memory")

// CommentMemoryConfig holds configuration for the comment_memory safe output type.
type CommentMemoryConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string   `yaml:"target,omitempty"`        // Target: "triggering" (default), "*" or explicit issue/PR number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in owner/repo format
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // Additional allowed repositories
	MemoryID             string   `yaml:"memory-id,omitempty"`     // Default memory identifier when item does not provide memory_id
	Footer               *string  `yaml:"footer,omitempty"`        // Footer visibility control ("true"/"false" templatable string); nil defaults to visible footer
}

// parseCommentMemoryConfig handles comment-memory configuration.
func (c *Compiler) parseCommentMemoryConfig(outputMap map[string]any) *CommentMemoryConfig {
	// Support explicit configuration if present.
	if _, exists := outputMap["comment-memory"]; !exists {
		return nil
	}

	commentMemoryLog.Print("Parsing comment-memory configuration")

	configData, _ := outputMap["comment-memory"].(map[string]any)
	if err := preprocessIntFieldAsString(configData, "max", commentMemoryLog); err != nil {
		commentMemoryLog.Printf("Invalid max value: %v", err)
		return nil
	}
	if err := preprocessBoolFieldAsString(configData, "footer", commentMemoryLog); err != nil {
		commentMemoryLog.Printf("Invalid footer value: %v", err)
		return nil
	}

	var config CommentMemoryConfig
	if err := unmarshalConfig(outputMap, "comment-memory", &config, commentMemoryLog); err != nil {
		commentMemoryLog.Printf("Failed to unmarshal config: %v", err)
		config = CommentMemoryConfig{}
	}

	if config.Max == nil {
		config.Max = defaultIntStr(1)
	}
	if config.MemoryID == "" {
		config.MemoryID = "default"
	}

	return &config
}
