---
"gh-aw": minor
---

Migrate terminology from "agent task" to "agent session".

This change updates the CLI, JSON schemas, codemods, docs, and tests to use
the new "agent session" terminology. A codemod (`gh aw fix`) is included to
automatically migrate workflows; the old `create-agent-task` key remains
supported with a deprecation warning to preserve backward compatibility.

## Codemod

If your workflows use the old `create-agent-task` frontmatter key, update them:

Before:

```yaml
create-agent-task: true
```

After:

```yaml
create-agent-session: true
```

Run `gh aw fix --write` to apply automatic updates across your repository.

