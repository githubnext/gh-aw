package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var downloadResourceFileFromGitHub = parser.DownloadFileFromGitHub

// extractResources extracts file paths from the top-level "resources" frontmatter field
// and validated grader evaluator paths.
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

	var paths []string
	if resourcesField, exists := result.Frontmatter["resources"]; exists {
		switch v := resourcesField.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					paths = append(paths, s)
				}
			}
		case []string:
			paths = append(paths, v...)
		}
	}

	graders, err := workflow.ParseGradersFromFrontmatter(result.Frontmatter)
	if err != nil {
		return nil, err
	}
	if graders != nil {
		for _, grader := range graders.Graders {
			if grader != nil && grader.Run != "" {
				paths = append(paths, grader.Run)
			}
		}
	}

	// Reject entries that contain GitHub Actions expression syntax — macros are not allowed.
	unique := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		if strings.Contains(p, "${{") {
			return nil, fmt.Errorf("resources entry %q contains GitHub Actions expression syntax (${{) which is not allowed; use static paths only", p)
		}
		if _, exists := seen[p]; exists {
			continue
		}
		seen[p] = struct{}{}
		unique = append(unique, p)
	}

	return unique, nil
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
func fetchAndSaveRemoteResources(ctx context.Context, content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker) error { //nolint:largefunc // Keep resource conflict, download, and tracking behavior together.
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
		defaultBranch, err := getRepoDefaultBranch(ctx, spec.RepoSlug)
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

	for _, resourcePath := range resourcePaths {
		// Early rejection of path traversal patterns. This is a fast first-pass check;
		// the filepath.Rel boundary check below is the authoritative security control.
		if strings.Contains(resourcePath, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping resource with unsafe path: %q", resourcePath)))
			}
			continue
		}

		// Resolve the remote file path. Grader evaluators are repository-relative;
		// ordinary resources remain relative to the source workflow directory.
		var remoteFilePath string
		isGraderEvaluator := strings.HasPrefix(resourcePath, constants.GithubDir+"graders/")
		if isGraderEvaluator {
			remoteFilePath = resourcePath
		} else if rest, ok := strings.CutPrefix(resourcePath, "/"); ok {
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
		targetBaseDir := targetDir
		if isGraderEvaluator {
			targetBaseDir, err = gitutil.FindGitRootFrom(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve repository root for grader resource %q: %w", resourcePath, err)
			}
			localRelPath = filepath.FromSlash(resourcePath)
		}
		targetPath := filepath.Join(targetBaseDir, localRelPath)

		// Belt-and-suspenders: verify the resolved path stays inside its target base.
		absTargetBase, absErr := filepath.Abs(targetBaseDir)
		if absErr != nil {
			remoteWorkflowLog.Printf("Failed to resolve absolute resource target directory %s: %v", targetBaseDir, absErr)
			continue
		}
		absTargetPath, absErr := filepath.Abs(targetPath)
		if absErr != nil {
			remoteWorkflowLog.Printf("Failed to resolve absolute path for resource %s: %v", resourcePath, absErr)
			continue
		}
		if rel, relErr := filepath.Rel(absTargetBase, absTargetPath); relErr != nil || strings.HasPrefix(rel, "..") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Refusing to write resource outside target directory: %q", resourcePath)))
			}
			continue
		}

		// Check whether the target file already exists.
		fileExists := false
		if fileutil.FileExists(targetPath) {
			fileExists = true
			// Shared evaluators may be referenced by multiple workflows in one package.
			// Their conflict handling is deferred until the source content can be compared.
			if !force && !isGraderEvaluator {
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
		fileContent, err := downloadResourceFileFromGitHub(ctx, owner, repo, remoteFilePath, ref)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch resource %s: %v", remoteFilePath, err)))
			}
			continue
		}
		if fileExists && !force && isGraderEvaluator {
			existingContent, err := os.ReadFile(targetPath)
			if err != nil {
				return fmt.Errorf("failed to read existing grader resource %q: %w", targetPath, err)
			}
			if bytes.Equal(existingContent, fileContent) {
				continue
			}
			return fmt.Errorf("resource %q already exists at %s; remove the file or use --force to overwrite", resourcePath, targetPath)
		}

		// For markdown resources, embed the source field for future conflict detection.
		if strings.HasSuffix(strings.ToLower(remoteFilePath), ".md") {
			depSourceString := path.Join(spec.RepoSlug, remoteFilePath) + "@" + ref
			if updated, srcErr := addSourceToWorkflow(string(fileContent), depSourceString); srcErr == nil {
				fileContent = []byte(updated)
			}
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), constants.DirPermPublic); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create directory for resource %s: %v", remoteFilePath, err)))
			}
			continue
		}

		// Write the file
		if err := os.WriteFile(targetPath, fileContent, constants.FilePermSensitive); err != nil {
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
