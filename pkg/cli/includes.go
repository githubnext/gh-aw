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
		content, err := parser.DownloadFileFromGitHub(ctx, owner, repo, filePath, ref)
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
				fullPath = constants.GithubDir + filePath
			} else {
				// Otherwise, resolve relative to the workflow path directory
				baseDir := getParentDir(baseSpec.WorkflowPath)
				if baseDir != "" {
					fullPath = baseDir + "/" + filePath
				} else {
					fullPath = filePath
				}
			}

			content, err := parser.DownloadFileFromGitHub(ctx, owner, repo, fullPath, ref)
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

	importsField, exists := result.Frontmatter["imports"]
	if !exists {
		return
	}

	var importPaths []string
	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
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
	case []string:
		importPaths = v
	}

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

		// Resolve the remote file path to an absolute repo path.
		// Use path (not filepath) because this is always a forward-slash URL/API path.
		var remoteFilePath string
		if rest, ok := strings.CutPrefix(filePath, "/"); ok {
			// Absolute path from repo root (e.g. "/scripts/helper.md")
			remoteFilePath = rest
		} else if strings.HasPrefix(filePath, "./") || strings.HasPrefix(filePath, "../") {
			// Explicitly-relative path (e.g. "./serena.md"): resolve relative to the
			// current importing file's directory so that sibling-file references work
			// correctly regardless of nesting depth.
			if currentBaseDir != "" {
				remoteFilePath = path.Join(currentBaseDir, filePath)
			} else {
				remoteFilePath = filePath
			}
		} else {
			// Non-explicit relative path: resolution strategy mirrors the compiler's
			// determineNestedBaseDir logic — it depends on whether the path contains a
			// "/" separator.
			//
			// Paths WITHOUT "/" (e.g. "helper.md"): the importing file treats them as
			// siblings in its own directory, so resolve relative to currentBaseDir.
			// Example: "control-precompute.md" from ".github/workflows/shared/control.md"
			//          → ".github/workflows/shared/control-precompute.md"
			//
			// Paths WITH "/" (e.g. "shared/foo.md"): workflows in this repository write
			// these paths relative to the workflow root (.github/workflows), not relative to
			// the importing file's directory, so resolve against originalBaseDir.
			// Example: "shared/reporting.md" from ".github/workflows/shared/daily-audit-base.md"
			//          → ".github/workflows/shared/reporting.md"
			var baseDir string
			if !strings.Contains(filePath, "/") {
				baseDir = currentBaseDir
			} else {
				baseDir = opts.originalBaseDir
			}
			if baseDir != "" {
				remoteFilePath = path.Join(baseDir, filePath)
			} else {
				remoteFilePath = filePath
			}
		}
		remoteFilePath = path.Clean(remoteFilePath)

		// Reject paths that try to escape the repository root (e.g. "../../etc/passwd")
		if remoteFilePath == ".." || strings.HasPrefix(remoteFilePath, "../") {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping import with unsafe path: %q", importPath)))
			}
			continue
		}

		// Cycle/duplicate prevention: use the fully-resolved remote path as the key.
		if setutil.Contains(opts.seen, remoteFilePath) {
			remoteWorkflowLog.Printf("Skipping already-seen import: %s", remoteFilePath)
			continue
		}
		opts.seen[remoteFilePath] = struct{}{}

		// Derive the local path relative to targetDir by stripping the original base-dir
		// prefix from the remote path. This ensures that imports in nested files resolve
		// to the correct location regardless of how many levels deep the recursion goes.
		//
		// Example: originalBaseDir=".github/workflows"
		//   remoteFilePath=".github/workflows/shared/analysis.md" → localRelPath="shared/analysis.md"
		//   (nested) remoteFilePath=".github/workflows/other.md"  → localRelPath="other.md"
		var localRelPath string
		if opts.originalBaseDir != "" && strings.HasPrefix(remoteFilePath, opts.originalBaseDir+"/") {
			localRelPath = remoteFilePath[len(opts.originalBaseDir)+1:]
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
		targetPath := filepath.Join(opts.targetDir, localRelPath)

		// Belt-and-suspenders: verify the resolved path is inside targetDir
		absTargetPath, absErr := filepath.Abs(targetPath)
		if absErr != nil {
			continue
		}
		if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write import outside target directory: %q", importPath)))
			}
			continue
		}

		// Check existence before downloading: if the file already exists and force=false,
		// skip the download entirely (no unnecessary network round-trip). However, still
		// recurse into the existing file's imports so that any transitive dependencies
		// it references are fetched even when the parent file was already present.
		fileExists := false
		if fileutil.FileExists(targetPath) {
			fileExists = true
			if !opts.force {
				if opts.verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Import file already exists, skipping: "+targetPath))
				}
				// Read the existing file so we can recurse into its own imports.
				// If the read fails, log it and skip — the compiler will report
				// any missing transitive dependencies.
				if existingContent, readErr := os.ReadFile(targetPath); readErr == nil {
					importedBaseDir := path.Dir(remoteFilePath)
					fetchFrontmatterImportsRecursive(ctx, string(existingContent), importedBaseDir, opts)
				} else {
					remoteWorkflowLog.Printf("Failed to read existing import %s for recursion: %v", targetPath, readErr)
				}
				continue
			}
		}

		// Download from the source repository
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
			continue
		}

		// Create the parent directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermPublic); err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for import %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// Write the file
		if err := os.WriteFile(targetPath, importContent, constants.FilePermSensitive); err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write import %s: %v", remoteFilePath, err)))
			}
			continue
		}

		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched import: "+targetPath))
		}

		// Track the file for git staging and potential rollback
		if opts.tracker != nil {
			if fileExists {
				opts.tracker.TrackModified(targetPath)
			} else {
				opts.tracker.TrackCreated(targetPath)
			}
		}

		// Recurse into the imported file's imports. Use the imported file's directory as
		// currentBaseDir so that relative paths inside it resolve correctly.
		importedBaseDir := path.Dir(remoteFilePath)
		fetchFrontmatterImportsRecursive(ctx, string(importContent), importedBaseDir, opts)
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

		isOptional := matches[1] == "?"
		includePath := strings.TrimSpace(matches[2])

		// Remove section reference for file fetching
		filePath := includePath
		if before, _, ok := strings.Cut(includePath, "#"); ok {
			filePath = before
		}

		// Skip if already processed
		if setutil.Contains(seen, filePath) {
			continue
		}
		seen[filePath] = struct {
		}{}

		// Fetch the include file
		includeContent, _, err := FetchIncludeFromSource(ctx, includePath, spec, verbose)
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
		if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermPublic); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Check if file already exists
		fileExists := false
		if fileutil.FileExists(targetPath) {
			fileExists = true
			if !force {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Include file already exists, skipping: "+targetPath))
				}
				continue
			}
		}

		// Write the include file
		if err := os.WriteFile(targetPath, includeContent, constants.FilePermSensitive); err != nil {
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
		if err := fetchAndSaveRemoteIncludes(ctx, string(includeContent), spec, targetDir, verbose, force, tracker); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch nested includes from %s: %v", filePath, err)))
			}
		}
	}

	return nil
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
