package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var resolutionLog = logger.New("cli:add_workflow_resolution")
var fetchWorkflowFromSourceWithContextFn = FetchWorkflowFromSourceWithContext

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
	// IsPrivate indicates if the workflow has private: true in its frontmatter
	IsPrivate bool
	// IsActionWorkflow indicates that the source is a raw GitHub Actions YAML file (.yml)
	// rather than an agentic workflow markdown file (.md). When true, the file is installed
	// directly to .github/workflows/ without frontmatter processing or compilation.
	IsActionWorkflow bool
	// IsPackageSkillFile is true when the file belongs to a skill directory from an aw.yml
	// package manifest. The file is installed as-is to the agentic engine skill folder.
	IsPackageSkillFile bool
	// IsPackageAgentFile is true when the file is an agent .md from an aw.yml package
	// manifest. The file is installed as-is to the agentic engine agents folder.
	IsPackageAgentFile bool
	// SkillName is the skill directory name for package skill files (e.g. "my-skill").
	// Only meaningful when IsPackageSkillFile is true.
	SkillName string
}

// ResolvedWorkflows contains all resolved workflows ready to be added
type ResolvedWorkflows struct {
	// Workflows is the list of resolved workflows
	Workflows []*ResolvedWorkflow
	// HasWildcard indicates if any of the original specs contained wildcards (local only)
	HasWildcard bool
	// HasWorkflowDispatch is true if any of the workflows has a workflow_dispatch trigger
	HasWorkflowDispatch bool
	// Warnings contains non-fatal package-resolution warnings to show during add
	Warnings []string
}

