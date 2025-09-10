---
name: Test Codex Repository Security Advisory
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  create-repository-security-advisory:
    max: 10
---

# Test Codex Create Repository Security Advisory

Create a new repository security advisory for the repository with title "Codex wants security review." and adding a couple of sentences about why security is important.