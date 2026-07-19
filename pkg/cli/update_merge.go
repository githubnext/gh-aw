package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var updateMergeLog = logger.New("cli:update_merge")

// hasLocalModifications checks if the local workflow file has been modified from its source
// It resolves the source field and imports on the remote content, then compares with local
// Note: stop-after field is ignored during comparison as it's a deployment-specific setting
// localWorkflowDir, if non-empty, is passed to import processing so that relative import paths
// whose files exist locally are preserved — giving an accurate comparison against local content.
func hasLocalModifications(sourceContent, localContent, sourceSpec, localWorkflowDir string, verbose bool) bool {
	updateMergeLog.Printf("Checking for local modifications: source_spec=%s", sourceSpec)
	// Normalize both contents
	sourceNormalized, localNormalized := hasLocalModificationsNormalize(sourceContent, localContent)

	// Parse the source spec to get repo and ref information
	parsedSourceSpec, err := parseSourceSpec(sourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to parse source spec: %v", err)))
		}
		// Fall back to simple comparison
		return sourceNormalized != localNormalized
	}

	sourceResolvedNormalized, localNormalized := hasLocalModificationsResolveSource(sourceNormalized, localNormalized, sourceSpec, parsedSourceSpec, localWorkflowDir, verbose)
	hasModifications := sourceResolvedNormalized != localNormalized
	updateMergeLog.Printf("Local modifications detected: %v", hasModifications)

	if verbose && hasModifications {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Local modifications detected"))
	}
	return hasModifications
}

func hasLocalModificationsNormalize(sourceContent, localContent string) (string, string) {
	sourceNormalized := stringutil.NormalizeWhitespace(sourceContent)
	localNormalized := stringutil.NormalizeWhitespace(localContent)
	sourceNormalized, _ = RemoveFieldFromOnTrigger(sourceNormalized, "stop-after")
	localNormalized, _ = RemoveFieldFromOnTrigger(localNormalized, "stop-after")
	return sourceNormalized, localNormalized
}

func hasLocalModificationsResolveSource(sourceNormalized, localNormalized, sourceSpec string, parsedSourceSpec *SourceSpec, localWorkflowDir string, verbose bool) (string, string) {
	sourceWithSource, err := UpdateFieldInFrontmatter(sourceNormalized, "source", sourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to add source field to remote content: %v", err)))
		}
		return sourceNormalized, localNormalized
	}

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: parsedSourceSpec.Repo,
			Version:  parsedSourceSpec.Ref,
		},
		WorkflowPath: parsedSourceSpec.Path,
	}
	sourceResolved, err := processIncludesInContent(sourceWithSource, workflow, parsedSourceSpec.Ref, localWorkflowDir, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to process imports on remote content: %v", err)))
		}
		// Use the version with source field but without resolved imports
		sourceResolved = sourceWithSource
	}

	// Normalize again after processing
	sourceResolvedNormalized := stringutil.NormalizeWhitespace(sourceResolved)
	if normalized, normalizeErr := UpdateFieldInFrontmatter(sourceResolvedNormalized, "source", "__gh_aw_source__"); normalizeErr == nil {
		sourceResolvedNormalized = normalized
	}
	if normalized, normalizeErr := UpdateFieldInFrontmatter(localNormalized, "source", "__gh_aw_source__"); normalizeErr == nil {
		localNormalized = normalized
	}
	return sourceResolvedNormalized, localNormalized
}

// MergeWorkflowContent performs a 3-way merge of workflow content using git merge-file
// It returns the merged content, whether conflicts exist, and any error
// localWorkflowPath is the filesystem path of the local workflow file being updated;
// when non-empty its directory is used to preserve relative import paths whose files
// exist locally rather than rewriting them to cross-repo references.
func MergeWorkflowContent(base, current, new, oldSourceSpec, newRefOrSourceSpec, localWorkflowPath string, verbose bool) (string, bool, error) {
	updateMergeLog.Printf("Starting 3-way merge: old_source=%s, new_ref_or_source=%s", oldSourceSpec, newRefOrSourceSpec)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Performing 3-way merge using git merge-file"))
	}

	specs, err := mergeWorkflowContentSpecs(oldSourceSpec, newRefOrSourceSpec)
	if err != nil {
		return "", false, err
	}
	baseNormalized, currentNormalized, newNormalized := mergeWorkflowContentNormalize(base, current, new, specs.currentSourceSpec, specs.newSourceSpec, verbose)

	currentFile, cleanup, err := mergeWorkflowContentWriteTempFiles(baseNormalized, currentNormalized, newNormalized)
	if err != nil {
		return "", false, err
	}
	defer cleanup()
	hasConflicts, err := mergeWorkflowContentRunGitMerge(currentFile, verbose)
	if err != nil {
		return "", false, err
	}

	updateMergeLog.Printf("Merge completed: has_conflicts=%v", hasConflicts)
	mergedContent, err := os.ReadFile(currentFile)
	if err != nil {
		return "", false, fmt.Errorf("failed to read merged content: %w", err)
	}

	mergedStr := string(mergedContent)
	mergedStr = mergeWorkflowContentProcessIncludes(mergedStr, hasConflicts, specs.parsedNewSourceSpec, specs.newRef, localWorkflowPath, verbose)
	return mergedStr, hasConflicts, nil
}

