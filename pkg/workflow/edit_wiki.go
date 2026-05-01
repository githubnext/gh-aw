package workflow

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/logger"
)

var editWikiLog = logger.New("workflow:edit_wiki")

// EditWikiConfig holds configuration for pushing changes to a repository's wiki
type EditWikiConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TargetRepoSlug       string   `yaml:"repo,omitempty"`                // Target repository in format "owner/repo". Defaults to the current repository.
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"`       // List of repositories in format "owner/repo" that the wiki edit can target
	IfNoChanges          string   `yaml:"if-no-changes,omitempty"`       // Behavior when no changes to push: "warn", "error", or "ignore" (default: "warn")
	CommitTitleSuffix    string   `yaml:"commit-title-suffix,omitempty"` // Optional suffix to append to generated commit titles
}

// parseEditWikiConfig handles edit-wiki configuration
func (c *Compiler) parseEditWikiConfig(outputMap map[string]any) *EditWikiConfig {
	if configData, exists := outputMap["edit-wiki"]; exists {
		editWikiLog.Print("Parsing edit-wiki configuration")
		editWikiConfig := &EditWikiConfig{
			IfNoChanges: "warn", // Default behavior: warn when no changes
		}

		// Handle the case where configData is nil (edit-wiki: with no value)
		if configData == nil {
			return editWikiConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse repo (optional, defaults to current repository)
			if repo, exists := configMap["repo"]; exists {
				if repoStr, ok := repo.(string); ok {
					editWikiConfig.TargetRepoSlug = repoStr
				}
			}

			// Parse allowed-repos (expression-aware)
			editWikiConfig.AllowedRepos = ParseStringArrayOrExprFromConfig(configMap, "allowed-repos", editWikiLog)

			// Parse if-no-changes (optional, defaults to "warn")
			if ifNoChanges, exists := configMap["if-no-changes"]; exists {
				if ifNoChangesStr, ok := ifNoChanges.(string); ok {
					switch ifNoChangesStr {
					case "warn", "error", "ignore":
						editWikiConfig.IfNoChanges = ifNoChangesStr
					default:
						if c.verbose {
							fmt.Fprintf(os.Stderr, "Warning: invalid if-no-changes value '%s' for edit-wiki, using default 'warn'\n", ifNoChangesStr)
						}
						editWikiConfig.IfNoChanges = "warn"
					}
				}
			}

			// Parse commit-title-suffix (optional)
			if commitTitleSuffix, exists := configMap["commit-title-suffix"]; exists {
				if commitTitleSuffixStr, ok := commitTitleSuffix.(string); ok {
					editWikiConfig.CommitTitleSuffix = commitTitleSuffixStr
				}
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &editWikiConfig.BaseSafeOutputConfig, 1)
		}

		return editWikiConfig
	}

	return nil
}
