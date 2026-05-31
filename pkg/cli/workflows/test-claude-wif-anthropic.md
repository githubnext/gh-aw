---
on:
  workflow_dispatch:

permissions:
  contents: read
  id-token: write

engine:
  id: claude
  auth:
    type: github-oidc
    provider: anthropic
    federation-rule-id: fdrl_test
    organization-id: org_test
    service-account-id: svac_test
    workspace-id: ws_test

network: defaults

timeout-minutes: 5
---

# Anthropic WIF schema test

Echo "ok".
