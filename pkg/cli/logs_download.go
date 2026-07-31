// This file provides command-line interface functionality for gh-aw.
// This file (logs_download.go) contains functions for downloading and extracting
// GitHub Actions workflow artifacts and logs.
//
// Key responsibilities:
//   - Downloading workflow run artifacts via gh CLI
//   - Extracting and organizing zip archives
//   - Flattening single-file artifact directories
//   - Managing local file system operations

package cli

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/errorutil"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsDownloadLog = logger.New("cli:logs_download")

// downloadArtifactsOptions bundles the common parameters shared by the artifact
// download helpers, avoiding repeated positional argument lists.
// runID identifies the workflow run; outputDir is the local destination directory;
// verbose enables progress messages; owner, repo, hostname identify the GitHub repository;
// artifactFilter is an optional list of artifact base names to download (nil means all).
type downloadArtifactsOptions struct {
	runID          int64
	outputDir      string
	verbose        bool
	owner          string
	repo           string
	hostname       string
	artifactFilter []string
}

// isUsageOnlyArtifactFilter reports whether the caller requested only the compact
// usage artifact. In this mode, workflow-run log downloads are intentionally skipped
// to minimize API and transfer volume for lightweight reporting paths.
func isUsageOnlyArtifactFilter(artifactFilter []string) bool {
	return len(artifactFilter) == 1 && artifactFilter[0] == constants.UsageArtifactName
}

func shouldDownloadWorkflowRunLogs(artifactFilter []string) bool {
	if len(artifactFilter) == 0 {
		return true
	}
	for _, artifact := range artifactFilter {
		if artifact != constants.ActivationArtifactName && artifact != constants.UsageArtifactName {
			return true
		}
	}
	return false
}

// flattenSingleFileArtifacts checks artifact directories and flattens any that contain a single file
// This handles the case where gh CLI creates a directory for each artifact, even if it's just one file
func flattenSingleFileArtifacts(outputDir string, verbose bool) error {
	logsDownloadLog.Printf("Flattening single-file artifacts in: %s", outputDir)
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == downloadedArtifactsMarkerDir {
			continue
		}
		artifactDir := filepath.Join(outputDir, entry.Name())
		flattenSingleArtifactEntry(outputDir, artifactDir, entry, verbose)
	}
	return nil
}

// flattenSingleArtifactEntry attempts to unfold an artifact directory that contains
// exactly one file, moving the file up to the parent and removing the now-empty dir.
func flattenSingleArtifactEntry(outputDir, artifactDir string, entry os.DirEntry, verbose bool) {
	artifactEntries, err := os.ReadDir(artifactDir)
	if err != nil {
		logsDownloadLog.Printf("Failed to read artifact directory %s: %v", artifactDir, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read artifact directory %s: %v", artifactDir, err)))
		}
		return
	}
	logsDownloadLog.Printf("Artifact directory %s contains %d entries", entry.Name(), len(artifactEntries))
	if len(artifactEntries) != 1 {
		if verbose && len(artifactEntries) > 1 {
			var fileNames []string
			for _, e := range artifactEntries {
				fileNames = append(fileNames, e.Name())
			}
			logsDownloadLog.Printf("Artifact directory %s has %d files, not flattening: %v", entry.Name(), len(artifactEntries), fileNames)
		}
		return
	}
	singleEntry := artifactEntries[0]
	if singleEntry.IsDir() {
		logsDownloadLog.Printf("Artifact directory %s contains a subdirectory, not flattening", entry.Name())
		return
	}
	sourcePath := filepath.Join(artifactDir, singleEntry.Name())
	destPath := filepath.Join(outputDir, singleEntry.Name())
	logsDownloadLog.Printf("Flattening: %s → %s", sourcePath, destPath)
	if err := os.Rename(sourcePath, destPath); err != nil {
		logsDownloadLog.Printf("Failed to move file %s to %s: %v", sourcePath, destPath, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to move file %s to %s: %v", sourcePath, destPath, err)))
		}
		return
	}
	if err := os.Remove(artifactDir); err != nil {
		logsDownloadLog.Printf("Failed to remove empty directory %s: %v", artifactDir, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove empty directory %s: %v", artifactDir, err)))
		}
		return
	}
	logsDownloadLog.Printf("Successfully flattened: %s/%s → %s", entry.Name(), singleEntry.Name(), singleEntry.Name())
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Unfolded single-file artifact: %s → %s", filepath.Join(entry.Name(), singleEntry.Name()), singleEntry.Name())))
	}
}

// findArtifactDir looks for an artifact directory by its base name (suffix) in outputDir.
// It handles three cases:
//  1. Exact match: "agent" → outputDir/agent
//  2. Legacy name: for "agent", also checks "agent-artifacts"
//  3. Prefixed name (workflow_call): "*-agent" → outputDir/<hash>-agent
//
// Returns the first matching directory path, or empty string if none found.
func findArtifactDir(outputDir, baseName string, legacyName string) string {
	// First, try exact match
	exactPath := filepath.Join(outputDir, baseName)
	if fileutil.DirExists(exactPath) {
		return exactPath
	}

	// Try legacy name if provided
	if legacyName != "" {
		legacyPath := filepath.Join(outputDir, legacyName)
		if fileutil.DirExists(legacyPath) {
			return legacyPath
		}
	}

	// Scan for prefixed names (workflow_call context): any directory ending with "-{baseName}"
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return ""
	}
	suffix := "-" + baseName
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			return filepath.Join(outputDir, entry.Name())
		}
	}

	return ""
}

