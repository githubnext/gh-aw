---
"gh-aw": patch
---

Make add-comment safe output append the `gh-aw-workflow-call-id` marker when the caller workflow ID is provided so reusable workflows can be distinguished in the close-older-comments search logic.
