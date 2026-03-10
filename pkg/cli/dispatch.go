package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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
type fileDownloadFn func(owner, repo, path, ref string) ([]byte, error)

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
func fetchAndSaveRemoteDispatchWorkflows(content string, spec *WorkflowSpec, targetDir string, verbose bool, force bool, tracker *FileTracker, downloaders ...fileDownloadFn) error {
	downloader := fileDownloadFn(parser.DownloadFileFromGitHub)
	if len(downloaders) > 0 && downloaders[0] != nil {
		downloader = downloaders[0]
	}
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
		workflowContent, err := downloader(owner, repo, remoteFilePath, ref)
		if err != nil {
			// .md not found — try .yml fallback (e.g. plain GitHub Actions workflow)
			ymlRemotePath := path.Clean(strings.TrimSuffix(remoteFilePath, ".md") + ".yml")
			ymlLocalPath := filepath.Join(targetDir, filepath.Clean(workflowName+".yml"))

			ymlContent, ymlErr := downloader(owner, repo, ymlRemotePath, ref)
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
