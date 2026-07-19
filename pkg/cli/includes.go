package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/setutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
)

// includeDirectivePattern matches @include or @include? directives with their path argument
var includeDirectivePattern = regexp.MustCompile(`^@include(\?)?\s+(.+)$`)
var downloadRemoteImportFile = parser.DownloadFileFromGitHub

// FetchIncludeFromSource fetches an include file from GitHub directly using a workflowspec format path.
// The includePath should be in the format: owner/repo/path/to/file.md[@ref]
// If the includePath is a relative path, it's resolved relative to the baseSpec.
// Returns: (content, section, error) where section is the #fragment from the path (e.g., "#section-name").
func FetchIncludeFromSource(ctx context.Context, includePath string, baseSpec *WorkflowSpec, verbose bool) ([]byte, string, error) {
	baseSpecStr := "<nil>"
	if baseSpec != nil {
		baseSpecStr = baseSpec.String()
	}
	remoteWorkflowLog.Printf("Fetching include from source: path=%s, base=%s", includePath, baseSpecStr)

	// Extract section reference (e.g., "#section-name") from the path upfront
	// This ensures consistent behavior regardless of which code path is taken
	cleanPath, section := fetchIncludeFromSourceSplitSection(includePath)

	// Check if this is a workflowspec format (owner/repo/path[@ref])
	if isWorkflowSpecFormat(cleanPath) {
		return fetchIncludeFromSourceWorkflowSpec(ctx, includePath, cleanPath, section)
	}

	// For relative paths, resolve against the base spec
	if baseSpec != nil && baseSpec.RepoSlug != "" {
		if content, ok, err := fetchIncludeFromSourceRelative(ctx, cleanPath, baseSpec); ok || err != nil {
			return content, section, err
		}
	}

	return nil, section, fmt.Errorf("cannot resolve include path: %s (no base spec provided)", includePath)
}

func fetchIncludeFromSourceSplitSection(includePath string) (string, string) {
	cleanPath := includePath
	var section string
	if idx := strings.Index(includePath, "#"); idx != -1 {
		cleanPath = includePath[:idx]
		section = includePath[idx:]
	}
	return cleanPath, section
}

func fetchIncludeFromSourceWorkflowSpec(ctx context.Context, includePath, cleanPath, section string) ([]byte, string, error) {
	// Split on @ to get path and ref
	parts := strings.SplitN(cleanPath, "@", 2)
	pathPart := parts[0]
	ref := "main"
	if len(parts) == 2 {
		ref = parts[1]
	}

	// Parse path: owner/repo/path/to/file.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		return nil, section, errors.New("invalid workflowspec: must be owner/repo/path[@ref]")
	}

	owner := slashParts[0]
	repo := slashParts[1]
	filePath := strings.Join(slashParts[2:], "/")
	content, err := parser.DownloadFileFromGitHub(ctx, owner, repo, filePath, ref)
	if err != nil {
		return nil, section, fmt.Errorf("failed to fetch include from %s: %w", includePath, err)
	}
	return content, section, nil
}

func fetchIncludeFromSourceRelative(ctx context.Context, cleanPath string, baseSpec *WorkflowSpec) ([]byte, bool, error) {
	parts := strings.SplitN(baseSpec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil, false, nil
	}
	owner, repo := parts[0], parts[1]
	ref := baseSpec.Version
	if ref == "" {
		ref = "main"
	}

	filePath := cleanPath
	if idx := strings.Index(filePath, "@"); idx != -1 {
		filePath = filePath[:idx]
	}
	fullPath := fetchIncludeFromSourceRelativeFullPath(filePath, baseSpec.WorkflowPath)
	content, err := parser.DownloadFileFromGitHub(ctx, owner, repo, fullPath, ref)
	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch include %s from %s/%s: %w", filePath, owner, repo, err)
	}
	return content, true, nil
}

func fetchIncludeFromSourceRelativeFullPath(filePath, workflowPath string) string {
	if strings.HasPrefix(filePath, "shared/") {
		return constants.GithubDir + filePath
	}
	baseDir := getParentDir(workflowPath)
	if baseDir != "" {
		return baseDir + "/" + filePath
	}
	return filePath
}

