package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

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

// fileDownloadFn is the type for a function that downloads a file from a GitHub repository.
// It is used for dependency injection in fetchAndSaveRemoteDispatchWorkflows to allow tests
// to provide a fast-failing mock instead of making real network calls.
type fileDownloadFn func(ctx context.Context, owner, repo, path, ref string) ([]byte, error)

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
//
// An optional downloader function may be provided as the last argument to override the default
// parser.DownloadFileFromGitHub implementation (used in tests to avoid real network calls).
func fetchAndSaveRemoteDispatchWorkflows(ctx context.Context, content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker, downloaders ...fileDownloadFn) error {
	remoteWorkflowLog.Printf("Fetching remote dispatch workflows: repo=%s, targetDir=%s, force=%v", spec.RepoSlug, targetDir, force)
	downloader := fetchAndSaveRemoteDispatchWorkflowsDownloader(downloaders)
	owner, repo, ref, ok := fetchAndSaveRemoteDispatchWorkflowsRepoRef(ctx, spec)
	if !ok {
		return nil
	}

	workflowNames := extractDispatchWorkflowNames(content)
	if len(workflowNames) == 0 {
		return nil
	}
	remoteWorkflowLog.Printf("Found %d dispatch workflow(s) to fetch from %s@%s", len(workflowNames), spec.RepoSlug, ref)

	workflowBaseDir := getParentDir(spec.WorkflowPath)
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for target directory %s: %v", targetDir, err)
		return nil
	}

	for _, workflowName := range workflowNames {
		if err := fetchAndSaveRemoteDispatchWorkflowsOne(fetchAndSaveRemoteDispatchWorkflowsOneParams{
			Ctx:             ctx,
			WorkflowName:    workflowName,
			Spec:            spec,
			TargetDir:       targetDir,
			Verbose:         verbose,
			Force:           force,
			Tracker:         tracker,
			Downloader:      downloader,
			Owner:           owner,
			Repo:            repo,
			Ref:             ref,
			WorkflowBaseDir: workflowBaseDir,
			AbsTargetDir:    absTargetDir,
		}); err != nil {
			return err
		}
	}
	return nil
}

func fetchAndSaveRemoteDispatchWorkflowsDownloader(downloaders []fileDownloadFn) fileDownloadFn {
	downloader := fileDownloadFn(parser.DownloadFileFromGitHub)
	if len(downloaders) > 0 && downloaders[0] != nil {
		downloader = downloaders[0]
	}
	return downloader
}

func fetchAndSaveRemoteDispatchWorkflowsRepoRef(ctx context.Context, spec *WorkflowSpec) (string, string, string, bool) {
	if spec.RepoSlug == "" {
		return "", "", "", false
	}
	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}
	ref := spec.Version
	if ref == "" {
		defaultBranch, err := getRepoDefaultBranch(ctx, spec.RepoSlug)
		if err != nil {
			remoteWorkflowLog.Printf("Failed to resolve default branch for %s, falling back to 'main': %v", spec.RepoSlug, err)
			ref = "main"
		} else {
			ref = defaultBranch
		}
		spec.Version = ref
	}
	return parts[0], parts[1], ref, true
}

type fetchAndSaveRemoteDispatchWorkflowsOneParams struct {
	Ctx             context.Context
	WorkflowName    string
	Spec            *WorkflowSpec
	TargetDir       string
	Verbose         bool
	Force           bool
	Tracker         *FileTracker
	Downloader      fileDownloadFn
	Owner           string
	Repo            string
	Ref             string
	WorkflowBaseDir string
	AbsTargetDir    string
}

func fetchAndSaveRemoteDispatchWorkflowsOne(p fetchAndSaveRemoteDispatchWorkflowsOneParams) error {
	remoteFilePath, targetPath, ok := fetchAndSaveRemoteDispatchWorkflowsPaths(p.WorkflowName, p.WorkflowBaseDir, p.TargetDir, p.AbsTargetDir, p.Verbose)
	if !ok {
		return nil
	}
	fileExists, skip, err := fetchAndSaveRemoteDispatchWorkflowsExisting(p.WorkflowName, targetPath, p.Spec.RepoSlug, p.Verbose, p.Force)
	if err != nil || skip {
		return err
	}

	workflowContent, err := p.Downloader(p.Ctx, p.Owner, p.Repo, remoteFilePath, p.Ref)
	if err != nil {
		fetchAndSaveRemoteDispatchWorkflowsYMLFallback(fetchAndSaveRemoteDispatchWorkflowsYMLFallbackParams{
			Ctx:            p.Ctx,
			WorkflowName:   p.WorkflowName,
			RemoteFilePath: remoteFilePath,
			TargetDir:      p.TargetDir,
			Verbose:        p.Verbose,
			Tracker:        p.Tracker,
			Downloader:     p.Downloader,
			Owner:          p.Owner,
			Repo:           p.Repo,
			Ref:            p.Ref,
			OriginalErr:    err,
		})
		return nil
	}
	return fetchAndSaveRemoteDispatchWorkflowsWriteMarkdown(fetchAndSaveRemoteDispatchWorkflowsWriteMarkdownParams{
		Ctx:             p.Ctx,
		WorkflowContent: workflowContent,
		Spec:            p.Spec,
		RemoteFilePath:  remoteFilePath,
		TargetPath:      targetPath,
		TargetDir:       p.TargetDir,
		Verbose:         p.Verbose,
		Force:           p.Force,
		Tracker:         p.Tracker,
		FileExists:      fileExists,
		Ref:             p.Ref,
	})
}

