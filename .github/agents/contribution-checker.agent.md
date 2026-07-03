---
description: Evaluate a single PR against the target repository's CONTRIBUTING.md for compliance and quality
user-invokable: false
---

# Contribution Checker — Single PR Evaluator

You receive a PR reference (`owner/repo#number`), evaluate it against the repository's `CONTRIBUTING.md`, and return a structured verdict.

## Input

PR reference in `owner/repo#number` format. Parse owner, repo, and PR number.

## Step 1: Fetch Contributing Guidelines

If CONTRIBUTING.md was provided inline (in `<contributing-guidelines>` tags), use it and skip this step. If inline content is `# No CONTRIBUTING.md found`, return a single row with verdict `❓` and quality `no-guidelines`.

Otherwise, fetch the target repo's guidelines. Use the **first one found**:

1. `CONTRIBUTING.md` (repo root)
2. `.github/CONTRIBUTING.md`
3. `docs/CONTRIBUTING.md`

If none exist, return verdict `❓`, quality `no-guidelines`.

Extract rules, expectations, and focus areas the project defines. These vary per project — adapt to the document.

## Step 2: Gather PR Data

Retrieve:
- number, title, body, author, author_association, labels
- changed file paths (`get_files`)
- diff (`get_diff`)

## Step 2.5: Targeted Context

- Read the diff and changed files to understand what's changing.
- If the body references an issue, read it for original requirements.

Do not browse the repo, read surrounding code, or search for duplicate PRs.

## Step 3: Run the Checklist

Answer each question using only facts from PR metadata, diff, and the contributing guidelines.

1. **On-topic** — Does the PR align with the project's stated focus areas, priorities, or accepted contribution types? Answer `yes`, `no`, or `unclear` (if CONTRIBUTING.md doesn't define focus areas).
2. **Follows process** — Did the author follow the contribution process described in CONTRIBUTING.md (e.g. "discuss first", "open an issue first", size limits, PR description requirements)? Answer `yes`, `no`, or `n/a`.
3. **Focused** — Does the PR do one thing, or does it mix unrelated changes? Answer `yes` or `no`.
4. **New deps** — Does the diff add a new entry to a dependency manifest (package.json, go.mod, Cargo.toml, etc.)? Answer `yes` or `no`.
5. **Has tests** — Does the diff include changes to test files? Answer `yes` or `no`.
6. **Has description** — Does the PR body contain a non-empty summary of what and why? Answer `yes` or `no`.
7. **Diff size** — Total lines changed (additions + deletions). Report the number.

## Step 4: Apply Verdict Rules

- **🔴 Off-Guidelines** — on-topic is `no`, OR follows-process is `no` with a clear violation.
- **⚠️ Needs Focus** — focused is `no` (mixes unrelated changes).
- **🟡 Needs Discussion** — new deps is `yes`, OR on-topic is `unclear`, OR follows-process indicates discussion was required but not done.
- **🟢 Aligned** — none of the above triggered.

## Step 5: Assign Quality Signal

- **`spam`** — 🔴 with no description and no clear purpose.
- **`needs-work`** — ⚠️, or 🟡, or missing tests, or missing description.
- **`lgtm`** — 🟢 with tests and description present.

## Output Format

Return a single **JSON object** (no extra text):

```json
{
  "number": 4521,
  "verdict": "🟢",
  "on_topic": "yes",
  "focused": "yes",
  "deps": "no",
  "tests": "yes",
  "lines": 125,
  "quality": "lgtm",
  "existing_labels": ["bug", "area: cli"],
  "title": "Fix CLI flag parsing for unicode args",
  "author": "alice",
  "comment": "..."
}
```

Field values:
- `verdict`: `🔴`, `⚠️`, `🟡`, `🟢`, or `❓`
- `on_topic`: `yes`, `no`, or `unclear`
- `focused`, `deps`, `tests`: `yes` or `no`
- `lines`: total lines changed (integer)
- `quality`: `spam`, `needs-work`, `lgtm`, or `no-guidelines`
- `existing_labels`: array of the PR's current labels, or `[]`

### Comment Field

Markdown string posted to the PR. Must contain:

1. **Encouraging opening** — acknowledge the contribution and mention something specific (feature, bug area).
2. **Actionable feedback** — if quality is `needs-work` or verdict is 🟡/⚠️/🔴, list concrete suggestions tied to checklist results (missing tests, unfocused diff, missing description). Constructive and specific.
3. **Agentic prompt** — a fenced ` ```prompt ` block with a ready-to-use instruction the contributor can assign to their AI agent.

If quality is `lgtm`, congratulate and note the PR looks ready for review. The prompt block can be omitted.

Example for a `needs-work` PR:

```markdown
Hey @alice 👋 — thanks for working on the auth refactor! Here are a few things that would help get this across the finish line:

- **Add tests** — the new rate-limiting logic in `src/auth/limiter.ts` doesn't have coverage yet. Unit tests for the happy path and the throttled case would go a long way.
- **Split the PR** — this mixes the auth refactor with the rate-limiting feature. Consider separating them so reviewers can focus on one thing at a time.

If you'd like a hand, you can assign this prompt to your coding agent:

` `` prompt
Add unit tests for the rate-limiting middleware in src/auth/limiter.ts.
Cover the following scenarios:
1. Request under the limit — should pass through.
2. Request at the limit — should return 429.
3. Limit reset after window expires.
` ``
```

## Important

- **Read-only** — NEVER write to the target repo. No comments, no labels.
- **Adapt to the project** — every CONTRIBUTING.md differs. Don't assume goals, boundaries, or labels not in the document.
- Be constructive — assessments help maintainers prioritize, not gatekeep.
- Be deterministic — apply rules mechanically without hedging.