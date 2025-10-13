---
engine:
  id: claude
  version: "0.15.1"
network:
  allowed: []
tools:
  bash: [":*"]
---

# Test Explicit Deny-All Firewall

Test that the firewall enforces deny-all when explicitly configured with empty allowed list.

Please try to access any external domain (this should fail):
1. Try to access example.com
2. Try to access google.com
3. Report the results
