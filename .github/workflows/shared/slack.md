---
safe-outputs:
  jobs:
    post-to-slack-channel:
      description: "Post a message to a Slack channel. Message must be 200 characters or less. Supports basic Slack markdown: *bold*, _italic_, ~strike~, `code`, ```code block```, >quote, and links <url|text>."
      runs-on: ubuntu-latest
      output: "Message posted to Slack successfully!"
      inputs:
        channel_id:
          description: "The Slack channel ID (e.g., C1234567890)"
          required: true
          type: string
        message:
          description: "The message to post (max 200 characters, supports Slack markdown)"
          required: true
          type: string
      permissions:
        contents: read
      steps:
        - name: Validate and post message to Slack
          uses: actions/github-script@v8
          env:
            SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
            SLACK_CHANNEL_ID: "${{ inputs.channel_id }}"
            SLACK_MESSAGE: "${{ inputs.message }}"
          with:
            script: |
              const slackBotToken = process.env.SLACK_BOT_TOKEN;
              const slackChannelId = process.env.SLACK_CHANNEL_ID;
              const slackMessage = process.env.SLACK_MESSAGE;
              
              // Validate required environment variables
              if (!slackBotToken) {
                core.setFailed('SLACK_BOT_TOKEN secret is not configured. Please add it to your repository secrets.');
                return;
              }
              
              if (!slackChannelId) {
                core.setFailed('SLACK_CHANNEL_ID is required');
                return;
              }
              
              if (!slackMessage) {
                core.setFailed('SLACK_MESSAGE is required');
                return;
              }
              
              // Validate message length (max 200 characters)
              const maxLength = 200;
              if (slackMessage.length > maxLength) {
                core.setFailed(`Message length (${slackMessage.length} characters) exceeds maximum allowed length of ${maxLength} characters`);
                return;
              }
              
              core.info(`Posting message to Slack channel: ${slackChannelId}`);
              core.info(`Message length: ${slackMessage.length} characters`);
              
              try {
                const response = await fetch('https://slack.com/api/chat.postMessage', {
                  method: 'POST',
                  headers: {
                    'Content-Type': 'application/json; charset=utf-8',
                    'Authorization': `Bearer ${slackBotToken}`
                  },
                  body: JSON.stringify({
                    channel: slackChannelId,
                    text: slackMessage
                  })
                });
                
                const data = await response.json();
                
                if (!response.ok) {
                  core.setFailed(`Slack API HTTP error (${response.status}): ${response.statusText}`);
                  return;
                }
                
                if (!data.ok) {
                  core.setFailed(`Slack API error: ${data.error || 'Unknown error'}`);
                  if (data.error === 'invalid_auth') {
                    core.error('Authentication failed. Please verify your SLACK_BOT_TOKEN is correct.');
                  } else if (data.error === 'channel_not_found') {
                    core.error('Channel not found. Please verify the SLACK_CHANNEL_ID is correct and the bot has access to it.');
                  }
                  return;
                }
                
                core.info('Message posted successfully to Slack');
                core.info(`Message timestamp: ${data.ts}`);
                core.info(`Channel: ${data.channel}`);
                
                // Add step summary
                await core.summary
                  .addHeading('Slack Message Posted', 2)
                  .addRaw(`✅ Successfully posted message to channel \`${slackChannelId}\``)
                  .addBreak()
                  .addRaw(`**Message:** ${slackMessage}`)
                  .addBreak()
                  .addRaw(`**Timestamp:** ${data.ts}`)
                  .write();
                
              } catch (error) {
                core.setFailed(`Failed to post message to Slack: ${error instanceof Error ? error.message : String(error)}`);
              }
---

## Slack Integration

This shared configuration provides a custom safe-job for posting messages to Slack channels.

### Safe Job: post-to-slack-channel

The `post-to-slack-channel` safe-job allows agentic workflows to post messages to Slack channels through the Slack API.

**Required Inputs:**
- `channel_id`: The Slack channel ID (e.g., C1234567890) to post the message to
- `message`: The message text to post (maximum 200 characters)

**Message Length Limit:**
Messages are limited to 200 characters to ensure concise, focused updates. The safe-job will fail if the message exceeds this limit.

**Supported Slack Markdown:**
The message supports basic Slack markdown syntax:
- `*bold*` - Bold text
- `_italic_` - Italic text
- `~strike~` - Strikethrough text
- `` `code` `` - Inline code
- ` ```code block``` ` - Code block
- `>quote` - Block quote
- `<url|text>` - Hyperlink with custom text

**Example Usage in Workflow:**

```
Please post a summary to our Slack channel C1234567890 using the post-to-slack-channel safe-job.
Keep the message under 200 characters.
```

**Audit Mode Support:**

This safe-job fully supports audit/staged mode. When `staged: true` is set in the workflow's safe-outputs configuration, any errors (such as authentication failures or message length violations) will be shown as preview issues instead of failing the workflow.

### Setup

1. **Create a Slack App** with a Bot User OAuth Token:
   - Go to https://api.slack.com/apps
   - Create a new app or select an existing one
   - Navigate to "OAuth & Permissions"
   - Add the `chat:write` bot token scope
   - Install the app to your workspace
   - Copy the "Bot User OAuth Token" (starts with `xoxb-`)

2. **Add the bot to your channel**:
   - In Slack, go to the channel where you want to post messages
   - Type `/invite @YourBotName` to add the bot
   - Get the channel ID from the channel details

3. **Configure GitHub Secrets**:
   - Add `SLACK_BOT_TOKEN` secret to your repository with the Bot User OAuth Token
   - The channel ID can be passed as input to the safe-job

4. **Include this configuration in your workflow**:
   ```yaml
   imports:
     - shared/slack.md
   ```
