---
description: Style guide for workflow status messages (run-started, run-success, run-failure, footer).
---

# Workflow Status Messages

Apply this guide when writing `safe-outputs.messages` in any workflow. Messages appear in GitHub issues, PR comments, and discussions.

## Rules

**Tone:** Plain and professional. Describe what the workflow does or what happened. No casual phrases ("Mission accomplished!", "Knowledge acquired!"), no dramatic language ("interrupted!", "crashed!"), no excitement punctuation (`!!`).

**Emoji:** One per message, at the start. Use the same emoji across all four message types for consistency. Do not append trailing emojis (`🏆`, `✅`, `📋`, etc.).

Emoji by domain: 🔍 search · 📐 architecture · 🔬 analysis/security · 📦 dependencies · 📝 docs · 🧪 testing · 🚀 release · 👀 review

## Message Types and Variables

### `run-started`
Available variables: `{workflow_name}`, `{run_url}`, `{event_type}`

```
"{emoji} [{workflow_name}]({run_url}) is [present-tense verb] for this {event_type}..."
```
End with `...`. Use `{event_type}` to show what triggered the run.

### `run-success`
Available variables: `{workflow_name}`, `{run_url}`

```
"{emoji} [{workflow_name}]({run_url}) has [past-tense completion phrase]."
```
End with `.`. Be specific about what was produced or verified.

### `run-failure`
Available variables: `{workflow_name}`, `{run_url}`, `{status}`

```
"{emoji} [{workflow_name}]({run_url}) {status}. [One sentence on what could not be completed]."
```
Include `{status}` to surface the failure reason. Keep the follow-up sentence factual.

### `footer`
Available variables: `{workflow_name}`, `{run_url}`, `{history_link}`

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

```yaml
run-started: "🔬 Breaking Change Checker online! [{workflow_name}]({run_url}) is analyzing compatibility..."
run-success: "✅ Analysis complete! [{workflow_name}]({run_url}) has reviewed all changes. Verdict delivered! 📋"
run-failure: "🔬 Analysis interrupted! [{workflow_name}]({run_url}) {status}. Compatibility status unknown..."
```
