package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

var compileOrchestratorLog = logger.New("cli:compile_orchestrator")

// getRepositoryRelativePath converts an absolute file path to a repository-relative path
// This ensures stable workflow identifiers regardless of where the repository is cloned
func getRepositoryRelativePath(absPath string) (string, error) {
	// Get the repository root for the specific file
	repoRoot, err := findGitRootForPath(absPath)
	if err != nil {
		// If we can't get the repo root, just use the basename as fallback
		compileOrchestratorLog.Printf("Warning: could not get repository root for %s: %v, using basename", absPath, err)
		return filepath.Base(absPath), nil
	}

	// Convert both paths to absolute to ensure they can be compared
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the relative path from repo root
	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Normalize path separators to forward slashes for consistency across platforms
	// This ensures the same hash value on Windows, Linux, and macOS
	relPath = filepath.ToSlash(relPath)

	return relPath, nil
}

func renderGeneratedCampaignOrchestratorMarkdown(data *workflow.WorkflowData, sourceCampaignPath string) string {
	// Produce a conventional gh-aw workflow markdown file so users can review
	// the generated orchestrator and recompile it like any other workflow.
	//
	// NOTE: The generated .campaign.g.md file is a debug artifact that is NOT
	// committed to git (it's in .gitignore). Users can review it locally to
	// understand the generated workflow structure. Only the source .campaign.md
	// and the compiled .campaign.lock.yml files are committed.
	b := &strings.Builder{}
	b.WriteString("---\n")
	if strings.TrimSpace(data.Name) != "" {
		fmt.Fprintf(b, "name: %q\n", data.Name)
	}
	if strings.TrimSpace(data.Description) != "" {
		fmt.Fprintf(b, "description: %q\n", data.Description)
	}
	if strings.TrimSpace(data.On) != "" {
		b.WriteString(strings.TrimSuffix(data.On, "\n"))
		b.WriteString("\n")
	}
	if strings.TrimSpace(data.Concurrency) != "" {
		b.WriteString(strings.TrimSuffix(data.Concurrency, "\n"))
		b.WriteString("\n")
	}

	// Make the orchestrator runnable by default.
	// Use engine from EngineConfig if available, otherwise default to claude.
	engineID := "claude"
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	}
	fmt.Fprintf(b, "engine: %s\n", engineID)

	// Render safe-outputs if configured by the campaign orchestrator generator.
	// This enables campaign orchestrators to update their Projects dashboard and
	// post tracker comments without requiring manual edits to generated files.
	if data.SafeOutputs != nil {
		// NOTE: We must emit the public frontmatter keys (e.g. "add-comment") rather
		// than the internal struct YAML tags (e.g. "add-comments").
		outputs := map[string]any{}
		if data.SafeOutputs.CreateIssues != nil {
			outputs["create-issue"] = map[string]any{
				"max": data.SafeOutputs.CreateIssues.Max,
			}
		}
		if data.SafeOutputs.AddComments != nil {
			outputs["add-comment"] = map[string]any{
				"max": data.SafeOutputs.AddComments.Max,
			}
		}
		if data.SafeOutputs.UpdateProjects != nil {
			updateProjectConfig := map[string]any{
				"max": data.SafeOutputs.UpdateProjects.Max,
			}
			// Include github-token if specified
			if strings.TrimSpace(data.SafeOutputs.UpdateProjects.GitHubToken) != "" {
				updateProjectConfig["github-token"] = data.SafeOutputs.UpdateProjects.GitHubToken
			}
			outputs["update-project"] = updateProjectConfig
		}
		if data.SafeOutputs.CreateProjectStatusUpdates != nil {
			statusUpdateConfig := map[string]any{
				"max": data.SafeOutputs.CreateProjectStatusUpdates.Max,
			}
			// Include github-token if specified
			if strings.TrimSpace(data.SafeOutputs.CreateProjectStatusUpdates.GitHubToken) != "" {
				statusUpdateConfig["github-token"] = data.SafeOutputs.CreateProjectStatusUpdates.GitHubToken
			}
			outputs["create-project-status-update"] = statusUpdateConfig
		}
		if len(outputs) > 0 {
			payload := map[string]any{"safe-outputs": outputs}
			if out, err := yaml.Marshal(payload); err == nil {
				b.WriteString(string(out))
			} else {
				compileOrchestratorLog.Printf("Failed to render safe-outputs for generated campaign orchestrator: %v", err)
			}
		}
	}

	// Intentionally omit permissions from generated campaign orchestrator frontmatter.
	// Workflow/job permissions are handled during compilation.
	if strings.TrimSpace(data.RunsOn) != "" {
		b.WriteString(strings.TrimSuffix(data.RunsOn, "\n"))
		b.WriteString("\n")
	}
	if len(data.Roles) > 0 {
		b.WriteString("roles:\n")
		for _, role := range data.Roles {
			if strings.TrimSpace(role) == "" {
				continue
			}
			fmt.Fprintf(b, "  - %q\n", role)
		}
	}
	// Render tools configuration if present
	if len(data.Tools) > 0 {
		payload := map[string]any{"tools": data.Tools}
		if out, err := yaml.Marshal(payload); err == nil {
			b.WriteString(string(out))
		} else {
			compileOrchestratorLog.Printf("Failed to render tools for generated campaign orchestrator: %v", err)
		}
	}
	// Render custom steps if present (e.g., discovery precomputation)
	if strings.TrimSpace(data.CustomSteps) != "" {
		// CustomSteps is already YAML-formatted, just write it as is
		b.WriteString("steps:\n")
		b.WriteString(data.CustomSteps)
	}
	b.WriteString("---\n\n")
	// Include version for released builds only (not "dev", "dirty", or "test")
	version := workflow.GetVersion()
	if workflow.IsReleasedVersion(version) {
		fmt.Fprintf(b, "<!-- This file was automatically generated by gh-aw (%s). DO NOT EDIT. -->\n", version)
	} else {
		b.WriteString("<!-- This file was automatically generated by gh-aw. DO NOT EDIT. -->\n")
	}
	if strings.TrimSpace(sourceCampaignPath) != "" {
		// Normalize path to be relative to git root (where .github folder exists)
		// This ensures stable paths regardless of current working directory
		relativePath := ToGitRootRelativePath(sourceCampaignPath)
		fmt.Fprintf(b, "<!-- Source: %s -->\n", relativePath)
	}
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(data.MarkdownContent))
	b.WriteString("\n")
	return b.String()
}

