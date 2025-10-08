---
"gh-aw": minor
---

Add comment creation for issue/PR reactions with workflow run links

When an agentic workflow is triggered by an issue or pull request and has the "reaction" frontmatter enabled, the `add_reaction` job now creates a comment pointing to the workflow run (in addition to adding the reaction). The comment ID and URL are also exposed as outputs of the job, making them available to downstream jobs.
