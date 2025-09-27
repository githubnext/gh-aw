---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
engine: copilot
safe-outputs:
    staged: true
    edit-wiki:
      path: ["dev/", "docs/"]
      max: 3
---
Write a poem and add it to the wiki.