// flattenArtifactTree moves all files from sourceDir into outputDir, preserving relative paths,
// then removes artifactDir (which may equal sourceDir, or be a parent of it in the old-structure
// case). label is used in log and user-facing messages.
// Cleanup failures are non-fatal: they are logged (and optionally printed) but do not return an error.
func flattenArtifactTree(sourceDir, artifactDir, outputDir, label string, verbose bool) error {
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if path == sourceDir {
			return nil
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		destPath := filepath.Join(outputDir, relPath)

		if info.IsDir() {
			// Create directory in destination with world-readable permissions (0755)
			if err := os.MkdirAll(destPath, constants.DirPermPublic); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			logsDownloadLog.Printf("Created directory: %s", destPath)
		} else {
			// Ensure parent directory exists with world-readable permissions (0755)
			if err := os.MkdirAll(filepath.Dir(destPath), constants.DirPermPublic); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
			}

			if err := os.Rename(path, destPath); err != nil {
				return fmt.Errorf("failed to move file %s to %s: %w", path, destPath, err)
			}
			logsDownloadLog.Printf("Moved file: %s → %s", path, destPath)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Flattened: %s → %s", relPath, relPath)))
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to flatten %s: %w", label, err)
	}

	cleanupFlattenedArtifactDir(artifactDir, label, verbose)
	return nil
}

// cleanupFlattenedArtifactDir removes the now-empty artifact directory after flattening.
func cleanupFlattenedArtifactDir(artifactDir, label string, verbose bool) {
	if err := os.RemoveAll(artifactDir); err != nil {
		logsDownloadLog.Printf("Failed to remove %s directory %s: %v", label, artifactDir, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove %s directory: %v", label, err)))
		}
	} else {
		logsDownloadLog.Printf("Removed %s directory: %s", label, artifactDir)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Flattened %s and removed nested structure", label)))
		}
	}
}

// flattenUnifiedArtifact flattens the unified agent artifact directory structure.
// The artifact is uploaded with all paths under /tmp/gh-aw/, so the action strips the
// common prefix and files land directly inside the artifact directory (new structure).
// For backward compatibility, it also handles the old structure where the full
// tmp/gh-aw/ path was preserved inside the artifact directory.
// New artifact name: "agent"   (preferred)
// Legacy artifact name: "agent-artifacts" (backward compat for older workflow runs)
// In workflow_call context, the artifact may be prefixed: "<hash>-agent"
func flattenUnifiedArtifact(outputDir string, verbose bool) error {
	agentArtifactsDir := findArtifactDir(outputDir, "agent", "agent-artifacts")
	if agentArtifactsDir == "" {
		// No unified artifact, nothing to flatten
		return nil
	}

	logsDownloadLog.Printf("Flattening unified agent artifact directory: %s", agentArtifactsDir)

	// Determine the source path: old structure preserves the tmp/gh-aw/ prefix inside the artifact
	sourceDir := agentArtifactsDir
	tmpGhAwPath := filepath.Join(agentArtifactsDir, "tmp", "gh-aw")
	if fileutil.DirExists(tmpGhAwPath) {
		logsDownloadLog.Printf("Found old artifact structure with tmp/gh-aw prefix")
		sourceDir = tmpGhAwPath
	} else {
		logsDownloadLog.Printf("Found new artifact structure without tmp/gh-aw prefix")
	}

	return flattenArtifactTree(sourceDir, agentArtifactsDir, outputDir, "unified agent artifact", verbose)
}

// flattenActivationArtifact flattens the activation artifact directory structure.
// The activation artifact contains aw_info.json and aw-prompts/prompt.txt.
// This function moves those files to the root output directory and removes the nested structure.
// In workflow_call context, the artifact may be prefixed: "<hash>-activation"
func flattenActivationArtifact(outputDir string, verbose bool) error {
	activationDir := findArtifactDir(outputDir, "activation", "")
	if activationDir == "" {
		// No activation artifact, nothing to flatten
		return nil
	}

	logsDownloadLog.Printf("Flattening activation artifact directory: %s", activationDir)

	return flattenArtifactTree(activationDir, activationDir, outputDir, "activation artifact", verbose)
}

// flattenAgentOutputsArtifact flattens the agent_outputs artifact directory structure.
// The agent_outputs artifact contains session logs with detailed token usage data
// that are critical for accurate token count parsing.
func flattenAgentOutputsArtifact(outputDir string, verbose bool) error {
	agentOutputsDir := filepath.Join(outputDir, "agent_outputs")

	// Check if agent_outputs directory exists
	if _, err := os.Stat(agentOutputsDir); os.IsNotExist(err) {
		// No agent_outputs artifact, nothing to flatten
		logsDownloadLog.Print("No agent_outputs artifact found (session logs may be missing)")
		return nil
	}

	logsDownloadLog.Printf("Flattening agent_outputs directory: %s", agentOutputsDir)

	return flattenArtifactTree(agentOutputsDir, agentOutputsDir, outputDir, "agent_outputs artifact", verbose)
}

