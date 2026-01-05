package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var runPushLog = logger.New("cli:run_push")

// collectWorkflowFiles collects the workflow .md file, its corresponding .lock.yml file,
// and the transitive closure of all imported files
func collectWorkflowFiles(workflowPath string, verbose bool) ([]string, error) {
	runPushLog.Printf("Collecting files for workflow: %s", workflowPath)

	files := make(map[string]bool) // Use map to avoid duplicates
	visited := make(map[string]bool)

	// Get absolute path for the workflow
	absWorkflowPath, err := filepath.Abs(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for workflow: %w", err)
	}

	// Add the workflow .md file
	files[absWorkflowPath] = true
	runPushLog.Printf("Added workflow file: %s", absWorkflowPath)

	// Check if lock file needs recompilation
	lockFilePath := strings.TrimSuffix(absWorkflowPath, ".md") + ".lock.yml"
	needsRecompile := false
	
	if lockStat, err := os.Stat(lockFilePath); err == nil {
		// Lock file exists - check if it's outdated
		if mdStat, err := os.Stat(absWorkflowPath); err == nil {
			if mdStat.ModTime().After(lockStat.ModTime()) {
				needsRecompile = true
				runPushLog.Printf("Lock file is outdated (md: %v, lock: %v)", mdStat.ModTime(), lockStat.ModTime())
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Detected outdated lock file, recompiling workflow..."))
				}
			}
		}
	} else if os.IsNotExist(err) {
		// Lock file doesn't exist - needs compilation
		needsRecompile = true
		runPushLog.Printf("Lock file not found: %s", lockFilePath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Lock file not found, compiling workflow..."))
		}
	}

	// Recompile if needed
	if needsRecompile {
		if err := recompileWorkflow(absWorkflowPath, verbose); err != nil {
			return nil, fmt.Errorf("failed to recompile workflow: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow compiled successfully"))
		}
	}

	// Add the corresponding .lock.yml file
	if _, err := os.Stat(lockFilePath); err == nil {
		files[lockFilePath] = true
		runPushLog.Printf("Added lock file: %s", lockFilePath)
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Lock file not found after compilation: %s", lockFilePath)))
	}

	// Collect transitive closure of imported files
	if err := collectImports(absWorkflowPath, files, visited, verbose); err != nil {
		return nil, fmt.Errorf("failed to collect imports: %w", err)
	}

	// Convert map to slice
	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}

	runPushLog.Printf("Collected %d files total", len(result))
	return result, nil
}

// recompileWorkflow compiles a workflow using CompileWorkflows
func recompileWorkflow(workflowPath string, verbose bool) error {
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
	}
	
	// Use background context for compilation
	ctx := context.Background()
	if _, err := CompileWorkflows(ctx, config); err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}
	
	runPushLog.Printf("Successfully recompiled workflow: %s", workflowPath)
	return nil
}

// collectImports recursively collects all imported files (transitive closure)
func collectImports(workflowPath string, files map[string]bool, visited map[string]bool, verbose bool) error {
	// Avoid processing the same file multiple times
	if visited[workflowPath] {
		return nil
	}
	visited[workflowPath] = true

	runPushLog.Printf("Processing imports for: %s", workflowPath)

	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file %s: %w", workflowPath, err)
	}

	// Extract frontmatter to get imports field
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		// No frontmatter is okay - might be a simple file
		runPushLog.Printf("No frontmatter in %s, skipping imports extraction", workflowPath)
		return nil
	}

	// Get imports from frontmatter
	importsField, exists := result.Frontmatter["imports"]
	if !exists {
		runPushLog.Printf("No imports field in %s", workflowPath)
		return nil
	}

	// Parse imports field - can be array of strings or objects with path
	workflowDir := filepath.Dir(workflowPath)
	var imports []string

	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
			switch importItem := item.(type) {
			case string:
				// Simple string import
				imports = append(imports, importItem)
			case map[string]any:
				// Object import with path field
				if pathValue, hasPath := importItem["path"]; hasPath {
					if pathStr, ok := pathValue.(string); ok {
						imports = append(imports, pathStr)
					}
				}
			}
		}
	case []string:
		imports = v
	}

	runPushLog.Printf("Found %d imports in %s", len(imports), workflowPath)

	// Process each import
	for _, importPath := range imports {
		// Resolve the import path
		resolvedPath := resolveImportPathLocal(importPath, workflowDir)
		if resolvedPath == "" {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not resolve import: %s", importPath)))
			}
			continue
		}

		// Get absolute path
		var absImportPath string
		if filepath.IsAbs(resolvedPath) {
			absImportPath = resolvedPath
		} else {
			absImportPath = filepath.Join(workflowDir, resolvedPath)
		}

		// Check if file exists
		if _, err := os.Stat(absImportPath); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Import file not found: %s", absImportPath)))
			}
			continue
		}

		// Add the import file
		files[absImportPath] = true
		runPushLog.Printf("Added import file: %s", absImportPath)

		// Recursively collect imports from this file
		if err := collectImports(absImportPath, files, visited, verbose); err != nil {
			return err
		}
	}

	return nil
}

