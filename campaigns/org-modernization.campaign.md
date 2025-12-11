---
id: org-modernization
version: "v1"
name: "Org-wide Modernization Campaign"
description: "Cross-repo modernization with human-in-loop approvals and intelligence reporting."

workflows:
  - org-wide-rollout        # rollout / coordinator
  - human-ai-collaboration  # decision / approval pattern
  - intelligence            # reporting / trend analysis

memory_paths:
  - "memory/campaigns/org-modernization-*/**"

owners:
  - "platform-team"
  - "devx-team"

executive_sponsors:
  - "vp-engineering"
  - "cto"

risk_level: "medium"
state: "planned"
tags:
  - "modernization"
  - "org-wide"
  - "rollout"

tracker_label: "campaign:org-modernization"

metrics_glob: "memory/campaigns/org-modernization-*/metrics/*.json"

allowed_safe_outputs:
  - "create-issue"
  - "add-comment"
  - "create-pull-request"

approval_policy:
  required_approvals: 2
  required_roles:
    - "platform-lead"
    - "team-lead"
  change_control: true
---

# Org-wide Modernization Campaign

Describe the modernization target state, rollout waves, dependency
constraints, and approval model. Clarify how agents should coordinate
across many repositories, when to pause or roll back, and how
intelligence reporting should summarize progress and risk.