// flattenSafeOutputsItemsArtifact flattens the safe-outputs-items artifact directory
// structure. The safe-outputs-items artifact contains safe-output-items.jsonl and
// temporary-id-map.json. After flattening, these files land at the run directory root
// where extractCreatedItemsFromManifest and loadResolvedTemporaryIDTargets expect them.
// The artifact may be prefixed in workflow_call context: "<hash>-safe-outputs-items".
func flattenSafeOutputsItemsArtifact(outputDir string, verbose bool) error {
	safeOutputsItemsDir := findArtifactDir(outputDir, constants.SafeOutputItemsArtifactName, "")
	if safeOutputsItemsDir == "" {
		// No safe-outputs-items artifact, nothing to flatten
		return nil
	}

	logsDownloadLog.Printf("Flattening safe-outputs-items artifact directory: %s", safeOutputsItemsDir)

	return flattenArtifactTree(safeOutputsItemsDir, safeOutputsItemsDir, outputDir, "safe-outputs-items artifact", verbose)
}

// downloadWorkflowRunLogs downloads and unzips workflow run logs using GitHub API
func downloadWorkflowRunLogs(ctx context.Context, runID int64, outputDir string, verbose bool, owner, repo, hostname string) error {
	logsDownloadLog.Printf("Downloading workflow run logs: run_id=%d, output_dir=%s, owner=%s, repo=%s", runID, outputDir, owner, repo)

	// Create a temporary file for the zip download
	tmpZip := filepath.Join(os.TempDir(), fmt.Sprintf("workflow-logs-%d.zip", runID))
	defer os.RemoveAll(tmpZip)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Downloading workflow run logs for run %d...", runID)))
	}

	args := buildWorkflowLogsDownloadArgs(owner, repo, hostname, runID)
	output, err := workflow.RunGHContext(ctx, "Downloading workflow logs...", args...)
	if err != nil {
		// Check for authentication errors
		if isPermissionError(err) {
			return errors.New("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		// If logs are not found or run has no logs, this is not a critical error.
		if errorutil.IsNotFoundError(err) || errorutil.IsNotFoundError(errors.New(string(output))) || errorutil.IsGoneError(err) {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No logs found for run %d (may be expired or unavailable)", runID)))
			}
			return nil
		}
		return fmt.Errorf("failed to download workflow run logs for run %d: %w", runID, err)
	}

	// Write the downloaded zip content to temporary file
	if err := os.WriteFile(tmpZip, output, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write logs zip file: %w", err)
	}

	// Create a subdirectory for workflow logs to keep the run directory organized
	workflowLogsDir := filepath.Join(outputDir, "workflow-logs")
	if err := os.MkdirAll(workflowLogsDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create workflow-logs directory: %w", err)
	}

	if err := unzipFile(tmpZip, workflowLogsDir, verbose); err != nil {
		return fmt.Errorf("failed to unzip workflow logs: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Downloaded and extracted workflow run logs to "+workflowLogsDir))
	}

	return nil
}

// buildWorkflowLogsDownloadArgs constructs the gh api arguments for downloading workflow run logs.
func buildWorkflowLogsDownloadArgs(owner, repo, hostname string, runID int64) []string {
	var endpoint string
	if owner != "" && repo != "" {
		endpoint = fmt.Sprintf("repos/%s/%s/actions/runs/%d/logs", owner, repo, runID)
	} else {
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/logs", runID)
	}
	args := []string{"api", endpoint}
	if hostname != "" && hostname != "github.com" {
		args = append(args, "--hostname", hostname)
	}
	return args
}

