---
"gh-aw": patch
---

Replace channel_id input with GH_AW_SLACK_CHANNEL_ID environment variable in Slack shared workflow

Updates the Slack shared workflow to use a required environment variable `GH_AW_SLACK_CHANNEL_ID` instead of accepting the channel ID as a `channel_id` input parameter. This simplifies the interface and aligns with best practices for configuration management. Workflows using the Slack shared workflow will need to set `GH_AW_SLACK_CHANNEL_ID` as an environment variable or repository variable instead of passing `channel_id` as an input.
