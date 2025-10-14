---
engine:
  id: claude
  version: "0.5.0"
network:
  allowed:
    - api.anthropic.com
    - httpbin.org
tools:
  bash: [":*"]
  web-fetch: {}
---

# Test Containerized Agent Execution with Proxy

Test the containerized Claude execution with proxy-based network traffic control.

## Test Cases

1. **Allowed Domain Test**
   - Access httpbin.org (should succeed)
   - Verify response is received

2. **Blocked Domain Test**
   - Try to access example.com (should be blocked by proxy)
   - Verify access is denied

## Tasks

Please run these tests:

1. Use web-fetch to access http://httpbin.org/get - this should work
2. Use web-fetch to access http://example.com - this should be blocked
3. Report the results of both attempts
