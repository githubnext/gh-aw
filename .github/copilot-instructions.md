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

Assign one lifecycle health label per workflow based on its outcome history across consecutive report cycles. Use cache-memory from the previous run to determine trend.

| Label | Emoji | When to assign |
|---|---|---|
| resolving | 🟢 | Pending items are moving to accepted/rejected at a healthy rate over recent runs |
| in flight | 🟡 | Outcomes are still being evaluated; no concerning stagnation pattern yet |
| aging | 🟠 | One or more items have been pending for >48 hours without resolution |
| stuck | 🔴 | Pending/unknown outcomes persist across two or more consecutive report cycles; pending count has not decreased |
| underdefined | ⚪ | Most outcomes land in unknown or ignored; acceptance/rejection criteria are unclear, the evaluator lacks signal, or the outcome model for this workflow has not yet matured |

**Lifecycle health decision rules:**
- A workflow is **stuck** (🔴) if its pending count has not decreased over two or more consecutive evaluation cycles. This is a signal that items need human review, the workflow has a timeout problem, or the evaluator is not reaching a terminal state.
- A workflow is **underdefined** (⚪) if its unknown or ignored share consistently exceeds 50% of its outcomes. This means the outcome model needs work: the safe output type may lack a dedicated evaluator, the acceptance/rejection criteria may not be clear, or the workflow's outputs may not map well to existing outcome categories.
- A workflow is **aging** (🟠) if any individual item has been pending for >48 hours, even if the overall pending count is decreasing.
- A workflow is **in flight** (🟡) when outcomes are actively being evaluated and no stagnation or quality problem is detected yet.
- A workflow is **resolving** (🟢) when its pending count is decreasing across consecutive cycles and a reasonable share of items are reaching accepted or rejected.

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