func fetchAndSaveRemoteDispatchWorkflowsPaths(workflowName, workflowBaseDir, targetDir, absTargetDir string, verbose bool) (string, string, bool) {
	var remoteFilePath string
	if workflowBaseDir != "" {
		remoteFilePath = path.Join(workflowBaseDir, workflowName+".md")
	} else {
		remoteFilePath = workflowName + ".md"
	}
	remoteFilePath = path.Clean(remoteFilePath)
	targetPath := filepath.Join(targetDir, filepath.Clean(workflowName+".md"))
	absTargetPath, absErr := filepath.Abs(targetPath)
	if absErr != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for dispatch workflow %s: %v", workflowName, absErr)
		return "", "", false
	}
	if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write dispatch workflow outside target directory: %q", workflowName)))
		}
		return "", "", false
	}
	return remoteFilePath, targetPath, true
}

func fetchAndSaveRemoteDispatchWorkflowsExisting(workflowName, targetPath, repoSlug string, verbose bool, force bool) (bool, bool, error) {
	if _, statErr := os.Stat(targetPath); statErr != nil {
		return false, false, nil
	}
	if force {
		return true, false, nil
	}
	existingSourceRepo := readSourceRepoFromFile(targetPath)
	if existingSourceRepo == repoSlug {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dispatch workflow from same source already exists, skipping: "+targetPath))
		}
		return true, true, nil
	}
	return true, false, fmt.Errorf(
		"dispatch workflow %q already exists at %s (existing source: %q, installing from: %q); remove the file or use --force to overwrite",
		workflowName, targetPath, sourceRepoLabel(existingSourceRepo), repoSlug,
	)
}

type fetchAndSaveRemoteDispatchWorkflowsYMLFallbackParams struct {
	Ctx            context.Context
	WorkflowName   string
	RemoteFilePath string
	TargetDir      string
	Verbose        bool
	Tracker        *FileTracker
	Downloader     fileDownloadFn
	Owner          string
	Repo           string
	Ref            string
	OriginalErr    error
}

func fetchAndSaveRemoteDispatchWorkflowsYMLFallback(p fetchAndSaveRemoteDispatchWorkflowsYMLFallbackParams) {
	remoteWorkflowLog.Printf(".md fetch failed for dispatch workflow %s, trying .yml fallback", p.WorkflowName)
	ymlRemotePath := path.Clean(strings.TrimSuffix(p.RemoteFilePath, ".md") + ".yml")
	ymlLocalPath := filepath.Join(p.TargetDir, filepath.Clean(p.WorkflowName+".yml"))
	ymlContent, ymlErr := p.Downloader(p.Ctx, p.Owner, p.Repo, ymlRemotePath, p.Ref)
	if ymlErr != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch dispatch workflow %s: %v", p.RemoteFilePath, p.OriginalErr)))
		}
		return
	}
	fetchAndSaveRemoteDispatchWorkflowsWriteYML(ymlRemotePath, ymlLocalPath, ymlContent, p.Verbose, p.Tracker)
}

func fetchAndSaveRemoteDispatchWorkflowsWriteYML(ymlRemotePath, ymlLocalPath string, ymlContent []byte, verbose bool, tracker *FileTracker) {
	if mkErr := os.MkdirAll(filepath.Dir(ymlLocalPath), constants.DirPermPublic); mkErr != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", ymlRemotePath, mkErr)))
		}
		return
	}
	_, ymlFileExistsErr := os.Stat(ymlLocalPath)
	ymlFileExists := ymlFileExistsErr == nil
	if writeErr := os.WriteFile(ymlLocalPath, ymlContent, constants.FilePermSensitive); writeErr != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", ymlRemotePath, writeErr)))
		}
		return
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow (.yml): "+ymlLocalPath))
	}
	fetchAndSaveRemoteDispatchWorkflowsTrack(tracker, ymlLocalPath, ymlFileExists)
}

