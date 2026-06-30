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

	parsedSpecs, resolutionWarnings, err := parseWorkflowSpecs(workflows)
	if err != nil {
		return nil, err
	}

	if repoErr := validateCurrentRepoSpecs(parsedSpecs); repoErr != nil {
		return nil, repoErr
	}

	hasWildcard := sliceutil.Any(parsedSpecs, func(spec *WorkflowSpec) bool {
		return spec.IsWildcard
	})
	if hasWildcard {
		parsedSpecs, err = expandLocalWildcardWorkflows(parsedSpecs, verbose)
		if err != nil {
			return nil, err
		}
	}

	resolvedWorkflows, hasWorkflowDispatch, moreWarnings, err := fetchWorkflowContents(ctx, parsedSpecs, verbose)
	if err != nil {
		return nil, err
	}
	resolutionWarnings = append(resolutionWarnings, moreWarnings...)

	resolutionLog.Printf("Resolution complete: resolved=%d workflows, has_wildcard=%t, has_dispatch=%t",
		len(resolvedWorkflows), hasWildcard, hasWorkflowDispatch)

	return &ResolvedWorkflows{
		Workflows:           resolvedWorkflows,
		HasWildcard:         hasWildcard,
		HasWorkflowDispatch: hasWorkflowDispatch,
		Warnings:            resolutionWarnings,
	}, nil
}

// parseWorkflowSpecs parses workflow specification strings into WorkflowSpec objects,
// handling repository package specs and direct workflow paths.
func parseWorkflowSpecs(workflows []string) ([]*WorkflowSpec, []string, error) {
	parsedSpecs := make([]*WorkflowSpec, 0, len(workflows))
	var resolutionWarnings []string

	for _, workflow := range workflows {
		if repoSpec, ok, repoErr := parseRepositoryPackageSpec(workflow); ok {
			if repoErr != nil {
				return nil, nil, repoErr
			}
			pkg, pkgErr := resolveRepositoryPackage(repoSpec, explicitHostForRepo(repoSpec.RepoSlug))
			if pkgErr == nil {
				resolutionWarnings = append(resolutionWarnings, pkg.Warnings...)
				parsedSpecs = appendRepositoryPackageWorkflowSpecs(parsedSpecs, repoSpec, pkg)
				continue
			}
			if repoSpec.PackagePath == "" || !isRepositoryPackageManifestNotFound(pkgErr) {
				return nil, nil, pkgErr
			}
		}

		spec, err := parseWorkflowSpec(workflow)
		if err != nil {
			repoSpec, repoErr := parseRepoSpec(workflow)
			if repoErr != nil {
				return nil, nil, fmt.Errorf("invalid specification '%s': not a valid workflow path or repository package: %w", workflow, repoErr)
			}
			pkg, pkgErr := resolveRepositoryPackage(repoSpec, explicitHostForRepo(repoSpec.RepoSlug))
			if pkgErr != nil {
				return nil, nil, pkgErr
			}
			resolutionWarnings = append(resolutionWarnings, pkg.Warnings...)
			parsedSpecs = appendRepositoryPackageWorkflowSpecs(parsedSpecs, repoSpec, pkg)
			continue
		}
		if spec.IsWildcard && !isLocalWorkflowPath(spec.WorkflowPath) {
			return nil, nil, fmt.Errorf("wildcards are only supported for local workflows, not remote repositories: %s", workflow)
		}
		parsedSpecs = append(parsedSpecs, spec)
	}
	return parsedSpecs, resolutionWarnings, nil
}

// validateCurrentRepoSpecs checks that no spec refers to the current repository.
func validateCurrentRepoSpecs(parsedSpecs []*WorkflowSpec) error {
	currentRepoSlug, repoErr := GetCurrentRepoSlug()
	if repoErr != nil {
		resolutionLog.Printf("Could not determine current repository: %v", repoErr)
		return nil
	}
	resolutionLog.Printf("Current repository: %s", currentRepoSlug)
	for _, spec := range parsedSpecs {
		if isLocalWorkflowPath(spec.WorkflowPath) {
			continue
		}
		if spec.RepoSlug == currentRepoSlug {
			return fmt.Errorf("cannot add workflows from the current repository (%s). The 'add' command is for installing workflows from other repositories", currentRepoSlug)
		}
	}
	return nil
}

