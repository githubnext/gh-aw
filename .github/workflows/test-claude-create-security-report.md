---
name: Test Claude Security Report
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  create-security-report:
    max: 10
---

# Test Claude Create Security Report

Create a new security report for the repository with title "Claude wants security review." and adding a couple of sentences about why security is important.