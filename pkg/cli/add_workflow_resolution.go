package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var resolutionLog = logger.New("cli:add_workflow_resolution")

// ResolvedWorkflow contains metadata about a workflow that has been resolved and is ready to add
type ResolvedWorkflow struct {
	// Spec is the parsed workflow specification
	Spec *WorkflowSpec
	// Content is the raw workflow content (convenience accessor, same as SourceInfo.Content)
	Content []byte
	// SourceInfo contains fetched workflow data including content, commit SHA, and source path
	SourceInfo *FetchedWorkflow
	// Description is the workflow description extracted from frontmatter
	Description string
	// Engine is the preferred engine extracted from frontmatter (empty if not specified)
	Engine string
	// HasWorkflowDispatch indicates if the workflow has workflow_dispatch trigger
	HasWorkflowDispatch bool
}

// ResolvedWorkflows contains all resolved workflows ready to be added
type ResolvedWorkflows struct {
	// Workflows is the list of resolved workflows
	Workflows []*ResolvedWorkflow
	// HasWildcard indicates if any of the original specs contained wildcards (local only)
	HasWildcard bool
	// HasWorkflowDispatch is true if any of the workflows has a workflow_dispatch trigger
	HasWorkflowDispatch bool
}

// ResolveWorkflows resolves workflow specifications by parsing specs and fetching workflow content.
// For remote workflows, content is fetched directly from GitHub without cloning.
// Wildcards are only supported for local workflows (not remote repositories).
func ResolveWorkflows(workflows []string, verbose bool) (*ResolvedWorkflows, error) {
	resolutionLog.Printf("Resolving workflows: count=%d", len(workflows))

	if len(workflows) == 0 {
		return nil, errors.New("at least one workflow name is required")
	}

	for i, workflow := range workflows {
		if workflow == "" {
			return nil, fmt.Errorf("workflow name cannot be empty (workflow %d)", i+1)
		}
	}

	// Parse workflow specifications
	parsedSpecs := []*WorkflowSpec{}

	for _, workflow := range workflows {
		spec, err := parseWorkflowSpec(workflow)
		if err != nil {
			return nil, fmt.Errorf("invalid workflow specification '%s': %w", workflow, err)
		}

		// Wildcards are only supported for local workflows
		if spec.IsWildcard && !isLocalWorkflowPath(spec.WorkflowPath) {
			return nil, fmt.Errorf("wildcards are only supported for local workflows, not remote repositories: %s", workflow)
		}

		parsedSpecs = append(parsedSpecs, spec)
	}

	// Check if any workflow is from the current repository
	// Skip this check if we can't determine the current repository (e.g., not in a git repo)
	currentRepoSlug, repoErr := GetCurrentRepoSlug()
	if repoErr == nil {
		// We successfully determined the current repository, check all workflow specs
		for _, spec := range parsedSpecs {
			// Skip local workflow specs
			if isLocalWorkflowPath(spec.WorkflowPath) {
				continue
			}

			if spec.RepoSlug == currentRepoSlug {
				return nil, fmt.Errorf("cannot add workflows from the current repository (%s). The 'add' command is for installing workflows from other repositories", currentRepoSlug)
			}
		}
	}
	// If we can't determine the current repository, proceed without the check

	// Check if any workflow specs contain wildcards (local only)
	hasWildcard := false
	for _, spec := range parsedSpecs {
		if spec.IsWildcard {
			hasWildcard = true
			break
		}
	}

	// Expand wildcards for local workflows only
	if hasWildcard {
		var err error
		parsedSpecs, err = expandLocalWildcardWorkflows(parsedSpecs, verbose)
		if err != nil {
			return nil, err
		}
	}

	// Fetch workflow content and metadata for each workflow
	resolvedWorkflows := make([]*ResolvedWorkflow, 0, len(parsedSpecs))
	hasWorkflowDispatch := false

	for _, spec := range parsedSpecs {
		// Fetch workflow content - FetchWorkflowFromSource handles both local and remote
		fetched, err := FetchWorkflowFromSource(spec, verbose)
		if err != nil {
			return nil, fmt.Errorf("workflow '%s' not found: %w", spec.String(), err)
		}

		// Extract description from content
		description := ExtractWorkflowDescription(string(fetched.Content))

		// Extract engine from content (if specified in frontmatter)
		engine := ExtractWorkflowEngine(string(fetched.Content))

		// Check for workflow_dispatch trigger in content
		workflowHasDispatch := checkWorkflowHasDispatchFromContent(string(fetched.Content))
		if workflowHasDispatch {
			hasWorkflowDispatch = true
		}

		resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
			Spec:                spec,
			Content:             fetched.Content,
			SourceInfo:          fetched,
			Description:         description,
			Engine:              engine,
			HasWorkflowDispatch: workflowHasDispatch,
		})
	}

	return &ResolvedWorkflows{
		Workflows:           resolvedWorkflows,
		HasWildcard:         hasWildcard,
		HasWorkflowDispatch: hasWorkflowDispatch,
	}, nil
}

