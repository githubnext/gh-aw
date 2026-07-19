package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var runPushLog = logger.New("cli:run_push")

// collectWorkflowFiles collects the workflow .md file, its corresponding .lock.yml file,
// and the transitive closure of all imported files.
// Note: This function always recompiles the workflow to ensure the lock file is up-to-date,
// regardless of the frontmatter hash status.
func collectWorkflowFiles(ctx context.Context, workflowPath string, verbose bool, approve bool) ([]string, error) {
	runPushLog.Printf("Collecting files for workflow: %s", workflowPath)
	files, visited, absWorkflowPath, lockFilePath, err := collectWorkflowFilesInit(workflowPath)
	if err != nil {
		return nil, err
	}

	collectWorkflowFilesLogLockStatus(absWorkflowPath, lockFilePath, verbose, err)
	if err := recompileWorkflow(ctx, absWorkflowPath, verbose, approve); err != nil {
		runPushLog.Printf("Failed to recompile workflow: %v", err)
		return nil, fmt.Errorf("failed to recompile workflow: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow compiled successfully"))
	}
	runPushLog.Printf("Recompilation completed successfully")
	collectWorkflowFilesAddLock(files, lockFilePath, verbose)

	runPushLog.Printf("Starting import collection for %s", absWorkflowPath)
	if err := collectImports(absWorkflowPath, files, visited, verbose); err != nil {
		runPushLog.Printf("Failed to collect imports: %v", err)
		return nil, fmt.Errorf("failed to collect imports: %w", err)
	}
	result := sliceutil.MapKeys(files)
	sort.Strings(result)
	runPushLog.Printf("Collected %d files total", len(result))
	return result, nil
}

func collectWorkflowFilesInit(workflowPath string) (map[string]struct{}, map[string]struct{}, string, string, error) {
	files := make(map[string]struct{})
	visited := make(map[string]struct{})
	absWorkflowPath, err := filepath.Abs(workflowPath)
	if err != nil {
		runPushLog.Printf("Failed to get absolute path for %s: %v", workflowPath, err)
		return nil, nil, "", "", fmt.Errorf("failed to get absolute path for workflow: %w", err)
	}
	runPushLog.Printf("Resolved absolute workflow path: %s", absWorkflowPath)
	absWorkflowPath, err = fileutil.ValidateAbsolutePath(absWorkflowPath)
	if err != nil {
		runPushLog.Printf("Invalid workflow path: %v", err)
		return nil, nil, "", "", fmt.Errorf("invalid workflow path: %w", err)
	}
	files[absWorkflowPath] = struct{}{}
	runPushLog.Printf("Added workflow file: %s", absWorkflowPath)
	lockFilePath := stringutil.MarkdownToLockFile(absWorkflowPath)
	runPushLog.Printf("Checking lock file: %s", lockFilePath)
	return files, visited, absWorkflowPath, lockFilePath, nil
}

func collectWorkflowFilesLogLockStatus(absWorkflowPath, lockFilePath string, verbose bool, err error) {
	if fileutil.FileExists(lockFilePath) {
		runPushLog.Printf("Lock file exists: %s", lockFilePath)
		runPushLog.Print("Checking frontmatter hash for observability")
		if hashMismatch, err := checkFrontmatterHashMismatch(absWorkflowPath, lockFilePath); err != nil {
			runPushLog.Printf("Error checking frontmatter hash: %v", err)
		} else if hashMismatch {
			runPushLog.Print("Lock file frontmatter hash changed (will recompile)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Frontmatter hash changed, recompiling workflow..."))
			}
		} else {
			runPushLog.Print("Lock file frontmatter hash unchanged (will still recompile)")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Recompiling workflow..."))
			}
		}
	} else if os.IsNotExist(err) {
		runPushLog.Printf("Lock file not found: %s", lockFilePath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Lock file not found, compiling workflow..."))
		}
	} else {
		runPushLog.Printf("Error checking lock file: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling workflow..."))
		}
	}
	runPushLog.Printf("Always recompiling workflow: %s", absWorkflowPath)
}

func collectWorkflowFilesAddLock(files map[string]struct{}, lockFilePath string, verbose bool) {
	if fileutil.FileExists(lockFilePath) {
		files[lockFilePath] = struct{}{}
		runPushLog.Printf("Added lock file: %s", lockFilePath)
	} else if verbose {
		runPushLog.Printf("Lock file not found after compilation: %s", lockFilePath)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Lock file not found after compilation: "+lockFilePath))
	}
}