// unzipFile extracts a zip file to a destination directory
func unzipFile(zipPath, destDir string, verbose bool) error {
	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	// Extract each file in the zip
	for _, f := range r.File {
		if err := extractZipFile(f, destDir, verbose); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive
func extractZipFile(f *zip.File, destDir string, verbose bool) error {
	// #nosec G305 -- Path traversal is prevented by filepath.Clean and prefix check below
	cleanName := filepath.Clean(f.Name)
	if strings.Contains(cleanName, "..") {
		return fmt.Errorf("invalid file path in zip (contains ..): %s", f.Name)
	}

	filePath := filepath.Join(destDir, cleanName)

	// Prevent zip slip vulnerability - ensure extracted path is within destDir
	cleanDest := filepath.Clean(destDir)
	if !strings.HasPrefix(filepath.Clean(filePath), cleanDest+string(os.PathSeparator)) && filepath.Clean(filePath) != cleanDest {
		return fmt.Errorf("invalid file path in zip (outside destination): %s", f.Name)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Extracting: "+cleanName))
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(filePath, constants.DirPermPublic)
	}

	// #nosec G110 -- Decompression bomb is mitigated by size check below
	const maxFileSize = 1 * 1024 * 1024 * 1024 // 1GB
	if f.UncompressedSize64 > maxFileSize {
		return fmt.Errorf("file too large in zip (>1GB): %s (%d bytes)", f.Name, f.UncompressedSize64)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return writeZipFileContent(f, filePath, maxFileSize)
}

// writeZipFileContent writes the content of a zip.File entry to filePath.
func writeZipFileContent(f *zip.File, filePath string, maxFileSize uint64) (writeErr error) {
	srcFile, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in zip: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		// Errors from closing the writable file must be captured to prevent silent data loss.
		if err := destFile.Close(); writeErr == nil && err != nil {
			writeErr = fmt.Errorf("failed to close destination file: %w", err)
		}
	}()

	limitedReader := io.LimitReader(srcFile, int64(maxFileSize))
	written, err := io.Copy(destFile, limitedReader)
	if err != nil {
		writeErr = fmt.Errorf("failed to extract file: %w", err)
		return writeErr
	}

	if uint64(written) > maxFileSize {
		writeErr = fmt.Errorf("file extraction exceeded size limit: %s", f.Name)
		return writeErr
	}

	return nil
}

// listArtifacts creates a list of all artifact files in the output directory
func listArtifacts(outputDir string) ([]string, error) {
	var artifacts []string

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and the summary file itself
		if info.IsDir() || filepath.Base(path) == runSummaryFileName {
			return nil
		}

		// Get relative path from outputDir
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		artifacts = append(artifacts, relPath)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

// isNonZipArtifactError reports whether the output from gh run download indicates
// that the failure was caused by one or more non-zip artifacts (e.g. .dockerbuild files).
// Such artifacts cannot be extracted as zip archives and should be skipped rather than
// failing the entire download.
func isNonZipArtifactError(output []byte) bool {
	s := string(output)
	return strings.Contains(s, "zip: not a valid zip file")
}

// isCaseCollisionArtifactError reports whether gh run download failed because
// a zip extraction attempted to write a file that already exists. This can
// happen on case-insensitive filesystems (e.g. macOS) when an artifact
// contains files whose names differ only by case.
func isCaseCollisionArtifactError(output []byte) bool {
	s := string(output)
	return strings.Contains(s, "error extracting zip archive") && strings.Contains(s, "file exists")
}

// isDockerBuildArtifact reports whether an artifact name represents a .dockerbuild artifact.
// These are not zip archives and cannot be extracted by gh run download.
func isDockerBuildArtifact(name string) bool {
	return strings.HasSuffix(name, ".dockerbuild")
}

// listRunArtifactNames returns the names of all artifacts for the given workflow run
// by querying the GitHub Actions API. Returns an error if the API call fails.
func listRunArtifactNames(ctx context.Context, runID int64, owner, repo, hostname string, verbose bool) ([]string, error) {
	var endpoint string
	if owner != "" && repo != "" {
		endpoint = fmt.Sprintf("repos/%s/%s/actions/runs/%d/artifacts", owner, repo, runID)
	} else {
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/artifacts", runID)
	}

	args := []string{"api", "--paginate", endpoint, "--jq", ".artifacts[].name"}
	if hostname != "" && hostname != "github.com" {
		args = append(args, "--hostname", hostname)
	}

	logsDownloadLog.Printf("Listing artifacts for run %d: gh %s", runID, strings.Join(args, " "))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Listing artifacts: gh "+strings.Join(args, " ")))
	}

	cmd := workflow.ExecGHContext(ctx, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts for run %d: %w", runID, err)
	}

	var names []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// downloadArtifactsByName downloads a list of artifacts individually by name.
// This is used when some artifacts (e.g. .dockerbuild) need to be skipped and
// only a subset of the run's artifacts should be downloaded.
func downloadArtifactsByName(ctx context.Context, opts downloadArtifactsOptions, names []string) error {
	var repoFlag string
	shouldLogProgress := IsRunningInCI() || opts.verbose
	if opts.owner != "" && opts.repo != "" {
		if opts.hostname != "" && opts.hostname != "github.com" {
			repoFlag = opts.hostname + "/" + opts.owner + "/" + opts.repo
		} else {
			repoFlag = opts.owner + "/" + opts.repo
		}
	}

	for _, name := range names {
		args := []string{"run", "download", strconv.FormatInt(opts.runID, 10), "--name", name, "--dir", opts.outputDir}
		if repoFlag != "" {
			args = append(args, "-R", repoFlag)
		}

		logsDownloadLog.Printf("Downloading artifact %q individually: gh %s", name, strings.Join(args, " "))
		if shouldLogProgress {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Downloading artifact: "+name))
		}

		cmd := workflow.ExecGHContext(ctx, args...)
		cmdOutput, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			logsDownloadLog.Printf("Failed to download artifact %q: %v (%s)", name, cmdErr, string(cmdOutput))
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifact %q: %v", name, cmdErr)))
			}
			// Non-fatal: continue downloading other artifacts
		} else {
			logsDownloadLog.Printf("Downloaded artifact %q", name)
			if err := markArtifactDownloaded(opts.outputDir, name); err != nil {
				return err
			}
		}
	}

	return nil
}

// criticalArtifactNames lists the artifact names that are essential for audit analysis.
// When a bulk download fails partially (e.g., due to non-zip artifacts), these artifacts
// are retried individually so that flattening and audit extraction have data to work with.
var criticalArtifactNames = []string{"activation", "agent"}

// retryCriticalArtifacts downloads critical artifacts individually when the bulk download
// was only partially successful. gh run download aborts on the first non-zip artifact,
// which may prevent valid artifacts from being downloaded.
// artifactFilter limits which critical artifacts are retried; nil means retry all.
func retryCriticalArtifacts(ctx context.Context, opts downloadArtifactsOptions) {
	// Build the repo flag once for reuse across retries
	var repoFlag string
	if opts.owner != "" && opts.repo != "" {
		if opts.hostname != "" && opts.hostname != "github.com" {
			repoFlag = opts.hostname + "/" + opts.owner + "/" + opts.repo
		} else {
			repoFlag = opts.owner + "/" + opts.repo
		}
	}

	for _, name := range criticalArtifactNames {
		// Skip artifacts not included in the active filter.
		if !artifactMatchesFilter(name, opts.artifactFilter) {
			logsDownloadLog.Printf("Skipping critical artifact %q (not in artifact filter)", name)
			continue
		}
		artifactDir := filepath.Join(opts.outputDir, name)
		if fileutil.DirExists(artifactDir) {
			logsDownloadLog.Printf("Critical artifact %q already present, skipping retry", name)
			continue
		}

		retryArgs := []string{"run", "download", strconv.FormatInt(opts.runID, 10), "--name", name, "--dir", opts.outputDir}
		if repoFlag != "" {
			retryArgs = append(retryArgs, "-R", repoFlag)
		}

		logsDownloadLog.Printf("Retrying individual download for artifact %q: gh %s", name, strings.Join(retryArgs, " "))
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Retrying download for missing artifact: "+name))
		}

		retryCmd := workflow.ExecGHContext(ctx, retryArgs...)
		retryOutput, retryErr := retryCmd.CombinedOutput()
		if retryErr != nil {
			logsDownloadLog.Printf("Failed to download artifact %q individually: %v (%s)", name, retryErr, string(retryOutput))
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not download artifact %q: %v", name, retryErr)))
			}
		} else {
			logsDownloadLog.Printf("Successfully downloaded artifact %q individually", name)
			// Marker write failures are non-fatal in the retry path: retryCriticalArtifacts
			// is a best-effort recovery after a partial bulk download, so a missing marker
			// only causes a redundant re-download on the next run (not data loss).
			if err := markArtifactDownloaded(opts.outputDir, name); err != nil {
				logsDownloadLog.Printf("Failed to mark artifact %q as downloaded: %v", name, err)
			}
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Downloaded missing artifact: "+name))
			}
		}
	}
}

