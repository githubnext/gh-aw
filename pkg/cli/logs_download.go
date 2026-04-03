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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsDownloadLog = logger.New("cli:logs_download")

// flattenSingleFileArtifacts checks artifact directories and flattens any that contain a single file
// This handles the case where gh CLI creates a directory for each artifact, even if it's just one file
func flattenSingleFileArtifacts(outputDir string, verbose bool) error {
	logsDownloadLog.Printf("Flattening single-file artifacts in: %s", outputDir)
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		artifactDir := filepath.Join(outputDir, entry.Name())

		// Read contents of artifact directory
		artifactEntries, err := os.ReadDir(artifactDir)
		if err != nil {
			logsDownloadLog.Printf("Failed to read artifact directory %s: %v", artifactDir, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read artifact directory %s: %v", artifactDir, err)))
			}
			continue
		}

		logsDownloadLog.Printf("Artifact directory %s contains %d entries", entry.Name(), len(artifactEntries))

		// Apply unfold rule: Check if directory contains exactly one entry and it's a file
		if len(artifactEntries) != 1 {
			if verbose && len(artifactEntries) > 1 {
				// Log what's in multi-file artifacts for debugging
				var fileNames []string
				for _, e := range artifactEntries {
					fileNames = append(fileNames, e.Name())
				}
				logsDownloadLog.Printf("Artifact directory %s has %d files, not flattening: %v", entry.Name(), len(artifactEntries), fileNames)
			}
			continue
		}

		singleEntry := artifactEntries[0]
		if singleEntry.IsDir() {
			logsDownloadLog.Printf("Artifact directory %s contains a subdirectory, not flattening", entry.Name())
			continue
		}

		// Unfold: Move the single file to parent directory and remove the artifact folder
		sourcePath := filepath.Join(artifactDir, singleEntry.Name())
		destPath := filepath.Join(outputDir, singleEntry.Name())

		logsDownloadLog.Printf("Flattening: %s → %s", sourcePath, destPath)

		// Move the file to root (parent directory)
		if err := os.Rename(sourcePath, destPath); err != nil {
			logsDownloadLog.Printf("Failed to move file %s to %s: %v", sourcePath, destPath, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to move file %s to %s: %v", sourcePath, destPath, err)))
			}
			continue
		}

		// Delete the now-empty artifact folder
		if err := os.Remove(artifactDir); err != nil {
			logsDownloadLog.Printf("Failed to remove empty directory %s: %v", artifactDir, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove empty directory %s: %v", artifactDir, err)))
			}
			continue
		}

		logsDownloadLog.Printf("Successfully flattened: %s/%s → %s", entry.Name(), singleEntry.Name(), singleEntry.Name())
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Unfolded single-file artifact: %s → %s", filepath.Join(entry.Name(), singleEntry.Name()), singleEntry.Name())))
		}
	}

	return nil
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
	if _, err := os.Stat(exactPath); err == nil {
		return exactPath
	}

	// Try legacy name if provided
	if legacyName != "" {
		legacyPath := filepath.Join(outputDir, legacyName)
		if _, err := os.Stat(legacyPath); err == nil {
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
			// Create directory in destination with owner+group permissions only (0750)
			if err := os.MkdirAll(destPath, 0750); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			logsDownloadLog.Printf("Created directory: %s", destPath)
		} else {
			// Ensure parent directory exists with owner+group permissions only (0750)
			if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
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

	// Remove the now-empty artifact directory structure.
	// Don't fail the entire operation if cleanup fails.
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

	return nil
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
	if _, err := os.Stat(tmpGhAwPath); err == nil {
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

// downloadWorkflowRunLogs downloads and unzips workflow run logs using GitHub API
func downloadWorkflowRunLogs(runID int64, outputDir string, verbose bool, owner, repo, hostname string) error {
	logsDownloadLog.Printf("Downloading workflow run logs: run_id=%d, output_dir=%s, owner=%s, repo=%s", runID, outputDir, owner, repo)

	// Create a temporary file for the zip download
	tmpZip := filepath.Join(os.TempDir(), fmt.Sprintf("workflow-logs-%d.zip", runID))
	defer os.RemoveAll(tmpZip)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Downloading workflow run logs for run %d...", runID)))
	}

	// Use gh api to download the logs zip file
	// The endpoint returns a 302 redirect to the actual zip file
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

	output, err := workflow.RunGH("Downloading workflow logs...", args...)
	if err != nil {
		// Check for authentication errors
		if strings.Contains(err.Error(), "exit status 4") {
			return errors.New("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		// If logs are not found or run has no logs, this is not a critical error
		if strings.Contains(string(output), "not found") || strings.Contains(err.Error(), "410") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No logs found for run %d (may be expired or unavailable)", runID)))
			}
			return nil
		}
		return fmt.Errorf("failed to download workflow run logs for run %d: %w", runID, err)
	}

	// Write the downloaded zip content to temporary file
	if err := os.WriteFile(tmpZip, output, 0644); err != nil {
		return fmt.Errorf("failed to write logs zip file: %w", err)
	}

	// Create a subdirectory for workflow logs to keep the run directory organized
	workflowLogsDir := filepath.Join(outputDir, "workflow-logs")
	if err := os.MkdirAll(workflowLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow-logs directory: %w", err)
	}

	// Unzip the logs into the workflow-logs subdirectory
	if err := unzipFile(tmpZip, workflowLogsDir, verbose); err != nil {
		return fmt.Errorf("failed to unzip workflow logs: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Downloaded and extracted workflow run logs to "+workflowLogsDir))
	}

	return nil
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
func extractZipFile(f *zip.File, destDir string, verbose bool) (extractErr error) {
	// #nosec G305 - Path traversal is prevented by filepath.Clean and prefix check below
	// Validate file name doesn't contain path traversal attempts
	cleanName := filepath.Clean(f.Name)
	if strings.Contains(cleanName, "..") {
		return fmt.Errorf("invalid file path in zip (contains ..): %s", f.Name)
	}

	// Construct the full path for the file
	filePath := filepath.Join(destDir, cleanName)

	// Prevent zip slip vulnerability - ensure extracted path is within destDir
	cleanDest := filepath.Clean(destDir)
	if !strings.HasPrefix(filepath.Clean(filePath), cleanDest+string(os.PathSeparator)) && filepath.Clean(filePath) != cleanDest {
		return fmt.Errorf("invalid file path in zip (outside destination): %s", f.Name)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Extracting: "+cleanName))
	}

	// Create directory if it's a directory entry
	if f.FileInfo().IsDir() {
		return os.MkdirAll(filePath, 0750)
	}

	// Decompression bomb protection - limit individual file size to 1GB
	// #nosec G110 - Decompression bomb is mitigated by size check below
	const maxFileSize = 1 * 1024 * 1024 * 1024 // 1GB
	if f.UncompressedSize64 > maxFileSize {
		return fmt.Errorf("file too large in zip (>1GB): %s (%d bytes)", f.Name, f.UncompressedSize64)
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open the file in the zip
	srcFile, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in zip: %w", err)
	}
	defer srcFile.Close()

	// Create the destination file
	destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		// Handle errors from closing the writable file to prevent data loss
		// Data written to a file may be cached in memory and only flushed when the file is closed.
		// If Close() fails and the error is ignored, data loss can occur silently.
		if err := destFile.Close(); extractErr == nil && err != nil {
			extractErr = fmt.Errorf("failed to close destination file: %w", err)
		}
	}()

	// Copy the content with size limit enforcement
	// Use LimitReader to prevent reading more than declared size
	limitedReader := io.LimitReader(srcFile, int64(maxFileSize))
	written, err := io.Copy(destFile, limitedReader)
	if err != nil {
		extractErr = fmt.Errorf("failed to extract file: %w", err)
		return extractErr
	}

	// Verify we didn't exceed the size limit
	if uint64(written) > maxFileSize {
		extractErr = fmt.Errorf("file extraction exceeded size limit: %s", f.Name)
		return extractErr
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
	return strings.Contains(s, "zip: not a valid zip file") || strings.Contains(s, "error extracting zip archive")
}

// isDockerBuildArtifact reports whether an artifact name represents a .dockerbuild artifact.
// These are not zip archives and cannot be extracted by gh run download.
func isDockerBuildArtifact(name string) bool {
	return strings.HasSuffix(name, ".dockerbuild")
}

// listRunArtifactNames returns the names of all artifacts for the given workflow run
// by querying the GitHub Actions API. Returns an error if the API call fails.
func listRunArtifactNames(runID int64, owner, repo, hostname string, verbose bool) ([]string, error) {
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

	cmd := workflow.ExecGH(args...)
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
func downloadArtifactsByName(runID int64, outputDir string, names []string, verbose bool, owner, repo, hostname string) error {
	var repoFlag string
	if owner != "" && repo != "" {
		if hostname != "" && hostname != "github.com" {
			repoFlag = hostname + "/" + owner + "/" + repo
		} else {
			repoFlag = owner + "/" + repo
		}
	}

	for _, name := range names {
		args := []string{"run", "download", strconv.FormatInt(runID, 10), "--name", name, "--dir", outputDir}
		if repoFlag != "" {
			args = append(args, "-R", repoFlag)
		}

		logsDownloadLog.Printf("Downloading artifact %q individually: gh %s", name, strings.Join(args, " "))
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Downloading artifact: "+name))
		}

		cmd := workflow.ExecGH(args...)
		cmdOutput, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			logsDownloadLog.Printf("Failed to download artifact %q: %v (%s)", name, cmdErr, string(cmdOutput))
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifact %q: %v", name, cmdErr)))
			}
			// Non-fatal: continue downloading other artifacts
		} else {
			logsDownloadLog.Printf("Downloaded artifact %q", name)
		}
	}

	return nil
}

