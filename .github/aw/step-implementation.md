---
description: Guidance for implementing custom steps in agentic workflows, covering preferred technologies and security rules.
---

# Step Implementation Guidance

Use this guidance when writing `steps:`, `pre-steps:`, `post-steps:`, and safe-output job `steps:` in agentic workflows.

## Priority Order

Choose the implementation in this order:

1. **`actions/github-script`** (preferred)
2. **Shell script** (`run:`)
3. **Python** (last resort, data science only)

---

## 1. Prefer `actions/github-script`

Use `actions/github-script` for steps that call the GitHub API, process structured data, or require reliable JavaScript logic.

- Use a recent major version such as `v9` — do **not** pin to a full SHA in the workflow markdown. The workflow compiler pins actions to verified SHAs automatically via `actions-lock.json`.
- JavaScript runs in Node.js: standard library (`fs`, `path`, `child_process`), the pre-authenticated `github` Octokit client, and `core`/`exec` helpers from `@actions/core` and `@actions/exec` are available.
- Pass untrusted values through `process.env` (set with `env:`) instead of interpolating `${{ ... }}` expressions directly into the `script:` block.

```yaml
steps:
  - name: Fetch open issues
    uses: actions/github-script@v9
    env:
      LABEL: ${{ github.event.label.name }}
    with:
      script: |
        const label = process.env.LABEL;
        const { data: issues } = await github.rest.issues.listForRepo({
          owner: context.repo.owner,
          repo: context.repo.repo,
          labels: label,
          state: 'open',
        });
        const fs = require('fs');
        fs.writeFileSync('/tmp/gh-aw/agent/issues.json', JSON.stringify(issues));
```

---

## 2. Shell Script (`run:`)

Use shell when the step is a straightforward sequence of CLI commands (`gh`, `jq`, `curl`, standard Unix utilities) and JavaScript would add unnecessary complexity.

### Shell injection rules

Shell injection occurs when untrusted content from GitHub context expressions is interpolated into a `run:` command. Follow these rules:

- **Never interpolate `${{ ... }}` expressions directly into `run:` scripts.** Instead, assign them to environment variables with `env:` and reference `$VAR_NAME` in the script.
- Treat all `github.event.*` values, user-supplied inputs, issue/PR titles, and comment bodies as untrusted.

**Unsafe — do not do this:**

```yaml
steps:
  - name: Comment
    run: gh issue comment ${{ github.event.issue.number }} --body "${{ github.event.issue.title }}"
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Safe — pass untrusted values through environment variables:**

```yaml
steps:
  - name: Comment
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      ISSUE_NUMBER: ${{ github.event.issue.number }}
      ISSUE_TITLE: ${{ github.event.issue.title }}
    run: gh issue comment "$ISSUE_NUMBER" --body "$ISSUE_TITLE"
```

Additional shell rules:

- Quote all variable expansions: `"$VAR"`, not `$VAR`.
- Use `--` or named flags to separate options from arguments when passing values to CLI tools.
- Set `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` on every step that calls `gh`.
- Write prepared data to `/tmp/gh-aw/agent/` and trim large outputs before handing them to the agent.
- Use `jq` to reduce JSON payload size.

---

## 3. Python (last resort, data science only)

Use Python only when the task genuinely requires scientific computing, statistical analysis, or data-science libraries (`pandas`, `numpy`, `matplotlib`, etc.) that have no shell or JavaScript equivalent.

- Prefer the system Python or a pinned runtime via `runtimes:` rather than installing Python in the step itself.
- Install dependencies via `pip install` only when necessary; pin versions.
- Apply the same injection rules as shell: pass `${{ ... }}` values through environment variables (`os.environ`), never via f-strings or string concatenation into subcommands.

```yaml
steps:
  - name: Analyse metrics
    env:
      DATA_PATH: /tmp/gh-aw/agent/metrics.csv
    run: |
      python3 - <<'EOF'
      import os, json, csv, statistics
      path = os.environ["DATA_PATH"]
      with open(path) as f:
          rows = list(csv.DictReader(f))
      values = [float(r["latency_ms"]) for r in rows]
      result = {"p50": statistics.median(values), "mean": statistics.mean(values)}
      with open("/tmp/gh-aw/agent/summary.json", "w") as out:
          json.dump(result, out)
      EOF
```

---

## Summary Table

| Need | Use |
|---|---|
| GitHub API calls, structured JSON processing | `actions/github-script@v9` |
| CLI commands (`gh`, `jq`, `curl`, Unix tools) | Shell `run:` |
| Scientific computing, data-science libraries | Python (last resort) |