// recompileWorkflow compiles a workflow using CompileWorkflows
func recompileWorkflow(ctx context.Context, workflowPath string, verbose bool, approve bool) error {
	runPushLog.Printf("Recompiling workflow: %s", workflowPath)

	config := CompileConfig{
		MarkdownFiles:        []string{workflowPath},
		Verbose:              verbose,
		EngineOverride:       "",
		Validate:             true,
		Watch:                false,
		WorkflowDir:          "",
		SkipInstructions:     false,
		NoEmit:               false,
		Purge:                false,
		TrialMode:            false,
		TrialLogicalRepoSlug: "",
		Strict:               false,
		Approve:              approve,
	}

	runPushLog.Printf("Compilation config: Validate=%v, NoEmit=%v", config.Validate, config.NoEmit)

	runPushLog.Printf("Starting compilation with CompileWorkflows")
	if _, err := CompileWorkflows(ctx, config); err != nil {
		runPushLog.Printf("Compilation failed: %v", err)
		return fmt.Errorf("compilation failed: %w", err)
	}

	runPushLog.Printf("Successfully recompiled workflow: %s", workflowPath)
	return nil
}

// checkLockFileStatus checks if a lock file is missing or outdated and returns status info.
// Note: This is used for display/warning purposes only. The actual compilation decision
// is made in collectWorkflowFiles, which always recompiles regardless of hash status.
type LockFileStatus struct {
	Missing  bool
	Outdated bool
	LockPath string
}

