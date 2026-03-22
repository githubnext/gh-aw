---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  dispatch_repository:
    trigger_ci:
      description: Trigger CI pipeline in a target repository
      workflow: ci.yml
      event_type: ci_trigger
      repository: org/target-repo
      inputs:
        environment:
          type: choice
          options:
            - staging
            - production
          default: staging
          description: Target deployment environment
      max: 1
    notify_service:
      description: Notify external service workflow
      workflow: notify.yml
      event_type: notify_event
      allowed_repositories:
        - org/service-repo
        - org/backup-repo
      inputs:
        message:
          type: string
          description: Notification message
      max: 2
---

# Test Copilot Dispatch Repository

Test the `dispatch_repository` safe output type with multiple tools using the Copilot engine.

## Task

Dispatch the `trigger_ci` tool to trigger CI in the target repository with `environment: staging`.

Optionally, call `notify_service` with a status message after the dispatch.
