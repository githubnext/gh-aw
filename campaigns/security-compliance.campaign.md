---
id: security-compliance
version: "v1"
name: "Security Compliance Campaign"
description: "Security remediation with compliance audit trail and executive reporting."

workflows:
  - security-compliance

memory-paths:
  - "memory/campaigns/security-compliance-*/**"

owners:
  - "security-team"
  - "platform-engineering"

executive-sponsors:
  - "ciso"
  - "vp-engineering"

risk-level: "high"
state: "planned"
tags:
  - "security"
  - "compliance"
  - "audit"

tracker-label: "campaign:security-compliance"

# Metrics snapshots for this campaign are expected under the
# memory/campaigns branch, following the structure described in the
# campaigns guide (metrics/YYYY-MM-DD.json).
metrics-glob: "memory/campaigns/security-compliance-*/metrics/*.json"

allowed-safe-outputs:
  - "create-issue"
  - "add-comment"
  - "create-pull-request"

approval-policy:
  required-approvals: 2
  required-roles:
    - "security-lead"
    - "engineering-lead"
  change-control: true
---

# Security Compliance Campaign

Describe the compliance frameworks in scope (e.g., SOC2, GDPR, HIPAA),
remediation priorities, and approval expectations. Document how agents
should generate evidence, maintain an audit trail in repo-memory, and
escalate high-risk changes for human review.