// fetchAndSaveRemoteFrontmatterImports fetches and saves files referenced in the frontmatter
// 'imports:' field of a remote workflow. These relative-path imports are resolved against
// the workflow's location in the source repository and saved locally so compilation can find them.
// This is analogous to fetchAndSaveRemoteIncludes, which handles @include directives in the
// markdown body; this function handles the YAML frontmatter 'imports:' field.
// Import failures are non-fatal (best-effort); the compiler will report any still-missing files.
func fetchAndSaveRemoteFrontmatterImports(ctx context.Context, content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	if spec.RepoSlug == "" {
		return nil
	}

	remoteWorkflowLog.Printf("Fetching frontmatter imports for workflow: repo=%s, path=%s", spec.RepoSlug, spec.WorkflowPath)

	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	owner, repo := parts[0], parts[1]
	ref := spec.Version
	if ref == "" {
		// Resolve the actual default branch of the source repo rather than assuming "main"
		defaultBranch, err := getRepoDefaultBranch(ctx, spec.RepoSlug)
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
	seen := make(map[string]struct {
	})
	fetchFrontmatterImportsRecursive(ctx, content, workflowBaseDir, frontmatterImportsOpts{
		owner:           owner,
		repo:            repo,
		ref:             ref,
		originalBaseDir: workflowBaseDir,
		targetDir:       targetDir,
		verbose:         verbose,
		force:           force,
		tracker:         tracker,
		seen:            seen,
	})
	return nil
}

// frontmatterImportsOpts holds the constant parameters for fetchFrontmatterImportsRecursive.
// Only `content` and `currentBaseDir` change per recursion level; everything else is constant.
type frontmatterImportsOpts struct {
	owner           string
	repo            string
	ref             string
	originalBaseDir string
	targetDir       string
	verbose         bool
	force           bool
	tracker         *FileTracker
	seen            map[string]struct{}
	// downloadFn is the function used to fetch file content from the source repository.
	// When nil, parser.DownloadFileFromGitHub is used. Tests may inject a stub to avoid
	// network calls and observe which paths were requested.
	downloadFn func(ctx context.Context, owner, repo, path, ref string) ([]byte, error)
}

// fetchFrontmatterImportsRecursive is the internal worker for fetchAndSaveRemoteFrontmatterImports.
//
// Parameters that change per recursion level:
//   - content: the text of the file whose imports are being processed
//   - currentBaseDir: directory of that file inside the source repo (used to resolve relative paths)
//
// Parameters that remain constant across all recursion levels (in opts):
//   - owner, repo, ref: source repository coordinates
//   - originalBaseDir: directory of the top-level workflow (used to map remote paths → local paths)
//   - targetDir: the `.github/workflows` directory in the user's repo
//   - seen: shared visited set (keyed by fully-resolved remote path) — prevents cycles & duplicates
func fetchFrontmatterImportsRecursive(ctx context.Context, content, currentBaseDir string, opts frontmatterImportsOpts) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil || result.Frontmatter == nil {
		return
	}

	importPaths := fetchFrontmatterImportsRecursivePaths(result.Frontmatter)
	if len(importPaths) == 0 {
		return
	}

	remoteWorkflowLog.Printf("Processing %d frontmatter imports recursively: owner=%s, repo=%s, ref=%s", len(importPaths), opts.owner, opts.repo, opts.ref)

	// Pre-compute the absolute target directory once for path-traversal boundary checks.
	absTargetDir, err := filepath.Abs(opts.targetDir)
	if err != nil {
		return
	}

	for _, importPath := range importPaths {
		remoteFilePath, ok := fetchFrontmatterImportsRecursiveRemotePath(importPath, currentBaseDir, opts)
		if !ok {
			continue
		}

		// Cycle/duplicate prevention: use the fully-resolved remote path as the key.
		if setutil.Contains(opts.seen, remoteFilePath) {
			remoteWorkflowLog.Printf("Skipping already-seen import: %s", remoteFilePath)
			continue
		}
		opts.seen[remoteFilePath] = struct{}{}

		targetPath, ok := fetchFrontmatterImportsRecursiveTargetPath(remoteFilePath, importPath, absTargetDir, opts)
		if !ok {
			continue
		}
		fileExists, skip := fetchFrontmatterImportsRecursiveHandleExisting(ctx, targetPath, remoteFilePath, opts)
		if skip {
			continue
		}

		importContent, ok := fetchFrontmatterImportsRecursiveDownloadAndWrite(ctx, targetPath, remoteFilePath, opts)
		if !ok {
			continue
		}
		fetchFrontmatterImportsRecursiveTrack(targetPath, fileExists, opts.tracker)
		importedBaseDir := path.Dir(remoteFilePath)
		fetchFrontmatterImportsRecursive(ctx, string(importContent), importedBaseDir, opts)
	}
}

