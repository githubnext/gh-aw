---
private: true
name: Smoke Admission
description: Smoke-test the admission client against a service container and validate its random AIC allowance downstream with Pi on Copilot.
intent: Detect admission protocol or job-output regressions that prevent a valid positive AIC allowance from reaching the agent.
on:
  workflow_dispatch:
  slash_command:
    name: smoke-admission
    strategy: centralized
    events: [issues, issue_comment, pull_request, pull_request_comment]
permissions:
  contents: read
  copilot-requests: write
engine: pi
model: copilot/auto
timeout-minutes: 5
max-ai-credits: 100
features:
  gh-aw-detection: false
network:
  allowed: [defaults]
safe-outputs:
  staged: true
  threat-detection: false
  activation-comments: false
  report-failure-as-issue: false
  report-failed-jobs: false
  noop:
    report-as-issue: false
jobs:
  admission_probe:
    runs-on: ubuntu-latest
    needs: [pre_activation]
    permissions:
      contents: read
    timeout-minutes: 3
    services:
      admission:
        image: wiremock/wiremock:3.13.1
        ports:
          - 8080:8080
        options: >-
          --health-cmd "curl --fail --silent http://localhost:8080/__admin/health"
          --health-interval 5s
          --health-timeout 3s
          --health-retries 10
    outputs:
      result: ${{ steps.gate.outputs.result }}
      max-ai-credits: ${{ steps.gate.outputs.max-ai-credits }}
    steps:
      - uses: actions/checkout@v7.0.1
        with:
          persist-credentials: false
      - name: Configure random admission service
        uses: actions/github-script@v9.0.0
        with:
          script: |
            const response = await fetch("http://127.0.0.1:8080/__admin/mappings", {
              method: "POST",
              headers: { "content-type": "application/json" },
              body: JSON.stringify({
                request: {
                  method: "POST",
                  url: "/api/admission",
                  headers: {
                    "content-type": { equalTo: "application/json" },
                    authorization: { matches: "bearer synthetic\\.[A-Za-z0-9_-]+\\.signature" },
                    traceparent: { matches: "00-[0-9a-f]{32}-[0-9a-f]{16}-01" },
                  },
                },
                response: {
                  status: 200,
                  headers: { "content-type": "application/json", "cache-control": "no-store" },
                  body: '{"max_ai_credits": {{randomInt lower=101 upper=200}} }',
                  transformers: ["response-template"],
                },
              }),
            });
            if (response.status !== 201) throw new Error("Unable to configure smoke admission service");
      - name: Check fixture admission with the production client
        id: gate
        uses: actions/github-script@v9.0.0
        with:
          script: |
            const assert = require("node:assert/strict");
            const { checkAdmission } = require("./actions/setup/js/check_admission.cjs");
            const endpoint = "https://smoke-admission.invalid/api/admission";
            const issuer = "https://smoke-oidc.invalid/token";
            const token = `synthetic.${Buffer.from(JSON.stringify({
              exp: Math.floor(Date.now() / 1000) + 300,
            })).toString("base64url")}.signature`;
            let expected;
            let requests = 0;
            const decision = await checkAdmission({
              env: {
                ...process.env,
                GH_AW_DEFAULT_ADMISSION_URL: endpoint,
                GH_AW_DEFAULT_ADMISSION_AUDIENCE: endpoint,
                GH_AW_DEFAULT_ADMISSION_ON_OUTAGE: "block",
                GH_AW_INFO_WORKFLOW_ID: `${process.env.GITHUB_REPOSITORY}/.github/workflows/smoke-admission.lock.yml@${process.env.GITHUB_REF}`,
                GH_AW_ADMISSION_WORKFLOW_HASH: `${"a".repeat(64)}:${"b".repeat(64)}`,
                ACTIONS_ID_TOKEN_REQUEST_URL: issuer,
                ACTIONS_ID_TOKEN_REQUEST_TOKEN: "synthetic-bootstrap",
              },
              fetchImpl: async (url, options) => {
                if (new URL(url).origin === new URL(issuer).origin) {
                  assert.equal(new URL(url).searchParams.get("audience"), endpoint);
                  return Response.json({ value: token });
                }
                assert.equal(url, endpoint);
                requests++;
                // Only the fixture transport uses HTTP; production HTTPS validation is unchanged.
                const response = await fetch("http://127.0.0.1:8080/api/admission", options);
                if (response.status === 200) expected = (await response.clone().json()).max_ai_credits;
                return response;
              },
            });
            assert.equal(requests, 1);
            assert.equal(decision.result, "admitted");
            assert.equal(decision.maxAiCredits, expected);
            assert.ok(Number.isSafeInteger(expected) && expected >= 101 && expected <= 200);
            core.setOutput("result", decision.result);
            core.setOutput("max-ai-credits", String(expected));
  activation:
    needs: [admission_probe]
    if: needs.admission_probe.outputs.result == 'admitted'
  agent:
    needs: [admission_probe]
steps:
  - name: Validate downstream admission outputs
    uses: actions/github-script@v9.0.0
    env:
      SMOKE_ADMISSION_RESULT: ${{ needs.admission_probe.outputs.result }}
      SMOKE_ADMISSION_AIC: ${{ needs.admission_probe.outputs.max-ai-credits }}
    with:
      script: |
        const assert = require("node:assert/strict");
        const fs = require("node:fs");
        const result = process.env.SMOKE_ADMISSION_RESULT;
        const allowance = process.env.SMOKE_ADMISSION_AIC;
        assert.equal(result, "admitted");
        assert.match(allowance, /^[1-9][0-9]*$/);
        const maxAiCredits = Number(allowance);
        assert.ok(Number.isSafeInteger(maxAiCredits) && maxAiCredits >= 101 && maxAiCredits <= 200);
        fs.mkdirSync("/tmp/gh-aw/agent", { recursive: true });
        fs.writeFileSync("/tmp/gh-aw/agent/smoke-admission.json", JSON.stringify({ result, maxAiCredits }));
post-steps:
  - name: Verify agent admission evidence
    uses: actions/github-script@v9.0.0
    env:
      SMOKE_ADMISSION_AIC: ${{ needs.admission_probe.outputs.max-ai-credits }}
    with:
      script: |
        const assert = require("node:assert/strict");
        const fs = require("node:fs");
        const evidence = JSON.parse(fs.readFileSync("/tmp/gh-aw/agent/smoke-admission-verified.json", "utf8"));
        assert.deepEqual(evidence, {
          result: "admitted",
          maxAiCredits: Number(process.env.SMOKE_ADMISSION_AIC),
          status: "PASS",
        });
---

# Smoke Admission

Read `/tmp/gh-aw/agent/smoke-admission.json`. Verify that `result` is exactly
`admitted` and `maxAiCredits` is an integer between 101 and 200 inclusive.
Write `/tmp/gh-aw/agent/smoke-admission-verified.json` with the same `result`
and `maxAiCredits` plus `"status":"PASS"`. Never substitute a fixed allowance.
If validation fails, stop and report the failure rather than writing PASS.
On success, call `noop` with a concise admission smoke-test success message.

This isolated test gate runs the production admission client against the
`admission_probe` job's service container using synthetic OIDC credentials and
a test-only HTTP transport. It gates activation and checks exact allowance
propagation to the agent; it does not test OIDC signature verification, TLS,
or AWF budget enforcement. The compiler-owned admission job still honors
managed repository/organization variables independently; this workflow never
changes them or replaces its production admission budget.
