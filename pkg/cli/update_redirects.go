package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var updateRedirectsLog = logger.New("cli:update_redirects")

const maxRedirectDepth = 20

var resolveLatestRefFn = resolveLatestRef
var downloadWorkflowContentFn = downloadWorkflowContent

type resolvedUpdateLocation struct {
	sourceSpec      *SourceSpec
	currentRef      string
	latestRef       string
	sourceFieldRef  string
	content         []byte
	redirectHistory []string
}

func resolveRedirectedUpdateLocation(ctx context.Context, workflowName string, initialSource *SourceSpec, allowMajor, verbose bool, noRedirect bool, coolDown time.Duration) (*resolvedUpdateLocation, error) {
	updateRedirectsLog.Printf("Resolving update location: workflow=%s, source=%s/%s@%s", workflowName, initialSource.Repo, initialSource.Path, initialSource.Ref)
	current := &SourceSpec{
		Repo: initialSource.Repo,
		Path: initialSource.Path,
		Ref:  initialSource.Ref,
	}
	visited := make(map[string]struct{})
	history := make([]string, 0, 2)

	for range maxRedirectDepth {
		result, next, err := resolveRedirectedUpdateLocationStep(resolveRedirectedUpdateLocationStepParams{
			Ctx:          ctx,
			WorkflowName: workflowName,
			Current:      current,
			Visited:      visited,
			History:      history,
			AllowMajor:   allowMajor,
			Verbose:      verbose,
			NoRedirect:   noRedirect,
			CoolDown:     coolDown,
		})
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
		history = append(history, next.message)
		current = next.source
	}

	updateRedirectsLog.Printf("Redirect chain exceeded max depth: workflow=%s, depth=%d", workflowName, maxRedirectDepth)
	return nil, fmt.Errorf("redirect chain exceeded maximum depth (%d) while updating %s", maxRedirectDepth, workflowName)
}

type resolveRedirectedUpdateLocationNext struct {
	source  *SourceSpec
	message string
}

type resolveRedirectedUpdateLocationStepParams struct {
	Ctx          context.Context
	WorkflowName string
	Current      *SourceSpec
	Visited      map[string]struct{}
	History      []string
	AllowMajor   bool
	Verbose      bool
	NoRedirect   bool
	CoolDown     time.Duration
}

func resolveRedirectedUpdateLocationStep(p resolveRedirectedUpdateLocationStepParams) (*resolvedUpdateLocation, resolveRedirectedUpdateLocationNext, error) {
	currentRef := p.Current.Ref
	if currentRef == "" {
		currentRef = resolveDefaultBranchRef(p.Ctx, p.Current.Repo)
	}
	locationKey := sourceSpecWithRef(p.Current, currentRef)
	if _, exists := p.Visited[locationKey]; exists {
		updateRedirectsLog.Printf("Redirect loop detected: workflow=%s, location=%s", p.WorkflowName, locationKey)
		return nil, resolveRedirectedUpdateLocationNext{}, fmt.Errorf("redirect loop detected while updating %s at %s", p.WorkflowName, locationKey)
	}
	p.Visited[locationKey] = struct{}{}

	latestRef, content, redirect, err := resolveRedirectedUpdateLocationFetch(p.Ctx, p.Current, currentRef, p.AllowMajor, p.Verbose, p.CoolDown)
	if err != nil {
		return nil, resolveRedirectedUpdateLocationNext{}, err
	}
	sourceFieldRef := latestRef
	if isBranchRef(currentRef) {
		sourceFieldRef = currentRef
	}
	if redirect == "" {
		updateRedirectsLog.Printf("Resolved update location: workflow=%s, repo=%s, latestRef=%s, redirects=%d", p.WorkflowName, p.Current.Repo, latestRef, len(p.History))
		return &resolvedUpdateLocation{sourceSpec: p.Current, currentRef: currentRef, latestRef: latestRef, sourceFieldRef: sourceFieldRef, content: content, redirectHistory: p.History}, resolveRedirectedUpdateLocationNext{}, nil
	}
	next, err := resolveRedirectedUpdateLocationRedirect(p.Ctx, p.WorkflowName, p.Current, latestRef, redirect, p.NoRedirect)
	return nil, next, err
}

func resolveRedirectedUpdateLocationFetch(ctx context.Context, current *SourceSpec, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, []byte, string, error) {
	latestRef, err := resolveLatestRefFn(ctx, current.Repo, currentRef, allowMajor, verbose, coolDown)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to resolve latest ref for %s: %w", sourceSpecWithRef(current, currentRef), err)
	}
	content, err := downloadWorkflowContentFn(ctx, current.Repo, current.Path, latestRef, verbose)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to download workflow %s: %w", sourceSpecWithRef(current, latestRef), err)
	}
	redirect, err := extractRedirectFromContent(string(content))
	if err != nil {
		return "", nil, "", err
	}
	return latestRef, content, redirect, nil
}