// GenerateCampaignOrchestratorOptions holds the options for generateAndCompileCampaignOrchestrator
type GenerateCampaignOrchestratorOptions struct {
	Compiler             *workflow.Compiler
	Spec                 *campaign.CampaignSpec
	CampaignSpecPath     string
	Verbose              bool
	NoEmit               bool
	RunZizmorPerFile     bool
	RunPoutinePerFile    bool
	RunActionlintPerFile bool
	Strict               bool
	ValidateActionSHAs   bool
}

func generateAndCompileCampaignOrchestrator(opts GenerateCampaignOrchestratorOptions) (string, error) {
	data, orchestratorPath := campaign.BuildOrchestrator(opts.Spec, opts.CampaignSpecPath)
	if data == nil || orchestratorPath == "" {
		return "", nil
	}

	// Ensure we pick a real engine in the YAML compiler path.
	// Campaign orchestrators should default to claude (matching the orchestrator generator).
	if strings.TrimSpace(data.AI) == "" {
		engineID := "claude"
		if data.EngineConfig != nil && strings.TrimSpace(data.EngineConfig.ID) != "" {
			engineID = strings.TrimSpace(data.EngineConfig.ID)
		}
		data.AI = engineID
	}

	if !opts.NoEmit {
		content := renderGeneratedCampaignOrchestratorMarkdown(data, opts.CampaignSpecPath)
		// Write with restrictive permissions (0600) to follow security best practices
		if err := os.WriteFile(orchestratorPath, []byte(content), 0600); err != nil {
			return "", fmt.Errorf("failed to write generated campaign orchestrator %s: %w", orchestratorPath, err)
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Generated campaign orchestrator %s", filepath.Base(orchestratorPath))))
		}
	}

	// Prefer compiling from the generated markdown so defaults and validation behavior
	// match normal workflows (including computed permissions).
	if !opts.NoEmit {
		if err := CompileWorkflowWithValidation(opts.Compiler, orchestratorPath, opts.Verbose, opts.RunZizmorPerFile, opts.RunPoutinePerFile, opts.RunActionlintPerFile, opts.Strict, opts.ValidateActionSHAs); err != nil {
			return orchestratorPath, err
		}
		return orchestratorPath, nil
	}

	// No-emit mode: compile from the in-memory WorkflowData.
	if err := CompileWorkflowDataWithValidation(opts.Compiler, data, orchestratorPath, opts.Verbose, opts.RunZizmorPerFile, opts.RunPoutinePerFile, opts.RunActionlintPerFile, opts.Strict, opts.ValidateActionSHAs); err != nil {
		return orchestratorPath, err
	}

	return orchestratorPath, nil
}

