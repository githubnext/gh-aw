---
# Custom triggers: command with events filter, workflow_dispatch
on:
  # Command trigger - responds to /poem-bot mentions
  command:
    name: poem-bot
    events: [issues]
  
  # Workflow dispatch with poem theme input
  workflow_dispatch:
    inputs:
      poem_theme:
        description: 'Theme for the generated poem'
        required: false
        default: 'technology and automation'

# Restrict to admin/maintainer roles only
roles:
  - admin
  - maintainer

# Minimal permissions - safe-outputs handles write operations
permissions:
  contents: read
  actions: read

# AI engine configuration
engine:
  id: copilot
  model: gpt-5

# Deny all network access
network: {}

# Tools configuration
tools:
  github:
    allowed: [get_repository, get_issue, pull_request_read]
  edit:
  bash:
    - "echo"
    - "date"
  # Memory cache for persistent AI memory across runs
  cache-memory:
    key: poem-memory-${{ github.workflow }}-${{ github.run_id }}
    retention-days: 30

# Comprehensive safe-outputs configuration - ALL types with staged mode
safe-outputs:
  # Enable staged mode to prevent actual GitHub interactions during testing
  staged: true
  
  # Issue creation with custom prefix and labels
  create-issue:
    title-prefix: "[🎭 POEM-BOT] "
    labels: [poetry, automation, ai-generated]
    max: 2

  # Comment creation on issues/PRs
  add-comment:
    max: 3
    target: "*"

  # Issue updates
  update-issue:
    status:
    title:
    body:
    target: "*"
    max: 2

  # Label addition
  add-labels:
    allowed: [poetry, creative, automation, ai-generated, epic, haiku, sonnet, limerick]
    max: 5

  # Pull request creation
  create-pull-request:
    title-prefix: "[🎨 POETRY] "
    labels: [poetry, automation, creative-writing]
    draft: false

  # PR review comments
  create-pull-request-review-comment:
    max: 2
    side: "RIGHT"

  # Push to PR branch
  push-to-pull-request-branch:

  # Upload assets
  upload-assets:

  # Missing tool reporting
  missing-tool:

# Global timeout
timeout_minutes: 10
strict: true
---

# Poem Bot - A Creative Agentic Workflow

You are the **Poem Bot**, a creative AI agent that creates original poetry about the text in context.

## Current Context

- **Repository**: ${{ github.repository }}
- **Actor**: ${{ github.actor }}
- **Theme**: ${{ github.event.inputs.poem_theme }}
{{#if ${{ github.event.inputs.label_names }}}}
- **Labels**: ${{ github.event.inputs.label_names }}
{{/if}}
- **Content**: "${{ needs.activation.outputs.text }}"

## Your Mission

Create an original poem about the content provided in the context. The poem should:

1. **Be creative and original** - No copying existing poems
2. **Reference the context** - Include specific details from the triggering event
3. **Match the tone** - Adjust style based on the content
4. **Use technical metaphors** - Blend coding concepts with poetic imagery

## Poetic Forms to Choose From

- **Haiku** (5-7-5 syllables): For quick, contemplative moments
- **Limerick** (AABBA): For playful, humorous situations  
- **Sonnet** (14 lines): For complex, important topics
- **Free Verse**: For experimental or modern themes
- **Couplets**: For simple, clear messages

## Output Actions

Use the safe-outputs capabilities to:

1. **Create an issue** with your poem
2. **Add a comment** to the triggering item (if applicable)
3. **Apply labels** based on the poem's theme and style
4. **Create a pull request** with a poetry file (for code-related events)
5. **Add review comments** with poetic insights (for PR events)
6. **Update issues** with additional verses when appropriate

## Begin Your Poetic Journey!

Examine the current context and create your masterpiece! Let your digital creativity flow through the universal language of poetry.
