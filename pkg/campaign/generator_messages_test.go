package campaign

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignGeneratorMessageTemplates(t *testing.T) {
	data := BuildCampaignGenerator()
	require.NotNil(t, data, "workflow data should not be nil")
	require.NotNil(t, data.SafeOutputs, "safe outputs should be configured")
	require.NotNil(t, data.SafeOutputs.Messages, "messages should be configured")

	msgs := data.SafeOutputs.Messages

	expectedFooter := "> *Campaign coordination by [{workflow_name}]({run_url})*\n" +
		"Docs: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"

	expectedRunStarted := "[{workflow_name}]({run_url}) started generating your campaign (trigger: {event_type}).\n\n" +
		"**What’s happening**\n" +
		"1. Read requirements from this issue\n" +
		"2. Create a GitHub Project (views + standard fields)\n" +
		"3. Generate a campaign spec + orchestrator workflow\n" +
		"4. Update this issue with a handoff checklist\n\n" +
		"You don’t need to do anything yet. When it finishes, look for a PR link in the issue update.\n\n" +
		"Learn more: https://githubnext.github.io/gh-aw/guides/campaigns/flow/"

	expectedRunSuccess := "[{workflow_name}]({run_url}) finished the initial campaign setup.\n\n" +
		"**Next steps**\n" +
		"1. Review the pull request created by the Copilot Coding Agent\n" +
		"2. Merge it\n" +
		"3. Run the campaign orchestrator from the Actions tab\n\n" +
		"Docs: https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/"

	expectedRunFailure := "[{workflow_name}]({run_url}) {status}.\n\n" +
		"**What to do**\n" +
		"- Open the run link above and check the logs for the first error\n" +
		"- Fix the issue (permissions/secret/config), then re-run by re-applying the label\n\n" +
		"Troubleshooting: https://githubnext.github.io/gh-aw/guides/campaigns/flow/#when-something-goes-wrong"

	assert.Equal(t, expectedFooter, msgs.Footer, "Footer template should match")
	assert.Equal(t, expectedRunStarted, msgs.RunStarted, "RunStarted template should match")
	assert.Equal(t, expectedRunSuccess, msgs.RunSuccess, "RunSuccess template should match")
	assert.Equal(t, expectedRunFailure, msgs.RunFailure, "RunFailure template should match")
}