// CompileWorkflows compiles workflows based on the provided configuration
func CompileWorkflows(ctx context.Context, config CompileConfig) ([]*workflow.WorkflowData, error) {
	compileOrchestratorLog.Printf("Starting workflow compilation: files=%d, validate=%v, watch=%v, noEmit=%v",
		len(config.MarkdownFiles), config.Validate, config.Watch, config.NoEmit)

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return nil, ctx.Err()
	default:
	}

	// Validate configuration
	if err := validateCompileConfig(config); err != nil {
		return nil, err
	}

	// Validate action mode if specified
	if err := validateActionModeConfig(config.ActionMode); err != nil {
		return nil, err
	}

	// Initialize actionlint statistics if actionlint is enabled
	if config.Actionlint && !config.NoEmit {
		initActionlintStats()
	}

	// Track compilation statistics
	stats := &CompilationStats{}

	// Track validation results for JSON output
	var validationResults []ValidationResult

	// Set up workflow directory (using default if not specified)
	workflowDir := config.WorkflowDir
	if workflowDir == "" {
		workflowDir = ".github/workflows"
		compileOrchestratorLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		workflowDir = filepath.Clean(workflowDir)
		compileOrchestratorLog.Printf("Using custom workflow directory: %s", workflowDir)
	}

	// Create and configure compiler
	compiler := createAndConfigureCompiler(config)

	// Handle watch mode (early return)
	if config.Watch {
		// Watch mode: watch for file changes and recompile automatically
		// For watch mode, we only support a single file for now
		var markdownFile string
		if len(config.MarkdownFiles) > 0 {
			if len(config.MarkdownFiles) > 1 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Watch mode only supports a single file, using the first one"))
			}
			// Resolve the workflow file to get the full path
			resolvedFile, err := resolveWorkflowFile(config.MarkdownFiles[0], config.Verbose)
			if err != nil {
				// Return error directly without wrapping - it already contains formatted message with suggestions
				return nil, err
			}
			markdownFile = resolvedFile
		}
		return nil, watchAndCompileWorkflows(markdownFile, compiler, config.Verbose)
	}

	// Compile specific files or all files in directory
	if len(config.MarkdownFiles) > 0 {
		// Compile specific workflow files
		return compileSpecificFiles(compiler, config, stats, &validationResults)
	}

	// Compile all workflow files in directory
	return compileAllFilesInDirectory(compiler, config, workflowDir, stats, &validationResults)
}
