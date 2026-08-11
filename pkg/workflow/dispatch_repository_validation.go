package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var dispatchRepositoryValidationLog = logger.New("workflow:dispatch_repository_validation")

// repoSlugPattern matches a valid owner/repo GitHub repository slug.
// Owner names: alphanumerics and hyphens (no dots - GitHub usernames/org names cannot have dots).
// Repository names: alphanumerics, hyphens, dots and underscores.
var repoSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9._-]+$`)

// validateDispatchRepository validates that the dispatch_repository configuration is correct.
func (c *Compiler) validateDispatchRepository(data *WorkflowData, workflowPath string) error {
	dispatchRepositoryValidationLog.Print("Starting dispatch_repository validation")

	if data.SafeOutputs == nil || data.SafeOutputs.DispatchRepository == nil {
		dispatchRepositoryValidationLog.Print("No dispatch_repository configuration found")
		return nil
	}

	config := data.SafeOutputs.DispatchRepository

	if len(config.Tools) == 0 {
		return errors.New("dispatch_repository configuration has no tools and must specify at least one dispatch tool. Expected at least one tool under safe-outputs.dispatch_repository. Example configuration in workflow frontmatter. Example:\nsafe-outputs:\n  dispatch_repository:\n    trigger_ci:\n      description: Trigger CI in another repository\n      workflow: ci.yml\n      event_type: ci_trigger\n      repository: org/target-repo")
	}

	collector := NewErrorCollector(c.failFast)

	for toolKey, tool := range config.Tools {
		dispatchRepositoryValidationLog.Printf("Validating dispatch_repository tool: %s", toolKey)

		// Validate workflow field is present
		if strings.TrimSpace(tool.Workflow) == "" {
			workflowErr := fmt.Errorf("dispatch_repository: tool %q must specify a 'workflow' field (target workflow name for traceability)\n\nExample:\n  dispatch_repository:\n    %s:\n      workflow: ci.yml\n      event_type: ci_trigger\n      repository: org/target-repo", toolKey, toolKey)
			if returnErr := collector.Add(workflowErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		// Validate event_type field is present
		if strings.TrimSpace(tool.EventType) == "" {
			eventTypeErr := fmt.Errorf("dispatch_repository: tool %q must specify an 'event_type' field\n\nExample:\n  dispatch_repository:\n    %s:\n      workflow: %s\n      event_type: my_event\n      repository: org/target-repo", toolKey, toolKey, tool.Workflow)
			if returnErr := collector.Add(eventTypeErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		// Validate that at least one repository target is specified
		hasRepository := strings.TrimSpace(tool.Repository) != ""
		hasAllowedRepos := len(tool.AllowedRepositories) > 0

		if !hasRepository && !hasAllowedRepos {
			repoErr := fmt.Errorf("dispatch_repository tool %q has no repository target. Expected either 'repository' or 'allowed_repositories'. Example:\n  dispatch_repository:\n    %s:\n      workflow: %s\n      event_type: %s\n      repository: org/target-repo\nAlternative Example:\n  dispatch_repository:\n    %s:\n      workflow: %s\n      event_type: %s\n      allowed_repositories:\n        - org/repo1\n        - org/repo2", toolKey, toolKey, tool.Workflow, tool.EventType, toolKey, tool.Workflow, tool.EventType)
			if returnErr := collector.Add(repoErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		// Validate single repository format (skip if it looks like a GitHub Actions expression)
		if hasRepository && !hasExpressionMarker(tool.Repository) {
			if !repoSlugPattern.MatchString(tool.Repository) {
				repoFmtErr := fmt.Errorf("dispatch_repository tool %q has invalid repository value %q in an unsupported format. Expected 'owner/repo'. Example: repository: github/gh-aw", toolKey, tool.Repository)
				if returnErr := collector.Add(repoFmtErr); returnErr != nil {
					return returnErr
				}
			}
		}

		// Validate allowed_repositories format
		for _, repo := range tool.AllowedRepositories {
			if hasExpressionMarker(repo) {
				continue
			}
			// Allow glob patterns like "org/*"
			if strings.Contains(repo, "*") {
				continue
			}
			if !repoSlugPattern.MatchString(repo) {
				allowedRepoErr := fmt.Errorf("dispatch_repository tool %q has allowed_repositories entry %q in an unsupported format. Expected entries like 'owner/repo' or 'owner/*'. Example:\nallowed_repositories:\n  - github/gh-aw\n  - github/*", toolKey, repo)
				if returnErr := collector.Add(allowedRepoErr); returnErr != nil {
					return returnErr
				}
			}
		}

		dispatchRepositoryValidationLog.Printf("Tool %q validation passed", toolKey)
	}

	dispatchRepositoryValidationLog.Printf("dispatch_repository validation completed: error_count=%d, total_tools=%d",
		collector.Count(), len(config.Tools))

	return collector.FormattedError("dispatch_repository")
}
