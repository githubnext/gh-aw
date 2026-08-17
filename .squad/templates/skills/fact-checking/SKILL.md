---
name: fact-checking
description: Review and validate claims using counter-hypothesis testing. Use when verifying technical content, checking references, validating API endpoints, or performing quality assurance on deliverables.
license: MIT
compatibility: Requires access to external references and APIs
metadata:
  author: squad
  version: "1.0.0"
  domain: quality, verification
  confidence: low
  last_validated: "2026-03-01"
---

# Skill: Fact Checking

## Context
Codifies the challenger agent review output format and methodology so any agent performing fact-checking or review produces consistent, structured output.

## Pattern

### Review Methodology

For every claim or deliverable under review:
1. Ask: "What evidence supports this? What would disprove it?"
2. Generate counter-hypotheses and test them against available data
3. Verify URLs, package names, API endpoints, and external references actually exist
4. Flag confidence levels: ✅ Verified, ⚠️ Unverified, ❌ Contradicted

### Review Output Format

When reviewing another agent's work, use this template:

```
### Fact Check — {deliverable name}
**Claims verified:** {count}
**Issues found:** {count}

| # | Claim | Status | Evidence/Notes |
|---|-------|--------|---------------|
| 1 | {claim} | ✅/⚠️/❌ | {supporting or contradicting evidence} |

**Counter-hypotheses tested:**
- {alternative explanation + result}

**Verdict:** {PASS / PASS WITH NOTES / NEEDS REVISION}
```

### Confidence Levels

- ✅ **Verified** — evidence confirms the claim
- ⚠️ **Unverified** — cannot confirm or deny; suggest verification method
- ❌ **Contradicted** — evidence disproves the claim

### Ceremony Integration

Auto-trigger this skill before any architecture decision, or when an agent claim contains superlatives or percentage thresholds (e.g., "saves 75%", "always", "never"). The coordinator spawns the challenger agent with:

```
Challenger — fact-check {agent}'s claim: "{claim}"
Cite evidence for every verdict. Max 3 investigation cycles.
```