// resolveImportPathLocal is a local version of resolveImportPath for push functionality
// This is needed to avoid circular dependencies with imports.go
func resolveImportPathLocal(importPath, baseDir string) string {
	// Handle section references (file.md#Section) - strip the section part
	if strings.Contains(importPath, "#") {
		parts := strings.SplitN(importPath, "#", 2)
		importPath = parts[0]
	}

	// Skip workflowspec format imports (owner/repo/path@sha)
	if strings.Contains(importPath, "@") || isWorkflowSpecFormatLocal(importPath) {
		runPushLog.Printf("Skipping workflowspec format import: %s", importPath)
		return ""
	}

	// If the import path is absolute (starts with /), use it relative to repo root
	if strings.HasPrefix(importPath, "/") {
		// Find git root
		gitRoot, err := findGitRoot()
		if err != nil {
			return ""
		}
		return filepath.Join(gitRoot, strings.TrimPrefix(importPath, "/"))
	}

	// Otherwise, resolve relative to the workflow file's directory
	return filepath.Join(baseDir, importPath)
}

// isWorkflowSpecFormatLocal is a local version of isWorkflowSpecFormat for push functionality
// This is duplicated from imports.go to avoid circular dependencies
func isWorkflowSpecFormatLocal(path string) bool {
	// Check if it contains @ (ref separator) or looks like owner/repo/path
	if strings.Contains(path, "@") {
		return true
	}

	// Remove section reference if present
	cleanPath := path
	if idx := strings.Index(path, "#"); idx != -1 {
		cleanPath = path[:idx]
	}

	// Check if it has at least 3 parts and doesn't start with . or /
	parts := strings.Split(cleanPath, "/")
	if len(parts) >= 3 && !strings.HasPrefix(cleanPath, ".") && !strings.HasPrefix(cleanPath, "/") {
		return true
	}

	return false
}

// pushWorkflowFiles commits and pushes the workflow files to the repository
func pushWorkflowFiles(workflowName string, files []string, verbose bool) error {
	runPushLog.Printf("Pushing %d files for workflow: %s", len(files), workflowName)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Staging %d files for commit", len(files))))
		for _, file := range files {
			fmt.Fprintf(os.Stderr, "  - %s\n", file)
		}
	}

	// Stage all files
	gitArgs := append([]string{"add"}, files...)
	cmd := exec.Command("git", gitArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		runPushLog.Printf("Failed to stage files: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to stage files: %w\nOutput: %s", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Files staged successfully"))
	}

	// Create commit message
	commitMessage := fmt.Sprintf("Updated agentic workflow %s", workflowName)
	runPushLog.Printf("Creating commit with message: %s", commitMessage)

	// Show what will be committed and ask for confirmation
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Ready to commit and push the following files:"))
	for _, file := range files {
		fmt.Fprintf(os.Stderr, "  - %s\n", file)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Commit message: %s\n"), commitMessage)
	fmt.Fprintln(os.Stderr, "")

	// Ask for confirmation
	fmt.Fprint(os.Stderr, console.FormatPromptMessage("Do you want to commit and push these changes? [y/N]: "))

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		response = "n" // Default to no on error
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		runPushLog.Print("Push cancelled by user")
		return fmt.Errorf("push cancelled by user")
	}

	// Commit the changes
	cmd = exec.Command("git", "commit", "-m", commitMessage)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Check if there are no changes to commit
		if strings.Contains(string(output), "nothing to commit") {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to commit"))
			}
			runPushLog.Print("No changes to commit")
			return nil
		}
		runPushLog.Printf("Failed to commit: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to commit changes: %w\nOutput: %s", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Changes committed successfully"))
	}

	// Push the changes
	runPushLog.Print("Pushing changes to remote")
	cmd = exec.Command("git", "push")
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
