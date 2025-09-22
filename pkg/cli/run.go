package cli

import (
	"encoding/json"
	"os/exec"
	"strconv"

	"github.com/githubnext/gh-aw/pkg/console"
)

func cancelWorkflowRuns(workflowID int64) error {
	// Start spinner for network operation
	spinner := console.NewSpinner("Cancelling workflow runs...")
	spinner.Start()

	// Get running workflow runs
	cmd := exec.Command("gh", "run", "list", "--workflow", strconv.FormatInt(workflowID, 10), "--status", "in_progress", "--json", "databaseId")
	output, err := cmd.Output()
	if err != nil {
		spinner.Stop()
		return err
	}

	var runs []struct {
		DatabaseID int64 `json:"databaseId"`
	}
	if err := json.Unmarshal(output, &runs); err != nil {
		spinner.Stop()
		return err
	}

	// Cancel each running workflow
	for _, run := range runs {
		cancelCmd := exec.Command("gh", "run", "cancel", strconv.FormatInt(run.DatabaseID, 10))
		_ = cancelCmd.Run() // Ignore errors for individual cancellations
	}

	spinner.Stop()
	return nil
}

// cancelWorkflowRunsByLockFile cancels in-progress runs for a workflow identified by its lock file name
func cancelWorkflowRunsByLockFile(lockFileName string) error {
	// Start spinner for network operation
	spinner := console.NewSpinner("Cancelling workflow runs...")
	spinner.Start()

	// Get running workflow runs by lock file name
	cmd := exec.Command("gh", "run", "list", "--workflow", lockFileName, "--status", "in_progress", "--json", "databaseId")
	output, err := cmd.Output()
	if err != nil {
		spinner.Stop()
		return err
	}

	var runs []struct {
		DatabaseID int64 `json:"databaseId"`
	}
	if err := json.Unmarshal(output, &runs); err != nil {
		spinner.Stop()
		return err
	}

	// Cancel each running workflow
	for _, run := range runs {
		cancelCmd := exec.Command("gh", "run", "cancel", strconv.FormatInt(run.DatabaseID, 10))
		_ = cancelCmd.Run() // Ignore errors for individual cancellations
	}

	spinner.Stop()
	return nil
}
