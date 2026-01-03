// Package cli provides command-line interface functionality for gh-aw.
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/githubnext/gh-aw/pkg/cli/fileutil"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var logsDownloadLog = logger.New("cli:logs_download")

// flattenSingleFileArtifacts checks artifact directories and flattens any that contain a single file
// This handles the case where gh CLI creates a directory for each artifact, even if it's just one file
func flattenSingleFileArtifacts(outputDir string, verbose bool) error {
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
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read artifact directory %s: %v", artifactDir, err)))
			}
			continue
		}

		// Apply unfold rule: Check if directory contains exactly one entry and it's a file
		if len(artifactEntries) != 1 {
			continue
		}

		singleEntry := artifactEntries[0]
		if singleEntry.IsDir() {
			continue
		}

		// Unfold: Move the single file to parent directory and remove the artifact folder
		sourcePath := filepath.Join(artifactDir, singleEntry.Name())
		destPath := filepath.Join(outputDir, singleEntry.Name())

		// Move the file to root (parent directory)
		if err := os.Rename(sourcePath, destPath); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to move file %s to %s: %v", sourcePath, destPath, err)))
			}
			continue
		}

		// Delete the now-empty artifact folder
		if err := os.Remove(artifactDir); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove empty directory %s: %v", artifactDir, err)))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Unfolded single-file artifact: %s → %s", filepath.Join(entry.Name(), singleEntry.Name()), singleEntry.Name())))
		}
	}

	return nil
}

// downloadWorkflowRunLogs downloads and unzips workflow run logs using GitHub API
func downloadWorkflowRunLogs(runID int64, outputDir string, verbose bool) error {
	logsDownloadLog.Printf("Downloading workflow run logs: run_id=%d, output_dir=%s", runID, outputDir)

	// Create a temporary file for the zip download
	tmpZip := filepath.Join(os.TempDir(), fmt.Sprintf("workflow-logs-%d.zip", runID))
	defer os.RemoveAll(tmpZip)

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Downloading workflow run logs for run %d...", runID)))
	}

	// Use gh api to download the logs zip file
	// The endpoint returns a 302 redirect to the actual zip file
	cmd := workflow.ExecGH("api", "repos/{owner}/{repo}/actions/runs/"+strconv.FormatInt(runID, 10)+"/logs")
	output, err := cmd.Output()
	if err != nil {
		// Check for authentication errors
		if strings.Contains(err.Error(), "exit status 4") {
			return fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		// If logs are not found or run has no logs, this is not a critical error
		if strings.Contains(string(output), "not found") || strings.Contains(err.Error(), "410") {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("No logs found for run %d (may be expired or unavailable)", runID)))
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
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Downloaded and extracted workflow run logs to %s", workflowLogsDir)))
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

	// Calculate total size for progress bar
	var totalSize int64
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			totalSize += int64(f.UncompressedSize64)
		}
	}

	// Create progress bar for extraction
	var progressBar *console.ProgressBar
	var extractedSize int64
	if totalSize > 0 && !verbose {
		progressBar = console.NewProgressBar(totalSize)
		// Show initial progress
		fmt.Fprintf(os.Stderr, "\rExtracting: %s", progressBar.Update(0))
	}

	// Extract each file in the zip
	for _, f := range r.File {
		if err := extractZipFile(f, destDir, verbose); err != nil {
			if progressBar != nil {
				fmt.Fprintln(os.Stderr) // New line after progress bar
			}
			return err
		}

		// Update progress bar if not in verbose mode
		if progressBar != nil && !f.FileInfo().IsDir() {
			extractedSize += int64(f.UncompressedSize64)
			fmt.Fprintf(os.Stderr, "\rExtracting: %s", progressBar.Update(extractedSize))
		}
	}

	// Clear progress bar line and show completion
	if progressBar != nil {
		fmt.Fprintln(os.Stderr) // New line after progress bar
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive
func extractZipFile(f *zip.File, destDir string, verbose bool) error {
	// Construct the full path for the file
	filePath := filepath.Join(destDir, f.Name)

	// Prevent zip slip vulnerability
	if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path in zip: %s", f.Name)
	}

	if verbose {
		fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Extracting: %s", f.Name)))
	}

	// Create directory if it's a directory entry
	if f.FileInfo().IsDir() {
		return os.MkdirAll(filePath, os.ModePerm)
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
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
	defer destFile.Close()

	// Copy the content
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
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

// downloadRunArtifacts downloads artifacts for a specific workflow run
func downloadRunArtifacts(runID int64, outputDir string, verbose bool) error {
	logsDownloadLog.Printf("Downloading run artifacts: run_id=%d, output_dir=%s", runID, outputDir)

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
		fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Created output directory %s", outputDir)))
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Executing: gh run download %s --dir %s", strconv.FormatInt(runID, 10), outputDir)))
	}

	// Start spinner for network operation
	spinner := console.NewSpinner(fmt.Sprintf("Downloading artifacts for run %d...", runID))
	if !verbose {
		spinner.Start()
	}

	cmd := workflow.ExecGH("run", "download", strconv.FormatInt(runID, 10), "--dir", outputDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Stop spinner on error
		if !verbose {
			spinner.Stop()
		}
		if verbose {
			fmt.Println(console.FormatVerboseMessage(string(output)))
		}

		// Check if it's because there are no artifacts
		if strings.Contains(string(output), "no valid artifacts") || strings.Contains(string(output), "not found") {
			// Clean up empty directory
			if err := os.RemoveAll(outputDir); err != nil && verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up empty directory %s: %v", outputDir, err)))
			}
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("No artifacts found for run %d (gh run download reported none)", runID)))
			}
			return ErrNoArtifacts
		}
		// Check for authentication errors
		if strings.Contains(err.Error(), "exit status 4") {
			return fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		return fmt.Errorf("failed to download artifacts for run %d: %w (output: %s)", runID, err, string(output))
	}

	// Stop spinner with success message
	if !verbose {
		spinner.StopWithMessage(fmt.Sprintf("✓ Downloaded artifacts for run %d", runID))
	}

	// Flatten single-file artifacts
	if err := flattenSingleFileArtifacts(outputDir, verbose); err != nil {
		return fmt.Errorf("failed to flatten artifacts: %w", err)
	}

	// Download and unzip workflow run logs
	if err := downloadWorkflowRunLogs(runID, outputDir, verbose); err != nil {
		// Log the error but don't fail the entire download process
		// Logs may not be available for all runs (e.g., expired or deleted)
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to download workflow run logs: %v", err)))
		}
	}

	if verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Downloaded artifacts for run %d to %s", runID, outputDir)))
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
			fmt.Println(console.FormatWarningMessage("Download completed but no artifact files were created (empty run)"))
		} else {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Artifact file count: %d", fileCount)))
			for _, f := range firstFiles {
				fmt.Println(console.FormatVerboseMessage("  • " + f))
			}
			if fileCount > len(firstFiles) {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("  … %d more files omitted", fileCount-len(firstFiles))))
			}
		}
	}

	return nil
}
