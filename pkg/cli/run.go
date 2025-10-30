package cli

import (
	"encoding/json"
	"os/exec"
	"strconv"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var cancelLog = logger.New("cli:cancel")

func cancelWorkflowRuns(workflowID int64) error {
	cancelLog.Printf("Cancelling workflow runs for workflow ID: %d", workflowID)

	// Start spinner for network operation
	spinner := console.NewSpinner("Cancelling workflow runs...")
	spinner.Start()

	// Get running workflow runs
	cmd := exec.Command("gh", "run", "list", "--workflow", strconv.FormatInt(workflowID, 10), "--status", "in_progress", "--json", "databaseId")
	output, err := cmd.Output()
	if err != nil {
		cancelLog.Printf("Failed to list workflow runs: %v", err)
		spinner.Stop()
		return err
	}

	var runs []struct {
		DatabaseID int64 `json:"databaseId"`
	}
	if err := json.Unmarshal(output, &runs); err != nil {
		cancelLog.Printf("Failed to parse workflow runs JSON: %v", err)
		spinner.Stop()
		return err
	}

	cancelLog.Printf("Found %d in-progress workflow runs to cancel", len(runs))

	// Cancel each running workflow
	for _, run := range runs {
		cancelLog.Printf("Cancelling workflow run: %d", run.DatabaseID)
		cancelCmd := exec.Command("gh", "run", "cancel", strconv.FormatInt(run.DatabaseID, 10))
		_ = cancelCmd.Run() // Ignore errors for individual cancellations
	}

	spinner.Stop()
	cancelLog.Print("Workflow run cancellation completed")
	return nil
}

// cancelWorkflowRunsByLockFile cancels in-progress runs for a workflow identified by its lock file name
func cancelWorkflowRunsByLockFile(lockFileName string) error {
	cancelLog.Printf("Cancelling workflow runs for lock file: %s", lockFileName)

	// Start spinner for network operation
	spinner := console.NewSpinner("Cancelling workflow runs...")
	spinner.Start()

	// Get running workflow runs by lock file name
	cmd := exec.Command("gh", "run", "list", "--workflow", lockFileName, "--status", "in_progress", "--json", "databaseId")
	output, err := cmd.Output()
	if err != nil {
		cancelLog.Printf("Failed to list workflow runs by lock file: %v", err)
		spinner.Stop()
		return err
	}

	var runs []struct {
		DatabaseID int64 `json:"databaseId"`
	}
	if err := json.Unmarshal(output, &runs); err != nil {
		cancelLog.Printf("Failed to parse workflow runs JSON: %v", err)
		spinner.Stop()
		return err
	}

	cancelLog.Printf("Found %d in-progress workflow runs to cancel", len(runs))

	// Cancel each running workflow
	for _, run := range runs {
		cancelLog.Printf("Cancelling workflow run: %d", run.DatabaseID)
		cancelCmd := exec.Command("gh", "run", "cancel", strconv.FormatInt(run.DatabaseID, 10))
		_ = cancelCmd.Run() // Ignore errors for individual cancellations
	}

	spinner.Stop()
	cancelLog.Print("Workflow run cancellation completed")
	return nil
}
