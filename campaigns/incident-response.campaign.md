---
id: incident-response
version: "v1"
name: "Incident Response Campaign"
description: "Multi-team incident coordination with command center, SLA tracking, and post-mortem."

workflows:
  - incident-response

memory-paths:
  - "memory/campaigns/incident-*/**"

owners:
  - "oncall-incident-commander"
  - "sre-team"

executive-sponsors:
  - "vp-engineering"

risk-level: "high"
state: "planned"
tags:
  - "incident"
  - "operations"

tracker-label: "campaign:incident-response"

allowed-safe-outputs:
  - "create-issue"
  - "add-comment"
  - "create-pull-request"

approval-policy:
  required-approvals: 1
  required-roles:
    - "incident-commander"
  change-control: false
---

# Incident Response Campaign

Describe this campaign's goals, incident command structure, SLA targets,
and communication cadence. Include how AI should assist (hypothesis
generation, risk-tiered recommendations) and where humans must stay in
the loop for approvals.