type mergeWorkflowContentSpecInfo struct {
	currentSourceSpec   string
	newSourceSpec       string
	parsedNewSourceSpec *SourceSpec
	newRef              string
}

func mergeWorkflowContentSpecs(oldSourceSpec, newRefOrSourceSpec string) (mergeWorkflowContentSpecInfo, error) {
	sourceSpec, err := parseSourceSpec(oldSourceSpec)
	if err != nil {
		updateMergeLog.Printf("Failed to parse source spec: %v", err)
		return mergeWorkflowContentSpecInfo{}, fmt.Errorf("failed to parse source spec: %w", err)
	}
	currentSourceSpec := fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, sourceSpec.Ref)
	newSourceSpec := fmt.Sprintf("%s/%s@%s", sourceSpec.Repo, sourceSpec.Path, newRefOrSourceSpec)
	if tentativeSourceSpec, parseErr := parseSourceSpec(newRefOrSourceSpec); parseErr == nil {
		newSourceSpec = sourceSpecWithRef(tentativeSourceSpec, tentativeSourceSpec.Ref)
	}
	parsedNewSourceSpec, err := parseSourceSpec(newSourceSpec)
	if err != nil {
		return mergeWorkflowContentSpecInfo{}, fmt.Errorf("failed to parse new source spec: %w", err)
	}
	return mergeWorkflowContentSpecInfo{currentSourceSpec: currentSourceSpec, newSourceSpec: newSourceSpec, parsedNewSourceSpec: parsedNewSourceSpec, newRef: parsedNewSourceSpec.Ref}, nil
}

func mergeWorkflowContentNormalize(base, current, new, currentSourceSpec, newSourceSpec string, verbose bool) (string, string, string) {
	baseWithSource, err := UpdateFieldInFrontmatter(base, "source", currentSourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add source to base content: %v", err)))
		}
		baseWithSource = base
	}
	newWithUpdatedSource, err := UpdateFieldInFrontmatter(new, "source", newSourceSpec)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update source in new content: %v", err)))
		}
		newWithUpdatedSource = new
	}
	baseNormalized := stringutil.NormalizeWhitespace(baseWithSource)
	currentNormalized := stringutil.NormalizeWhitespace(current)
	newNormalized := stringutil.NormalizeWhitespace(newWithUpdatedSource)
	if normalizedCurrent, normalizeErr := UpdateFieldInFrontmatter(currentNormalized, "source", currentSourceSpec); normalizeErr == nil {
		currentNormalized = normalizedCurrent
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to normalize source in current content: %v", normalizeErr)))
	}
	return baseNormalized, currentNormalized, newNormalized
}

func mergeWorkflowContentWriteTempFiles(baseNormalized, currentNormalized, newNormalized string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "gh-aw-merge-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }
	baseFile := filepath.Join(tmpDir, "base.md")
	currentFile := filepath.Join(tmpDir, "current.md")
	newFile := filepath.Join(tmpDir, "new.md")
	if err := os.WriteFile(baseFile, []byte(baseNormalized), constants.FilePermPublic); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to write base file: %w", err)
	}
	if err := os.WriteFile(currentFile, []byte(currentNormalized), constants.FilePermPublic); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to write current file: %w", err)
	}
	if err := os.WriteFile(newFile, []byte(newNormalized), constants.FilePermPublic); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to write new file: %w", err)
	}
	return currentFile, cleanup, nil
}

func mergeWorkflowContentRunGitMerge(currentFile string, verbose bool) (bool, error) {
	baseFile := filepath.Join(filepath.Dir(currentFile), "base.md")
	newFile := filepath.Join(filepath.Dir(currentFile), "new.md")
	cmd := exec.Command("git", "merge-file", "-L", "current (local changes)", "-L", "base (original)", "-L", "new (upstream)", "--diff3", currentFile, baseFile, newFile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false, fmt.Errorf("failed to execute git merge-file: %w", err)
	}
	exitCode := exitErr.ExitCode()
	if exitCode <= 0 || exitCode >= 128 {
		updateMergeLog.Printf("Git merge-file failed: exit_code=%d", exitCode)
		return false, fmt.Errorf("git merge-file failed: %w\nOutput: %s", err, output)
	}
	updateMergeLog.Printf("Merge conflicts detected: exit_code=%d", exitCode)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Merge conflicts detected (exit code: %d)", exitCode)))
	}
	return true, nil
}

func mergeWorkflowContentProcessIncludes(mergedStr string, hasConflicts bool, parsedNewSourceSpec *SourceSpec, newRef, localWorkflowPath string, verbose bool) string {
	if hasConflicts {
		return mergedStr
	}
	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: parsedNewSourceSpec.Repo,
			Version:  newRef,
		},
		WorkflowPath: parsedNewSourceSpec.Path,
	}
	localWorkflowDir := ""
	if localWorkflowPath != "" {
		localWorkflowDir = filepath.Dir(localWorkflowPath)
	}
	processedContent, err := processIncludesInContent(mergedStr, workflow, newRef, localWorkflowDir, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
		}
		return mergedStr
	}
	return processedContent
}
