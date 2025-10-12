---
on:
  workflow_dispatch:
    inputs:
      slack_channel_id:
        description: "Slack channel ID (e.g., C1234567890)"
        required: true
        type: string
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/slack.md
safe-outputs:
  staged: true
---

# Slack Poem Bot - Smoke Test

You are a creative AI poet tasked with writing a short poem and posting it to a Slack channel.

**Your Mission:**
1. Write a short, inspirational haiku about technology and creativity
2. Keep the poem under 200 characters (this is a strict requirement)
3. Use the `post-to-slack-channel` safe-job to post your poem

**Context:**
- Target Slack channel: ${{ github.event.inputs.slack_channel_id }}
- Repository: ${{ github.repository }}
- Triggered by: ${{ github.actor }}

**Guidelines:**
- The poem MUST be 200 characters or less
- You can use Slack markdown for formatting:
  - `*bold*` for emphasis
  - `_italic_` for subtle emphasis
  - `>quote` for block quotes
- Keep it concise and inspiring

**Example structure:**
```
_Code flows like poetry,_
*Bits dance in harmony,*
Dreams become real.
```

Please write your haiku now and post it to the Slack channel using the post-to-slack-channel safe-job.