// fetchWorkflowContents fetches content for all parsed specs and returns resolved workflows.
func fetchWorkflowContents(ctx context.Context, parsedSpecs []*WorkflowSpec, verbose bool) ([]*ResolvedWorkflow, bool, []string, error) {
	resolvedWorkflows := make([]*ResolvedWorkflow, 0, len(parsedSpecs))
	var resolutionWarnings []string
	hasWorkflowDispatch := false

	for _, spec := range parsedSpecs {
		resolvedSpec, fetched, err := resolveAddWorkflowSpecAndContent(ctx, spec, verbose)
		if err != nil {
			return nil, false, nil, fmt.Errorf("workflow '%s' not found: %w", spec.String(), err)
		}

		if spec.IsPackageSkillFile {
			resolutionLog.Printf("Resolved package skill file: spec=%s, skill=%s, content_size=%d bytes",
				spec.String(), spec.SkillName, len(fetched.Content))
			resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
				Spec: resolvedSpec, Content: fetched.Content, SourceInfo: fetched,
				IsPackageSkillFile: true, SkillName: spec.SkillName,
			})
			continue
		}
		if spec.IsPackageAgentFile {
			resolutionLog.Printf("Resolved package agent file: spec=%s, content_size=%d bytes",
				spec.String(), len(fetched.Content))
			resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
				Spec: resolvedSpec, Content: fetched.Content, SourceInfo: fetched, IsPackageAgentFile: true,
			})
			continue
		}
		if isActionWorkflowPath(resolvedSpec.WorkflowPath) {
			resolutionLog.Printf("Resolved action workflow: spec=%s, content_size=%d bytes",
				spec.String(), len(fetched.Content))
			resolvedWorkflows = append(resolvedWorkflows, &ResolvedWorkflow{
				Spec: resolvedSpec, Content: fetched.Content, SourceInfo: fetched, IsActionWorkflow: true,
			})
			continue
		}

		rw, warns, dispatches, err := buildRegularResolvedWorkflow(spec, resolvedSpec, fetched)
		if err != nil {
			return nil, false, nil, err
		}
		resolutionWarnings = append(resolutionWarnings, warns...)
		if dispatches {
			hasWorkflowDispatch = true
		}
		resolvedWorkflows = append(resolvedWorkflows, rw)
	}
	return resolvedWorkflows, hasWorkflowDispatch, resolutionWarnings, nil
}

// buildRegularResolvedWorkflow builds a ResolvedWorkflow for a standard markdown agentic workflow.
func buildRegularResolvedWorkflow(spec *WorkflowSpec, resolvedSpec *WorkflowSpec, fetched *FetchedWorkflow) (*ResolvedWorkflow, []string, bool, error) {
	var warns []string
	description := ExtractWorkflowDescription(string(fetched.Content))
	engine := ExtractWorkflowEngine(string(fetched.Content))

	if spec.FromRepositoryManifest {
		privateValue, hasPrivate := ExtractWorkflowPrivateSetting(string(fetched.Content))
		if hasPrivate && privateValue {
			manifestPath := joinRepositoryPackagePath(spec.PackagePath, repositoryPackageManifestFileName)
			return nil, nil, false, fmt.Errorf("invalid Agentic Workflow manifest %q: workflow %q sets private: true and cannot be included because private workflows cannot be added", manifestPath, resolvedSpec.WorkflowPath)
		}
	}

	isPrivate := ExtractWorkflowPrivate(string(fetched.Content))
	if isPrivate {
		return nil, nil, false, fmt.Errorf("workflow '%s' is private and cannot be added to other repositories", spec.String())
	}

	dispatches := checkWorkflowHasDispatchFromContent(string(fetched.Content))
	if fetched.ConvertedFromJSON {
		warns = append(warns, fmt.Sprintf("JSON workflow import for %q was best-effort; run an agentic prompt to refine .github/workflows/%s.md", resolvedSpec.WorkflowName, resolvedSpec.WorkflowName))
	}
	resolutionLog.Printf("Resolved workflow: spec=%s, engine=%s, has_dispatch=%t, content_size=%d bytes",
		spec.String(), engine, dispatches, len(fetched.Content))

	return &ResolvedWorkflow{
		Spec:                resolvedSpec,
		Content:             fetched.Content,
		SourceInfo:          fetched,
		Description:         description,
		Engine:              engine,
		HasWorkflowDispatch: dispatches,
		IsPrivate:           isPrivate,
	}, warns, dispatches, nil
}

