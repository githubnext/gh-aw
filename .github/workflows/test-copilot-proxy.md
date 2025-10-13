---
engine:
  id: copilot
  version: "1.2.3"
network:
  allowed:
    - "api.githubcopilot.com"
    - "httpbin.org"
tools:
  bash: [":*"]
---

# Test Copilot with Proxy

Test the containerized Copilot execution with proxy-based network traffic control.

Please run these tests:

1. Access httpbin.org (should work)
2. Try to access example.com (should be blocked)
3. Report the results
