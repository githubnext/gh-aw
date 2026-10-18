---
private: true
emoji: "🔒"
name: The Great Escapi
description: Bounded AWF network policy and filesystem regression checks

on:
  workflow_dispatch:
  schedule: daily
  pull_request:
    types: [labeled]
    names: ['firewall-escape-test']

permissions:
  contents: read
  copilot-requests: write
strict: true

engine:
  id: copilot
  copilot-sdk: true

max-tool-denials: 3
timeout-minutes: 15
tracker-id: firewall-escape

network:
  allowed:
    - defaults
    - node
    - github.com
    - api.github.com

sandbox:
  agent:
    id: awf

safe-outputs:
  create-discussion:
    expires: 1d
    category: "audits"
    title-prefix: "[Firewall Escape] "
    max: 1

imports:
  - shared/otlp.md
  - shared/reporting.md
tools:
  edit: false
  bash: ["node scripts/firewall-policy-check.cjs:*"]

steps:
  - name: Stage trusted firewall verifier before agent execution
    run: install -m 0444 scripts/firewall-policy-check.cjs "${RUNNER_TEMP}/gh-aw/firewall-policy-check.cjs"

post-steps:
  - name: Verify bounded firewall check evidence
    if: always()
    env:
      FIREWALL_REPORT: /tmp/gh-aw/agent/firewall-policy-${{ github.run_id }}.json
    run: node "${RUNNER_TEMP}/gh-aw/firewall-policy-check.cjs" --verify "$FIREWALL_REPORT"

jobs:
  post-issue:
    needs: agent
    if: failure()
    runs-on: ubuntu-latest
    permissions:
      issues: write
    steps:
      - name: Create issue on test failure
        uses: actions/github-script@v9.0.0
        with:
          script: |
            await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: '[Firewall Test] Validation failed or incomplete',
              body: `## Firewall Policy Validation

            **Workflow Run:** ${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}

            The bounded checks failed or could not complete. Review the agent logs and deterministic check evidence before classifying the failure.

            A content-filtered model response, unavailable tool, DNS failure, timeout, or missing report is inconclusive, not evidence of a firewall escape or a successful test.

            Only an observed response from the non-allowed test domain establishes unexpected network access. Passing these bounded checks does not prove the sandbox is secure.`,
              labels: ['bug', 'firewall', 'automated']
            });
---

# Bounded Firewall Policy Validation

Run the repository's fixed regression checks once, inside the existing AWF
sandbox, with this exact command:

```bash
node scripts/firewall-policy-check.cjs /tmp/gh-aw/agent/firewall-policy-${{ github.run_id }}.json
```

The script checks API and web access to allowed GitHub domains, a single ordinary
HTTPS request to the non-allowed `example.com` domain, GitHub DNS resolution,
home-directory reads, home and agent-directory write/read/cleanup, and a local
HTTP server round trip. It does not change proxy settings or firewall policy.
It discards remote response bodies and emits only check metadata.

This is a bounded regression test, not an open-ended security assessment. Do not
invent additional probes, attempt bypasses, change the script or its report,
research attack techniques, or read prior attempt history.

Create one discussion summarizing the script's eight check results and its
overall `PASS`, `FAIL`, or `INCONCLUSIVE` outcome. Include this run link:
${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}.
Report only observed results; the post-agent verifier rejects missing, incomplete,
or non-passing evidence regardless of the discussion text.
The host-side verifier uses a pre-agent copy in the runtime directory mounted
read-only inside AWF, never executable source from the post-agent checkout.

A proxy CONNECT rejection is the expected negative-control result. DNS errors,
timeouts, TLS failures, unavailable commands, or model/tool failures are not proof
of a policy denial. If the command cannot run or its report cannot be read, report
`INCONCLUSIVE`. Do not retry a content-filtered response or claim unexecuted
checks passed. Even `PASS` means only that these eight assertions passed; never
claim that the sandbox is secure or that all possible escape paths were tested.