// downloadRunArtifacts downloads artifacts for a specific workflow run.
// artifactFilter is a list of artifact base names to download; nil means download all.
func downloadRunArtifacts(ctx context.Context, opts downloadArtifactsOptions) error {
	logsDownloadLog.Printf("Downloading run artifacts: run_id=%d, output_dir=%s, owner=%s, repo=%s, artifactFilter=%v", opts.runID, opts.outputDir, opts.owner, opts.repo, opts.artifactFilter)
	shouldLogProgress := IsRunningInCI() || opts.verbose

	opts, done, err := checkArtifactsCached(ctx, opts, shouldLogProgress)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	if err := os.MkdirAll(opts.outputDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create run output directory: %w", err)
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Created output directory "+opts.outputDir))
	}

	dockerBuildArtifacts, downloadableNames, listErr := classifyArtifactDownloadNames(ctx, opts)

	downloadableNames, incrementalUnfilteredDownload, done, err := checkIncrementalDownload(opts, downloadableNames, listErr, shouldLogProgress)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	spinner := console.NewSpinner(fmt.Sprintf("Downloading artifacts for run %d...", opts.runID))
	if !opts.verbose {
		spinner.Start()
	}

	if len(dockerBuildArtifacts) > 0 || len(opts.artifactFilter) > 0 || incrementalUnfilteredDownload {
		if err := runFilteredArtifactsDownload(ctx, opts, downloadableNames, shouldLogProgress, spinner); err != nil {
			return err
		}
	} else {
		if err := runBulkArtifactsDownload(ctx, opts, downloadableNames, shouldLogProgress, spinner); err != nil {
			return err
		}
	}

	if !opts.verbose {
		spinner.StopWithMessage(fmt.Sprintf("✓ Downloaded artifacts for run %d", opts.runID))
	}

	if err := flattenDownloadedArtifacts(ctx, opts); err != nil {
		return err
	}

	if opts.verbose {
		logVerboseArtifactSummary(opts.outputDir)
	}

	return nil
}