// criticalArtifactNames lists the artifact names that are essential for audit analysis.
// When a bulk download fails partially (e.g., due to non-zip artifacts), these artifacts
// are retried individually so that flattening and audit extraction have data to work with.
var criticalArtifactNames = []string{"activation", "agent", "firewall-audit-logs"}

// retryCriticalArtifacts downloads critical artifacts individually when the bulk download
// was only partially successful. gh run download aborts on the first non-zip artifact,
// which may prevent valid artifacts from being downloaded.
func retryCriticalArtifacts(runID int64, outputDir string, verbose bool, owner, repo, hostname string) {
	// Build the repo flag once for reuse across retries
	var repoFlag string
	if owner != "" && repo != "" {
		if hostname != "" && hostname != "github.com" {
			repoFlag = hostname + "/" + owner + "/" + repo
		} else {
			repoFlag = owner + "/" + repo
		}
	}

	for _, name := range criticalArtifactNames {
		artifactDir := filepath.Join(outputDir, name)
		if fileutil.DirExists(artifactDir) {
			logsDownloadLog.Printf("Critical artifact %q already present, skipping retry", name)
			continue
		}

		retryArgs := []string{"run", "download", strconv.FormatInt(runID, 10), "--name", name, "--dir", outputDir}
		if repoFlag != "" {
			retryArgs = append(retryArgs, "-R", repoFlag)
		}

		logsDownloadLog.Printf("Retrying individual download for artifact %q: gh %s", name, strings.Join(retryArgs, " "))
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Retrying download for missing artifact: "+name))
		}

		retryCmd := workflow.ExecGH(retryArgs...)
		retryOutput, retryErr := retryCmd.CombinedOutput()
		if retryErr != nil {
			logsDownloadLog.Printf("Failed to download artifact %q individually: %v (%s)", name, retryErr, string(retryOutput))
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not download artifact %q: %v", name, retryErr)))
			}
		} else {
			logsDownloadLog.Printf("Successfully downloaded artifact %q individually", name)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Downloaded missing artifact: "+name))
			}
		}
	}
}

