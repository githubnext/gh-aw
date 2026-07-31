**Structured Variant Body Template**:
```markdown
**Total PRs** (`agent_prs_total`): [count] | **Merged** (`agent_prs_merged`): [count] ([percentage]%) | **Avg Duration**: [time]

**Performance Metrics**

| Date | PRs | Merged | Success Rate | Avg Duration | Avg Comments |
|------|-----|--------|--------------|--------------|--------------|
| [today] | [count] | [count] | [%] | [time] | [count] |
| [today-1] | [count] | [count] | [%] | [time] | [count] |
| [today-2] | [count] | [count] | [%] | [time] | [count] |

**Trend**: [Only mention if significant change >10%]

<details>
<summary><b>Agent Task Texts</b></summary>

| PR # | Status | Task Text (first 100 chars) |
|------|--------|----------------------------|
| [#number]([url]) | [Merged/Closed/Open] | [PR body truncated to 100 chars, or "No description provided"] |

</details>

<details>
<summary><b>Notable PRs</b></summary>

[Only if failures, closures, or PRs open >24h — otherwise omit]

</details>

**Key Insights**

[1-2 bullets max — omit if nothing notable]
```
