package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var remoteWorkflowLog = logger.New("cli:remote_workflow")

// FetchedWorkflow contains content and metadata from a directly fetched workflow file.
// This is the unified type that combines content with source information.
type FetchedWorkflow struct {
	Content    []byte // The raw content of the workflow file
	CommitSHA  string // The resolved commit SHA at the time of fetch (empty for local)
	IsLocal    bool   // true if this is a local workflow (from filesystem)
	SourcePath string // The original source path (local path or remote path)
}

// FetchWorkflowFromSource fetches a workflow file directly from GitHub without cloning.
// This is the preferred way to add remote workflows as it only fetches the specific
// files needed rather than cloning the entire repository.
//
// For local workflows (local filesystem paths), it reads from the local filesystem.
// For remote workflows, it uses the GitHub API to fetch the file content.
func FetchWorkflowFromSource(spec *WorkflowSpec, verbose bool) (*FetchedWorkflow, error) {
	remoteWorkflowLog.Printf("Fetching workflow from source: spec=%s", spec.String())

	// Handle local workflows
	if isLocalWorkflowPath(spec.WorkflowPath) {
		return fetchLocalWorkflow(spec, verbose)
	}

	// Handle remote workflows from GitHub
	return fetchRemoteWorkflow(spec, verbose)
}

// fetchLocalWorkflow reads a workflow file from the local filesystem
func fetchLocalWorkflow(spec *WorkflowSpec, verbose bool) (*FetchedWorkflow, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Reading local workflow: "+spec.WorkflowPath))
	}

	content, err := os.ReadFile(spec.WorkflowPath)
	if err != nil {
		return nil, fmt.Errorf("local workflow '%s' not found: %w", spec.WorkflowPath, err)
	}

	return &FetchedWorkflow{
		Content:    content,
		CommitSHA:  "", // Local workflows don't have a commit SHA
		IsLocal:    true,
		SourcePath: spec.WorkflowPath,
	}, nil
}

// fetchRemoteWorkflow fetches a workflow file directly from GitHub using the API
func fetchRemoteWorkflow(spec *WorkflowSpec, verbose bool) (*FetchedWorkflow, error) {
	remoteWorkflowLog.Printf("Fetching remote workflow: repo=%s, path=%s, version=%s",
		spec.RepoSlug, spec.WorkflowPath, spec.Version)

	// Parse owner and repo from the slug
	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository slug: %s", spec.RepoSlug)
	}
	owner := parts[0]
	repo := parts[1]

	// Determine the ref to use
	ref := spec.Version
	if ref == "" {
		ref = "main" // Default to main branch
		remoteWorkflowLog.Print("No version specified, defaulting to 'main'")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching %s/%s/%s@%s...", owner, repo, spec.WorkflowPath, ref)))
	}

	// Resolve the ref to a commit SHA for source tracking
	commitSHA, err := parser.ResolveRefToSHAForHost(owner, repo, ref, spec.Host)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to resolve ref to SHA: %v", err)
		// Continue without SHA - we can still fetch the content
		commitSHA = ""
	} else {
		remoteWorkflowLog.Printf("Resolved ref %s to SHA: %s", ref, commitSHA)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Resolved to commit: "+commitSHA[:7]))
		}
	}

	// Download the workflow file from GitHub
	content, err := parser.DownloadFileFromGitHubForHost(owner, repo, spec.WorkflowPath, ref, spec.Host)
	if err != nil {
		// Try with common workflow directory prefixes if the direct path fails.
		// This handles short workflow names without path separators (e.g. "my-workflow.md").
		if !strings.HasPrefix(spec.WorkflowPath, "workflows/") && !strings.Contains(spec.WorkflowPath, "/") {
			for _, prefix := range []string{"workflows/", ".github/workflows/"} {
				altPath := prefix + spec.WorkflowPath
				if !strings.HasSuffix(altPath, ".md") {
					altPath += ".md"
				}
				remoteWorkflowLog.Printf("Direct path failed, trying: %s", altPath)
				if altContent, altErr := parser.DownloadFileFromGitHubForHost(owner, repo, altPath, ref, spec.Host); altErr == nil {
					return &FetchedWorkflow{
						Content:    altContent,
						CommitSHA:  commitSHA,
						IsLocal:    false,
						SourcePath: altPath,
					}, nil
				}
			}
		}
		return nil, fmt.Errorf("failed to download workflow from %s/%s/%s@%s: %w", owner, repo, spec.WorkflowPath, ref, err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded workflow (%d bytes)", len(content))))
	}

	return &FetchedWorkflow{
		Content:    content,
		CommitSHA:  commitSHA,
		IsLocal:    false,
		SourcePath: spec.WorkflowPath,
	}, nil
}