// checkArtifactsCached checks whether artifacts are already cached and returns early if so.
// It may update opts.artifactFilter to restrict the download to only missing artifacts.
// Returns (updatedOpts, done, err): when done=true the caller should return nil.
func checkArtifactsCached(ctx context.Context, opts downloadArtifactsOptions, shouldLogProgress bool) (downloadArtifactsOptions, bool, error) {
	if !fileutil.DirExists(opts.outputDir) || fileutil.IsDirEmpty(opts.outputDir) {
		return opts, false, nil
	}
	if len(opts.artifactFilter) > 0 {
		missing := findMissingFilterEntries(opts.artifactFilter, opts.outputDir)
		if len(missing) == 0 {
			logsDownloadLog.Printf("All requested artifacts already on disk for run %d", opts.runID)
			if shouldLogProgress {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("All requested artifacts already present for run %d, skipping download", opts.runID)))
			}
			ensureUsageAwInfoFallback(ctx, opts)
			return opts, true, nil
		}
		logsDownloadLog.Printf("Downloading missing artifacts for run %d: %v (already have: %v)", opts.runID, missing, opts.artifactFilter)
		if shouldLogProgress {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Downloading missing artifacts for run %d: %v", opts.runID, missing)))
		}
		opts.artifactFilter = missing
		return opts, false, nil
	}
	// No filter — check if all artifacts are present via the complete-download marker.
	if len(findMissingFilterEntries([]string{string(ArtifactSetAll)}, opts.outputDir)) == 0 {
		logsDownloadLog.Printf("Using cached artifacts for run %d", opts.runID)
		if shouldLogProgress {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("All artifacts already present for run %d, skipping download", opts.runID)))
		}
		return opts, true, nil
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run folder for %d is missing the complete artifact marker; downloading all artifacts", opts.runID)))
	}
	return opts, false, nil
}

// classifyArtifactDownloadNames lists run artifacts and separates dockerbuild artifacts
// (which cannot be extracted) from those eligible for download.
func classifyArtifactDownloadNames(ctx context.Context, opts downloadArtifactsOptions) (dockerBuildArtifacts, downloadableNames []string, listErr error) {
	artifactNames, listErr := listRunArtifactNames(ctx, opts.runID, opts.owner, opts.repo, opts.hostname, opts.verbose)
	if listErr == nil {
		for _, name := range artifactNames {
			if isDockerBuildArtifact(name) {
				dockerBuildArtifacts = append(dockerBuildArtifacts, name)
			} else if artifactMatchesFilter(name, opts.artifactFilter) {
				downloadableNames = append(downloadableNames, name)
			}
		}
		if len(dockerBuildArtifacts) > 0 {
			logsDownloadLog.Printf("Found %d .dockerbuild artifact(s) that will be skipped: %v", len(dockerBuildArtifacts), dockerBuildArtifacts)
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Skipping %d .dockerbuild artifact(s) (not valid zip archives): %s", len(dockerBuildArtifacts), strings.Join(dockerBuildArtifacts, ", "))))
			}
		}
	} else {
		if len(opts.artifactFilter) > 0 {
			downloadableNames = append(downloadableNames, opts.artifactFilter...)
			logsDownloadLog.Printf("Could not list artifacts (will try requested artifact names directly): %v", listErr)
		} else {
			logsDownloadLog.Printf("Could not list artifacts (will use bulk download): %v", listErr)
		}
	}
	return
}

// checkIncrementalDownload checks whether some artifacts are already on disk and narrows
// the download to only the missing ones.
// Returns (updatedNames, incrementalUnfilteredDownload, done, err).
// When done=true the caller should return nil.
func checkIncrementalDownload(opts downloadArtifactsOptions, downloadableNames []string, listErr error, shouldLogProgress bool) ([]string, bool, bool, error) {
	if listErr != nil || len(downloadableNames) == 0 || len(opts.artifactFilter) != 0 ||
		!fileutil.DirExists(opts.outputDir) || fileutil.IsDirEmpty(opts.outputDir) {
		return downloadableNames, false, false, nil
	}
	missingNames := findMissingFilterEntries(downloadableNames, opts.outputDir)
	if len(missingNames) == 0 {
		logsDownloadLog.Printf("All %d artifacts already present for run %d (incremental check)", len(downloadableNames), opts.runID)
		if shouldLogProgress {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("All artifacts already present for run %d, skipping download", opts.runID)))
		}
		if markerErr := markArtifactDownloaded(opts.outputDir, string(ArtifactSetAll)); markerErr != nil {
			return downloadableNames, false, true, markerErr
		}
		return downloadableNames, false, true, nil
	}
	if len(missingNames) < len(downloadableNames) {
		logsDownloadLog.Printf("Incremental download for run %d: %d/%d artifacts missing: %v",
			opts.runID, len(missingNames), len(downloadableNames), missingNames)
		if shouldLogProgress {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf(
				"Incremental download for run %d: fetching %d missing artifact(s): %v",
				opts.runID, len(missingNames), missingNames)))
		}
		return missingNames, true, false, nil
	}
	return downloadableNames, false, false, nil
}

// runFilteredArtifactsDownload downloads artifacts individually when a name filter,
// dockerbuild artifacts, or an incremental top-up requires per-name downloads.
func runFilteredArtifactsDownload(ctx context.Context, opts downloadArtifactsOptions, downloadableNames []string, shouldLogProgress bool, spinner *console.SpinnerWrapper) error {
	if !opts.verbose {
		spinner.Stop()
	}
	if len(downloadableNames) == 0 {
		if shouldDownloadWorkflowRunLogs(opts.artifactFilter) {
			if logErr := downloadWorkflowRunLogs(ctx, opts.runID, opts.outputDir, opts.verbose, opts.owner, opts.repo, opts.hostname); logErr != nil {
				if opts.verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", logErr)))
				}
				if fileutil.IsDirEmpty(opts.outputDir) {
					if removeErr := os.RemoveAll(opts.outputDir); removeErr != nil && opts.verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up empty directory %s: %v", opts.outputDir, removeErr)))
					}
				}
			}
		}
		return ErrNoArtifacts
	}
	if err := downloadArtifactsByName(ctx, opts, downloadableNames); err != nil {
		return err
	}
	if fileutil.IsDirEmpty(opts.outputDir) {
		return ErrNoArtifacts
	}
	if len(opts.artifactFilter) == 0 && len(downloadableNames) > 0 {
		if len(findMissingFilterEntries(downloadableNames, opts.outputDir)) == 0 {
			if markerErr := markArtifactDownloaded(opts.outputDir, string(ArtifactSetAll)); markerErr != nil {
				return markerErr
			}
		}
	}
	return nil
}