func fetchFrontmatterImportsRecursivePaths(frontmatter map[string]any) []string {
	importsField, exists := frontmatter["imports"]
	if !exists {
		return nil
	}
	switch v := importsField.(type) {
	case []any:
		return fetchFrontmatterImportsRecursiveAnyPaths(v)
	case []string:
		return v
	default:
		return nil
	}
}

func fetchFrontmatterImportsRecursiveAnyPaths(items []any) []string {
	var importPaths []string
	for _, item := range items {
		switch importItem := item.(type) {
		case string:
			importPaths = append(importPaths, importItem)
		case map[string]any:
			// Handle uses: and path: forms (mirrors GitHub Actions reusable workflow syntax)
			if usesVal, ok := importItem["uses"]; ok {
				if p, ok := usesVal.(string); ok {
					importPaths = append(importPaths, p)
				}
			} else if pathVal, ok := importItem["path"]; ok {
				if p, ok := pathVal.(string); ok {
					importPaths = append(importPaths, p)
				}
			}
		}
	}
	return importPaths
}

func fetchFrontmatterImportsRecursiveRemotePath(importPath, currentBaseDir string, opts frontmatterImportsOpts) (string, bool) {
	if isWorkflowSpecFormat(importPath) {
		return "", false
	}
	filePath := importPath
	if before, _, hasSec := strings.Cut(importPath, "#"); hasSec {
		filePath = before
	}
	if filePath == "" {
		return "", false
	}
	remoteFilePath := fetchFrontmatterImportsRecursiveResolveRemote(filePath, currentBaseDir, opts.originalBaseDir)
	remoteFilePath = path.Clean(remoteFilePath)
	if remoteFilePath == ".." || strings.HasPrefix(remoteFilePath, "../") {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping import with unsafe path: %q", importPath)))
		}
		return "", false
	}
	return remoteFilePath, true
}

func fetchFrontmatterImportsRecursiveResolveRemote(filePath, currentBaseDir, originalBaseDir string) string {
	// Use path (not filepath) because this is always a forward-slash URL/API path.
	if rest, ok := strings.CutPrefix(filePath, "/"); ok {
		return rest
	}
	if strings.HasPrefix(filePath, "./") || strings.HasPrefix(filePath, "../") {
		if currentBaseDir != "" {
			return path.Join(currentBaseDir, filePath)
		}
		return filePath
	}
	baseDir := originalBaseDir
	if !strings.Contains(filePath, "/") {
		baseDir = currentBaseDir
	}
	if baseDir != "" {
		return path.Join(baseDir, filePath)
	}
	return filePath
}

func fetchFrontmatterImportsRecursiveTargetPath(remoteFilePath, importPath, absTargetDir string, opts frontmatterImportsOpts) (string, bool) {
	localRelPath := remoteFilePath
	if opts.originalBaseDir != "" && strings.HasPrefix(remoteFilePath, opts.originalBaseDir+"/") {
		localRelPath = remoteFilePath[len(opts.originalBaseDir)+1:]
	}
	localRelPath = filepath.Clean(filepath.FromSlash(localRelPath))
	localRelPath = strings.TrimLeft(localRelPath, string(filepath.Separator))
	if localRelPath == "" || localRelPath == "." {
		return "", false
	}
	targetPath := filepath.Join(opts.targetDir, localRelPath)
	absTargetPath, absErr := filepath.Abs(targetPath)
	if absErr != nil {
		return "", false
	}
	if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write import outside target directory: %q", importPath)))
		}
		return "", false
	}
	return targetPath, true
}

