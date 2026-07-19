package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var removeLog = logger.New("cli:remove_command")

// RemoveWorkflows removes workflows matching a pattern
func RemoveWorkflows(pattern string, keepOrphans bool, workflowDir string) error {
	removeLog.Printf("Removing workflows: pattern=%q, keepOrphans=%v, workflowDir=%q", pattern, keepOrphans, workflowDir)
	workflowsDir := RemoveWorkflowsDir(workflowDir)
	mdFiles, err := RemoveWorkflowsMarkdownFiles(workflowsDir)
	if err != nil || len(mdFiles) == 0 {
		return err
	}

	if pattern == "" {
		RemoveWorkflowsPrintAvailable(mdFiles)
		return nil
	}

	filesToRemove := RemoveWorkflowsMatchingFiles(mdFiles, pattern)
	if len(filesToRemove) == 0 {
		removeLog.Printf("No workflows matched pattern: %q", pattern)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflows found matching pattern: "+pattern))
		return nil
	}
	removeLog.Printf("Found %d workflows to remove", len(filesToRemove))

	orphanedIncludes := RemoveWorkflowsPreviewOrphans(filesToRemove, keepOrphans)
	RemoveWorkflowsPrintPlan(filesToRemove, orphanedIncludes)
	confirmed, err := RemoveWorkflowsConfirm()
	if err != nil || !confirmed {
		return err
	}

	removedFiles := RemoveWorkflowsDeleteFiles(filesToRemove)
	RemoveWorkflowsCleanupOrphans(removedFiles, keepOrphans)
	if len(removedFiles) > 0 && isGitRepo() {
		stageWorkflowChanges()
	}
	return nil
}

func RemoveWorkflowsDir(workflowDir string) string {
	if workflowDir != "" {
		return workflowDir
	}
	return getWorkflowsDir()
}

func RemoveWorkflowsMarkdownFiles(workflowsDir string) ([]string, error) {
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No .github/workflows directory found."))
		return nil, nil
	}
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow files: %w", err)
	}
	mdFiles = filterWorkflowFiles(mdFiles)
	removeLog.Printf("Found %d workflow files", len(mdFiles))
	if len(mdFiles) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflow files found to remove."))
	}
	return mdFiles, nil
}

func RemoveWorkflowsPrintAvailable(mdFiles []string) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Available workflows to remove:"))
	for _, file := range mdFiles {
		workflowName, _ := extractWorkflowNameFromFile(file)
		name := normalizeWorkflowID(filepath.Base(file))
		if workflowName != "" {
			fmt.Fprintf(os.Stderr, "  %-20s - %s\n", name, workflowName)
		} else {
			fmt.Fprintf(os.Stderr, "  %s\n", name)
		}
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("\nUsage: %s remove <filter>", string(constants.CLIExtensionPrefix))))
}

func RemoveWorkflowsMatchingFiles(mdFiles []string, pattern string) []string {
	var filesToRemove []string
	for _, file := range mdFiles {
		filename := normalizeWorkflowID(filepath.Base(file))
		workflowName, _ := extractWorkflowNameFromFile(file)
		if strings.Contains(strings.ToLower(filename), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(workflowName), strings.ToLower(pattern)) {
			filesToRemove = append(filesToRemove, file)
		}
	}
	return filesToRemove
}

func RemoveWorkflowsPreviewOrphans(filesToRemove []string, keepOrphans bool) []string {
	if keepOrphans {
		return nil
	}
	orphanedIncludes, err := previewOrphanedIncludes(filesToRemove, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to preview orphaned includes: %v", err)))
		return []string{}
	}
	return orphanedIncludes
}

func RemoveWorkflowsPrintPlan(filesToRemove, orphanedIncludes []string) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("The following workflows will be removed:"))
	for _, file := range filesToRemove {
		RemoveWorkflowsPrintPlanFile(file)
	}
	if len(orphanedIncludes) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nThe following orphaned include files will also be removed (suppress with --keep-orphans):"))
		for _, include := range orphanedIncludes {
			fmt.Fprintf(os.Stderr, "  %s (orphaned include)\n", include)
		}
	}
}