type fetchAndSaveRemoteDispatchWorkflowsWriteMarkdownParams struct {
	Ctx             context.Context
	WorkflowContent []byte
	Spec            *WorkflowSpec
	RemoteFilePath  string
	TargetPath      string
	TargetDir       string
	Verbose         bool
	Force           bool
	Tracker         *FileTracker
	FileExists      bool
	Ref             string
}

func fetchAndSaveRemoteDispatchWorkflowsWriteMarkdown(p fetchAndSaveRemoteDispatchWorkflowsWriteMarkdownParams) error {
	depSourceString := p.Spec.RepoSlug + "/" + p.RemoteFilePath + "@" + p.Ref
	if updated, srcErr := addSourceToWorkflow(string(p.WorkflowContent), depSourceString); srcErr == nil {
		p.WorkflowContent = []byte(updated)
	}
	if err := os.MkdirAll(filepath.Dir(p.TargetPath), constants.DirPermPublic); err != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", p.RemoteFilePath, err)))
		}
		return nil
	}
	if err := os.WriteFile(p.TargetPath, p.WorkflowContent, constants.FilePermSensitive); err != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", p.RemoteFilePath, err)))
		}
		return nil
	}
	if p.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow: "+p.TargetPath))
	}
	fetchAndSaveRemoteDispatchWorkflowsTrack(p.Tracker, p.TargetPath, p.FileExists)
	fetchDownloadedWorkflowFrontmatterImports(p.Ctx, p.WorkflowContent, p.Spec, p.RemoteFilePath, p.TargetDir, p.Verbose, p.Force, p.Tracker)
	return nil
}

func fetchAndSaveRemoteDispatchWorkflowsTrack(tracker *FileTracker, path string, fileExists bool) {
	if tracker == nil {
		return
	}
	if fileExists {
		tracker.TrackModified(path)
	} else {
		tracker.TrackCreated(path)
	}
}

func fetchDownloadedWorkflowFrontmatterImports(ctx context.Context, workflowContent []byte, parentSpec *WorkflowSpec, remoteFilePath, targetDir string, verbose bool, force bool, tracker *FileTracker) {
	depSpec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: parentSpec.RepoSlug,
			Version:  parentSpec.Version,
		},
		WorkflowPath: remoteFilePath,
	}
	if err := fetchAndSaveRemoteFrontmatterImports(ctx, string(workflowContent), depSpec, targetDir, verbose, force, tracker); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch frontmatter imports for %s: %v", remoteFilePath, err)))
	}
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
func fetchAndSaveDispatchWorkflowsFromParsedFile(ctx context.Context, destFile string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) {
	remoteWorkflowLog.Printf("Fetching import-derived dispatch workflows from parsed file: %s, repo=%s", destFile, spec.RepoSlug)
	owner, repo, ref, ok := fetchAndSaveDispatchWorkflowsFromParsedFileRepo(spec)
	if !ok {
		return
	}

	workflowNames, originalCount := fetchAndSaveDispatchWorkflowsFromParsedFileNames(destFile)
	if len(workflowNames) == 0 {
		return
	}
	remoteWorkflowLog.Printf("Processing %d import-derived dispatch workflow(s) (filtered from %d)", len(workflowNames), originalCount)

	workflowBaseDir := getParentDir(spec.WorkflowPath)
	absTargetDir, absErr := filepath.Abs(targetDir)
	if absErr != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for target directory %s: %v", targetDir, absErr)
		return
	}
	for _, workflowName := range workflowNames {
		fetchAndSaveDispatchWorkflowsFromParsedFileOne(fetchAndSaveDispatchWorkflowsFromParsedFileOneParams{
			Ctx:             ctx,
			WorkflowName:    workflowName,
			Spec:            spec,
			TargetDir:       targetDir,
			Verbose:         verbose,
			Force:           force,
			Tracker:         tracker,
			Owner:           owner,
			Repo:            repo,
			Ref:             ref,
			WorkflowBaseDir: workflowBaseDir,
			AbsTargetDir:    absTargetDir,
		})
	}
}

func fetchAndSaveDispatchWorkflowsFromParsedFileRepo(spec *WorkflowSpec) (string, string, string, bool) {
	if spec.RepoSlug == "" {
		return "", "", "", false
	}
	parts := strings.SplitN(spec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}
	ref := spec.Version
	if ref == "" {
		ref = "main"
	}
	return parts[0], parts[1], ref, true
}