// expandLocalWildcardWorkflows expands wildcard workflow specifications for local workflows only.
func expandLocalWildcardWorkflows(specs []*WorkflowSpec, verbose bool) ([]*WorkflowSpec, error) {
	expandedWorkflows := []*WorkflowSpec{}

	for _, spec := range specs {
		if spec.IsWildcard && isLocalWorkflowPath(spec.WorkflowPath) {
			resolutionLog.Printf("Expanding local wildcard: %s", spec.WorkflowPath)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Discovering local workflows matching %s...", spec.WorkflowPath)))
			}

			// Expand local wildcard (e.g., ./*.md or ./workflows/*.md)
			discovered, err := expandLocalWildcard(spec)
			if err != nil {
				return nil, fmt.Errorf("failed to expand wildcard %s: %w", spec.WorkflowPath, err)
			}

			if len(discovered) == 0 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflows found matching "+spec.WorkflowPath))
			} else {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %d workflow(s)", len(discovered))))
				}
				expandedWorkflows = append(expandedWorkflows, discovered...)
			}
		} else {
			expandedWorkflows = append(expandedWorkflows, spec)
		}
	}

	if len(expandedWorkflows) == 0 {
		return nil, errors.New("no workflows to add after expansion")
	}

	return expandedWorkflows, nil
}

// checkWorkflowHasDispatchFromContent checks if workflow content has a workflow_dispatch trigger
func checkWorkflowHasDispatchFromContent(content string) bool {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return false
	}

	onSection, exists := result.Frontmatter["on"]
	if !exists {
		return false
	}

	switch on := onSection.(type) {
	case map[string]any:
		_, hasDispatch := on["workflow_dispatch"]
		return hasDispatch
	case string:
		return strings.Contains(strings.ToLower(on), "workflow_dispatch")
	case []any:
		for _, item := range on {
			if str, ok := item.(string); ok && strings.ToLower(str) == "workflow_dispatch" {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// expandLocalWildcard expands a local wildcard path (e.g., ./*.md) into individual workflow specs
func expandLocalWildcard(spec *WorkflowSpec) ([]*WorkflowSpec, error) {
	pattern := spec.WorkflowPath

	// Use filepath.Glob to expand the pattern
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid wildcard pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	var result []*WorkflowSpec
	for _, match := range matches {
		// Only include .md files
		if !strings.HasSuffix(match, ".md") {
			continue
		}

		// Create a new spec for each matched file
		workflowName := normalizeWorkflowID(match)
		result = append(result, &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug: spec.RepoSlug,
				Version:  spec.Version,
			},
			WorkflowPath: match,
			WorkflowName: workflowName,
			IsWildcard:   false,
		})
	}

	return result, nil
}
