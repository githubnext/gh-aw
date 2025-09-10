---
name: Test Codex Security Report
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: codex

safe-outputs:
  create-security-report:
    max: 10
---

# Test Codex Create Security Report

Create a new security report for the repository with title "Codex wants security review." and adding a couple of sentences about why security is important.