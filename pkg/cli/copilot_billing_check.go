package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

const copilotBillingTimeout = 3 * time.Second

// copilotBillingInconclusiveNote is the user-facing message printed when the
// org's Copilot CLI billing status cannot be confirmed (non-200 response,
// network error, missing field, or no org login available).
const copilotBillingInconclusiveNote = "Could not confirm org Copilot CLI billing — check with your org admin."

// detectOrgCopilotCLIBillingWithClient calls GET /orgs/{org}/copilot/billing with
// a 3 s timeout and returns the raw "cli" field. Any non-200 response or error
// results in ("", err).
func detectOrgCopilotCLIBillingWithClient(ctx context.Context, orgLogin string, client *api.RESTClient) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, copilotBillingTimeout)
	defer cancel()
	var result struct {
		CLI string `json:"cli"`
	}
	if err := client.DoWithContext(ctx, "GET", fmt.Sprintf("orgs/%s/copilot/billing", orgLogin), nil, &result); err != nil {
		return "", err
	}
	return result.CLI, nil
}

// orgCopilotBillingProbeResult holds the outcome of probing the org's Copilot CLI
// billing status together with derived UI hints for the auth method selection form.
type orgCopilotBillingProbeResult struct {
	// BillingStatus is the raw "cli" field value returned by the API (e.g.
	// "enabled", "disabled", another policy string, or "" if inconclusive).
	BillingStatus string
	// LabelSuffix is appended to the copilot-requests option label in the form.
	// It is empty when the result is inconclusive.
	LabelSuffix string
	// Disabled indicates whether the copilot-requests option should be blocked
	// by a validation guard.
	Disabled bool
	// InfoNote is a non-empty one-line message to print to stderr when the
	// billing check is inconclusive (error, non-200, or missing "cli" field).
	InfoNote string
}

// probeCopilotBillingForOrg probes the org's Copilot CLI billing status and
// returns derived UI hints for the auth method selection form.
func probeCopilotBillingForOrg(ctx context.Context, orgLogin string) orgCopilotBillingProbeResult {
	client, err := api.NewRESTClient(api.ClientOptions{})
	if err != nil {
		return orgCopilotBillingProbeResult{
			InfoNote: copilotBillingInconclusiveNote,
		}
	}
	return probeCopilotBillingForOrgWithClient(ctx, orgLogin, client)
}

// probeCopilotBillingForOrgWithClient is the testable core of probeCopilotBillingForOrg.
// It calls detectOrgCopilotCLIBillingWithClient and maps the result to UI hints.
func probeCopilotBillingForOrgWithClient(ctx context.Context, orgLogin string, client *api.RESTClient) orgCopilotBillingProbeResult {
	cliStatus, err := detectOrgCopilotCLIBillingWithClient(ctx, orgLogin, client)
	switch {
	case err != nil || cliStatus == "":
		return orgCopilotBillingProbeResult{
			BillingStatus: cliStatus,
			InfoNote:      copilotBillingInconclusiveNote,
		}
	case cliStatus == "enabled":
		return orgCopilotBillingProbeResult{
			BillingStatus: cliStatus,
			LabelSuffix:   " [recommended — org Copilot CLI billing enabled]",
		}
	default: // "disabled" or any other policy value
		return orgCopilotBillingProbeResult{
			BillingStatus: cliStatus,
			LabelSuffix:   fmt.Sprintf(" [not available — org Copilot CLI billing: %s]", cliStatus),
			Disabled:      true,
		}
	}
}