// ResolveWorkflows resolves workflow specifications by parsing specs and fetching workflow content.
// For remote workflows, content is fetched directly from GitHub without cloning.
// Wildcards are only supported for local workflows (not remote repositories).
func ResolveWorkflows(ctx context.Context, workflows []string, verbose bool) (*ResolvedWorkflows, error) {
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
	parsedSpecs := make([]*WorkflowSpec, 0, len(workflows))
	var resolutionWarnings []string

	for _, workflow := range workflows {
		if repoSpec, ok, repoErr := parseRepositoryPackageSpec(workflow); ok {
			if repoErr != nil {
				return nil, repoErr
			}

			pkg, pkgErr := resolveRepositoryPackage(repoSpec, explicitHostForRepo(repoSpec.RepoSlug))
			if pkgErr == nil {
				resolutionWarnings = append(resolutionWarnings, pkg.Warnings...)
				parsedSpecs = appendRepositoryPackageWorkflowSpecs(parsedSpecs, repoSpec, pkg)
				continue
			}
			if repoSpec.PackagePath == "" || !isRepositoryPackageManifestNotFound(pkgErr) {
				return nil, pkgErr
			}
		}

		spec, err := parseWorkflowSpec(workflow)
		if err != nil {
			repoSpec, repoErr := parseRepoSpec(workflow)
			if repoErr != nil {
				return nil, fmt.Errorf("invalid specification '%s': not a valid workflow path or repository package: %w", workflow, repoErr)
			}

			pkg, pkgErr := resolveRepositoryPackage(repoSpec, explicitHostForRepo(repoSpec.RepoSlug))
			if pkgErr != nil {
				return nil, pkgErr
			}
			resolutionWarnings = append(resolutionWarnings, pkg.Warnings...)
			parsedSpecs = appendRepositoryPackageWorkflowSpecs(parsedSpecs, repoSpec, pkg)
			continue
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
		resolutionLog.Printf("Current repository: %s", currentRepoSlug)
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
	} else {
		resolutionLog.Printf("Could not determine current repository: %v", repoErr)
	}
	// If we can't determine the current repository, proceed without the check

	// Check if any workflow specs contain wildcards (local only)
	hasWildcard := sliceutil.Any(parsedSpecs, func(spec *WorkflowSpec) bool {
		return spec.IsWildcard
	})

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
		// Fetch workflow content (including redirect resolution for remote workflows)
		resolvedSpec, fetched, err := resolveAddWorkflowSpecAndContent(ctx, spec, verbose)
		if err != nil {
			return nil, fmt.Errorf("workflow '%s' not found: %w", spec.String(), err)
		}

		// Package skill files are installed as-is to the engine's skill directory.
		if spec.IsPackageSkillFile {
			resolutionLog.Printf("Resolved package skill file: spec=%s, skill=%s, content_size=%d bytes",
				spec.String(), spec.SkillName, len(fetched.Content))
			resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
				Spec:               resolvedSpec,
				Content:            fetched.Content,
				SourceInfo:         fetched,
				IsPackageSkillFile: true,
				SkillName:          spec.SkillName,
			})
			continue
		}

		// Package agent files are installed as-is to the engine's agents directory.
		if spec.IsPackageAgentFile {
			resolutionLog.Printf("Resolved package agent file: spec=%s, content_size=%d bytes",
				spec.String(), len(fetched.Content))
			resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
				Spec:               resolvedSpec,
				Content:            fetched.Content,
				SourceInfo:         fetched,
				IsPackageAgentFile: true,
			})
			continue
		}

		// Action workflow files (.yml) are raw GitHub Actions YAML — skip all markdown
		// frontmatter processing and install them as-is.
		if isActionWorkflowPath(resolvedSpec.WorkflowPath) {
			resolutionLog.Printf("Resolved action workflow: spec=%s, content_size=%d bytes",
				spec.String(), len(fetched.Content))
			resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
				Spec:             resolvedSpec,
				Content:          fetched.Content,
				SourceInfo:       fetched,
				IsActionWorkflow: true,
			})
			continue
		}

		// Extract description from content
		description := ExtractWorkflowDescription(string(fetched.Content))

		// Extract engine from content (if specified in frontmatter)
		engine := ExtractWorkflowEngine(string(fetched.Content))

		if spec.FromRepositoryManifest {
			privateValue, hasPrivate := ExtractWorkflowPrivateSetting(string(fetched.Content))
			if hasPrivate && privateValue {
				manifestPath := joinRepositoryPackagePath(spec.PackagePath, repositoryPackageManifestFileName)
				return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: workflow %q sets private: true and cannot be included because private workflows cannot be added", manifestPath, resolvedSpec.WorkflowPath)
			}
		}

		// Check if workflow is private - private workflows cannot be added to other repositories
		isPrivate := ExtractWorkflowPrivate(string(fetched.Content))
		if isPrivate {
			return nil, fmt.Errorf("workflow '%s' is private and cannot be added to other repositories", spec.String())
		}

		// Check for workflow_dispatch trigger in content
		workflowHasDispatch := checkWorkflowHasDispatchFromContent(string(fetched.Content))
		if workflowHasDispatch {
			hasWorkflowDispatch = true
		}

		if fetched.ConvertedFromJSON {
			resolutionWarnings = append(resolutionWarnings,
				fmt.Sprintf("JSON workflow import for %q was best-effort; run an agentic prompt to refine .github/workflows/%s.md", resolvedSpec.WorkflowName, resolvedSpec.WorkflowName))
		}

		resolutionLog.Printf("Resolved workflow: spec=%s, engine=%s, has_dispatch=%t, content_size=%d bytes",
			spec.String(), engine, workflowHasDispatch, len(fetched.Content))

		resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
			Spec:                resolvedSpec,
			Content:             fetched.Content,
			SourceInfo:          fetched,
			Description:         description,
			Engine:              engine,
			HasWorkflowDispatch: workflowHasDispatch,
			IsPrivate:           isPrivate,
		})
	}

	resolutionLog.Printf("Resolution complete: resolved=%d workflows, has_wildcard=%t, has_dispatch=%t",
		len(resolvedWorkflows), hasWildcard, hasWorkflowDispatch)

	return &ResolvedWorkflows{
		Workflows:           resolvedWorkflows,
		HasWildcard:         hasWildcard,
		HasWorkflowDispatch: hasWorkflowDispatch,
		Warnings:            resolutionWarnings,
	}, nil
}

