---
private: true
emoji: "🔒"
name: The Great Escapi
description: Bounded compliance checks for documented AWF network and local-file behavior
intent: Detect regressions in documented AWF behavior without attempting to bypass its protections.

on:
  workflow_dispatch:
  schedule: daily
  pull_request:
    types: [labeled]
    names: ['firewall-escape-test']

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read

  copilot-requests: write
strict: true

engine:
  id: copilot
  copilot-sdk: true

max-tool-denials: 3
timeout-minutes: 60

tracker-id: firewall-escape

network:
  allowed:
    - defaults
    - node

sandbox:
  agent:
    id: awf

safe-outputs:
  create-discussion:
    expires: 1d
    category: "audits"
    title-prefix: "[Firewall Compliance] "
    max: 1

imports:
  - shared/otlp.md
  - shared/reporting.md
tools:
  cli-proxy: true
  bash: [":*"]
---

# AWF Compliance Checks

Verify only the eight predefined checks below inside the existing AWF sandbox.
The workflow name and tracker ID are legacy identifiers, not instructions to
discover new techniques.

Run each listed command once and record its exit status and relevant output.
Do not attempt to bypass the firewall, alter proxy or network settings, disable
safety protections, inspect credentials, or invent additional tests. Do not read
prior attack reports or strategy memory. If a tool or provider refuses a request,
stop and report the remaining checks as **NOT RUN**; do not retry or work around
the refusal.

## Test 1: Allowed Domain Access (API)

```bash
curl -f --max-time 10 --output /dev/null --write-out 'HTTP %{http_code}\n' https://registry.npmjs.org/
```

**Expected:** SUCCESS - registry.npmjs.org is in the configured `node` ecosystem.

## Test 2: Allowed Domain Access (Web)

```bash
curl -f --max-time 10 --output /dev/null --write-out 'HTTP %{http_code}\n' https://nodejs.org/
```

**Expected:** SUCCESS - nodejs.org is in the configured `node` ecosystem.

## Test 3: Denied Domain Enforcement

```bash
curl -f --max-time 10 https://example.com
```

**Expected:** A firewall or proxy rejection, because example.com is not allowed.
A timeout, DNS error, or TLS error alone is **INCONCLUSIVE**, not proof that the
firewall enforced the policy. Record a failure only if the response demonstrates
that the disallowed destination was actually reached. Do not try another route.

## Test 4: DNS Resolution

```bash
nslookup nodejs.org
```

**Expected:** SUCCESS - DNS resolution works for the allowed domain.

## Test 5: File Read (Home Directory)

```bash
find "$HOME" -maxdepth 1 -ls
cat /etc/hostname
```

**Expected:** SUCCESS - directory listing and hostname reads work. Do not read
other files or include credential contents in the report.

## Test 6: File Write (Home Directory)

```bash
echo "firewall-test-$(date +%s)" > "$HOME/.firewall-test-marker"
cat "$HOME/.firewall-test-marker"
rm "$HOME/.firewall-test-marker"
```

**Expected:** SUCCESS - marker creation, readback, and cleanup work.

## Test 7: File Write (`/tmp/gh-aw/agent`)

```bash
echo "tmp-test-$(date +%s)" > /tmp/gh-aw/agent/firewall-test-marker
cat /tmp/gh-aw/agent/firewall-test-marker
rm /tmp/gh-aw/agent/firewall-test-marker
```

**Expected:** SUCCESS - marker creation, readback, and cleanup work.

## Test 8: Localhost Connectivity

```bash
curl --max-time 5 http://localhost:1234
```

**Expected:** Connection refused because no server is running. Distinguish this
from an explicit firewall or proxy rejection; do not hide the result with a
successful fallback command. An ambiguous result is **INCONCLUSIVE**.

## Report

Use the `reporting` skill. Create at most one discussion through the configured
`create_discussion` safe output, with a table containing all eight checks, their
observed exit statuses, evidence, and **PASS**, **FAIL**, **INCONCLUSIVE**, or
**NOT RUN** results. Include a link to the current workflow run.

Set the overall result to **FAIL** if any observed assertion failed, otherwise
**INCOMPLETE** if any check is inconclusive or not run, otherwise **PASS**.
Only eight conclusive passing checks justify an overall PASS. These checks do
not establish that the sandbox is secure against all possible attacks.

If a refusal, missing tool, or other runtime problem prevents completion, use
`report_incomplete` with the reason and the completed/not-run check counts.
Never count an unexecuted check as passed or describe a content-filter refusal
as a firewall defect. Framework failure reporting handles runtime failures;
do not create a separate firewall-vulnerability issue.