func fetchAndSaveDispatchWorkflowsFromParsedFileNames(destFile string) ([]string, int) {
	compiler := workflow.NewCompiler()
	data, err := compiler.ParseWorkflowFile(destFile)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to parse workflow file %s for import-derived dispatch workflows: %v", destFile, err)
		return nil, 0
	}
	if data == nil || data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return nil, 0
	}
	workflowNames := data.SafeOutputs.DispatchWorkflow.Workflows
	filtered := make([]string, 0, len(workflowNames))
	for _, name := range workflowNames {
		if !strings.Contains(name, "${{") {
			filtered = append(filtered, name)
		}
	}
	return filtered, len(workflowNames)
}

type fetchAndSaveDispatchWorkflowsFromParsedFileOneParams struct {
	Ctx             context.Context
	WorkflowName    string
	Spec            *WorkflowSpec
	TargetDir       string
	Verbose         bool
	Force           bool
	Tracker         *FileTracker
	Owner           string
	Repo            string
	Ref             string
	WorkflowBaseDir string
	AbsTargetDir    string
}

func fetchAndSaveDispatchWorkflowsFromParsedFileOne(p fetchAndSaveDispatchWorkflowsFromParsedFileOneParams) {
	remoteFilePath, targetPath, ok := fetchAndSaveDispatchWorkflowsFromParsedFilePaths(p.WorkflowName, p.WorkflowBaseDir, p.TargetDir, p.AbsTargetDir, p.Verbose)
	if !ok {
		return
	}
	fileExists, skip := fetchAndSaveDispatchWorkflowsFromParsedFileExisting(p.WorkflowName, targetPath, p.Spec.RepoSlug, p.Verbose, p.Force)
	if skip {
		return
	}
	workflowContent, err := parser.DownloadFileFromGitHub(p.Ctx, p.Owner, p.Repo, remoteFilePath, p.Ref)
	if err != nil {
		fetchAndSaveDispatchWorkflowsFromParsedFileYMLFallback(fetchAndSaveDispatchWorkflowsFromParsedFileYMLFallbackParams{
			Ctx:            p.Ctx,
			WorkflowName:   p.WorkflowName,
			RemoteFilePath: remoteFilePath,
			TargetDir:      p.TargetDir,
			Verbose:        p.Verbose,
			Tracker:        p.Tracker,
			Owner:          p.Owner,
			Repo:           p.Repo,
			Ref:            p.Ref,
			OriginalErr:    err,
		})
		return
	}
	fetchAndSaveDispatchWorkflowsFromParsedFileWriteMarkdown(fetchAndSaveDispatchWorkflowsFromParsedFileWriteMarkdownParams{
		Ctx:             p.Ctx,
		WorkflowContent: workflowContent,
		Spec:            p.Spec,
		RemoteFilePath:  remoteFilePath,
		TargetPath:      targetPath,
		TargetDir:       p.TargetDir,
		Verbose:         p.Verbose,
		Force:           p.Force,
		Tracker:         p.Tracker,
		FileExists:      fileExists,
		Ref:             p.Ref,
	})
}

func fetchAndSaveDispatchWorkflowsFromParsedFilePaths(workflowName, workflowBaseDir, targetDir, absTargetDir string, verbose bool) (string, string, bool) {
	if strings.Contains(workflowName, "..") {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping dispatch workflow with unsafe name: %q", workflowName)))
		}
		return "", "", false
	}
	var remoteFilePath string
	if workflowBaseDir != "" {
		remoteFilePath = path.Join(workflowBaseDir, workflowName+".md")
	} else {
		remoteFilePath = workflowName + ".md"
	}
	remoteFilePath = path.Clean(remoteFilePath)
	targetPath := filepath.Join(targetDir, filepath.Clean(workflowName+".md"))
	absTargetPath, absErr := filepath.Abs(targetPath)
	if absErr != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for dispatch workflow %s: %v", workflowName, absErr)
		return "", "", false
	}
	if rel, relErr := filepath.Rel(absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write dispatch workflow outside target directory: %q", workflowName)))
		}
		return "", "", false
	}
	return remoteFilePath, targetPath, true
}

