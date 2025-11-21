---
"gh-aw": patch
---

Add shared safe output app configuration workflow

Added a new shared workflow `shared/safe-output-app.md` that configures GitHub App authentication for safe outputs using repository-level variables (`APP_ID`) and secrets (`APP_PRIVATE_KEY`).

This complements the existing `shared/app-config.md` which uses organization-level variables (`ORG_APP_ID` and `ORG_APP_PRIVATE_KEY`).

The changeset generator workflow now imports this shared configuration, enabling GitHub App token-based authentication for the `push-to-pull-request-branch` safe output.

**Technical improvements:**
- Enhanced `push_to_pull_request_branch` job to support GitHub App tokens
- Added automatic app token minting step when app configuration is detected
- Created `generateGitConfigurationStepsWithToken` helper for flexible Git authentication
- App tokens are automatically used in git operations and GitHub CLI commands when configured
