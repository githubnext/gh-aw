package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
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
	commitSHA, err := parser.ResolveRefToSHA(owner, repo, ref)
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
	content, err := parser.DownloadFileFromGitHub(owner, repo, spec.WorkflowPath, ref)
	if err != nil {
		// Try with a workflows/ prefix if the direct path fails
		if !strings.HasPrefix(spec.WorkflowPath, "workflows/") && !strings.Contains(spec.WorkflowPath, "/") {
			// Try workflows/filename.md
			altPath := "workflows/" + spec.WorkflowPath
			if !strings.HasSuffix(altPath, ".md") {
				altPath += ".md"
			}
			remoteWorkflowLog.Printf("Direct path failed, trying: %s", altPath)
			if altContent, altErr := parser.DownloadFileFromGitHub(owner, repo, altPath, ref); altErr == nil {
				return &FetchedWorkflow{
					Content:    altContent,
					CommitSHA:  commitSHA,
					IsLocal:    false,
					SourcePath: altPath,
				}, nil
			}

			// Try .github/workflows/filename.md
			altPath = ".github/workflows/" + spec.WorkflowPath
			if !strings.HasSuffix(altPath, ".md") {
				altPath += ".md"
			}
			remoteWorkflowLog.Printf("Trying: %s", altPath)
			if altContent, altErr := parser.DownloadFileFromGitHub(owner, repo, altPath, ref); altErr == nil {
				return &FetchedWorkflow{
					Content:    altContent,
					CommitSHA:  commitSHA,
					IsLocal:    false,
					SourcePath: altPath,
				}, nil
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

// FetchIncludeFromSource fetches an include file from GitHub directly using a workflowspec format path.
// The includePath should be in the format: owner/repo/path/to/file.md[@ref]
// If the includePath is a relative path, it's resolved relative to the baseSpec.
// Returns: (content, section, error) where section is the #fragment from the path (e.g., "#section-name").
func FetchIncludeFromSource(includePath string, baseSpec *WorkflowSpec, verbose bool) ([]byte, string, error) {
	baseSpecStr := "<nil>"
	if baseSpec != nil {
		baseSpecStr = baseSpec.String()
	}
	remoteWorkflowLog.Printf("Fetching include from source: path=%s, base=%s", includePath, baseSpecStr)

	// Extract section reference (e.g., "#section-name") from the path upfront
	// This ensures consistent behavior regardless of which code path is taken
	cleanPath := includePath
	var section string
	if idx := strings.Index(includePath, "#"); idx != -1 {
		cleanPath = includePath[:idx]
		section = includePath[idx:]
	}

	// Check if this is a workflowspec format (owner/repo/path[@ref])
	if isWorkflowSpecFormat(cleanPath) {
		// Split on @ to get path and ref
		parts := strings.SplitN(cleanPath, "@", 2)
		pathPart := parts[0]
		var ref string
		if len(parts) == 2 {
			ref = parts[1]
		} else {
			ref = "main"
		}

		// Parse path: owner/repo/path/to/file.md
		slashParts := strings.Split(pathPart, "/")
		if len(slashParts) < 3 {
			return nil, section, errors.New("invalid workflowspec: must be owner/repo/path[@ref]")
		}

		owner := slashParts[0]
		repo := slashParts[1]
		filePath := strings.Join(slashParts[2:], "/")

		// Download the file
		content, err := parser.DownloadFileFromGitHub(owner, repo, filePath, ref)
		if err != nil {
			return nil, section, fmt.Errorf("failed to fetch include from %s: %w", includePath, err)
		}

		return content, section, nil
	}

	// For relative paths, resolve against the base spec
	if baseSpec != nil && baseSpec.RepoSlug != "" {
		parts := strings.SplitN(baseSpec.RepoSlug, "/", 2)
		if len(parts) == 2 {
			owner := parts[0]
			repo := parts[1]
			ref := baseSpec.Version
			if ref == "" {
				ref = "main"
			}

			// Remove @ ref suffix if present in the clean path (for relative paths with explicit refs)
			filePath := cleanPath
			if idx := strings.Index(filePath, "@"); idx != -1 {
				filePath = filePath[:idx]
			}

			// If it's a relative path starting with shared/, it's relative to .github/
			var fullPath string
			if strings.HasPrefix(filePath, "shared/") {
				fullPath = ".github/" + filePath
			} else {
				// Otherwise, resolve relative to the workflow path directory
				baseDir := getParentDir(baseSpec.WorkflowPath)
				if baseDir != "" {
					fullPath = baseDir + "/" + filePath
				} else {
					fullPath = filePath
				}
			}

			content, err := parser.DownloadFileFromGitHub(owner, repo, fullPath, ref)
			if err != nil {
				return nil, section, fmt.Errorf("failed to fetch include %s from %s/%s: %w", filePath, owner, repo, err)
			}

			return content, section, nil
		}
	}

	return nil, section, fmt.Errorf("cannot resolve include path: %s (no base spec provided)", includePath)
}

// fetchAndSaveRemoteFrontmatterImports fetches and saves files referenced in the frontmatter
// 'imports:' field of a remote workflow. These relative-path imports are resolved against
// the workflow's location in the source repository and saved locally so compilation can find them.
// This is analogous to fetchAndSaveRemoteIncludes, which handles @include directives in the
// markdown body; this function handles the YAML frontmatter 'imports:' field.
// Import failures are non-fatal (best-effort); the compiler will report any still-missing files.
func fetchAndSaveRemoteFrontmatterImports(content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	if spec.RepoSlug == "" {
		return nil
	}

	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	owner, repo := parts[0], parts[1]
	ref := spec.Version
	if ref == "" {
		// Resolve the actual default branch of the source repo rather than assuming "main"
		defaultBranch, err := getRepoDefaultBranch(spec.RepoSlug)
		if err != nil {
			remoteWorkflowLog.Printf("Failed to resolve default branch for %s, falling back to 'main': %v", spec.RepoSlug, err)
			ref = "main"
		} else {
			ref = defaultBranch
		}
		// Persist the resolved default ref so other callers do not need to re-resolve it
		spec.Version = ref
	}

	// workflowBaseDir is the directory of the top-level workflow in the source repo
	// (e.g. ".github/workflows"). It serves as both the starting point for resolving
	// relative imports and as the prefix to strip when computing local target paths.
	workflowBaseDir := getParentDir(spec.WorkflowPath)

	// seen is keyed by fully-resolved remote file path. It is shared across all recursion
	// levels so that every import (at any depth) is downloaded at most once and import
	// cycles (A imports B, B imports A) are broken without infinite recursion.
	seen := make(map[string]bool)
	fetchFrontmatterImportsRecursive(content, owner, repo, ref, workflowBaseDir, workflowBaseDir, targetDir, verbose, force, tracker, seen)
	return nil
}

// fetchFrontmatterImportsRecursive is the internal worker for fetchAndSaveRemoteFrontmatterImports.
//
// Parameters that change per recursion level:
//   - content: the text of the file whose imports are being processed
//   - currentBaseDir: directory of that file inside the source repo (used to resolve relative paths)
//
// Parameters that remain constant across all recursion levels:
//   - owner, repo, ref: source repository coordinates
//   - originalBaseDir: directory of the top-level workflow (used to map remote paths → local paths)
//   - targetDir: the `.github/workflows` directory in the user's repo
//   - seen: shared visited set (keyed by fully-resolved remote path) — prevents cycles & duplicates
func fetchFrontmatterImportsRecursive(content, owner, repo, ref, currentBaseDir, originalBaseDir, targetDir string, verbose, force bool, tracker *FileTracker, seen map[string]bool) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil || result.Frontmatter == nil {
		return
	}

	importsField, exists := result.Frontmatter["imports"]
	if !exists {
		return
	}

	var importPaths []string
	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				importPaths = append(importPaths, s)
			}
		}
	case []string:
		importPaths = v
	}

	if len(importPaths) == 0 {
		return
	}

	// Pre-compute the absolute target directory once for path-traversal boundary checks.
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return
	}

	for _, importPath := range importPaths {
		// Skip workflowspec-format imports (already pinned to a remote ref)
		if isWorkflowSpecFormat(importPath) {
			continue
		}

		// Strip any section reference (file.md#Section → file.md)
		filePath := importPath
		if before, _, hasSec := strings.Cut(importPath, "#"); hasSec {
			filePath = before
		}
		if filePath == "" {
			continue
		}

		// Resolve the remote file path relative to the current file's directory.
		// Use path (not filepath) because this is always a forward-slash URL/API path.
		var remoteFilePath string
		if rest, ok := strings.CutPrefix(filePath, "/"); ok {
			// Absolute path from repo root (e.g. "/scripts/helper.md")
			remoteFilePath = rest
		} else if currentBaseDir != "" {
			remoteFilePath = path.Join(currentBaseDir, filePath)
		} else {
			remoteFilePath = filePath
		}
		remoteFilePath = path.Clean(remoteFilePath)

		// Reject paths that try to escape the repository root (e.g. "../../etc/passwd")
		if remoteFilePath == ".." || strings.HasPrefix(remoteFilePath, "../") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping import with unsafe path: %q", importPath)))
			}
			continue
		}

		// Cycle/duplicate prevention: use the fully-resolved remote path as the key.
		if seen[remoteFilePath] {
			continue
		}
		seen[remoteFilePath] = true

		// Derive the local path relative to targetDir by stripping the original base-dir
		// prefix from the remote path. This ensures that imports in nested files resolve
		// to the correct location regardless of how many levels deep the recursion goes.
		//
		// Example: originalBaseDir=".github/workflows"
		//   remoteFilePath=".github/workflows/shared/analysis.md" → localRelPath="shared/analysis.md"
		//   (nested) remoteFilePath=".github/workflows/other.md"  → localRelPath="other.md"
		var localRelPath string
		if originalBaseDir != "" && strings.HasPrefix(remoteFilePath, originalBaseDir+"/") {
			localRelPath = remoteFilePath[len(originalBaseDir)+1:]
		} else {
			// Workflow at repo root, or import outside the original base dir:
			// use the full remote path relative to targetDir.
			localRelPath = remoteFilePath
		}
		localRelPath = filepath.Clean(filepath.FromSlash(localRelPath))
		// Strip any leading separator produced by Clean on root-relative paths.
		localRelPath = strings.TrimLeft(localRelPath, string(filepath.Separator))
		// Reject empty or "." paths (would point to targetDir itself) as a safety guard.
		// ".." cannot appear here because remoteFilePath was already rejected above if it
		// started with "..", and path.Clean cannot introduce new ".." components.
		if localRelPath == "" || localRelPath == "." {
			continue
		}
		targetPath := filepath.Join(targetDir, localRelPath)

		// Belt-and-suspenders: verify the resolved path is inside targetDir
		absTargetPath, absErr := filepath.Abs(targetPath)
		if absErr != nil {
			continue
		}
		if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write import outside target directory: %q", importPath)))
			}
			continue
		}

		// Check existence before downloading: if the file already exists and force=false,
		// skip the download entirely (no unnecessary network round-trip).
		fileExists := false
		if _, statErr := os.Stat(targetPath); statErr == nil {
			fileExists = true
			if !force {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Import file already exists, skipping: "+targetPath))
				}
				continue
			}
		}

		// Download from the source repository
		importContent, err := parser.DownloadFileFromGitHub(owner, repo, remoteFilePath, ref)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch import %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// Create the parent directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for import %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// Write the file
		if err := os.WriteFile(targetPath, importContent, 0600); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write import %s: %v", remoteFilePath, err)))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched import: "+targetPath))
		}

		// Track the file for git staging and potential rollback
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(targetPath)
			} else {
				tracker.TrackCreated(targetPath)
			}
		}

		// Recurse into the imported file's imports. Use the imported file's directory as
		// currentBaseDir so that relative paths inside it resolve correctly.
		importedBaseDir := path.Dir(remoteFilePath)
		fetchFrontmatterImportsRecursive(string(importContent), owner, repo, ref, importedBaseDir, originalBaseDir, targetDir, verbose, force, tracker, seen)
	}
}

