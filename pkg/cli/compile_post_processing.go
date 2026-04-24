// This file provides post-processing operations for workflow compilation.
//
// This file contains functions that perform post-compilation operations such as
// generating Dependabot manifests and maintenance workflows.
//
// # Organization Rationale
//
// These post-processing functions are grouped here because they:
//   - Run after workflow compilation completes
//   - Generate auxiliary files and manifests
//   - Have a clear domain focus (post-compilation processing)
//   - Keep the main orchestrator focused on coordination
//
// # Key Functions
//
// Generation:
//   - generateDependabotManifestsWrapper() - Generate Dependabot manifests
//   - generateMaintenanceWorkflowWrapper() - Generate maintenance workflow
//
// These functions abstract post-processing operations, allowing the main compile
// orchestrator to focus on coordination while these handle generation and validation.

package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compilePostProcessingLog = logger.New("cli:compile_post_processing")

// generateDependabotManifestsWrapper generates Dependabot manifests for compiled workflows
func generateDependabotManifestsWrapper(
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	workflowsDir string,
	forceOverwrite bool,
	strict bool,
) error {
	compilePostProcessingLog.Print("Generating Dependabot manifests for compiled workflows")

	if err := compiler.GenerateDependabotManifests(workflowDataList, workflowsDir, forceOverwrite); err != nil {
		if strict {
			return fmt.Errorf("failed to generate Dependabot manifests: %w", err)
		}
		// Non-strict mode: just report as warning
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate Dependabot manifests: %v", err)))
	}

	return nil
}

// generateMaintenanceWorkflowWrapper generates maintenance workflow if any workflow uses expires field
func generateMaintenanceWorkflowWrapper(
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	workflowsDir string,
	gitRoot string,
	verbose bool,
	strict bool,
) error {
	compilePostProcessingLog.Print("Generating maintenance workflow")

	// Load repo-level configuration (optional file).
	repoConfig, err := workflow.LoadRepoConfig(gitRoot)
	if err != nil {
		if strict {
			return fmt.Errorf("failed to load repo config: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to load repo config: %v", err)))
		repoConfig = nil
	}

	if err := workflow.GenerateMaintenanceWorkflow(workflowDataList, workflowsDir, compiler.GetVersion(), compiler.GetActionMode(), compiler.GetActionTag(), verbose, repoConfig, compiler.GetRepositorySlug()); err != nil {
		if strict {
			return fmt.Errorf("failed to generate maintenance workflow: %w", err)
		}
		// Non-strict mode: just report as warning
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate maintenance workflow: %v", err)))
	}

	return nil
}
