---
description: Smoke test to validate --allow-host-service-ports with Redis service container
on:
  workflow_dispatch:
  schedule: daily
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Service Ports
engine: copilot
strict: true
services:
  redis:
    image: redis:7
    ports:
      - 6379:6379
    options: >-
      --health-cmd "redis-cli ping"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
network:
  allowed:
    - defaults
    - github
tools:
  bash:
    - "*"
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 2
    messages:
      footer: "> 🔌 *Service ports validation by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
      run-started: "🔌 Starting service ports validation... [{workflow_name}]({run_url}) is testing Redis connectivity..."
      run-success: "✅ Service ports validation passed! [{workflow_name}]({run_url}) confirms agent can reach Redis."
      run-failure: "❌ Service ports validation failed! [{workflow_name}]({run_url}) could not reach Redis: {status}"
timeout-minutes: 5
---

# Smoke Test: Service Ports (Redis)

**Purpose:** Validate that the `--allow-host-service-ports` feature works end-to-end. The compiler should have automatically detected the Redis service port and configured AWF to allow traffic to it.

**IMPORTANT:** Inside AWF's sandbox, you must connect to services via `host.docker.internal` (not `localhost`). The service containers run on the host, and AWF routes traffic through the host gateway. Since the workflow maps port 6379:6379, port 6379 should work. Keep all outputs concise.

## Required Tests

1. **Redis PING**: Run `redis-cli -h host.docker.internal -p 6379 ping` or `echo PING | nc host.docker.internal 6379` and verify the response contains `PONG`.

2. **Redis SET/GET**: Write a value to Redis and read it back:
   - `redis-cli -h host.docker.internal -p 6379 SET smoke_test "service-ports-ok"`
   - `redis-cli -h host.docker.internal -p 6379 GET smoke_test`
   - Verify the returned value is `service-ports-ok`

3. **Redis INFO**: Run `redis-cli -h host.docker.internal -p 6379 INFO server | head -5` to verify we can query Redis server info.

## Output Requirements

Add a **concise comment** to the pull request (if triggered by PR) with:

- Each test with a pass/fail status
- Overall status: PASS or FAIL
- Note whether `redis-cli` was available or if `nc`/netcat was used as fallback

Example:
```
## Service Ports Smoke Test (Redis)

| Test | Status |
|------|--------|
| Redis PING | ✅ PONG received |
| Redis SET/GET | ✅ Value round-tripped |
| Redis INFO | ✅ Server info retrieved |

**Result:** 3/3 tests passed ✅
```

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
