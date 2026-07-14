package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

const evalsBranchPrefix = "evals"

func ensureEvalsResultsFromBranch(ctx context.Context, run WorkflowRun, runDir, owner, repo, hostname string, verbose bool) bool {
	if runHasEvals(runDir, verbose) {
		return true
	}

	workflowID := workflowIDFromRunPath(run.WorkflowPath)
	if workflowID == "" {
		return false
	}
	branchName := evalsStateBranchName(workflowID)
	repoOverride, host := resolveRepoOverrideForRun(run, owner, repo, hostname)
	if repoOverride == "" {
		return false
	}

	decoded, err := readRemoteRepoBranchFileContext(ctx, repoOverride, branchName, constants.EvalsResultFilename, host)
	if err != nil {
		if !isRemoteFileNotFound(err) {
			logsOrchestratorLog.Printf("Failed to fetch evals branch file for run %d: branch=%s repo=%s err=%v", run.DatabaseID, branchName, repoOverride, err)
		}
		return false
	}

	if mkdirErr := os.MkdirAll(runDir, constants.DirPermPublic); mkdirErr != nil {
		logsOrchestratorLog.Printf("Failed to create run directory for evals branch file: run=%d dir=%s err=%v", run.DatabaseID, runDir, mkdirErr)
		return false
	}

	dest := filepath.Join(runDir, constants.EvalsResultFilename)
	if writeErr := os.WriteFile(dest, decoded, constants.FilePermPublic); writeErr != nil {
		logsOrchestratorLog.Printf("Failed to write evals branch file for run %d: %v", run.DatabaseID, writeErr)
		return false
	}
	logsOrchestratorLog.Printf("Loaded evals results from branch %s into %s for run %d", branchName, dest, run.DatabaseID)
	return true
}

func workflowIDFromRunPath(workflowPath string) string {
	if workflowPath == "" {
		return ""
	}
	base := filepath.Base(workflowPath)
	base = stringutil.NormalizeWorkflowName(base)
	if before, ok := strings.CutSuffix(base, ".yml"); ok {
		base = before
	}
	if before, ok := strings.CutSuffix(base, ".yaml"); ok {
		base = before
	}
	return strings.TrimSpace(base)
}

func evalsStateBranchName(workflowID string) string {
	sanitized := workflow.SanitizeWorkflowIDForCacheKey(workflowID)
	if sanitized == "" {
		sanitized = "default"
	}
	return evalsBranchPrefix + "/" + sanitized
}

func resolveRepoOverrideForRun(run WorkflowRun, owner, repo, hostname string) (string, string) {
	runOwner, runRepo, runHost := owner, repo, hostname
	if runOwner == "" && run.URL != "" {
		if c, err := parser.ParseRunURLExtended(run.URL); err == nil && c.Owner != "" {
			runOwner, runRepo, runHost = c.Owner, c.Repo, c.Host
		}
	}
	if runOwner == "" || runRepo == "" {
		return "", runHost
	}
	if runHost == "" {
		runHost = "github.com"
	}
	return fmt.Sprintf("%s/%s", runOwner, runRepo), runHost
}
