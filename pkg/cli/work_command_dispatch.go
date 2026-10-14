package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/spf13/cobra"
)

func workDispatchCommand() *cobra.Command {
	return workDispatchCommandWithRunner(RunWorkflowOnGitHub)
}

func workDispatchCommandWithRunner(run func(context.Context, string, RunOptions) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispatch <dispatcher>",
		Short: "Run a trusted dispatcher to grant and launch fair queue assignments",
		Long: "Run a verified work-queue dispatcher through the GitHub CLI on the target\n" +
			"repository's default branch. Its trusted controls select and launch assignments.\n" +
			"This command cannot choose a Work, reassign a Claim, or replay worker effects.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("dispatcher_invalid: expected exactly one dispatcher basename")
			}
			branch := workBranch(cmd)
			requestID, _ := cmd.Flags().GetString("request-id")
			if branch.Name != workqueue.DefaultBranch || requestID != "" {
				return errors.New("dispatcher_invalid: --branch and --request-id cannot override the dispatcher's compiled queue authority or native run identity")
			}
			route, err := branch.DispatcherFromConfig(cmd.Context(), args[0]) //nolint:uncheckedsliceindex // Exactly one argument is checked above.
			if err != nil {
				return err
			}
			result := struct {
				workqueue.DispatcherRoute
				Status string `json:"status"`
			}{route, "preview"}
			preview, _ := cmd.Flags().GetBool("dry-run")
			if preview {
				return workPrint(cmd, result, fmt.Sprintf("Dispatcher preview: %s at %s (%s); no Claims granted or workflows launched",
					route.Workflow, route.Ref, route.Revision))
			}
			wait, _ := cmd.Flags().GetBool("wait")
			if err := run(cmd.Context(), args[0], RunOptions{ //nolint:uncheckedsliceindex // Exactly one argument is checked above.
				WorkflowRunFlags:  WorkflowRunFlags{RepoOverride: route.Repository, RefOverride: route.Ref},
				WaitForCompletion: wait,
				ExpectedRevision:  route.Revision,
			}); err != nil {
				return err
			}
			result.Status = "activated"
			return workPrint(cmd, result, fmt.Sprintf("Dispatcher activated: %s at %s; inspect the ledger for Claims and Results",
				route.Workflow, route.Revision))
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview the verified dispatcher route without compiling, granting Claims, or launching workflows")
	cmd.Flags().Bool("wait", false, "Wait for the dispatcher run; worker Results still require independent verification")
	return cmd
}
