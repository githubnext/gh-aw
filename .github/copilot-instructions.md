# Instructions for updating daily outcome reports

Update the daily outcome report format to use progressive disclosure and an executive-first visual layout.

## Required top section

Each report should start with:

1. A concise **Executive read** sentence.
2. A compact table with these columns:
   - `Workflow`
   - `Status`
   - `Lifecycle health`
3. A **Legend** section immediately below the table.

### Workflow table requirements

- `Status` must be a compact visual bar using emoji only.
- Use inline HTML to prevent line wrapping in the Status column:
  - format bars as `<span style="white-space: nowrap;">...</span>`
- Use these status symbols:
  - 🟩 accepted
  - 🟥 rejected
  - 🟨 pending
  - ⬜ unknown
- Do not show numeric counts in the top executive table if the visual bar already communicates them.

### Lifecycle health values

Use one of:
- 🟢 Resolving
- 🟡 In flight
- 🟠 Aging
- 🔴 Stuck
- ⚪ Underdefined

The top section should be optimized for at-a-glance executive readability.

## Required legend

Add a legend directly below the executive table in this format:

- **Status:** 🟩 accepted · 🟥 rejected · 🟨 pending · ⬜ unknown
- **Lifecycle health:** 🟢 resolving · 🟡 in flight · 🟠 aging · 🔴 stuck · ⚪ underdefined

## Progressive disclosure

All detailed metrics, numeric breakdowns, evidence quality, trends, and action items must appear below the executive section inside an HTML `<details>` block.

Use this structure:

```html
<details>
<summary>Detailed metrics, evidence quality, workflow counts, and actions</summary>

...details here...

</details>
```

## Details section contents

Move the following into the `<details>` section:
- scorecard metrics and percentages
- per-workflow numeric counts
- trend analysis
- action items
- evidence quality notes
- fallback evaluator / weak-signal diagnostics

## General guidance

- Prefer progressive disclosure.
- Optimize the top section for executive scanning.
- Preserve rigorous metrics in the details section.
- Keep the visual style consistent with other reports.
