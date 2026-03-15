---
# Daily Audit Discussion — shared safe-output base configuration
#
# Usage:
#   imports:
#     - shared/daily-audit-discussion.md
#
# This component provides the standard safe-output configuration used by daily/weekly
# audit and reporting workflows that publish a discussion in the "audits" category:
#   - create-discussion: category "audits", max 1, close-older-discussions
#   - close-discussion: max 10
#
# Each importing workflow keeps only its workflow-specific settings inline.
# Because create-discussion uses field-level merging, any field that is NOT set
# in the workflow's own safe-outputs block is automatically filled from this component.
# Fields typically kept per-workflow: title-prefix and expires.
#
# Example — workflow-specific block after importing:
#   safe-outputs:
#     create-discussion:
#       title-prefix: "[my-workflow] "
#       expires: 3d

safe-outputs:
  create-discussion:
    category: "audits"
    max: 1
    close-older-discussions: true
  close-discussion:
    max: 10
---

## Audit Discussion Output

Create a new discussion in the `audits` category when the analysis is complete.
Previous discussions for the same workflow are automatically closed.
Charts and artifacts can be uploaded as assets and referenced in the discussion body.

### Discussion Format

Follow `shared/reporting.md` guidelines:
- Use `###` (h3) for all section headers (never `#` or `##`)
- Wrap verbose details in `<details><summary>` tags
- Include a summary table at the top with key metrics
- Reference the workflow run: `[${{ github.run_id }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})`
