---
name: Example Skip Roles Workflow
description: This workflow demonstrates skip-roles feature for AI moderation
on:
  issues:
    types: [opened]
  skip-roles: [admin, maintainer, write]
engine: copilot
---

# Moderate External Contributions

This workflow checks external contributions (from users without admin, maintainer, or write access) for compliance with community guidelines.

Internal team members with elevated permissions are skipped.
