---
name: resolve-merge-conflicts
description: Merge a base ref and safely regenerate compiled workflow lock-file conflicts.
tools:
  bash:
    - "./.github/skills/resolve-merge-conflicts/resolve.sh *"
    - "git diff *"
    - "git status *"
---

# Resolve Merge Conflicts

Use this skill when merging `origin/main` into a branch, especially when the
only conflicts are generated `.github/workflows/*.lock.yml` files.

## One-step path

From the repository root, run:

```bash
./.github/skills/resolve-merge-conflicts/resolve.sh origin/main
```

The command works both before a merge and after another command has stopped on
conflicts. It:

1. Starts the merge with `--no-commit`, or resumes the current merge.
2. Refuses to auto-resolve if any conflict is not a workflow `.lock.yml`.
3. Runs `make recompile` once so generated files come from the merged Markdown.
4. Stages the regenerated conflicting lock files.
5. Verifies that no unresolved paths or whitespace errors remain.

The script does not fetch, commit, push, abort, or edit workflow Markdown.
Refresh `origin/main` first only when credentials are available. After success,
review the staged merge, run the repository's final validation gate, then
commit and push.

## Safety rules

- Never choose `ours` or `theirs` for compiled lock files; regenerate them.
- Never manually remove conflict markers from `.lock.yml` files.
- Never auto-resolve a source `.md`, Go, JavaScript, or other mixed conflict.
- If the script refuses a mixed conflict, resolve source conflicts on their
  merits, stage them, and rerun the same command. It will regenerate the
  remaining lock conflicts.
- Do not abort an existing merge unless the user explicitly requests it.

