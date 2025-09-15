---
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
safe-outputs:
  missing-tool:
  staged: true
engine: 
  id: claude
  max-turns: 5
tools:
  cache-memory: true
  playwright:
    allowed_domains: ["github.com", "*.github.com"]
permissions: read-all
concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.ref }}"
---

Before starting, read the entire memory graph and print it to the output as "My past poems..."

Then:

Write a short poem.
- check if this poem is already in memory
- if already in memory, generate a new poem

Before returning the poem:
- store generated poem in memory

<!-- This workflow tests the integration with the Claude AI engine. 
  Meant as a scratchpad in pull requests. -->