// fetchAndSaveRemoteIncludes parses the workflow content for @include directives and fetches them from the remote source
func fetchAndSaveRemoteIncludes(content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	remoteWorkflowLog.Printf("Fetching remote includes for workflow: %s", spec.String())

	// Parse the workflow content to find @include directives
	includePattern := regexp.MustCompile(`^@include(\?)?\s+(.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		matches := includePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		isOptional := matches[1] == "?"
		includePath := strings.TrimSpace(matches[2])

		// Remove section reference for file fetching
		filePath := includePath
		if before, _, ok := strings.Cut(includePath, "#"); ok {
			filePath = before
		}

		// Skip if already processed
		if seen[filePath] {
			continue
		}
		seen[filePath] = true

		// Fetch the include file
		includeContent, _, err := FetchIncludeFromSource(includePath, spec, verbose)
		if err != nil {
			if isOptional {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Optional include not found: "+includePath))
				}
				continue
			}
			return fmt.Errorf("failed to fetch include %s: %w", includePath, err)
		}

		// Determine target path for the include file
		var targetPath string
		if strings.HasPrefix(filePath, "shared/") {
			// shared/ files go to .github/shared/
			targetPath = filepath.Join(filepath.Dir(targetDir), filePath)
		} else if isWorkflowSpecFormat(filePath) {
			// Workflowspec includes: extract just the filename and put in shared/
			parts := strings.Split(filePath, "/")
			filename := parts[len(parts)-1]
			targetPath = filepath.Join(filepath.Dir(targetDir), "shared", filename)
		} else {
			// Relative includes go alongside the workflow
			targetPath = filepath.Join(targetDir, filePath)
		}

		// Create target directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Check if file already exists
		fileExists := false
		if _, err := os.Stat(targetPath); err == nil {
			fileExists = true
			if !force {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Include file already exists, skipping: "+targetPath))
				}
				continue
			}
		}

		// Write the include file
		if err := os.WriteFile(targetPath, includeContent, 0600); err != nil {
			return fmt.Errorf("failed to write include file %s: %w", targetPath, err)
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched include: "+targetPath))
		}

		// Track the file
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(targetPath)
			} else {
				tracker.TrackCreated(targetPath)
			}
		}

		// Recursively fetch includes from the fetched file
		if err := fetchAndSaveRemoteIncludes(string(includeContent), spec, targetDir, verbose, force, tracker); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch nested includes from %s: %v", filePath, err)))
			}
		}
	}

	return nil
}

// getParentDir returns the directory part of a path
func getParentDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return ""
	}
	return path[:idx]
}

// readSourceRepoFromFile reads the 'source' frontmatter field from a local workflow file
// and returns the "owner/repo" portion (e.g. "github/gh-aw"). Returns "" if the file
// cannot be read, has no source field, or the field is not in the expected format.
func readSourceRepoFromFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result.Frontmatter == nil {
		return ""
	}
	sourceRaw, ok := result.Frontmatter["source"]
	if !ok {
		return ""
	}
	source, ok := sourceRaw.(string)
	if !ok || source == "" {
		return ""
	}
	// source format: "owner/repo/path/to/file.md@ref" — extract just "owner/repo"
	slashParts := strings.SplitN(source, "/", 3)
	if len(slashParts) < 2 {
		return ""
	}
	return slashParts[0] + "/" + slashParts[1]
}

// sourceRepoLabel returns the source repo string for display in error messages.
// When the repo string is empty (file has no source field or is not a markdown file),
// a human-readable placeholder is returned so the error message is not confusing.
func sourceRepoLabel(repo string) string {
	if repo == "" {
		return "(no source field)"
	}
	return repo
}

// extractDispatchWorkflowNames extracts workflow names from the safe-outputs.dispatch-workflow
// frontmatter field. It handles both array and map forms of the configuration.
// Workflow names that contain GitHub Actions expression syntax (e.g. "${{") are skipped.
func extractDispatchWorkflowNames(content string) []string {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil || result.Frontmatter == nil {
		return nil
	}

	safeOutputsMap, ok := result.Frontmatter["safe-outputs"].(map[string]any)
	if !ok {
		return nil
	}

	dispatchWorkflow, exists := safeOutputsMap["dispatch-workflow"]
	if !exists {
		return nil
	}

	var workflowNames []string

	switch v := dispatchWorkflow.(type) {
	case []any:
		// Array format: dispatch-workflow: [name1, name2]
		for _, item := range v {
			if name, ok := item.(string); ok && !strings.Contains(name, "${{") {
				workflowNames = append(workflowNames, name)
			}
		}
	case map[string]any:
		// Map format: dispatch-workflow: {workflows: [name1, name2]}
		if workflowsArray, ok := v["workflows"].([]any); ok {
			for _, item := range workflowsArray {
				if name, ok := item.(string); ok && !strings.Contains(name, "${{") {
					workflowNames = append(workflowNames, name)
				}
			}
		}
	}

	return workflowNames
}

// fetchAndSaveRemoteDispatchWorkflows fetches and saves the workflow files referenced in the
// safe-outputs.dispatch-workflow configuration of a remote workflow. Each listed workflow name
// (without extension) is resolved as a sibling file ("<name>.md") in the same directory as
// the source workflow and downloaded from the same remote repository.
//
// Workflow names that use GitHub Actions expression syntax (e.g. "${{") are silently skipped
// because they are dynamic values that cannot be resolved at add-time.
//
// If a target file already exists from a different source (different owner/repo in its
// 'source:' frontmatter field, or no source field at all), an error is returned.
// Files from the same source are silently skipped. Download failures are non-fatal.
func fetchAndSaveRemoteDispatchWorkflows(content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	if spec.RepoSlug == "" {
		return nil
	}

	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	owner, repo := parts[0], parts[1]
	ref := spec.Version
	if ref == "" {
		defaultBranch, err := getRepoDefaultBranch(spec.RepoSlug)
		if err != nil {
			remoteWorkflowLog.Printf("Failed to resolve default branch for %s, falling back to 'main': %v", spec.RepoSlug, err)
			ref = "main"
		} else {
			ref = defaultBranch
		}
		spec.Version = ref
	}

	workflowNames := extractDispatchWorkflowNames(content)
	if len(workflowNames) == 0 {
		return nil
	}

	// workflowBaseDir is the directory of the source workflow in the remote repo
	// (e.g. ".github/workflows"). Dispatch-workflow names are resolved relative to it.
	workflowBaseDir := getParentDir(spec.WorkflowPath)

	// Pre-compute the absolute target directory for path-traversal boundary checks.
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for target directory %s: %v", targetDir, err)
		return nil
	}

	for _, workflowName := range workflowNames {
		// Build the remote file path for this dispatch workflow
		var remoteFilePath string
		if workflowBaseDir != "" {
			remoteFilePath = path.Join(workflowBaseDir, workflowName+".md")
		} else {
			remoteFilePath = workflowName + ".md"
		}
		remoteFilePath = path.Clean(remoteFilePath)

		// The local path is just the workflow filename in targetDir
		localRelPath := filepath.Clean(workflowName + ".md")
		targetPath := filepath.Join(targetDir, localRelPath)

		// Belt-and-suspenders: verify the resolved path stays inside targetDir
		absTargetPath, absErr := filepath.Abs(targetPath)
		if absErr != nil {
			remoteWorkflowLog.Printf("Failed to resolve absolute path for dispatch workflow %s: %v", workflowName, absErr)
			continue
		}
		if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write dispatch workflow outside target directory: %q", workflowName)))
			}
			continue
		}

		// Check whether the target file already exists.
		fileExists := false
		if _, statErr := os.Stat(targetPath); statErr == nil {
			fileExists = true
			if !force {
				// Allow if the existing file comes from the same source repository.
				existingSourceRepo := readSourceRepoFromFile(targetPath)
				if existingSourceRepo == spec.RepoSlug {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dispatch workflow from same source already exists, skipping: "+targetPath))
					}
					continue
				}
				// Different or missing source — this is a conflict.
				return fmt.Errorf(
					"dispatch workflow %q already exists at %s (existing source: %q, installing from: %q); remove the file or use --force to overwrite",
					workflowName, targetPath, sourceRepoLabel(existingSourceRepo), spec.RepoSlug,
				)
			}
		}

		// Download from the source repository — try .md first, then .yml as fallback
		// (the dispatch-workflow validator accepts either .md or .yml files locally).
		workflowContent, err := parser.DownloadFileFromGitHub(owner, repo, remoteFilePath, ref)
		if err != nil {
			// .md not found — try .yml fallback (e.g. plain GitHub Actions workflow)
			ymlRemotePath := path.Clean(strings.TrimSuffix(remoteFilePath, ".md") + ".yml")
			ymlLocalPath := filepath.Join(targetDir, filepath.Clean(workflowName+".yml"))

			ymlContent, ymlErr := parser.DownloadFileFromGitHub(owner, repo, ymlRemotePath, ref)
			if ymlErr != nil {
				// Neither .md nor .yml found — best-effort, continue
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch dispatch workflow %s: %v", remoteFilePath, err)))
				}
				continue
			}
			// .yml fallback succeeded — write it (no source field for yml)
			if mkErr := os.MkdirAll(filepath.Dir(ymlLocalPath), 0755); mkErr != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", ymlRemotePath, mkErr)))
				}
				continue
			}
			// Capture whether file exists before writing (for correct tracker classification).
			_, ymlFileExistsErr := os.Stat(ymlLocalPath)
			ymlFileExists := ymlFileExistsErr == nil
			if writeErr := os.WriteFile(ymlLocalPath, ymlContent, 0600); writeErr != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", ymlRemotePath, writeErr)))
				}
				continue
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow (.yml): "+ymlLocalPath))
			}
			if tracker != nil {
				if ymlFileExists {
					tracker.TrackModified(ymlLocalPath)
				} else {
					tracker.TrackCreated(ymlLocalPath)
				}
			}
			continue
		}

		// Embed the source field so future adds can detect same-source conflicts.
		depSourceString := spec.RepoSlug + "/" + remoteFilePath + "@" + ref
		if updated, srcErr := addSourceToWorkflow(string(workflowContent), depSourceString); srcErr == nil {
			workflowContent = []byte(updated)
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// Write the file
		if err := os.WriteFile(targetPath, workflowContent, 0600); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", remoteFilePath, err)))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow: "+targetPath))
		}

		// Track the file
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(targetPath)
			} else {
				tracker.TrackCreated(targetPath)
			}
		}
	}

	return nil
}

// extractResources extracts file paths from the top-level "resources" frontmatter field.
// Returns an error if any entry contains GitHub Actions expression syntax (e.g. "${{"),
// since macros are not permitted in resource paths.
func extractResources(content string) ([]string, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to extract frontmatter for resources: %v", err)
		return nil, nil
	}
	if result.Frontmatter == nil {
		return nil, nil
	}

	resourcesField, exists := result.Frontmatter["resources"]
	if !exists {
		return nil, nil
	}

	var paths []string
	switch v := resourcesField.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				paths = append(paths, s)
			}
		}
	case []string:
		paths = v
	}

	// Reject entries that contain GitHub Actions expression syntax — macros are not allowed.
	for _, p := range paths {
		if strings.Contains(p, "${{") {
			return nil, fmt.Errorf("resources entry %q contains GitHub Actions expression syntax (${{) which is not allowed; use static paths only", p)
		}
	}

	return paths, nil
}

// fetchAndSaveRemoteResources fetches files listed in the top-level "resources" frontmatter
// field from the same remote repository and saves them locally. Resources are resolved as
// relative paths from the same directory as the source workflow in the remote repo.
//
// GitHub Actions expression syntax (e.g. "${{") is not allowed in resource paths and will
// cause an error. Download failures for individual files are non-fatal (best-effort).
//
// For Markdown resource files: if the target already exists from a different source repository
// (different 'source:' frontmatter field, or no source field), an error is returned. Files
// from the same source are silently skipped.
// For non-Markdown resource files: if the target already exists and force is false, an error
// is returned regardless of origin (non-markdown files have no source tracking).
func fetchAndSaveRemoteResources(content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	if spec.RepoSlug == "" {
		return nil
	}

	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	owner, repo := parts[0], parts[1]
	ref := spec.Version
	if ref == "" {
		defaultBranch, err := getRepoDefaultBranch(spec.RepoSlug)
		if err != nil {
			remoteWorkflowLog.Printf("Failed to resolve default branch for %s, falling back to 'main': %v", spec.RepoSlug, err)
			ref = "main"
		} else {
			ref = defaultBranch
		}
		spec.Version = ref
	}

	resourcePaths, err := extractResources(content)
	if err != nil {
		return err
	}
	if len(resourcePaths) == 0 {
		return nil
	}

	// Resources are resolved relative to the source workflow's directory in the remote repo.
	workflowBaseDir := getParentDir(spec.WorkflowPath)

	// Pre-compute the absolute target directory for path-traversal boundary checks.
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for target directory %s: %v", targetDir, err)
		return nil
	}

	for _, resourcePath := range resourcePaths {
		// Early rejection of path traversal patterns. This is a fast first-pass check;
		// the filepath.Rel boundary check below is the authoritative security control.
		if strings.Contains(resourcePath, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping resource with unsafe path: %q", resourcePath)))
			}
			continue
		}

		// Resolve the remote file path
		var remoteFilePath string
		if rest, ok := strings.CutPrefix(resourcePath, "/"); ok {
			remoteFilePath = rest
		} else if workflowBaseDir != "" {
			remoteFilePath = path.Join(workflowBaseDir, resourcePath)
		} else {
			remoteFilePath = resourcePath
		}
		remoteFilePath = path.Clean(remoteFilePath)

		// Derive the local relative path by stripping the workflow base dir prefix
		localRelPath := remoteFilePath
		if workflowBaseDir != "" && strings.HasPrefix(remoteFilePath, workflowBaseDir+"/") {
			localRelPath = remoteFilePath[len(workflowBaseDir)+1:]
		}
		localRelPath = filepath.Clean(filepath.FromSlash(localRelPath))
		localRelPath = strings.TrimLeft(localRelPath, string(filepath.Separator))
		if localRelPath == "" || localRelPath == "." {
			continue
		}
		targetPath := filepath.Join(targetDir, localRelPath)

		// Belt-and-suspenders: verify the resolved path stays inside targetDir
		absTargetPath, absErr := filepath.Abs(targetPath)
		if absErr != nil {
			remoteWorkflowLog.Printf("Failed to resolve absolute path for resource %s: %v", resourcePath, absErr)
			continue
		}
		if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write resource outside target directory: %q", resourcePath)))
			}
			continue
		}

		// Check whether the target file already exists.
		fileExists := false
		if _, statErr := os.Stat(targetPath); statErr == nil {
			fileExists = true
			if !force {
				isMarkdown := strings.HasSuffix(strings.ToLower(targetPath), ".md")
				if isMarkdown {
					// For markdown files, allow same-source overwrites.
					existingSourceRepo := readSourceRepoFromFile(targetPath)
					if existingSourceRepo == spec.RepoSlug {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Resource file from same source already exists, skipping: "+targetPath))
						}
						continue
					}
					return fmt.Errorf(
						"resource %q already exists at %s (existing source: %q, installing from: %q); remove the file or use --force to overwrite",
						resourcePath, targetPath, sourceRepoLabel(existingSourceRepo), spec.RepoSlug,
					)
				}
				// Non-markdown files have no source tracking — always conflict.
				return fmt.Errorf(
					"resource %q already exists at %s; remove the file or use --force to overwrite",
					resourcePath, targetPath,
				)
			}
		}

		// Download from source repository
		fileContent, err := parser.DownloadFileFromGitHub(owner, repo, remoteFilePath, ref)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch resource %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// For markdown resources, embed the source field for future conflict detection.
		if strings.HasSuffix(strings.ToLower(remoteFilePath), ".md") {
			depSourceString := spec.RepoSlug + "/" + remoteFilePath + "@" + ref
			if updated, srcErr := addSourceToWorkflow(string(fileContent), depSourceString); srcErr == nil {
				fileContent = []byte(updated)
			}
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for resource %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// Write the file
		if err := os.WriteFile(targetPath, fileContent, 0600); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write resource %s: %v", remoteFilePath, err)))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched resource: "+targetPath))
		}

		// Track the file
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(targetPath)
			} else {
				tracker.TrackCreated(targetPath)
			}
		}
	}

	return nil
}

// fetchAndSaveDispatchWorkflowsFromParsedFile parses a locally-saved workflow file to obtain
// the fully merged safe-outputs configuration (including dispatch workflows that originate
// from imported shared workflows), then fetches any referenced dispatch workflow files that
// don't already exist locally.
//
// This is needed because import-derived dispatch workflows cannot be discovered by static
// frontmatter inspection alone — they only become visible after the compiler processes all
// imports and merges the safe-outputs configuration.
//
// All early returns (empty RepoSlug, invalid slug, parse failure, no dispatch workflows) are
// intentional no-ops: this function is best-effort and must never block the add workflow flow.
// Parse failures are logged at debug level so they can be investigated when needed.
// Source conflicts are reported as warnings (not errors) because the main file is already written.
func fetchAndSaveDispatchWorkflowsFromParsedFile(destFile string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) {
	if spec.RepoSlug == "" {
		return
	}

	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return
	}
	owner, repo := parts[0], parts[1]
	ref := spec.Version
	if ref == "" {
		ref = "main"
	}

	// Parse the locally-saved workflow to get the full merged safe-outputs config.
	compiler := workflow.NewCompiler()
	data, err := compiler.ParseWorkflowFile(destFile)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to parse workflow file %s for import-derived dispatch workflows: %v", destFile, err)
		return
	}
	if data == nil || data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return
	}

	workflowNames := data.SafeOutputs.DispatchWorkflow.Workflows
	if len(workflowNames) == 0 {
		return
	}

	// Filter out GitHub Actions expression syntax
	filtered := make([]string, 0, len(workflowNames))
	for _, name := range workflowNames {
		if !strings.Contains(name, "${{") {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return
	}

	workflowBaseDir := getParentDir(spec.WorkflowPath)

	absTargetDir, absErr := filepath.Abs(targetDir)
	if absErr != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for target directory %s: %v", targetDir, absErr)
		return
	}

	for _, workflowName := range filtered {
		// Early rejection of path traversal patterns (authoritative check is filepath.Rel below).
		if strings.Contains(workflowName, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping dispatch workflow with unsafe name: %q", workflowName)))
			}
			continue
		}

		var remoteFilePath string
		if workflowBaseDir != "" {
			remoteFilePath = path.Join(workflowBaseDir, workflowName+".md")
		} else {
			remoteFilePath = workflowName + ".md"
		}
		remoteFilePath = path.Clean(remoteFilePath)

		localRelPath := filepath.Clean(workflowName + ".md")
		targetPath := filepath.Join(targetDir, localRelPath)

		absTargetPath, absErr2 := filepath.Abs(targetPath)
		if absErr2 != nil {
			remoteWorkflowLog.Printf("Failed to resolve absolute path for dispatch workflow %s: %v", workflowName, absErr2)
			continue
		}
		if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write dispatch workflow outside target directory: %q", workflowName)))
			}
			continue
		}

		// Check whether the target file already exists.
		fileExists := false
		if _, statErr := os.Stat(targetPath); statErr == nil {
			fileExists = true
			if !force {
				existingSourceRepo := readSourceRepoFromFile(targetPath)
				if existingSourceRepo == spec.RepoSlug {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dispatch workflow (from import) from same source already exists, skipping: "+targetPath))
					}
					continue
				}
				// Different or missing source — warn and skip (post-write best-effort).
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(
					"Dispatch workflow %q already exists at %s from a different source (existing: %q, needed: %q); use --force to overwrite",
					workflowName, targetPath, sourceRepoLabel(existingSourceRepo), spec.RepoSlug,
				)))
				continue
			}
		}

		// Download from source repository — try .md first, then .yml as fallback
		workflowContent, err := parser.DownloadFileFromGitHub(owner, repo, remoteFilePath, ref)
		if err != nil {
			// .md not found — try .yml fallback
			ymlRemotePath := path.Clean(strings.TrimSuffix(remoteFilePath, ".md") + ".yml")
			ymlLocalPath := filepath.Join(targetDir, filepath.Clean(workflowName+".yml"))

			ymlContent, ymlErr := parser.DownloadFileFromGitHub(owner, repo, ymlRemotePath, ref)
			if ymlErr != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch dispatch workflow %s: %v", remoteFilePath, err)))
				}
				continue
			}
			if mkErr := os.MkdirAll(filepath.Dir(ymlLocalPath), 0755); mkErr != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", ymlRemotePath, mkErr)))
				}
				continue
			}
			// Capture whether file exists before writing (for correct tracker classification).
			_, ymlFileExistsErr := os.Stat(ymlLocalPath)
			ymlFileExists := ymlFileExistsErr == nil
			if writeErr := os.WriteFile(ymlLocalPath, ymlContent, 0600); writeErr != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", ymlRemotePath, writeErr)))
				}
				continue
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow (.yml, from import): "+ymlLocalPath))
			}
			if tracker != nil {
				if ymlFileExists {
					tracker.TrackModified(ymlLocalPath)
				} else {
					tracker.TrackCreated(ymlLocalPath)
				}
			}
			continue
		}

		// Embed the source field for future conflict detection.
		depSourceString := spec.RepoSlug + "/" + remoteFilePath + "@" + ref
		if updated, srcErr := addSourceToWorkflow(string(workflowContent), depSourceString); srcErr == nil {
			workflowContent = []byte(updated)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", remoteFilePath, err)))
			}
			continue
		}

		if err := os.WriteFile(targetPath, workflowContent, 0600); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", remoteFilePath, err)))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow (from import): "+targetPath))
		}

		if tracker != nil {
			if fileExists {
				tracker.TrackModified(targetPath)
			} else {
				tracker.TrackCreated(targetPath)
			}
		}
	}
}
