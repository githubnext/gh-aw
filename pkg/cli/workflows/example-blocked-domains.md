---
engine: copilot
on:
  workflow_dispatch:
    
network:
  firewall: true
  allowed:
    - defaults
    - github
    - node
  blocked:
    - tracker.example.com
    - analytics.example.com
---

# Example: Blocked Domains

This workflow demonstrates using the `blocked` field in network configuration to block specific domains while allowing others.

The workflow allows access to:
- Basic infrastructure (`defaults`)
- GitHub domains (`github`)
- Node.js/NPM ecosystem (`node`)

But explicitly blocks:
- `tracker.example.com` (tracking domain)
- `analytics.example.com` (analytics domain)

Blocked domains take precedence over allowed domains, providing fine-grained control over network access.