// checkLockFileStatus checks the status of a workflow's lock file.
// This function is used to determine whether to show warnings to the user about
// outdated lock files. It does NOT control whether recompilation happens -
// collectWorkflowFiles always recompiles regardless of the hash status.
func checkLockFileStatus(workflowPath string) (*LockFileStatus, error) {
	runPushLog.Printf("Checking lock file status for: %s", workflowPath)

	// Get absolute path for the workflow
	absWorkflowPath, err := filepath.Abs(workflowPath)
	if err != nil {
		runPushLog.Printf("Failed to get absolute path for %s: %v", workflowPath, err)
		return nil, fmt.Errorf("failed to get absolute path for workflow: %w", err)
	}
	runPushLog.Printf("Resolved absolute path: %s", absWorkflowPath)

	// Validate the absolute path
	absWorkflowPath, err = fileutil.ValidateAbsolutePath(absWorkflowPath)
	if err != nil {
		runPushLog.Printf("Invalid workflow path: %v", err)
		return nil, fmt.Errorf("invalid workflow path: %w", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(absWorkflowPath)
	runPushLog.Printf("Expected lock file path: %s", lockFilePath)
	status := &LockFileStatus{
		LockPath: lockFilePath,
	}

	// Check if lock file exists
	if _, err := os.Stat(lockFilePath); err != nil {
		if os.IsNotExist(err) {
			status.Missing = true
			runPushLog.Printf("Lock file missing: %s", lockFilePath)
			return status, nil
		}
		runPushLog.Printf("Error stating lock file: %v", err)
		return nil, fmt.Errorf("failed to stat lock file: %w", err)
	}
	runPushLog.Printf("Lock file exists: %s", lockFilePath)

	// Lock file exists - check frontmatter hash
	hashMismatch, err := checkFrontmatterHashMismatch(absWorkflowPath, lockFilePath)
	if err != nil {
		runPushLog.Printf("Error checking frontmatter hash: %v", err)
		// Treat hash check error as outdated to be safe
		status.Outdated = true
		runPushLog.Printf("Lock file considered outdated due to hash check error")
	} else if hashMismatch {
		status.Outdated = true
		runPushLog.Printf("Lock file outdated (frontmatter hash mismatch)")
	} else {
		runPushLog.Printf("Lock file is up-to-date (frontmatter hash matches)")
	}

	return status, nil
}

// collectImports recursively collects all imported files (transitive closure)
func collectImports(workflowPath string, files map[string]struct{}, visited map[string]struct{}, verbose bool) error {
	if setutil.Contains(visited, workflowPath) {
		runPushLog.Printf("Skipping already visited file: %s", workflowPath)
		return nil
	}
	visited[workflowPath] = struct{}{}
	runPushLog.Printf("Processing imports for: %s", workflowPath)

	content, err := os.ReadFile(workflowPath)
	if err != nil {
		runPushLog.Printf("Failed to read workflow file %s: %v", workflowPath, err)
		return fmt.Errorf("failed to read workflow file %s: %w", workflowPath, err)
	}
	runPushLog.Printf("Read %d bytes from %s", len(content), workflowPath)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		runPushLog.Printf("No frontmatter in %s, skipping imports extraction: %v", workflowPath, err)
		return nil
	}
	runPushLog.Printf("Extracted frontmatter from %s", workflowPath)
	importsField, exists := result.Frontmatter["imports"]
	if !exists {
		runPushLog.Printf("No imports field in %s", workflowPath)
		return nil
	}

	workflowDir := filepath.Dir(workflowPath)
	imports := collectImportsParseImports(importsField, workflowPath, workflowDir)
	if err := collectImportsProcess(imports, workflowDir, files, visited, verbose); err != nil {
		return err
	}
	runPushLog.Printf("Finished processing imports for: %s", workflowPath)
	return nil
}

func collectImportsParseImports(importsField any, workflowPath, workflowDir string) []string {
	runPushLog.Printf("Found imports field in %s", workflowPath)
	runPushLog.Printf("Workflow directory: %s", workflowDir)
	switch v := importsField.(type) {
	case []any:
		runPushLog.Printf("Parsing imports as []any with %d items", len(v))
		return collectImportsExtractPathsFromArray(v)
	case []string:
		runPushLog.Printf("Parsing imports as []string with %d items", len(v))
		return v
	case map[string]any:
		return collectImportsParseMap(v)
	default:
		runPushLog.Printf("Imports field has unexpected type: %T", v)
		return nil
	}
}

func collectImportsParseMap(v map[string]any) []string {
	if awAny, hasAW := v["aw"]; hasAW {
		switch aw := awAny.(type) {
		case []any:
			runPushLog.Printf("Parsing imports.aw as []any with %d items", len(aw))
			return collectImportsExtractPathsFromArray(aw)
		case []string:
			runPushLog.Printf("Parsing imports.aw as []string with %d items", len(aw))
			return aw
		}
	}
	return nil
}

func collectImportsExtractPathsFromArray(items []any) []string {
	var paths []string
	for i, item := range items {
		switch importItem := item.(type) {
		case string:
			runPushLog.Printf("Import %d: string format: %s", i, importItem)
			paths = append(paths, importItem)
		case map[string]any:
			paths = collectImportsAppendObjectPath(paths, i, importItem)
		default:
			runPushLog.Printf("Import %d: unknown type: %T", i, importItem)
		}
	}
	return paths
}

func collectImportsAppendObjectPath(paths []string, i int, importItem map[string]any) []string {
	if pathValue, hasPath := importItem["path"]; hasPath {
		if pathStr, ok := pathValue.(string); ok {
			runPushLog.Printf("Import %d: object format with path: %s", i, pathStr)
			return append(paths, pathStr)
		}
		runPushLog.Printf("Import %d: object has path but not string type", i)
	} else {
		runPushLog.Printf("Import %d: object missing path field", i)
	}
	return paths
}

func collectImportsProcess(imports []string, workflowDir string, files map[string]struct{}, visited map[string]struct{}, verbose bool) error {
	runPushLog.Printf("Found %d imports", len(imports))
	for i, importPath := range imports {
		if err := collectImportsProcessOne(importPath, i, len(imports), workflowDir, files, visited, verbose); err != nil {
			return err
		}
	}
	return nil
}

func collectImportsProcessOne(importPath string, i, total int, workflowDir string, files map[string]struct{}, visited map[string]struct{}, verbose bool) error {
	runPushLog.Printf("Processing import %d/%d: %s", i+1, total, importPath)
	resolvedPath := resolveImportPath(importPath, workflowDir, importPathRunPushOpts)
	if resolvedPath == "" {
		runPushLog.Printf("Could not resolve import path: %s", importPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not resolve import: "+importPath))
		}
		return nil
	}
	absImportPath := collectImportsAbsolutePath(resolvedPath, workflowDir)
	if _, err := os.Stat(absImportPath); err != nil {
		runPushLog.Printf("Import file not found: %s (error: %v)", absImportPath, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Import file not found: "+absImportPath))
		}
		return nil
	}
	files[absImportPath] = struct{}{}
	runPushLog.Printf("Added import file: %s", absImportPath)
	return collectImports(absImportPath, files, visited, verbose)
}

func collectImportsAbsolutePath(resolvedPath, workflowDir string) string {
	if filepath.IsAbs(resolvedPath) {
		runPushLog.Printf("Import path is absolute: %s", resolvedPath)
		return resolvedPath
	}
	absImportPath := filepath.Join(workflowDir, resolvedPath)
	runPushLog.Printf("Joined relative path: %s + %s = %s", workflowDir, resolvedPath, absImportPath)
	return absImportPath
}

// pushWorkflowFiles commits and pushes the workflow files to the repository
func pushWorkflowFiles(ctx context.Context, workflowName string, files []string, refOverride string, verbose bool) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	runPushLog.Printf("Pushing %d files for workflow: %s", len(files), workflowName)
	runPushLog.Printf("Files to push: %v", files)
	pushWorkflowFilesPrintStaging(files, verbose)

	statusOutput, err := pushWorkflowFilesStageAndStatus(ctx, files, verbose)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(statusOutput)) == "" {
		runPushLog.Printf("No staged changes detected")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to commit"))
		}
		runPushLog.Print("No changes to commit")
		return nil
	}
	if err := pushWorkflowFilesVerifyRef(refOverride, verbose); err != nil {
		return err
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
	runPushLog.Printf("Found %d staged files: %v", len(stagedFiles), stagedFiles)
	if err := pushWorkflowFilesCheckExtra(files, stagedFiles); err != nil {
		return err
	}
	commitMessage := "Updated agentic workflow " + workflowName
	if err := pushWorkflowFilesConfirm(files, commitMessage); err != nil {
		return err
	}
	return pushWorkflowFilesCommitAndPush(ctx, commitMessage, verbose)
}