// runBulkArtifactsDownload uses gh run download for an efficient bulk download when no
// artifact filter or dockerbuild artifacts require per-name downloads.
func runBulkArtifactsDownload(ctx context.Context, opts downloadArtifactsOptions, downloadableNames []string, _ bool, spinner *console.SpinnerWrapper) error {
	ghArgs := []string{"run", "download", strconv.FormatInt(opts.runID, 10), "--dir", opts.outputDir}
	if opts.owner != "" && opts.repo != "" {
		if opts.hostname != "" && opts.hostname != "github.com" {
			ghArgs = append(ghArgs, "-R", opts.hostname+"/"+opts.owner+"/"+opts.repo)
		} else {
			ghArgs = append(ghArgs, "-R", opts.owner+"/"+opts.repo)
		}
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executing: gh "+strings.Join(ghArgs, " ")))
	}
	cmd := workflow.ExecGHContext(ctx, ghArgs...)
	output, err := cmd.CombinedOutput()

	var skippedNonZipArtifacts, skippedCaseCollisionArtifacts bool
	if err != nil {
		skippedNonZipArtifacts, skippedCaseCollisionArtifacts, err = handleBulkDownloadError(ctx, opts, output, err, spinner)
		if err != nil {
			return err
		}
	}

	if skippedNonZipArtifacts {
		retryCriticalArtifacts(ctx, opts)
	}
	if skippedCaseCollisionArtifacts {
		if err := handleCaseCollisionRetry(ctx, opts, downloadableNames); err != nil {
			return err
		}
	}
	if skippedNonZipArtifacts && fileutil.IsDirEmpty(opts.outputDir) {
		return ErrNoArtifacts
	}
	if err == nil || skippedNonZipArtifacts || skippedCaseCollisionArtifacts {
		if markerErr := markArtifactDownloaded(opts.outputDir, string(ArtifactSetAll)); markerErr != nil {
			return markerErr
		}
	}
	return nil
}

// handleBulkDownloadError processes an error from the bulk gh run download command.
// Returns (skippedNonZip, skippedCaseCollision, terminalErr): when terminalErr!=nil the
// caller should return it immediately.
func handleBulkDownloadError(ctx context.Context, opts downloadArtifactsOptions, output []byte, cmdErr error, spinner *console.SpinnerWrapper) (bool, bool, error) {
	if !opts.verbose {
		spinner.Stop()
	}
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(string(output)))
	}
	if strings.Contains(string(output), "no valid artifacts") || strings.Contains(string(output), "not found") {
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No artifacts found for run %d (gh run download reported none)", opts.runID)))
		}
		if logErr := downloadWorkflowRunLogs(ctx, opts.runID, opts.outputDir, opts.verbose, opts.owner, opts.repo, opts.hostname); logErr != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", logErr)))
			}
			if fileutil.IsDirEmpty(opts.outputDir) {
				if removeErr := os.RemoveAll(opts.outputDir); removeErr != nil && opts.verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up empty directory %s: %v", opts.outputDir, removeErr)))
				}
			}
		}
		return false, false, ErrNoArtifacts
	}
	if isPermissionError(cmdErr) {
		return false, false, errors.New("GitHub CLI authentication required. Run 'gh auth login' first")
	}
	if isNonZipArtifactError(output) {
		msg := stringutil.Truncate(string(output), 203)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Some artifacts could not be extracted (not a valid zip archive) and were skipped: "+msg))
		return true, false, nil
	}
	if isCaseCollisionArtifactError(output) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Some artifacts could not be fully extracted due to case-colliding file paths. Retrying artifacts individually and continuing."))
		return false, true, nil
	}
	return false, false, fmt.Errorf("failed to download artifacts for run %d: %w (output: %s)", opts.runID, cmdErr, string(output))
}

// handleCaseCollisionRetry retries artifact downloads individually after a case-collision
// failure in bulk download.
func handleCaseCollisionRetry(ctx context.Context, opts downloadArtifactsOptions, downloadableNames []string) error {
	retryNames := downloadableNames
	if len(retryNames) == 0 {
		artifactNamesRetry, retryListErr := listRunArtifactNames(ctx, opts.runID, opts.owner, opts.repo, opts.hostname, opts.verbose)
		if retryListErr != nil {
			return fmt.Errorf("bulk artifact download hit case-colliding entries and could not list artifacts for individual retry: %w", retryListErr)
		}
		for _, name := range artifactNamesRetry {
			if !isDockerBuildArtifact(name) && artifactMatchesFilter(name, opts.artifactFilter) {
				retryNames = append(retryNames, name)
			}
		}
	}
	if len(retryNames) > 0 {
		if err := downloadArtifactsByName(ctx, opts, retryNames); err != nil {
			return err
		}
	}
	return nil
}

