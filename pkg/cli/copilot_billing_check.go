package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

const copilotBillingTimeout = 3 * time.Second

// detectOrgCopilotCLIBilling returns "enabled" or "disabled" when the org's
// Copilot CLI billing status is definitively known (HTTP 200), or "" when the
// result is inconclusive (non-200 status, network error, or missing field).
// Only org owners receive a 200; for non-owners this always returns "".
func detectOrgCopilotCLIBilling(ctx context.Context, orgLogin string) (string, error) {
	client, err := api.NewRESTClient(api.ClientOptions{})
	if err != nil {
		return "", err
	}
	return detectOrgCopilotCLIBillingWithClient(ctx, orgLogin, client)
}

// detectOrgCopilotCLIBillingWithClient is the testable core of detectOrgCopilotCLIBilling.
// It calls GET /orgs/{org}/copilot/billing with a 3 s timeout and returns the "cli" field.
// Any non-200 response or error is treated as inconclusive (returns "", err).
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