func fetchFrontmatterImportsRecursiveHandleExisting(ctx context.Context, targetPath, remoteFilePath string, opts frontmatterImportsOpts) (bool, bool) {
	if !fileutil.FileExists(targetPath) {
		return false, false
	}
	if opts.force {
		return true, false
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Import file already exists, skipping: "+targetPath))
	}
	if existingContent, readErr := os.ReadFile(targetPath); readErr == nil {
		fetchFrontmatterImportsRecursive(ctx, string(existingContent), path.Dir(remoteFilePath), opts)
	} else {
		remoteWorkflowLog.Printf("Failed to read existing import %s for recursion: %v", targetPath, readErr)
	}
	return true, true
}

func fetchFrontmatterImportsRecursiveDownloadAndWrite(ctx context.Context, targetPath, remoteFilePath string, opts frontmatterImportsOpts) ([]byte, bool) {
	downloadFn := opts.downloadFn
	if downloadFn == nil {
		downloadFn = downloadRemoteImportFile
	}
	importContent, err := downloadFn(ctx, opts.owner, opts.repo, remoteFilePath, opts.ref)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to download import %s from %s/%s@%s: %v", remoteFilePath, opts.owner, opts.repo, opts.ref, err)
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch import %s: %v", remoteFilePath, err)))
		}
		return nil, false
	}
	if err := fetchFrontmatterImportsRecursiveWriteFile(targetPath, remoteFilePath, importContent, opts.verbose); err != nil {
		return nil, false
	}
	return importContent, true
}

func fetchFrontmatterImportsRecursiveWriteFile(targetPath, remoteFilePath string, importContent []byte, verbose bool) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermPublic); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for import %s: %v", remoteFilePath, err)))
		}
		return err
	}
	if err := os.WriteFile(targetPath, importContent, constants.FilePermSensitive); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write import %s: %v", remoteFilePath, err)))
		}
		return err
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched import: "+targetPath))
	}
	return nil
}

func fetchFrontmatterImportsRecursiveTrack(targetPath string, fileExists bool, tracker *FileTracker) {
	if tracker == nil {
		return
	}
	if fileExists {
		tracker.TrackModified(targetPath)
	} else {
		tracker.TrackCreated(targetPath)
	}
}

// fetchAndSaveRemoteIncludes parses the workflow content for @include directives and fetches them from the remote source
func fetchAndSaveRemoteIncludes(ctx context.Context, content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	remoteWorkflowLog.Printf("Fetching remote includes for workflow: %s", spec.String())

	// Parse the workflow content to find @include directives
	scanner := bufio.NewScanner(strings.NewReader(content))
	seen := make(map[string]struct {
	})

	for scanner.Scan() {
		line := scanner.Text()
		matches := includeDirectivePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		if err := fetchAndSaveRemoteIncludesDirective(ctx, matches, spec, targetDir, verbose, force, tracker, seen); err != nil {
			return err
		}
	}

	return nil
}

func fetchAndSaveRemoteIncludesDirective(
	ctx context.Context,
	matches []string,
	spec *WorkflowSpec,
	targetDir string,
	verbose bool,
	force bool,
	tracker *FileTracker,
	seen map[string]struct{},
) error {
	isOptional := matches[1] == "?"
	includePath := strings.TrimSpace(matches[2])
	filePath := fetchAndSaveRemoteIncludesFilePath(includePath)
	if setutil.Contains(seen, filePath) {
		return nil
	}
	seen[filePath] = struct{}{}

	includeContent, _, err := FetchIncludeFromSource(ctx, includePath, spec, verbose)
	if err != nil {
		if isOptional {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Optional include not found: "+includePath))
			}
			return nil
		}
		return fmt.Errorf("failed to fetch include %s: %w", includePath, err)
	}

	targetPath := fetchAndSaveRemoteIncludesTargetPath(filePath, targetDir)
	fileExists, skip, err := fetchAndSaveRemoteIncludesWrite(targetPath, includeContent, verbose, force)
	if err != nil || skip {
		return err
	}
	fetchAndSaveRemoteIncludesTrack(targetPath, fileExists, tracker)
	if err := fetchAndSaveRemoteIncludes(ctx, string(includeContent), spec, targetDir, verbose, force, tracker); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch nested includes from %s: %v", filePath, err)))
		}
	}
	return nil
}