func RemoveWorkflowsPrintPlanFile(file string) {
	workflowName, _ := extractWorkflowNameFromFile(file)
	if workflowName != "" {
		fmt.Fprintf(os.Stderr, "  %s - %s\n", filepath.Base(file), workflowName)
	} else {
		fmt.Fprintf(os.Stderr, "  %s\n", filepath.Base(file))
	}
	lockFile := stringutil.MarkdownToLockFile(file)
	if fileutil.FileExists(lockFile) {
		fmt.Fprintf(os.Stderr, "  %s (compiled workflow)\n", filepath.Base(lockFile))
	}
}

func RemoveWorkflowsConfirm() (bool, error) {
	confirmed, err := console.ConfirmAction("Are you sure you want to remove these workflows?", "Yes, remove", "No, cancel")
	if err != nil {
		return false, fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Operation cancelled."))
	}
	return confirmed, nil
}

func RemoveWorkflowsDeleteFiles(filesToRemove []string) []string {
	var removedFiles []string
	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove %s: %v", file, err)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Removed: "+filepath.Base(file)))
			removedFiles = append(removedFiles, file)
		}
		RemoveWorkflowsDeleteLockFile(file)
	}
	return removedFiles
}

func RemoveWorkflowsDeleteLockFile(file string) {
	lockFile := stringutil.MarkdownToLockFile(file)
	if !fileutil.FileExists(lockFile) {
		return
	}
	if err := os.Remove(lockFile); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove %s: %v", lockFile, err)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Removed: "+filepath.Base(lockFile)))
	}
}

func RemoveWorkflowsCleanupOrphans(removedFiles []string, keepOrphans bool) {
	if len(removedFiles) == 0 || keepOrphans {
		return
	}
	if err := cleanupOrphanedIncludes(false); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up orphaned includes: %v", err)))
	}
}

// cleanupOrphanedIncludes removes include files that are no longer used by any workflow
func cleanupOrphanedIncludes(verbose bool) error {
	removeLog.Print("Cleaning up orphaned include files")
	usedIncludes, err := cleanupOrphanedIncludesUsed(verbose)
	if err != nil {
		removeLog.Print("No markdown files found, cleaning up all includes")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No markdown files found, cleaning up all includes"))
		}
		return cleanupAllIncludes(verbose)
	}

	allIncludes, err := getAllIncludeFiles()
	if err != nil {
		return fmt.Errorf("failed to scan include files: %w", err)
	}
	cleanupOrphanedIncludesRemove(allIncludes, usedIncludes, verbose)
	return nil
}

func cleanupOrphanedIncludesUsed(verbose bool) (map[string]struct{}, error) {
	mdFiles, err := getMarkdownWorkflowFiles("")
	if err != nil {
		return nil, err
	}
	usedIncludes := make(map[string]struct{})
	for _, mdFile := range mdFiles {
		content, err := os.ReadFile(mdFile)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not read %s for include analysis: %v", mdFile, err)))
			}
			continue
		}
		includes, err := findIncludesInContent(string(content))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not analyze includes in %s: %v", mdFile, err)))
			}
			continue
		}
		for _, include := range includes {
			usedIncludes[include] = struct{}{}
		}
	}
	return usedIncludes, nil
}

func cleanupOrphanedIncludesRemove(allIncludes []string, usedIncludes map[string]struct{}, verbose bool) {
	workflowsDir := constants.GetWorkflowDir()
	for _, include := range allIncludes {
		if setutil.Contains(usedIncludes, include) {
			continue
		}
		includePath := filepath.Join(workflowsDir, include)
		if err := os.Remove(includePath); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove orphaned include %s: %v", include, err)))
			}
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Removed orphaned include: "+include))
		}
	}
}

// previewOrphanedIncludes returns a list of include files that would become orphaned if the specified files were removed
func previewOrphanedIncludes(filesToRemove []string, verbose bool) ([]string, error) {
	allMdFiles, err := getMarkdownWorkflowFiles("")
	if err != nil {
		return nil, err
	}
	remainingFiles := previewOrphanedIncludesRemainingFiles(allMdFiles, filesToRemove)
	if len(remainingFiles) == 0 {
		return getAllIncludeFiles()
	}

	usedIncludes := previewOrphanedIncludesUsed(remainingFiles, verbose)
	allIncludes, err := getAllIncludeFiles()
	if err != nil {
		return nil, err
	}

	var orphanedIncludes []string
	for _, include := range allIncludes {
		if !setutil.Contains(usedIncludes, include) {
			orphanedIncludes = append(orphanedIncludes, include)
		}
	}
	return orphanedIncludes, nil
}

