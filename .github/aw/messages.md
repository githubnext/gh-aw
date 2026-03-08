---
description: Style guide for workflow status messages (all safe-outputs.messages template types).
---

# Workflow Status Messages

Apply this guide when writing `safe-outputs.messages` in any workflow. Messages appear in GitHub issues, PR comments, and discussions.

## Rules

**Tone:** Plain and professional. Describe what the workflow does or what happened. No casual phrases ("Mission accomplished!", "Knowledge acquired!"), no dramatic language ("interrupted!", "crashed!"), no excitement punctuation (`!!`).

**Emoji:** One per message, at the start. Use the same emoji across `run-started`, `run-success`, `run-failure`, and `footer` for consistency. Do not append trailing emojis (`🏆`, `✅`, `📋`, etc.).

Emoji by domain: 🔍 search · 📐 architecture · 🔬 analysis/security · 📦 dependencies · 📝 docs · 🧪 testing · 🚀 release · 👀 review

## All Message Types

### Status messages (shown on the triggering issue/PR/discussion)

| Key | Variables | Default |
|-----|-----------|---------|
| `run-started` | `{workflow_name}`, `{run_url}`, `{event_type}` | `Agentic [{workflow_name}]({run_url}) triggered by this {event_type}.` |
| `run-success` | `{workflow_name}`, `{run_url}` | `✅ Agentic [{workflow_name}]({run_url}) completed successfully.` |
| `run-failure` | `{workflow_name}`, `{run_url}`, `{status}` | `❌ Agentic [{workflow_name}]({run_url}) {status} and wasn't able to produce a result.` |
| `detection-failure` | `{workflow_name}`, `{run_url}` | `⚠️ Security scanning failed for [{workflow_name}]({run_url}). Review the logs for details.` |

### Footer messages (appended to every AI-generated comment/issue/PR)

| Key | Variables | Default |
|-----|-----------|---------|
| `footer` | `{workflow_name}`, `{run_url}`, `{triggering_number}`, `{workflow_source}`, `{workflow_source_url}` | *(system default)* |
| `footer-install` | `{workflow_source}`, `{workflow_source_url}` | *(system default)* |
| `footer-workflow-recompile` | `{workflow_name}`, `{run_url}`, `{repository}` | `> Workflow sync report by [{workflow_name}]({run_url}) for {repository}` |
| `footer-workflow-recompile-comment` | `{workflow_name}`, `{run_url}`, `{repository}` | `> Update from [{workflow_name}]({run_url}) for {repository}` |
| `agent-failure-issue` | `{workflow_name}`, `{run_url}` | `> Agent failure tracked by [{workflow_name}]({run_url})` |
| `agent-failure-comment` | `{workflow_name}`, `{run_url}` | `> Agent failure update from [{workflow_name}]({run_url})` |

### Activation comment links (appended to `run-started` comment when resources are created)

| Key | Variables | Default |
|-----|-----------|---------|
| `pull-request-created` | `{item_number}`, `{item_url}` | `Pull request created: [#{item_number}]({item_url})` |
| `issue-created` | `{item_number}`, `{item_url}` | `Issue created: [#{item_number}]({item_url})` |
| `commit-pushed` | `{commit_sha}`, `{short_sha}`, `{commit_url}` | `Commit pushed: [\`{short_sha}\`]({commit_url})` |

### Staged mode messages (shown when `staged: true`)

| Key | Variables | Default |
|-----|-----------|---------|
| `staged-title` | `{operation}` | `🎭 Preview: {operation}` |
| `staged-description` | `{operation}` | `The following {operation} would occur if staged mode was disabled:` |

### Boolean options

| Key | Default | Description |
|-----|---------|-------------|
| `append-only-comments` | `false` | When `true`, creates a new comment for completion instead of editing the activation comment |

## Templates

### `run-started`
```
"{emoji} [{workflow_name}]({run_url}) is [present-tense verb] for this {event_type}..."
```
End with `...`. Use `{event_type}` to show what triggered the run.

### `run-success`
```
"{emoji} [{workflow_name}]({run_url}) has [past-tense completion phrase]."
```
End with `.`. Be specific about what was produced or verified.

### `run-failure`
```
"{emoji} [{workflow_name}]({run_url}) {status}. [One sentence on what could not be completed]."
```
Include `{status}` to surface the failure reason. Keep the follow-up sentence factual.

### `footer`
```
"> {emoji} *[Action noun] by [{workflow_name}]({run_url})*{history_link}"
```
Blockquote + italics. Include `{history_link}` for navigation to run history.

## Examples

✅ **Search workflow:**
```yaml
run-started: "🔍 [{workflow_name}]({run_url}) is searching the web for this {event_type}..."
run-success: "🔍 [{workflow_name}]({run_url}) has completed the web search and posted results."
run-failure: "🔍 [{workflow_name}]({run_url}) {status}. The search could not be completed."
footer:      "> 🔍 *Search results by [{workflow_name}]({run_url})*{history_link}"
```

✅ **Compatibility checker:**
```yaml
run-started: "🔬 [{workflow_name}]({run_url}) is analyzing API compatibility for this {event_type}..."
run-success: "🔬 [{workflow_name}]({run_url}) has completed the compatibility analysis."
run-failure: "🔬 [{workflow_name}]({run_url}) {status}. The compatibility analysis could not be completed."
footer:      "> 🔬 *Compatibility report by [{workflow_name}]({run_url})*{history_link}"
```

❌ **Avoid — casual language, mismatched emojis, trailing decorations:**
```yaml
run-started: "🔍 Brave Search activated! [{workflow_name}]({run_url}) is venturing into the web..."
run-success: "🦁 Mission accomplished! [{workflow_name}]({run_url}) returned with findings. Knowledge acquired! 🏆"
run-failure: "🔍 Search interrupted! [{workflow_name}]({run_url}) {status}. The web remains unexplored..."
footer:      "> 🦁 *Search results brought to you by [{workflow_name}]({run_url})*{history_link}"
```