func fetchAndSaveRemoteIncludesFilePath(includePath string) string {
	filePath := includePath
	if before, _, ok := strings.Cut(includePath, "#"); ok {
		filePath = before
	}
	return filePath
}

func fetchAndSaveRemoteIncludesTargetPath(filePath, targetDir string) string {
	if strings.HasPrefix(filePath, "shared/") {
		// shared/ files go to .github/shared/
		return filepath.Join(filepath.Dir(targetDir), filePath)
	}
	if isWorkflowSpecFormat(filePath) {
		// Workflowspec includes: extract just the filename and put in shared/
		parts := strings.Split(filePath, "/")
		filename := parts[len(parts)-1]
		return filepath.Join(filepath.Dir(targetDir), "shared", filename)
	}
	// Relative includes go alongside the workflow
	return filepath.Join(targetDir, filePath)
}

func fetchAndSaveRemoteIncludesWrite(targetPath string, includeContent []byte, verbose bool, force bool) (bool, bool, error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermPublic); err != nil {
		return false, false, fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
	}
	fileExists := false
	if fileutil.FileExists(targetPath) {
		fileExists = true
		if !force {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Include file already exists, skipping: "+targetPath))
			}
			return fileExists, true, nil
		}
	}
	if err := os.WriteFile(targetPath, includeContent, constants.FilePermSensitive); err != nil {
		return false, false, fmt.Errorf("failed to write include file %s: %w", targetPath, err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched include: "+targetPath))
	}
	return fileExists, false, nil
}

func fetchAndSaveRemoteIncludesTrack(targetPath string, fileExists bool, tracker *FileTracker) {
	if tracker == nil {
		return
	}
	if fileExists {
		tracker.TrackModified(targetPath)
	} else {
		tracker.TrackCreated(targetPath)
	}
}

// fetchAllRemoteDependencies fetches all remote dependencies for a workflow:
// includes (@include directives), frontmatter imports, dispatch workflows, and resources.
// This is the single entry point shared by both the add and trial commands.
//
// Error handling is intentionally asymmetric:
//   - @include and frontmatter import errors are best-effort: failures emit a warning when
//     verbose is true but do not stop the overall operation.
//   - Dispatch-workflow and resource errors are fatal and are returned to the caller.
func fetchAllRemoteDependencies(ctx context.Context, content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	remoteWorkflowLog.Printf("Fetching all remote dependencies: spec=%s, targetDir=%s, force=%v", spec.String(), targetDir, force)
	// Fetch and save @include directive dependencies (best-effort: errors are not fatal).
	if err := fetchAndSaveRemoteIncludes(ctx, content, spec, targetDir, verbose, force, tracker); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch include dependencies: %v", err)))
		}
	}
	// Fetch and save frontmatter 'imports:' dependencies so they are available
	// locally during compilation. Keeping these as relative paths (not workflowspecs)
	// ensures the compiler resolves them from disk rather than downloading from GitHub.
	// Best-effort: errors are not fatal.
	if err := fetchAndSaveRemoteFrontmatterImports(ctx, content, spec, targetDir, verbose, force, tracker); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch frontmatter import dependencies: %v", err)))
		}
	}
	// Fetch and save workflows referenced in safe-outputs.dispatch-workflow so they are
	// available locally. Workflow names using GitHub Actions expression syntax are skipped.
	if err := fetchAndSaveRemoteDispatchWorkflows(ctx, content, spec, targetDir, verbose, force, tracker); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch dispatch workflow dependencies: %v", err)))
		}
		return fmt.Errorf("failed to fetch dispatch workflow dependencies: %w", err)
	}
	// Fetch files listed in the 'resources:' frontmatter field (additional workflow or
	// action files that should be present alongside this workflow).
	if err := fetchAndSaveRemoteResources(ctx, content, spec, targetDir, verbose, force, tracker); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch resource dependencies: %v", err)))
		}
		return fmt.Errorf("failed to fetch resource dependencies: %w", err)
	}
	return nil
}
