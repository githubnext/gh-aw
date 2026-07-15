---
private: true
emoji: 🎓
name: Daily Exercise Creator
description: Generates a daily GitHub Skills exercise outline for a new gh-aw topic, following the exercise-creator guidelines from https://github.com/skills/exercise-creator
on:
  schedule: daily around 09:00 on weekdays
  workflow_dispatch:
max-daily-ai-credits: 10000
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-exercise-creator
strict: true
timeout-minutes: 30
network:
  allowed:
    - defaults
    - github
tools:
  github:
    mode: gh-proxy
    toolsets:
      - default
safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    title-prefix: "[exercise] "
    labels: [exercise, skills]
    expires: 90
  noop:
---

# Daily Exercise Creator

You create a daily GitHub Skills exercise outline for a new **gh-aw** (GitHub Agentic Workflows) topic,
following the guidelines from https://github.com/skills/exercise-creator.

## Step 1 — Check Existing Outlines

Search for exercise outlines already created to avoid repeating topics:

```bash
gh issue list --repo "$GITHUB_REPOSITORY" --label exercise --label skills --state all \
  --json number,title --limit 100
```

Record the topic covered in each issue title. Skip any topic that is already represented.

## Step 2 — Pick a New gh-aw Topic

Choose the **first topic from the list below that has not yet been covered**.
If all topics are covered, invent a new unique gh-aw topic not on the list.

### Suggested topic bank (in priority order)

1. Create your first agentic workflow
2. Scheduling workflows with cron triggers
3. Using safe-outputs for GitHub writes
4. Connecting MCP tools to workflows
5. Writing effective agent prompts
6. Using sub-agents to split complex tasks
7. Debugging workflow runs with `gh aw logs` and `gh aw audit`
8. A/B testing workflow variants with experiments
9. Keeping agent state with repo-memory
10. Configuring network access and firewall rules
11. Applying permissions and the read-only agent pattern
12. Using `noop` to signal explicit no-ops
13. Creating reusable shared workflow components
14. Generating pull requests from an agentic workflow
15. Triggering workflows on pull request events
16. Using `workflow_dispatch` for on-demand runs
17. Integrating Playwright for browser automation
18. Writing report workflows that post daily issues
19. Using `skip-if-match` to prevent duplicate runs
20. Compiling and validating workflows with `gh aw compile`

## Step 3 — Gather Context for the Chosen Topic

Use the GitHub tools to gather context from the gh-aw documentation and repository:

```bash
# Find relevant docs files
gh api "repos/$GITHUB_REPOSITORY/git/trees/HEAD?recursive=1" \
  --jq '.tree[] | select(.path | startswith("docs/")) | .path' | head -50
```

Read 1-3 relevant documentation files to ground the outline in accurate, specific content.

## Step 4 — Create the Exercise Outline

Using the selected topic and the context gathered above, write a complete exercise outline
that follows the **exercise-creator template** defined below.

### Template structure

```
# Logistics
- **Exercise Title:** <title>
- **Repo URL:** https://github.com/skills/<kebab-name> (tentative)
- **Experience Level:** Beginner / Intermediate / Advanced
- **Recommended Grouping:** GitHub Automation / Developer Workflows / AI & Copilot

### Relationships to other exercises
- **Previous Exercise:** (optional: name of prerequisite exercise)
- **Next Exercise:** (optional: name of follow-on exercise)

---

# Outline

## README

**Title:** <human-friendly title>

<Two-sentence introduction to the exercise.>

### Overview
1. <Learning objective 1>
2. <Short description of step 1>
3. ...

### What you will build
<Three-sentence description of the practical outcome.>

### Prerequisites
- <Prior knowledge or tool required>

## Step 1 — <Step Title>

### Theory
<Short description of the concept being taught.>
- <Key background concept>
- <Key concept to teach>

### References
- <Link to official GitHub/gh-aw documentation>

### Activity: <Activity Title>
1. <Actionable instruction>
2. ...

### Transition
- **Actions Trigger:** `<GitHub Actions event>`
- **Grading-Check:** <What to verify in the learner's repo>

## Step 2 — <Step Title>
...

## Step 3 — <Step Title>
...

## Review
<Two-sentence summary of the exercise.>
- <Skill learned>
- <Skill learned>

### What's next?
- <Link to gh-aw docs>
- <Link to a related GitHub Skills exercise>

# Future Considerations
- <Idea for a follow-on exercise or upgrade>
```

### Outline quality rules (from exercise-creator guidelines)

- **Theory** must cite real gh-aw documentation — no generic descriptions.
- **References** must be official sources: https://docs.github.com, https://github.com/github/gh-aw docs, or https://github.blog.
- **Activities** must have action-oriented titles and numbered steps in present-tense imperative form.
- **Transitions**: pick the smallest GitHub Actions trigger that fires when the learner completes the activity.
- **Grading-Check**: describe what a workflow will verify (e.g., file exists, keyphrase present, PR opened).
- Aim for 3 steps for beginner exercises; up to 5 for intermediate/advanced.
- Do not teach unrelated concepts — scope tightly to the chosen topic.

## Step 5 — Create the Issue

Write the issue body using the outline you composed.

### Issue title format

```
<Exercise Title> — <Experience Level> exercise outline
```

Example: `Create your first agentic workflow — Beginner exercise outline`

### Report formatting rules

- Use `###` (h3) or lower for all headers — never `##` or `#`.
- Wrap any extra-long sections in `<details><summary>...</summary>` blocks.
- Keep the top-level outline visible (do not collapse the main sections).

After creating the issue, call `noop` only if you could not find a new topic and could not invent one.
Always call `create_issue` when a valid new outline was produced.
