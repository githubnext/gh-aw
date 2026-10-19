---
on:
  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      types: [edited]
      status-transitions:
        - {from: Todo, to: Ready}
    security-alerts:
      sources: [code-scanning, dependabot]
      types: [created]
    jira-issue:
      project: APP
      types: [opened, closed]
    linear-issue:
      organization-id: organization-uuid
      team-id: team-uuid
      types: [opened, closed]
    azure-devops-issue:
      project_id: project-uuid
      types: [closed, reopened]
      closed-states: [Done, Closed]
    webhooks:
      - type: com.example.release
        filters:
          data:
            state: ready
permissions:
  contents: read
engine: copilot
---

# Gateway resource events

Process ${{ toJSON(gateway.events) }} using authenticated provider clients.