// flattenDownloadedArtifacts runs all artifact flattening steps and downloads workflow run logs.
func flattenDownloadedArtifacts(ctx context.Context, opts downloadArtifactsOptions) error {
	if err := flattenSingleFileArtifacts(opts.outputDir, opts.verbose); err != nil {
		return fmt.Errorf("failed to flatten artifacts: %w", err)
	}
	if err := flattenActivationArtifact(opts.outputDir, opts.verbose); err != nil {
		return fmt.Errorf("failed to flatten activation artifact: %w", err)
	}
	ensureUsageAwInfoFallback(ctx, opts)
	if err := flattenUnifiedArtifact(opts.outputDir, opts.verbose); err != nil {
		return fmt.Errorf("failed to flatten unified artifact: %w", err)
	}
	if err := flattenAgentOutputsArtifact(opts.outputDir, opts.verbose); err != nil {
		return fmt.Errorf("failed to flatten agent_outputs artifact: %w", err)
	}
	if err := flattenSafeOutputsItemsArtifact(opts.outputDir, opts.verbose); err != nil {
		return fmt.Errorf("failed to flatten safe-outputs-items artifact: %w", err)
	}
	if shouldDownloadWorkflowRunLogs(opts.artifactFilter) {
		if err := downloadWorkflowRunLogs(ctx, opts.runID, opts.outputDir, opts.verbose, opts.owner, opts.repo, opts.hostname); err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", err)))
			}
		}
	}
	return nil
}

// logVerboseArtifactSummary prints a summary of downloaded artifact files to stderr.
func logVerboseArtifactSummary(outputDir string) {
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Downloaded artifacts to "+outputDir))
	var fileCount int
	var firstFiles []string
	var walkFailed bool
	if walkErr := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logsDownloadLog.Printf("walk error at %s: %v", path, err)
			walkFailed = true
			return nil
		}
		if info.IsDir() {
			return nil
		}
		fileCount++
		if len(firstFiles) < 12 {
			rel, relErr := filepath.Rel(outputDir, path)
			if relErr == nil {
				firstFiles = append(firstFiles, rel)
			}
		}
		return nil
	}); walkErr != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("filesystem error enumerating artifacts in %s: %v", outputDir, walkErr)))
	}
	if fileCount == 0 {
		if walkFailed {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Download completed but artifact files could not be enumerated (filesystem error)"))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Download completed but no artifact files were created (empty run)"))
		}
	} else {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Artifact file count: %d", fileCount)))
		for _, f := range firstFiles {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("  • "+f))
		}
		if fileCount > len(firstFiles) {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("  … %d more files omitted", fileCount-len(firstFiles))))
		}
	}
}

func ensureUsageAwInfoFallback(ctx context.Context, opts downloadArtifactsOptions) {
	if !isUsageOnlyArtifactFilter(opts.artifactFilter) {
		return
	}

	awInfoPath := filepath.Join(opts.outputDir, "aw_info.json")
	if fileutil.FileExists(awInfoPath) {
		return
	}

	logsDownloadLog.Printf("aw_info.json missing from usage artifact, downloading activation artifact as fallback")
	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("aw_info.json missing from usage artifact; downloading activation artifact as fallback"))
	}

	activationNames := []string{constants.ActivationArtifactName}
	if artifactNames, err := listRunArtifactNames(ctx, opts.runID, opts.owner, opts.repo, opts.hostname, opts.verbose); err == nil {
		var matched []string
		for _, name := range artifactNames {
			if artifactMatchesFilter(name, []string{constants.ActivationArtifactName}) {
				matched = append(matched, name)
			}
		}
		if len(matched) > 0 {
			activationNames = matched
		}
	} else {
		logsDownloadLog.Printf("Failed to list artifacts for activation fallback: %v", err)
	}

	if err := downloadArtifactsByName(ctx, opts, activationNames); err != nil {
		logsDownloadLog.Printf("Activation artifact fallback download failed: %v", err)
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not download activation artifact fallback: %v", err)))
		}
	}
	foundActivationDir := false
	for _, name := range activationNames {
		activationDir := findArtifactDir(opts.outputDir, name, "")
		if activationDir != "" {
			foundActivationDir = true
			logsDownloadLog.Printf("Found activation artifact fallback directory: %s", activationDir)
			if err := flattenActivationArtifact(opts.outputDir, opts.verbose); err != nil {
				logsDownloadLog.Printf("Failed to flatten fallback activation artifact: %v", err)
				if opts.verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to flatten activation artifact fallback: %v", err)))
				}
			}
			break
		}
	}
	if !foundActivationDir {
		logsDownloadLog.Print("Activation artifact fallback directory not found after download attempt")
	}
	if _, err := os.Stat(awInfoPath); os.IsNotExist(err) {
		logsDownloadLog.Print("aw_info.json still absent after activation artifact fallback")
		if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("aw_info.json still absent after activation artifact fallback"))
		}
	}
}