func appendRepositoryPackageWorkflowSpecs(parsedSpecs []*WorkflowSpec, repoSpec *RepoSpec, pkg *resolvedRepositoryPackage) []*WorkflowSpec {
	if pkg == nil {
		return parsedSpecs
	}
	host := explicitHostForRepo(repoSpec.RepoSlug)
	effectiveVersion := repositoryPackageEffectiveRef(repoSpec, pkg)
	for _, installationSource := range pkg.InstallationSource {
		// installationSource is guaranteed by isSupportedPackageInstallablePath to be
		// either a .md agentic workflow or a .yml action workflow file; no other
		// extensions can reach this point.
		base := filepath.Base(installationSource)
		// Use filepath.Ext for case-insensitive extension removal (e.g. ".YML" or ".MD").
		workflowName := strings.TrimSuffix(base, filepath.Ext(base))
		parsedSpecs = append(parsedSpecs, &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug:    repoSpec.RepoSlug,
				Version:     effectiveVersion,
				PackagePath: repoSpec.PackagePath,
			},
			WorkflowPath:           installationSource,
			WorkflowName:           workflowName,
			Host:                   host,
			FromRepositoryManifest: true,
		})
	}

	// Append skill file specs. Each spec carries IsPackageSkillFile=true and the SkillName
	// so that the installation step can route the file to the correct skill directory.
	for _, skillFile := range pkg.SkillFiles {
		base := filepath.Base(skillFile.SourcePath)
		// WorkflowName is unused for skill files but set to a stable value for logging.
		workflowName := skillFile.SkillName + "/" + strings.TrimSuffix(base, filepath.Ext(base))
		parsedSpecs = append(parsedSpecs, &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug:    repoSpec.RepoSlug,
				Version:     effectiveVersion,
				PackagePath: repoSpec.PackagePath,
			},
			WorkflowPath:       skillFile.SourcePath,
			WorkflowName:       workflowName,
			Host:               host,
			IsPackageSkillFile: true,
			SkillName:          skillFile.SkillName,
		})
	}

	// Append agent file specs. Each spec carries IsPackageAgentFile=true so the installation
	// step routes the file to the correct agents directory.
	for _, agentFile := range pkg.AgentFiles {
		base := filepath.Base(agentFile)
		workflowName := strings.TrimSuffix(base, filepath.Ext(base))
		parsedSpecs = append(parsedSpecs, &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug:    repoSpec.RepoSlug,
				Version:     effectiveVersion,
				PackagePath: repoSpec.PackagePath,
			},
			WorkflowPath:       agentFile,
			WorkflowName:       workflowName,
			Host:               host,
			IsPackageAgentFile: true,
		})
	}

	return parsedSpecs
}

func resolveAddWorkflowSpecAndContent(ctx context.Context, initialSpec *WorkflowSpec, verbose bool) (*WorkflowSpec, *FetchedWorkflow, error) {
	currentSpec := *initialSpec
	visited := make(map[string]struct{})
	followedRedirect := false

	for range maxRedirectDepth {
		// Fetch workflow content - handles both local and remote.
		fetched, err := fetchWorkflowFromSourceWithContextFn(ctx, &currentSpec, verbose)
		if err != nil {
			return nil, nil, err
		}

		// Redirects only apply to remote workflows.
		if fetched.IsLocal {
			return &currentSpec, fetched, nil
		}

		currentRef := currentSpec.Version
		if currentRef == "" {
			currentRef = "main"
		}
		locationKey := fmt.Sprintf("%s/%s@%s", currentSpec.RepoSlug, currentSpec.WorkflowPath, currentRef)
		if _, exists := visited[locationKey]; exists {
			return nil, nil, fmt.Errorf("redirect loop detected at %s", locationKey)
		}
		visited[locationKey] = struct{}{}

		redirect, err := extractRedirectFromContent(string(fetched.Content))
		if err != nil {
			return nil, nil, err
		}
		if redirect == "" {
			// Preserve the original WorkflowName from the user's request only when
			// one or more redirects were followed, so the final local file keeps
			// the requested name.
			// Without redirects, keep any name derived during fetch, such as JSON
			// imports where conversion picks a better filename from `name`.
			if followedRedirect {
				currentSpec.WorkflowName = initialSpec.WorkflowName
			}
			return &currentSpec, fetched, nil
		}

		redirectedSource, err := normalizeRedirectToSourceSpec(redirect)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid redirect %q in %s: %w", redirect, locationKey, err)
		}

		nextSpec := &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug: redirectedSource.Repo,
				Version:  redirectedSource.Ref,
			},
			WorkflowPath: redirectedSource.Path,
			WorkflowName: normalizeWorkflowID(redirectedSource.Path),
			Host:         currentSpec.Host,
		}
		resolutionLog.Printf("Following redirect for add: from=%s to=%s", locationKey, nextSpec.String())
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow redirect: %s -> %s", locationKey, nextSpec.String())))
		}
		followedRedirect = true
		currentSpec = *nextSpec
	}

	return nil, nil, fmt.Errorf("redirect chain exceeded maximum depth (%d) for workflow '%s'", maxRedirectDepth, initialSpec.String())
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
			if str, ok := item.(string); ok && strings.EqualFold(str, "workflow_dispatch") {
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

	mdMatches := sliceutil.Filter(matches, func(m string) bool {
		return strings.HasSuffix(m, ".md")
	})
	result := sliceutil.Map(mdMatches, func(match string) *WorkflowSpec {
		return &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug: spec.RepoSlug,
				Version:  spec.Version,
			},
			WorkflowPath: match,
			WorkflowName: normalizeWorkflowID(match),
			IsWildcard:   false,
		}
	})

	return result, nil
}
