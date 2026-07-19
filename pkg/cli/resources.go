package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
)

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
func fetchAndSaveRemoteResources(ctx context.Context, content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error {
	owner, repo, ref, ok := fetchAndSaveRemoteResourcesRepoInfo(ctx, spec)
	if !ok {
		return nil
	}
	resourcePaths, err := extractResources(content)
	if err != nil || len(resourcePaths) == 0 {
		return err
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for target directory %s: %v", targetDir, err)
		return nil
	}
	state := fetchAndSaveRemoteResourcesContext{
		owner: owner, repo: repo, ref: ref, workflowBaseDir: getParentDir(spec.WorkflowPath),
		targetDir: targetDir, absTargetDir: absTargetDir, spec: spec,
		verbose: verbose, force: force, tracker: tracker,
	}
	for _, resourcePath := range resourcePaths {
		if err := fetchAndSaveRemoteResourcesOne(ctx, state, resourcePath); err != nil {
			return err
		}
	}
	return nil
}

type fetchAndSaveRemoteResourcesContext struct {
	owner           string
	repo            string
	ref             string
	workflowBaseDir string
	targetDir       string
	absTargetDir    string
	spec            *WorkflowSpec
	verbose         bool
	force           bool
	tracker         *FileTracker
}

func fetchAndSaveRemoteResourcesRepoInfo(ctx context.Context, spec *WorkflowSpec) (string, string, string, bool) {
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

func fetchAndSaveRemoteResourcesOne(ctx context.Context, state fetchAndSaveRemoteResourcesContext, resourcePath string) error {
	if strings.Contains(resourcePath, "..") {
		if state.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping resource with unsafe path: %q", resourcePath)))
		}
		return nil
	}

	remoteFilePath, targetPath, ok := fetchAndSaveRemoteResourcesPaths(state, resourcePath)
	if !ok {
		return nil
	}
	fileExists, err := fetchAndSaveRemoteResourcesCheckExisting(state, resourcePath, targetPath)
	if err != nil || fileExists && !state.force && strings.HasSuffix(strings.ToLower(targetPath), ".md") && readSourceRepoFromFile(targetPath) == state.spec.RepoSlug {
		return err
	}
	fileContent, ok := fetchAndSaveRemoteResourcesDownload(ctx, state, remoteFilePath)
	if !ok {
		return nil
	}
	if strings.HasSuffix(strings.ToLower(remoteFilePath), ".md") {
		depSourceString := state.spec.RepoSlug + "/" + remoteFilePath + "@" + state.ref
		if updated, srcErr := addSourceToWorkflow(string(fileContent), depSourceString); srcErr == nil {
			fileContent = []byte(updated)
		}
	}
	return fetchAndSaveRemoteResourcesWrite(state, remoteFilePath, targetPath, fileContent, fileExists)
}

func fetchAndSaveRemoteResourcesPaths(state fetchAndSaveRemoteResourcesContext, resourcePath string) (string, string, bool) {
	var remoteFilePath string
	if rest, ok := strings.CutPrefix(resourcePath, "/"); ok {
		remoteFilePath = rest
	} else if state.workflowBaseDir != "" {
		remoteFilePath = path.Join(state.workflowBaseDir, resourcePath)
	} else {
		remoteFilePath = resourcePath
	}
	remoteFilePath = path.Clean(remoteFilePath)
	localRelPath := fetchAndSaveRemoteResourcesLocalPath(state.workflowBaseDir, remoteFilePath)
	if localRelPath == "" || localRelPath == "." {
		return "", "", false
	}
	targetPath := filepath.Join(state.targetDir, localRelPath)
	absTargetPath, absErr := filepath.Abs(targetPath)
	if absErr != nil {
		remoteWorkflowLog.Printf("Failed to resolve absolute path for resource %s: %v", resourcePath, absErr)
		return "", "", false
	}
	if rel, relErr := filepath.Rel(state.absTargetDir, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
		if state.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write resource outside target directory: %q", resourcePath)))
		}
		return "", "", false
	}
	return remoteFilePath, targetPath, true
}

func fetchAndSaveRemoteResourcesLocalPath(workflowBaseDir, remoteFilePath string) string {
	localRelPath := remoteFilePath
	if workflowBaseDir != "" && strings.HasPrefix(remoteFilePath, workflowBaseDir+"/") {
		localRelPath = remoteFilePath[len(workflowBaseDir)+1:]
	}
	localRelPath = filepath.Clean(filepath.FromSlash(localRelPath))
	return strings.TrimLeft(localRelPath, string(filepath.Separator))
}

func fetchAndSaveRemoteResourcesCheckExisting(state fetchAndSaveRemoteResourcesContext, resourcePath, targetPath string) (bool, error) {
	if !fileutil.FileExists(targetPath) {
		return false, nil
	}
	if state.force {
		return true, nil
	}
	isMarkdown := strings.HasSuffix(strings.ToLower(targetPath), ".md")
	if isMarkdown {
		existingSourceRepo := readSourceRepoFromFile(targetPath)
		if existingSourceRepo == state.spec.RepoSlug {
			if state.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Resource file from same source already exists, skipping: "+targetPath))
			}
			return true, nil
		}
		return true, fmt.Errorf("resource %q already exists at %s (existing source: %q, installing from: %q); remove the file or use --force to overwrite", resourcePath, targetPath, sourceRepoLabel(existingSourceRepo), state.spec.RepoSlug)
	}
	return true, fmt.Errorf("resource %q already exists at %s; remove the file or use --force to overwrite", resourcePath, targetPath)
}

func fetchAndSaveRemoteResourcesDownload(ctx context.Context, state fetchAndSaveRemoteResourcesContext, remoteFilePath string) ([]byte, bool) {
	fileContent, err := parser.DownloadFileFromGitHub(ctx, state.owner, state.repo, remoteFilePath, state.ref)
	if err != nil {
		if state.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch resource %s: %v", remoteFilePath, err)))
		}
		return nil, false
	}
	return fileContent, true
}

func fetchAndSaveRemoteResourcesWrite(state fetchAndSaveRemoteResourcesContext, remoteFilePath, targetPath string, fileContent []byte, fileExists bool) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermPublic); err != nil {
		if state.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for resource %s: %v", remoteFilePath, err)))
		}
		return nil
	}
	if err := os.WriteFile(targetPath, fileContent, constants.FilePermSensitive); err != nil {
		if state.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to write resource %s: %v", remoteFilePath, err)))
		}
		return nil
	}
	if state.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Fetched resource: "+targetPath))
	}
	if state.tracker != nil {
		if fileExists {
			state.tracker.TrackModified(targetPath)
		} else {
			state.tracker.TrackCreated(targetPath)
		}
	}
	return nil
}