func fetchAndSaveDispatchWorkflowsFromParsedFileExisting(workflowName, targetPath, repoSlug string, verbose bool, force bool) (bool, bool) {
	if _, statErr := os.Stat(targetPath); statErr != nil {
		return false, false
	}
	if force {
		return true, false
	}
	existingSourceRepo := readSourceRepoFromFile(targetPath)
	if existingSourceRepo == repoSlug {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dispatch workflow (from import) from same source already exists, skipping: "+targetPath))
		}
		return true, true
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(
		"Dispatch workflow %q already exists at %s from a different source (existing: %q, needed: %q); use --force to overwrite",
		workflowName, targetPath, sourceRepoLabel(existingSourceRepo), repoSlug,
	)))
	return true, true
}

type fetchAndSaveDispatchWorkflowsFromParsedFileYMLFallbackParams struct {
	Ctx            context.Context
	WorkflowName   string
	RemoteFilePath string
	TargetDir      string
	Verbose        bool
	Tracker        *FileTracker
	Owner          string
	Repo           string
	Ref            string
	OriginalErr    error
}

func fetchAndSaveDispatchWorkflowsFromParsedFileYMLFallback(p fetchAndSaveDispatchWorkflowsFromParsedFileYMLFallbackParams) {
	ymlRemotePath := path.Clean(strings.TrimSuffix(p.RemoteFilePath, ".md") + ".yml")
	ymlLocalPath := filepath.Join(p.TargetDir, filepath.Clean(p.WorkflowName+".yml"))
	ymlContent, ymlErr := parser.DownloadFileFromGitHub(p.Ctx, p.Owner, p.Repo, ymlRemotePath, p.Ref)
	if ymlErr != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch dispatch workflow %s: %v", p.RemoteFilePath, p.OriginalErr)))
		}
		return
	}
	fetchAndSaveDispatchWorkflowsFromParsedFileWriteYML(ymlRemotePath, ymlLocalPath, ymlContent, p.Verbose, p.Tracker)
}

func fetchAndSaveDispatchWorkflowsFromParsedFileWriteYML(ymlRemotePath, ymlLocalPath string, ymlContent []byte, verbose bool, tracker *FileTracker) {
	if mkErr := os.MkdirAll(filepath.Dir(ymlLocalPath), constants.DirPermPublic); mkErr != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", ymlRemotePath, mkErr)))
		}
		return
	}
	_, ymlFileExistsErr := os.Stat(ymlLocalPath)
	ymlFileExists := ymlFileExistsErr == nil
	if writeErr := os.WriteFile(ymlLocalPath, ymlContent, constants.FilePermSensitive); writeErr != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", ymlRemotePath, writeErr)))
		}
		return
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow (.yml, from import): "+ymlLocalPath))
	}
	fetchAndSaveDispatchWorkflowsFromParsedFileTrack(tracker, ymlLocalPath, ymlFileExists)
}

type fetchAndSaveDispatchWorkflowsFromParsedFileWriteMarkdownParams struct {
	Ctx             context.Context
	WorkflowContent []byte
	Spec            *WorkflowSpec
	RemoteFilePath  string
	TargetPath      string
	TargetDir       string
	Verbose         bool
	Force           bool
	Tracker         *FileTracker
	FileExists      bool
	Ref             string
}

func fetchAndSaveDispatchWorkflowsFromParsedFileWriteMarkdown(p fetchAndSaveDispatchWorkflowsFromParsedFileWriteMarkdownParams) {
	depSourceString := p.Spec.RepoSlug + "/" + p.RemoteFilePath + "@" + p.Ref
	if updated, srcErr := addSourceToWorkflow(string(p.WorkflowContent), depSourceString); srcErr == nil {
		p.WorkflowContent = []byte(updated)
	}
	if err := os.MkdirAll(filepath.Dir(p.TargetPath), constants.DirPermPublic); err != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for dispatch workflow %s: %v", p.RemoteFilePath, err)))
		}
		return
	}
	if err := os.WriteFile(p.TargetPath, p.WorkflowContent, constants.FilePermSensitive); err != nil {
		if p.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write dispatch workflow %s: %v", p.RemoteFilePath, err)))
		}
		return
	}
	if p.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched dispatch workflow (from import): "+p.TargetPath))
	}
	fetchAndSaveDispatchWorkflowsFromParsedFileTrack(p.Tracker, p.TargetPath, p.FileExists)
	fetchDownloadedWorkflowFrontmatterImports(p.Ctx, p.WorkflowContent, p.Spec, p.RemoteFilePath, p.TargetDir, p.Verbose, p.Force, p.Tracker)
}

func fetchAndSaveDispatchWorkflowsFromParsedFileTrack(tracker *FileTracker, path string, fileExists bool) {
	if tracker == nil {
		return
	}
	if fileExists {
		tracker.TrackModified(path)
	} else {
		tracker.TrackCreated(path)
	}
}