func previewOrphanedIncludesRemainingFiles(allMdFiles, filesToRemove []string) []string {
	removeMap := make(map[string]struct{})
	for _, file := range filesToRemove {
		removeMap[file] = struct{}{}
	}
	var remainingFiles []string
	for _, file := range allMdFiles {
		if !setutil.Contains(removeMap, file) {
			remainingFiles = append(remainingFiles, file)
		}
	}
	return remainingFiles
}

func previewOrphanedIncludesUsed(remainingFiles []string, verbose bool) map[string]struct{} {
	usedIncludes := make(map[string]struct{})
	for _, mdFile := range remainingFiles {
		content, err := os.ReadFile(mdFile)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not read %s for include analysis: %v", mdFile, err)))
			}
			continue
		}
		includes, err := findIncludesInContent(string(content))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not analyze includes in %s: %v", mdFile, err)))
			}
			continue
		}
		for _, include := range includes {
			usedIncludes[include] = struct{}{}
		}
	}
	return usedIncludes
}

// getAllIncludeFiles returns all include files in .github/workflows subdirectories
func getAllIncludeFiles() ([]string, error) {
	workflowsDir := constants.GetWorkflowDir()
	var allIncludes []string

	err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, err := filepath.Rel(workflowsDir, path)
			if err != nil {
				return err
			}

			// Only consider files in subdirectories as potential include files
			// Root-level .md files are workflow files, not include files
			if strings.Contains(relPath, string(filepath.Separator)) {
				allIncludes = append(allIncludes, relPath)
			}
		}

		return nil
	})

	return allIncludes, err
}

// cleanupAllIncludes removes all include files when no workflows remain
func cleanupAllIncludes(verbose bool) error {
	workflowsDir := constants.GetWorkflowDir()

	err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, _ := filepath.Rel(workflowsDir, path)

			// Only remove files in subdirectories (like shared/) as these are include files
			// Root-level .md files are workflow files, not include files
			if strings.Contains(relPath, string(filepath.Separator)) {
				if err := os.Remove(path); err != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove include %s: %v", relPath, err)))
					}
				} else {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Removed include: "+relPath))
				}
			}
		}

		return nil
	})

	return err
}

// findIncludesInContent finds all import references in content
func findIncludesInContent(content string) ([]string, error) {
	var includes []string
	// Manual index-based scan avoids the iter.Seq yield overhead of strings.Lines.
	for remaining := content; remaining != ""; {
		var line string
		if idx := strings.IndexByte(remaining, '\n'); idx >= 0 {
			line = remaining[:idx]
			remaining = remaining[idx+1:]
		} else {
			line = remaining
			remaining = ""
		}
		if path := parseIncludePath(line); path != "" {
			if includes == nil {
				includes = make([]string, 0, 4)
			}
			includes = append(includes, path)
		}
	}

	if includes == nil {
		return []string{}, nil
	}
	return includes, nil
}

// parseIncludePath extracts the file path from @include/@import/{{#import}} directive lines
// without allocating a regex submatch slice or a directive struct.
// Returns an empty string if the line is not a recognised directive.
func parseIncludePath(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if trimmed[0] != '@' && trimmed[0] != '{' {
		return ""
	}

	var rest string
	switch {
	case strings.HasPrefix(trimmed, "@include"):
		rest = trimmed[len("@include"):]
	case strings.HasPrefix(trimmed, "@import"):
		rest = trimmed[len("@import"):]
	case strings.HasPrefix(trimmed, "{{#import"):
		return parseIncludePathBraceDirective(trimmed[len("{{#import"):])
	default:
		return ""
	}

	if rest != "" && rest[0] == '?' {
		rest = rest[1:]
	}
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return ""
	}
	return parseIncludePathStripSection(strings.TrimSpace(rest))
}

func parseIncludePathBraceDirective(rest string) string {
	if rest != "" && rest[0] == '?' {
		rest = rest[1:]
	}
	rest = strings.TrimSpace(rest)
	if rest != "" && rest[0] == ':' {
		rest = strings.TrimSpace(rest[1:])
	}
	before, after, ok := strings.Cut(rest, "}}")
	if !ok || strings.TrimSpace(after) != "" {
		return ""
	}
	return parseIncludePathStripSection(strings.TrimSpace(before))
}

func parseIncludePathStripSection(path string) string {
	if path == "" {
		return ""
	}
	if filePath, _, ok := strings.Cut(path, "#"); ok {
		return filePath
	}
	return path
}
