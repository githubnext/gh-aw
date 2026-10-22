package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/spf13/cobra"
)

func workSyncIssuesCommand() *cobra.Command {
	return workSyncIssuesCommandWithSynchronizer(func(ctx context.Context, branch workqueue.Branch, id string) (workqueue.IssueSynchronization, error) {
		return branch.SynchronizeIssueTransition(ctx, id)
	})
}

func workSyncIssuesCommandWithSynchronizer(synchronize func(context.Context, workqueue.Branch, string) (workqueue.IssueSynchronization, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "sync-issues",
		Short: "Synchronize existing mirrors for one committed terminal request",
		Long: "Requires repository maintain or administrator permission and an exact --request-id.\n" +
			"Updates only existing AW-managed Issue/status and summary receipts for\n" +
			"current terminal Work in that request. Never creates, rebinds, closes,\n" +
			"or globally scans Issues, and never steals uncertain projection locks.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("request-id")
			if id == "" {
				return errors.New("request_invalid: sync-issues requires --request-id for one committed terminal request")
			}

			result, err := synchronize(cmd.Context(), workBranch(cmd), id)
			if err != nil && len(result.Pending) == 0 && result.Synchronized == 0 {
				return err
			}
			message := fmt.Sprintf("Synchronized %d existing terminal Work mirrors; request=%s", result.Synchronized, id)
			if result.Disabled {
				message = "Issue synchronization is disabled in the verified repository configuration"
			}
			if len(result.Pending) != 0 {
				message += fmt.Sprintf("; %d terminal Work pending existing mirror receipts", len(result.Pending))
			}
			return errors.Join(err, workPrint(cmd, result, message))
		},
	}
}

func workSyncTerminalPublication(cmd *cobra.Command, publication *workqueue.Publication) error {
	if publication == nil || publication.Commit == nil {
		return nil
	}
	for _, operation := range publication.Commit.Operations {
		var fields struct{ Kind string }
		if err := json.Unmarshal(operation, &fields); err != nil {
			return err
		}
		if fields.Kind == "WorkCancellation" || fields.Kind == "Result" || fields.Kind == "DeliveryFailure" {
			id := publication.Commit.Request.ID
			if _, err := workBranch(cmd).SynchronizePublishedIssueTransition(cmd.Context(), publication); err != nil {
				return fmt.Errorf("queue transition committed; Issue synchronization pending; retry sync-issues --request-id %s: %w", id, err)
			}

			return nil
		}
	}
	return nil
}

func workPrintTerminalPublication(cmd *cobra.Command, publication *workqueue.Publication, result any, message string) error {
	syncErr := workSyncTerminalPublication(cmd, publication)
	return errors.Join(syncErr, workPrint(cmd, result, message))
}