func appendRepositoryPackageWorkflowSpecs(parsedSpecs []*WorkflowSpec, repoSpec *RepoSpec, pkg *resolvedRepositoryPackage) []*WorkflowSpec {
	if pkg == nil {
		return parsedSpecs
	}
	host := explicitHostForRepo(repoSpec.RepoSlug)
	effectiveVersion := repositoryPackageEffectiveRef(repoSpec, pkg)
	repoBase := RepoSpec{RepoSlug: repoSpec.RepoSlug, Version: effectiveVersion, PackagePath: repoSpec.PackagePath}

	for _, installationSource := range pkg.InstallationSource {
		base := filepath.Base(installationSource)
		workflowName := strings.TrimSuffix(base, filepath.Ext(base))
		parsedSpecs = append(parsedSpecs, &WorkflowSpec{
			RepoSpec: repoBase, WorkflowPath: installationSource,
			WorkflowName: workflowName, Host: host, FromRepositoryManifest: true,
		})
	}

	for _, skillFile := range pkg.SkillFiles {
		base := filepath.Base(skillFile.SourcePath)
		workflowName := skillFile.SkillName + "/" + strings.TrimSuffix(base, filepath.Ext(base))
		parsedSpecs = append(parsedSpecs, &WorkflowSpec{
			RepoSpec: repoBase, WorkflowPath: skillFile.SourcePath,
			WorkflowName: workflowName, Host: host,
			IsPackageSkillFile: true, SkillName: skillFile.SkillName,
		})
	}

	parsedSpecs = appendPackageAgentFileSpecs(parsedSpecs, repoBase, host, pkg.AgentFiles)
	return parsedSpecs
}

// appendPackageAgentFileSpecs appends agent file WorkflowSpecs from a resolved package.
func appendPackageAgentFileSpecs(parsedSpecs []*WorkflowSpec, repoBase RepoSpec, host string, agentFiles []string) []*WorkflowSpec {
	for _, agentFile := range agentFiles {
		base := filepath.Base(agentFile)
		workflowName := strings.TrimSuffix(base, filepath.Ext(base))
		parsedSpecs = append(parsedSpecs, &WorkflowSpec{
			RepoSpec: repoBase, WorkflowPath: agentFile,
			WorkflowName: workflowName, Host: host, IsPackageAgentFile: true,
		})
	}
	return parsedSpecs
}

func resolveAddWorkflowSpecAndContent(ctx context.Context, initialSpec *WorkflowSpec, verbose bool) (*WorkflowSpec, *FetchedWorkflow, error) {
	currentSpec := *initialSpec
	visited := make(map[string]struct{})
	followedRedirect := false

	for range maxRedirectDepth {
		fetched, err := fetchWorkflowFromSourceWithContextFn(ctx, &currentSpec, verbose)
		if err != nil {
			return nil, nil, err
		}
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
			if followedRedirect {
				currentSpec.WorkflowName = initialSpec.WorkflowName
			}
			return &currentSpec, fetched, nil
		}

		nextSpec, err := buildRedirectSpec(redirect, locationKey, currentSpec.Host, verbose)
		if err != nil {
			return nil, nil, err
		}
		followedRedirect = true
		currentSpec = *nextSpec
	}

	return nil, nil, fmt.Errorf("redirect chain exceeded maximum depth (%d) for workflow '%s'", maxRedirectDepth, initialSpec.String())
}

// buildRedirectSpec resolves a redirect string into the next WorkflowSpec to follow.
func buildRedirectSpec(redirect, locationKey, host string, verbose bool) (*WorkflowSpec, error) {
	redirectedSource, err := normalizeRedirectToSourceSpec(redirect)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect %q in %s: %w", redirect, locationKey, err)
	}
	nextSpec := &WorkflowSpec{
		RepoSpec:     RepoSpec{RepoSlug: redirectedSource.Repo, Version: redirectedSource.Ref},
		WorkflowPath: redirectedSource.Path,
		WorkflowName: normalizeWorkflowID(redirectedSource.Path),
		Host:         host,
	}
	resolutionLog.Printf("Following redirect for add: from=%s to=%s", locationKey, nextSpec.String())
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow redirect: %s -> %s", locationKey, nextSpec.String())))
	}
	return nextSpec, nil
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