func pushWorkflowFilesPrintStaging(files []string, verbose bool) {
	if !verbose {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Staging %d files for commit", len(files))))
	for _, file := range files {
		fmt.Fprintf(os.Stderr, "  - %s\n", file)
	}
}

func pushWorkflowFilesStageAndStatus(ctx context.Context, files []string, verbose bool) ([]byte, error) {
	gitArgs := append([]string{"add"}, files...)
	runPushLog.Printf("Executing git command: git %v", gitArgs)
	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		runPushLog.Printf("Failed to stage files: %v, output: %s", err, string(output))
		return nil, fmt.Errorf("failed to stage files: %w\nOutput: %s", err, string(output))
	}
	runPushLog.Printf("Successfully staged %d files", len(files))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Files staged successfully"))
	}

	runPushLog.Printf("Checking staged files with git diff --cached --name-only")
	statusCmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		runPushLog.Printf("Failed to check git status: %v, output: %s", err, string(statusOutput))
		return nil, fmt.Errorf("failed to check git status: %w\nOutput: %s", err, string(statusOutput))
	}
	runPushLog.Printf("Git status output: %s", string(statusOutput))
	return statusOutput, nil
}

func pushWorkflowFilesVerifyRef(refOverride string, verbose bool) error {
	if refOverride == "" {
		return nil
	}
	runPushLog.Printf("Checking if current branch matches --ref value: %s", refOverride)
	currentBranch, err := getCurrentBranch()
	if err != nil {
		runPushLog.Printf("Failed to determine current branch: %v", err)
		return fmt.Errorf("failed to determine current branch: %w", err)
	}
	if currentBranch != refOverride {
		runPushLog.Printf("Current branch (%s) does not match --ref value (%s)", currentBranch, refOverride)
		return fmt.Errorf("--push requires the current branch (%s) to match the --ref value (%s). Switching branches is not supported. Please checkout the target branch first", currentBranch, refOverride)
	}
	runPushLog.Printf("Current branch matches --ref value: %s", currentBranch)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Verified current branch matches --ref: "+currentBranch))
	}
	return nil
}

func pushWorkflowFilesCheckExtra(files, stagedFiles []string) error {
	ourFiles := pushWorkflowFilesMap(files)
	var extraStagedFiles []string
	for _, stagedFile := range stagedFiles {
		if pushWorkflowFilesContainsStaged(ourFiles, stagedFile) {
			continue
		}
		runPushLog.Printf("Extra staged file detected: %s", stagedFile)
		extraStagedFiles = append(extraStagedFiles, stagedFile)
	}
	if len(extraStagedFiles) == 0 {
		runPushLog.Printf("No extra staged files detected - all staged files are part of our workflow")
		return nil
	}
	pushWorkflowFilesPrintExtra(extraStagedFiles)
	return errors.New("git has staged files not part of workflow - commit or unstage them before using --push")
}

func pushWorkflowFilesMap(files []string) map[string]struct{} {
	runPushLog.Printf("Building map of our files for comparison")
	ourFiles := make(map[string]struct{})
	for _, file := range files {
		if absPath, err := filepath.Abs(file); err == nil {
			if validPath, validErr := fileutil.ValidateAbsolutePath(absPath); validErr == nil {
				ourFiles[validPath] = struct{}{}
				runPushLog.Printf("Added to our files map: %s (absolute: %s)", file, validPath)
			} else {
				runPushLog.Printf("Failed to validate path for %s: %v", absPath, validErr)
			}
		} else {
			runPushLog.Printf("Failed to get absolute path for %s: %v", file, err)
		}
		ourFiles[file] = struct{}{}
		runPushLog.Printf("Added to our files map: %s", file)
	}
	return ourFiles
}

