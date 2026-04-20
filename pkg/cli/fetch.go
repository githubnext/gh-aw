package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var remoteWorkflowLog = logger.New("cli:remote_workflow")

var resolveRefToSHAForHost = parser.ResolveRefToSHAForHost
var downloadFileFromGitHubForHost = parser.DownloadFileFromGitHubForHost
var sleepBeforeSHAResolutionRetry = time.Sleep

var shaResolutionRetryDelays = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	9 * time.Second,
}

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

	// Resolve the ref to a commit SHA for source tracking.
	commitSHA, err := resolveCommitSHAWithRetries(owner, repo, ref, spec.WorkflowPath, spec.Host, verbose)
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Resolved to commit: "+commitSHA[:7]))
	}

	// Download the workflow file from GitHub
	content, err := downloadFileFromGitHubForHost(owner, repo, spec.WorkflowPath, ref, spec.Host)
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
				if altContent, altErr := downloadFileFromGitHubForHost(owner, repo, altPath, ref, spec.Host); altErr == nil {
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

func resolveCommitSHAWithRetries(owner, repo, ref, workflowPath, host string, verbose bool) (string, error) {
	retryCommand := fmt.Sprintf("gh aw add %s/%s/%s@<exact-sha>", owner, repo, workflowPath)
	attempts := len(shaResolutionRetryDelays) + 1
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		commitSHA, err := resolveRefToSHAForHost(owner, repo, ref, host)
		if err == nil {
			remoteWorkflowLog.Printf("Resolved ref %s to SHA: %s", ref, commitSHA)
			return commitSHA, nil
		}

		lastErr = err
		remoteWorkflowLog.Printf("Failed to resolve ref %s to SHA (attempt %d/%d): %v", ref, attempt, attempts, err)

		if !isTransientSHAResolutionError(err) {
			message := fmt.Sprintf(
				"failed to resolve '%s' to commit SHA for '%s/%s': %v. Expected the GitHub API to return a commit SHA for the ref. Try: %s.",
				ref, owner, repo, err, retryCommand,
			)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(message))
			return "", fmt.Errorf("%s: %w", message, err)
		}

		if attempt < attempts {
			delay := shaResolutionRetryDelays[attempt-1]
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
					fmt.Sprintf("Transient SHA resolution failure for '%s' (attempt %d/%d). Retrying in %s...", ref, attempt, attempts, delay),
				))
			}
			sleepBeforeSHAResolutionRetry(delay)
		}
	}

	message := fmt.Sprintf(
		"failed to resolve '%s' to commit SHA after %d retries for '%s/%s': %v. Expected the GitHub API to return a commit SHA for the ref. Check rate limits or try: %s.",
		ref, len(shaResolutionRetryDelays), owner, repo, lastErr, retryCommand,
	)
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage(message))
	return "", fmt.Errorf("%s: %w", message, lastErr)
}

func isTransientSHAResolutionError(err error) bool {
	if err == nil {
		return false
	}

	errorText := strings.ToLower(err.Error())
	if strings.Contains(errorText, "http 429") ||
		strings.Contains(errorText, "rate limit") ||
		strings.Contains(errorText, "timeout") ||
		strings.Contains(errorText, "timed out") ||
		strings.Contains(errorText, "context deadline exceeded") ||
		strings.Contains(errorText, "temporary") ||
		strings.Contains(errorText, "connection reset") ||
		strings.Contains(errorText, "connection refused") ||
		strings.Contains(errorText, "eof") {
		return true
	}

	for statusCode := 500; statusCode <= 599; statusCode++ {
		if strings.Contains(errorText, fmt.Sprintf("http %d", statusCode)) {
			return true
		}
	}

	return false
}
