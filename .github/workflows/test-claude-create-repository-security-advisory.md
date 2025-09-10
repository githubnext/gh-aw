---
name: Test Claude Repository Security Advisory
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  create-repository-security-advisory:
    max: 10
---

# Test Claude Create Repository Security Advisory

Create a new repository security advisory for the repository with title "Claude wants security review." and adding a couple of sentences about why security is important.