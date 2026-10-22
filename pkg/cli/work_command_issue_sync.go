package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/spf13/cobra"
)

func workSyncIssuesCommand() *cobra.Command {
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

			result, err := workBranch(cmd).SynchronizeIssueTransition(cmd.Context(), id)
			if err != nil {
				return err
			}
			message := fmt.Sprintf("Synchronized %d existing terminal Work mirrors; request=%s", result.Synchronized, id)
			if result.Disabled {
				message = "Issue synchronization is disabled in the verified repository configuration"
			}
			return workPrint(cmd, result, message)
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
			if _, err := workBranch(cmd).SynchronizeIssueTransition(cmd.Context(), id); err != nil {
				return fmt.Errorf("queue transition committed; Issue synchronization pending; retry sync-issues --request-id %s: %w", id, err)
			}
			return nil
		}
	}
	return nil
}