func resolveRedirectedUpdateLocationRedirect(ctx context.Context, workflowName string, current *SourceSpec, latestRef, redirect string, noRedirect bool) (resolveRedirectedUpdateLocationNext, error) {
	if noRedirect {
		updateRedirectsLog.Printf("Redirect blocked by --no-redirect: workflow=%s, redirect=%s", workflowName, redirect)
		return resolveRedirectedUpdateLocationNext{}, fmt.Errorf("redirect is disabled by --no-redirect for %s: %s declares redirect to %s (remove redirect frontmatter or run update without --no-redirect)", workflowName, sourceSpecWithRef(current, latestRef), redirect)
	}
	redirectedSource, err := normalizeRedirectToSourceSpec(redirect)
	if err != nil {
		return resolveRedirectedUpdateLocationNext{}, fmt.Errorf("invalid redirect %q in %s: %w", redirect, sourceSpecWithRef(current, latestRef), err)
	}
	nextRef := redirectedSource.Ref
	if nextRef == "" {
		nextRef = resolveDefaultBranchRef(ctx, redirectedSource.Repo)
	}
	updateRedirectsLog.Printf("Following redirect: workflow=%s, from=%s, to=%s", workflowName, sourceSpecWithRef(current, latestRef), sourceSpecWithRef(redirectedSource, nextRef))
	redirectMessage := fmt.Sprintf("Workflow %s redirect: %s → %s", workflowName, sourceSpecWithRef(current, latestRef), sourceSpecWithRef(redirectedSource, nextRef))
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(redirectMessage))
	return resolveRedirectedUpdateLocationNext{source: redirectedSource, message: redirectMessage}, nil
}

// resolveDefaultBranchRef returns the repository's default branch name via the
// GitHub API. When the source field omits a branch ref we must track the repo's
// actual default branch (which may be "master", "trunk", etc.) rather than
// assuming "main". If the API lookup fails, it falls back to "main" so resolution
// can still proceed in offline or rate-limited scenarios.
func resolveDefaultBranchRef(ctx context.Context, repo string) string {
	branch, err := getRepoDefaultBranchCached(ctx, repo)
	if err != nil || strings.TrimSpace(branch) == "" {
		updateRedirectsLog.Printf("Falling back to 'main' as default branch for %s: %v", repo, err)
		return "main"
	}
	updateRedirectsLog.Printf("Resolved default branch for %s: %s", repo, branch)
	return branch
}

func extractRedirectFromContent(content string) (string, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirected workflow frontmatter: %w", err)
	}

	value, ok := result.Frontmatter["redirect"]
	if !ok {
		return "", nil
	}

	redirect, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("redirect must be a string, got %T", value)
	}

	return strings.TrimSpace(redirect), nil
}

func normalizeRedirectToSourceSpec(redirect string) (*SourceSpec, error) {
	updateRedirectsLog.Printf("Normalizing redirect to source spec: %q", redirect)
	redirect = strings.TrimSpace(redirect)
	if redirect == "" {
		return nil, errors.New("redirect cannot be empty")
	}

	if strings.Contains(redirect, "://") {
		workflowSpec, workflowErr := parseWorkflowSpec(redirect)
		if workflowErr != nil {
			return nil, fmt.Errorf("must be a workflow spec or GitHub URL: %w", workflowErr)
		}
		if workflowSpec.RepoSlug == "" {
			return nil, errors.New("redirect must point to a remote workflow location")
		}
		return &SourceSpec{
			Repo: workflowSpec.RepoSlug,
			Path: workflowSpec.WorkflowPath,
			Ref:  workflowSpec.Version,
		}, nil
	}

	// First try strict source syntax (owner/repo/path@ref).
	sourceSpec, sourceErr := parseSourceSpec(redirect)
	if sourceErr == nil {
		return sourceSpec, nil
	}

	// Fall back to workflow spec syntax (including GitHub URLs).
	workflowSpec, workflowErr := parseWorkflowSpec(redirect)
	if workflowErr != nil {
		return nil, fmt.Errorf("must be a workflow spec or GitHub URL: %w", workflowErr)
	}
	if workflowSpec.RepoSlug == "" {
		return nil, errors.New("redirect must point to a remote workflow location")
	}

	return &SourceSpec{
		Repo: workflowSpec.RepoSlug,
		Path: workflowSpec.WorkflowPath,
		Ref:  workflowSpec.Version,
	}, nil
}

func sourceSpecWithRef(spec *SourceSpec, ref string) string {
	if ref == "" {
		return fmt.Sprintf("%s/%s", spec.Repo, spec.Path)
	}
	return fmt.Sprintf("%s/%s@%s", spec.Repo, spec.Path, ref)
}
