# Writing Style Guidelines for Agentic Workflows

This guide provides style guidelines for creating effective agentic workflow prompts and user interactions.

## Interactive Mode Communication Style

When interacting with users in conversational mode, adopt the GitHub Copilot CLI chat style to create an engaging, helpful experience.

### Style Characteristics

- **Conversational and Friendly**: Format questions and responses similarly to the GitHub Copilot CLI chat style
- **Emoji Usage**: Use emojis to make the conversation more engaging and to highlight important points
- **Incremental Questions**: Don't overwhelm users with too many questions at once or long bullet points
- **User-Driven**: Ask users to express their intent in their own words, then translate it into an agentic workflow
- **Back-and-Forth**: Engage in natural back-and-forth conversation to gather necessary details

### Example Interactions

**Good** ✅:
```
🤖 What do you want to automate today?
```

**After user responds**:
```
💡 That sounds like a great use case for an issue triage workflow! 

📅 When should this run - when issues are opened, on a schedule, or both?
```

**Avoid** ❌:
```
Please provide:
- Workflow trigger (issues/PRs/schedule)
- Required tools
- Permission requirements
- Network access needs
- Safe output configuration
- Timeout settings
```

## Workflow Prompt Writing

The body of agentic workflow markdown files is a prompt for the AI agent. Follow these best practices:

### Clarity and Actionability

- **Be specific**: Clearly state what the agent should accomplish
- **Provide context**: Explain why the task matters and any relevant background
- **Set boundaries**: Specify what the agent should NOT do if relevant
- **Structure logically**: Use headers, lists, and clear sections

### Example Structure

```markdown
# <Workflow Name>

You are an AI agent that <what the agent does>.

## Your Task

<Clear, actionable instructions>

## Guidelines

<Specific guidelines for behavior>

## Examples

<Optional: Show examples of good output>

## Constraints

<Optional: Explicit boundaries>
```

### Tone and Voice

- **Direct**: Use imperative voice ("Analyze the issue", not "You should analyze")
- **Professional**: Maintain clarity without being overly casual
- **Specific**: Provide concrete instructions rather than vague guidance
- **Focused**: Keep prompts concise and on-topic

## Frontmatter Configuration Style

### Minimalism Principle

✨ **Keep frontmatter minimal** - Only include fields that differ from sensible defaults:

**DO NOT include**:
- `engine: copilot` - Copilot is the default engine. Only specify if using Claude, Codex, or custom
- `timeout-minutes:` - Has sensible defaults. Only specify if needing custom timeout
- `workflow_dispatch:` - Automatically added by compiler for scheduled workflows
- Other fields with good defaults - Let the compiler use sensible defaults unless customization is needed

**Example - Minimal Frontmatter**:
```yaml
---
description: Automatically label issues based on their content
on:
  issues:
    types: [opened, edited]
roles: read
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-comment:
    max: 1
---
```

This example omits unnecessary fields while including only what's essential for the workflow's behavior.

## Documentation Comments

### Workflow Files

Add helpful comments at the top of workflow files to guide users:

```markdown
<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
@./agentics/<workflow-id>.md
```

### Agentics Prompt Files

Add header comments explaining the purpose and editability:

```markdown
<!-- This prompt will be imported in the agentic workflow .github/workflows/<workflow-id>.md at runtime. -->
<!-- You can edit this file to modify the agent behavior without recompiling the workflow. -->
```

## Summary

- Use emoji-enhanced, conversational style for user interactions
- Write clear, actionable prompts for AI agents
- Keep frontmatter minimal - omit defaults
- Add helpful comments to guide users
- Maintain a professional, direct tone in workflow prompts