// downloadRunArtifacts downloads artifacts for a specific workflow run
func downloadRunArtifacts(runID int64, outputDir string, verbose bool, owner, repo, hostname string) error {
	logsDownloadLog.Printf("Downloading run artifacts: run_id=%d, output_dir=%s, owner=%s, repo=%s", runID, outputDir, owner, repo)

	// Check if artifacts already exist on disk (since they're immutable)
	if fileutil.DirExists(outputDir) && !fileutil.IsDirEmpty(outputDir) {
		// Try to load cached summary
		if summary, ok := loadRunSummary(outputDir, verbose); ok {
			// Valid cached summary exists, skip download
			logsDownloadLog.Printf("Using cached artifacts for run %d", runID)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using cached artifacts for run %d at %s (from %s)", runID, outputDir, summary.ProcessedAt.Format("2006-01-02 15:04:05"))))
			}
			return nil
		}
		// Summary doesn't exist or version mismatch - artifacts exist but need reprocessing
		// Don't re-download, just reprocess what's there
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run folder exists with artifacts, will reprocess run %d without re-downloading", runID)))
		}
		// Return nil to indicate success - the artifacts are already there
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create run output directory: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Created output directory "+outputDir))
	}

	// Proactively list artifacts to detect .dockerbuild files that gh run download cannot
	// extract (they are not zip archives). When found, skip them and download the
	// remaining artifacts individually so the bulk download never encounters them.
	artifactNames, listErr := listRunArtifactNames(runID, owner, repo, hostname, verbose)
	var dockerBuildArtifacts, downloadableNames []string
	if listErr == nil {
		for _, name := range artifactNames {
			if isDockerBuildArtifact(name) {
				dockerBuildArtifacts = append(dockerBuildArtifacts, name)
			} else {
				downloadableNames = append(downloadableNames, name)
			}
		}
		if len(dockerBuildArtifacts) > 0 {
			logsDownloadLog.Printf("Found %d .dockerbuild artifact(s) that will be skipped: %v", len(dockerBuildArtifacts), dockerBuildArtifacts)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %d .dockerbuild artifact(s) (not valid zip archives): %s", len(dockerBuildArtifacts), strings.Join(dockerBuildArtifacts, ", "))))
		}
	} else {
		logsDownloadLog.Printf("Could not list artifacts (will use bulk download): %v", listErr)
	}

	// Start spinner for network operation
	spinner := console.NewSpinner(fmt.Sprintf("Downloading artifacts for run %d...", runID))
	if !verbose {
		spinner.Start()
	}

	if len(dockerBuildArtifacts) > 0 {
		// .dockerbuild artifacts detected — download only the valid artifacts individually
		// to avoid the bulk download aborting on the non-zip files.
		if !verbose {
			spinner.Stop()
		}
		if len(downloadableNames) == 0 {
			// All artifacts are .dockerbuild; nothing can be downloaded.
			// Attempt workflow run logs for diagnostics before returning.
			if logErr := downloadWorkflowRunLogs(runID, outputDir, verbose, owner, repo, hostname); logErr != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", logErr)))
				}
				if fileutil.IsDirEmpty(outputDir) {
					if removeErr := os.RemoveAll(outputDir); removeErr != nil && verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up empty directory %s: %v", outputDir, removeErr)))
					}
				}
			}
			return ErrNoArtifacts
		}
		if err := downloadArtifactsByName(runID, outputDir, downloadableNames, verbose, owner, repo, hostname); err != nil {
			return err
		}
		if fileutil.IsDirEmpty(outputDir) {
			// Downloads were attempted but none succeeded; treat as no artifacts.
			return ErrNoArtifacts
		}
	} else {
		// No .dockerbuild artifacts detected (or listing failed) — use efficient bulk download.
		// Build gh run download command with optional repo/hostname override for cross-repo and multi-host support
		ghArgs := []string{"run", "download", strconv.FormatInt(runID, 10), "--dir", outputDir}
		if owner != "" && repo != "" {
			if hostname != "" && hostname != "github.com" {
				ghArgs = append(ghArgs, "-R", hostname+"/"+owner+"/"+repo)
			} else {
				ghArgs = append(ghArgs, "-R", owner+"/"+repo)
			}
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executing: gh "+strings.Join(ghArgs, " ")))
		}

		cmd := workflow.ExecGH(ghArgs...)
		output, err := cmd.CombinedOutput()

		// skippedNonZipArtifacts is set when gh run download fails due to non-zip artifacts
		// that were not detected during the listing phase (e.g., listing failed).
		var skippedNonZipArtifacts bool

		if err != nil {
			// Stop spinner on error
			if !verbose {
				spinner.Stop()
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(string(output)))
			}

			// Check if it's because there are no artifacts
			if strings.Contains(string(output), "no valid artifacts") || strings.Contains(string(output), "not found") {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No artifacts found for run %d (gh run download reported none)", runID)))
				}
				// Even with no artifacts, attempt to download workflow run logs so that
				// pre-agent step failures (e.g., activation job errors) can be diagnosed.
				if logErr := downloadWorkflowRunLogs(runID, outputDir, verbose, owner, repo, hostname); logErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", logErr)))
					}
					// Clean up empty directory only if logs download also produced nothing
					if fileutil.IsDirEmpty(outputDir) {
						if removeErr := os.RemoveAll(outputDir); removeErr != nil && verbose {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up empty directory %s: %v", outputDir, removeErr)))
						}
					}
				}
				return ErrNoArtifacts
			}
			// Check for authentication errors
			if strings.Contains(err.Error(), "exit status 4") {
				return errors.New("GitHub CLI authentication required. Run 'gh auth login' first")
			}
			// Check if the error is due to non-zip artifacts (e.g., .dockerbuild files).
			// The gh CLI fails when it encounters artifacts that are not valid zip archives.
			// We warn and continue with any artifacts that were successfully downloaded.
			if isNonZipArtifactError(output) {
				// Show a concise warning; the raw output may be verbose so truncate it.
				msg := string(output)
				if len(msg) > 200 {
					msg = msg[:200] + "..."
				}
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Some artifacts could not be extracted (not a valid zip archive) and were skipped: "+msg))
				skippedNonZipArtifacts = true
			} else {
				return fmt.Errorf("failed to download artifacts for run %d: %w (output: %s)", runID, err, string(output))
			}
		}

		// When the bulk download failed due to non-zip artifacts, gh CLI may have aborted
		// before downloading all valid artifacts. Retry individually for critical artifacts
		// that are missing, so flattening and audit analysis can proceed.
		if skippedNonZipArtifacts {
			retryCriticalArtifacts(runID, outputDir, verbose, owner, repo, hostname)
		}

		if skippedNonZipArtifacts && fileutil.IsDirEmpty(outputDir) {
			// All artifacts were non-zip (none could be extracted) so nothing was downloaded.
			// Treat this the same as a run with no artifacts — the audit will rely solely on
			// workflow logs rather than artifact content.
			return ErrNoArtifacts
		}
	}

	// Stop spinner with success message
	if !verbose {
		spinner.StopWithMessage(fmt.Sprintf("✓ Downloaded artifacts for run %d", runID))
	}

	// Flatten single-file artifacts
	if err := flattenSingleFileArtifacts(outputDir, verbose); err != nil {
		return fmt.Errorf("failed to flatten artifacts: %w", err)
	}

	// Flatten activation artifact directory structure (contains aw_info.json and prompt.txt)
	if err := flattenActivationArtifact(outputDir, verbose); err != nil {
		return fmt.Errorf("failed to flatten activation artifact: %w", err)
	}

	// Flatten unified agent directory structure
	if err := flattenUnifiedArtifact(outputDir, verbose); err != nil {
		return fmt.Errorf("failed to flatten unified artifact: %w", err)
	}

	// Flatten agent_outputs artifact if present
	if err := flattenAgentOutputsArtifact(outputDir, verbose); err != nil {
		return fmt.Errorf("failed to flatten agent_outputs artifact: %w", err)
	}

	// Download and unzip workflow run logs
	if err := downloadWorkflowRunLogs(runID, outputDir, verbose, owner, repo, hostname); err != nil {
		// Log the error but don't fail the entire download process
		// Logs may not be available for all runs (e.g., expired or deleted)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", err)))
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded artifacts for run %d to %s", runID, outputDir)))
		// Enumerate created files (shallow + summary) for immediate visibility
		var fileCount int
		var firstFiles []string
		_ = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			fileCount++
			if len(firstFiles) < 12 { // capture a reasonable preview
				rel, relErr := filepath.Rel(outputDir, path)
				if relErr == nil {
					firstFiles = append(firstFiles, rel)
				}
			}
			return nil
		})
		if fileCount == 0 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Download completed but no artifact files were created (empty run)"))
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

	return nil
}
