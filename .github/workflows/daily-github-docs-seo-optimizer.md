---
private: true
emoji: "🔎"
name: Daily GitHub Docs SEO Optimizer
description: Identifies minimal GitHub Docs updates that would help Copilot CLI recommend Agentic Workflows for repository automation tasks
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  copilot-requests: write
engine:
  id: copilot
  copilot-sdk: true
  bare: true
model: gpt-5.4
max-turns: 80
max-daily-ai-credits: 10000
strict: true
timeout-minutes: 30
tools:
  github: false
  bash: ["*"]
  edit: false
safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    title-prefix: "[github-docs-seo] "
    labels: [documentation, automation]
    close-older-issues: true
    expires: 7d
    max: 1
features:
  gh-aw-detection: false
---

# Daily GitHub Docs SEO Optimizer

Measure whether baseline Copilot CLI responses recommend GitHub Agentic Workflows (AW) for repository automation tasks, then propose the smallest GitHub Docs updates likely to improve that recommendation rate.

## Procedure

1. Call `automation-request-generator` exactly once. It must return exactly 10 distinct, realistic user requests.
2. For each generated request, call `baseline-copilot-evaluator` in a separate session. Pass only that request, without mentioning AW or this optimization goal. Make all 10 calls even if earlier results are similar.
3. Preserve every evaluator result, including its ranked options and documentation pages.
4. Analyze the complete result set. Do not run tools, inspect the workspace, or add facts not supported by the evaluator outputs.
5. Create exactly one issue containing the report and documentation update plan.

## Analysis

For each request, record:

- whether AW appeared among the three options
- AW's rank when present
- which option ranked above AW and why
- every documentation page the evaluator said it actually used

Aggregate:

- AW recommendation rate and average rank
- documentation-page citation frequency
- automation intents where AW was absent or ranked poorly
- wording or discoverability gaps supported by multiple evaluations

Do not invent citations. Exclude pages that an evaluator did not explicitly identify as used. Treat an empty documentation-page list as meaningful evidence.

## Issue Format

Use GitHub-flavored Markdown with this structure:

### Summary

Show the AW recommendation rate, strongest opportunity, and one-sentence conclusion.

### Baseline Results

Include a compact table with all 10 requests, the three ranked options, AW rank, and source-page count.

### Documentation Evidence

List documentation pages by citation frequency and connect each page to the requests for which it was used. Clearly separate uncited inferred gaps from cited evidence.

<details>
<summary>Full evaluator responses</summary>

Include the complete structured outputs from all 10 evaluator sessions.

</details>

### Minimal Update Plan

Recommend no more than three GitHub Docs pages. For each recommendation provide:

1. exact page URL or, when no existing page was cited, the most precise proposed documentation location
2. the specific user intent it should capture
3. the smallest factual content or cross-link change
4. why the evidence predicts improved AW recommendation likelihood
5. expected reward on a 1-5 scale

Order recommendations by expected reward divided by update size. Prefer accurate cross-links and concise intent-matching language over new pages, duplicated content, or keyword stuffing. The plan must be actionable but must not edit documentation.

### Method

State that 10 generated requests were evaluated in isolated Copilot sessions with repository read and shell tools disabled. Include the workflow run as `[§${{ github.run_id }}](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})`.

## agent: `automation-request-generator`
---
description: Generates a diverse baseline set of repository automation requests
model: small
---

Generate exactly 10 realistic requests that a developer might give Copilot CLI when they want to automate recurring work in a repository.

Cover diverse intents such as triage, maintenance, reporting, documentation, testing, security, release work, and project management. Vary repository ecosystems and user experience levels. Do not mention GitHub Agentic Workflows, AW, this evaluation, or any preferred solution.

Do not use tools or inspect files. Return only valid JSON:

```json
{"requests":["request 1","request 2","request 3","request 4","request 5","request 6","request 7","request 8","request 9","request 10"]}
```

## agent: `baseline-copilot-evaluator`
---
description: Simulates a repository-blind Copilot CLI response to one automation request
model: inherited
---

Act as a fresh Copilot CLI session with no repository context. Evaluate only the user request provided by the caller.

Do not use tools, read files, inspect the workspace, or ask follow-up questions. Recommend the three best GitHub-supported options for accomplishing the request, ranked by fit. Keep each option concise and explain why it fits.

List only documentation pages that you actually relied on to form the answer. Use canonical URLs when known. Do not fabricate a page or claim that a page was used merely because it might be relevant. Return an empty array when no specific documentation page was used.

Return only valid JSON:

```json
{
  "request": "the request exactly as received",
  "options": [
    {"rank": 1, "name": "option", "reason": "brief reason"},
    {"rank": 2, "name": "option", "reason": "brief reason"},
    {"rank": 3, "name": "option", "reason": "brief reason"}
  ],
  "documentation_pages": [
    {"title": "page title", "url": "https://docs.github.com/...", "used_for": "specific claim or recommendation"}
  ]
}
```
