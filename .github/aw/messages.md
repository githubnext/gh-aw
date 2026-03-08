---
description: Style guide for workflow status messages (run-started, run-success, run-failure) to establish consistent tone, emoji usage, and message templates across all workflows.
---

# Workflow Status Messages

Consult this file when writing or reviewing `run-started`, `run-success`, and `run-failure` messages in the `safe-outputs.messages` block of a workflow.

These messages appear in GitHub issues, pull request comments, and discussions when a workflow starts, completes, or fails. Consistent, professional messages help enterprise users understand what a workflow is doing and build confidence in the system.

## Tone Guidelines

Status messages must be **professional and clear**.

- Use plain, direct language that describes what the workflow is doing or what happened
- Avoid casual or playful phrases such as "Mission accomplished!", "Knowledge acquired!", or "online!"
- Avoid excitement punctuation (multiple exclamation marks, `!!`)
- Avoid filler words like "activated", "venturing", or "returns with findings"
- Write as if reporting to a colleague, not cheering at a sports event

## Emoji Conventions

Each message uses **one emoji** that reflects the workflow's primary purpose:

- Place the emoji at the **start** of the message
- Use the **same emoji** for `run-started` and `run-failure` so the workflow is recognizable even on error
- `run-success` may use the same emoji or a closely related one, but must not add decorative trailing emojis
- Do **not** append excitement or trophy emojis at the end of any message (e.g., no `🏆`, `✅`, `📋` appended after the sentence)

Choose an emoji that reflects the workflow's domain:

| Domain | Suggested emoji |
|--------|----------------|
| Search / discovery | 🔍 |
| Architecture / design | 📐 |
| Security / analysis | 🔬 |
| Dependency management | 📦 |
| Documentation | 📝 |
| Testing / quality | 🧪 |
| Release / deployment | 🚀 |
| Code review | 👀 |

## Message Templates

Use these templates as the starting point for all three message types. Variables in `{…}` are substituted at runtime by the safe-outputs system.

### `run-started`

```
"{emoji} [{workflow_name}]({run_url}) is [present-tense verb phrase] for this {event_type}..."
```

- Use a present-tense verb phrase that describes the ongoing action (e.g., "analyzing", "searching", "checking")
- End with `...` to indicate work is in progress
- Include `{event_type}` to give context on what triggered the run

### `run-success`

```
"{emoji} [{workflow_name}]({run_url}) has [past-tense completion phrase]."
```

- Use a past-tense verb phrase that describes what was accomplished (e.g., "completed the analysis", "published the report")
- End with a single period; no trailing emojis
- Be specific about what was produced or verified

### `run-failure`

```
"{emoji} [{workflow_name}]({run_url}) {status}. [One sentence describing what could not be completed]."
```

- Include `{status}` to surface the failure reason from GitHub Actions
- Follow with a single sentence explaining what the workflow was unable to do
- Keep the tone neutral and factual — avoid dramatic language like "interrupted" or "crashed"

## Examples

### ✅ Correct

**Search workflow:**
```yaml
run-started: "🔍 [{workflow_name}]({run_url}) is searching the web for this {event_type}..."
run-success: "🔍 [{workflow_name}]({run_url}) has completed the web search and posted results."
run-failure: "🔍 [{workflow_name}]({run_url}) {status}. The search could not be completed."
```

**Compatibility checker:**
```yaml
run-started: "🔬 [{workflow_name}]({run_url}) is analyzing API compatibility for this {event_type}..."
run-success: "🔬 [{workflow_name}]({run_url}) has completed the compatibility analysis."
run-failure: "🔬 [{workflow_name}]({run_url}) {status}. The compatibility analysis could not be completed."
```

### ❌ Incorrect

**Playful language and trailing emojis:**
```yaml
# BAD: casual language, excitement emojis
run-started: "🔍 Brave Search activated! [{workflow_name}]({run_url}) is venturing into the web for this {event_type}..."
run-success: "🦁 Mission accomplished! [{workflow_name}]({run_url}) has returned with the findings. Knowledge acquired! 🏆"
run-failure: "🔍 Search interrupted! [{workflow_name}]({run_url}) {status}. The web remains unexplored..."
```

**Trailing decorative emoji and mismatched emojis:**
```yaml
# BAD: run-success uses different emoji, trailing ✅ and 📋 added decoratively
run-started: "🔬 Breaking Change Checker online! [{workflow_name}]({run_url}) is analyzing API compatibility..."
run-success: "✅ Analysis complete! [{workflow_name}]({run_url}) has reviewed all changes. Compatibility verdict delivered! 📋"
run-failure: "🔬 Analysis interrupted! [{workflow_name}]({run_url}) {status}. Compatibility status unknown..."
```

## Footer Messages

The `footer` message is appended automatically to every comment or issue the workflow creates. Keep it short and factual:

```yaml
footer: "> 🔍 *Search results by [{workflow_name}]({run_url})*{history_link}"
```

- Use the same emoji as `run-started`
- Use italics (`*…*`) and blockquote (`> `) formatting
- Include `{history_link}` so readers can navigate to the run history
