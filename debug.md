# Debugging Agentic Workflows

This legacy URL remains available for existing links. The instructions live in
[the debugging bootstrap](.github/aw/debug-bootstrap.md).

Before setup, commands, or edits, load and follow that file in full. Resolve it
from the same repository/ref as this file; for GitHub `/blob/` URLs, use the
raw-content equivalent at the same ref without switching to `main`. In a local checkout, use
`.github/aw/debug-bootstrap.md`.

```text
Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md

Run URL: https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Keep the supplied run/job/step URL intact. The canonical bootstrap owns setup,
reference discovery, and delegation to the shared debugging strategy; this
router adds no execution authorization.
