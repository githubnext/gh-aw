---
"gh-aw": patch
---

Rename check-membership job to check_membership with constant

Refactored the check-membership job name to use underscores (check_membership) for consistency with Go naming conventions. Introduced CheckMembershipJobName constant in constants.go to centralize the job name and eliminate hardcoded strings throughout the codebase. Updated all references including step IDs, job dependencies, step outputs, tests, and recompiled all workflow files.