func pushWorkflowFilesContainsStaged(ourFiles map[string]struct{}, stagedFile string) bool {
	runPushLog.Printf("Checking staged file: %s", stagedFile)
	if absStagedPath, err := filepath.Abs(stagedFile); err == nil {
		if validPath, validErr := fileutil.ValidateAbsolutePath(absStagedPath); validErr == nil && setutil.Contains(ourFiles, validPath) {
			runPushLog.Printf("Staged file %s matches our file %s (absolute)", stagedFile, validPath)
			return true
		}
	}
	if setutil.Contains(ourFiles, stagedFile) {
		runPushLog.Printf("Staged file %s matches our file (relative)", stagedFile)
		return true
	}
	return false
}

func pushWorkflowFilesPrintExtra(extraStagedFiles []string) {
	runPushLog.Printf("Found %d extra staged files not in our list, refusing to proceed: %v", len(extraStagedFiles), extraStagedFiles)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Cannot proceed: there are already staged files in git that are not part of this workflow"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Extra staged files:"))
	for _, file := range extraStagedFiles {
		fmt.Fprintf(os.Stderr, "  - %s\n", file)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please commit or unstage these files before using --push"))
	fmt.Fprintln(os.Stderr, "")
}

func pushWorkflowFilesConfirm(files []string, commitMessage string) error {
	runPushLog.Printf("Creating commit with message: %s", commitMessage)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Ready to commit and push the following files:"))
	for _, file := range files {
		fmt.Fprintf(os.Stderr, "  - %s\n", file)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Commit message: %s\n"), commitMessage)
	fmt.Fprintln(os.Stderr, "")
	confirmed, err := console.ConfirmAction("Do you want to commit and push these changes?", "Yes, commit and push", "No, cancel")
	if err != nil {
		runPushLog.Printf("Confirmation failed: %v", err)
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		runPushLog.Print("Push cancelled by user")
		return errors.New("push cancelled by user")
	}
	runPushLog.Printf("User confirmed - proceeding with commit and push")
	return nil
}

func pushWorkflowFilesCommitAndPush(ctx context.Context, commitMessage string, verbose bool) error {
	runPushLog.Printf("Executing git commit with message: %s", commitMessage)
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", commitMessage)
	if output, err := cmd.CombinedOutput(); err != nil {
		runPushLog.Printf("Failed to commit: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to commit changes: %w\nOutput: %s", err, string(output))
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Changes committed successfully"))
	}
	runPushLog.Print("Pushing changes to remote")
	cmd = exec.CommandContext(ctx, "git", "push")
	if output, err := cmd.CombinedOutput(); err != nil {
		runPushLog.Printf("Failed to push: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to push changes: %w\nOutput: %s", err, string(output))
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Changes pushed to remote"))
	}
	runPushLog.Print("Push completed successfully")
	return nil
}

// checkFrontmatterHashMismatch checks if the frontmatter hash in the lock file
// matches the recomputed hash from the workflow file.
// Returns true if there's a mismatch (lock file is stale), false if they match.
// Note: This is used for logging/observability only. The compilation decision
// is made by collectWorkflowFiles, which always recompiles regardless of hash status.
func checkFrontmatterHashMismatch(workflowPath, lockFilePath string) (bool, error) {
	runPushLog.Printf("Checking frontmatter hash for %s", workflowPath)

	// Read lock file to extract existing hash
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read lock file: %w", err)
	}

	// Extract hash from lock file using the workflow package function
	metadata, _, err := workflow.ExtractMetadataFromLockFile(string(lockContent))
	var existingHash string
	if err == nil && metadata != nil {
		existingHash = metadata.FrontmatterHash
	}
	if existingHash == "" {
		runPushLog.Print("No frontmatter-hash found in lock file")
		// No hash in lock file - consider it stale to regenerate with hash
		return true, nil
	}
	runPushLog.Printf("Existing hash from lock file: %s", existingHash)

	// Compute current hash from workflow file
	cache := parser.NewImportCache("")
	currentHash, err := parser.ComputeFrontmatterHashFromFile(workflowPath, cache)
	if err != nil {
		return false, fmt.Errorf("failed to compute frontmatter hash: %w", err)
	}
	runPushLog.Printf("Current hash from workflow: %s", currentHash)

	// Compare hashes
	mismatch := existingHash != currentHash
	if mismatch {
		runPushLog.Printf("Hash mismatch: existing=%s, current=%s", existingHash, currentHash)
	} else {
		runPushLog.Print("Hashes match")
	}

	return mismatch, nil
}
